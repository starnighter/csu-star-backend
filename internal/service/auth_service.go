package service

import (
	"crypto/md5"
	"csu-star-backend/internal/constant"
	"csu-star-backend/internal/model"
	"csu-star-backend/internal/repo"
	"csu-star-backend/logger"
	"csu-star-backend/pkg/utils"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AuthService struct {
	userRepo       repo.UserRepository
	invitationRepo repo.InvitationRepository
	securitySvc    *SecurityService
}

var campusMailboxStatusChecker = utils.CheckCampusMailboxStatus

func NewAuthService(ur repo.UserRepository, ir repo.InvitationRepository) *AuthService {
	return &AuthService{userRepo: ur, invitationRepo: ir}
}

func (s *AuthService) SetSecurityService(securitySvc *SecurityService) {
	s.securitySvc = securitySvc
}

func (s *AuthService) SendCaptcha(email string, isNotExists bool) error {
	normalizedEmail, err := normalizeSchoolEmail(email)
	if err != nil {
		return err
	}

	if isNotExists {
		userByEmail, err := s.userRepo.FindUserByEmail(normalizedEmail)
		if userByEmail != nil {
			return &constant.UserHasRegisteredErr
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	}

	mailboxStatus, err := campusMailboxStatusChecker(normalizedEmail)
	if err != nil && logger.Log != nil {
		logger.Log.Warn("校园邮箱 SMTP 轻量握手校验失败", zap.String("email", normalizedEmail), zap.Error(err))
	}
	switch mailboxStatus {
	case utils.CampusMailboxStatusNotFound:
		return &constant.CampusMailboxNotFoundErr
	case utils.CampusMailboxStatusRetry:
		return &constant.CampusMailboxCheckRetryErr
	case utils.CampusMailboxStatusUnknown:
		if logger.Log != nil {
			logger.Log.Warn("校园邮箱 SMTP 轻量握手校验降级为仅发送验证码", zap.String("email", normalizedEmail))
		}
	}

	// 检查是否在60s内重复调用
	stuNumber := GetStuNumberByEmail(normalizedEmail)
	result, err := utils.RDB.Get(utils.Ctx, constant.CaptchaRepeatPrefix+stuNumber).Result()
	if errors.Is(err, redis.Nil) {
		result = ""
	} else if err != nil {
		return err
	}
	if result != "" {
		return &constant.SendCaptchaRepeatedlyIn60sErr
	}

	// 发送验证码邮件，统一走全局 SMTP provider 池轮询。
	to := []string{normalizedEmail}
	captcha, err := utils.GenerateCaptcha(6)
	if err != nil {
		return err
	}
	err = utils.SendVerificationEmail(to, captcha)
	if err != nil {
		return err
	}

	// 存入redis防止60s内重复访问并供后续校验
	if err = utils.RDB.Set(utils.Ctx, constant.CaptchaRepeatPrefix+stuNumber, captcha, 60*time.Second).Err(); err != nil {
		return err
	}
	// 存验证码，10min有效期
	if err = utils.RDB.Set(utils.Ctx, constant.CaptchaPrefix+stuNumber, captcha, 600*time.Second).Err(); err != nil {
		return err
	}

	return nil
}

func (s *AuthService) VerifyCaptcha(email string, captcha string) error {
	normalizedEmail, err := normalizeSchoolEmail(email)
	if err != nil {
		return err
	}

	stuNumber := GetStuNumberByEmail(normalizedEmail)
	captchaKey := constant.CaptchaPrefix + stuNumber
	result, err := utils.RDB.Get(utils.Ctx, captchaKey).Result()
	if errors.Is(err, redis.Nil) {
		return &constant.CaptchaNotMatchErr
	}
	if err != nil {
		return err
	}

	if result == captcha {
		if err = utils.RDB.Del(utils.Ctx, captchaKey).Err(); err != nil {
			return err
		}
		return nil
	}

	return &constant.CaptchaNotMatchErr
}

func (s *AuthService) Register(email, password, nickName, avatarUrl, inviteCode string) error {
	normalizedEmail, err := normalizeSchoolEmail(email)
	if err != nil {
		return err
	}

	userByEmail, err := s.userRepo.FindUserByEmail(normalizedEmail)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		userByEmail = nil
	} else if err != nil {
		return err
	}
	if userByEmail != nil {
		return &constant.UserHasRegisteredErr
	}

	hashPassword, err := utils.HashPassword(password)
	if err != nil {
		return err
	}

	if nickName == "" {
		nickName, err = utils.GenerateNickname()
		if err != nil {
			return err
		}
	}

	var inviterID *int64
	if inviteCode != "" {
		foundInviterID, findErr := s.invitationRepo.FindInviterIDByCode(inviteCode)
		if findErr != nil {
			err = findErr
			return err
		}
		inviterID = &foundInviterID
	}

	user := &model.Users{
		Email:         &normalizedEmail,
		Password:      hashPassword,
		Nickname:      nickName,
		AvatarUrl:     avatarUrl,
		EmailVerified: true,
		InviterID:     inviterID,
	}

	if err := s.userRepo.CreateUser(user); err != nil {
		logger.Log.Error("使用邮箱创建新用户失败：", zap.Error(err))
		return err
	}

	if inviteCode != "" {
		consumedInviterID, err := s.invitationRepo.ConsumeInvitation(inviteCode, user.ID)
		if err != nil {
			logger.Log.Error("使用邀请码创建新用户后，更新邀请信息失败：", zap.Error(err))
			return err
		}
		if inviterID == nil || consumedInviterID != *inviterID {
			logger.Log.Error("邀请码邀请人不一致", zap.Int64p("expected_inviter_id", inviterID), zap.Int64("actual_inviter_id", consumedInviterID))
			return &constant.InviteCodeNotExistErr
		}
		if err := s.userRepo.RewardInvitee(user.ID); err != nil {
			logger.Log.Error("使用邀请码创建新用户后，发放被邀请奖励失败：", zap.Error(err))
			return err
		}
		if err := s.userRepo.RewardInviter(consumedInviterID); err != nil {
			logger.Log.Error("使用邀请码创建新用户后，发放邀请奖励失败：", zap.Error(err))
			return err
		}
	}

	return nil
}

func (s *AuthService) Login(email, password string) (*model.Users, string, string, error) {
	normalizedEmail, err := normalizeSchoolEmail(email)
	if err != nil {
		return nil, "", "", err
	}

	user, err := s.userRepo.FindUserByEmail(normalizedEmail)
	if user == nil {
		return user, "", "", &constant.UserNotExistErr
	}
	if err != nil {
		return user, "", "", err
	}
	if s.securitySvc != nil {
		user, banDecision, accessErr := s.securitySvc.EnforceUserAccess(user.ID)
		if accessErr != nil {
			if errors.Is(accessErr, ErrSecurityUserBanned) {
				return user, "", "", &constant.UserBannedErr
			}
			return nil, "", "", accessErr
		}
		_ = banDecision
	} else if user.Status == model.UserStatusBanned {
		return user, "", "", &constant.UserBannedErr
	}
	if !utils.CheckPasswordHash(password, user.Password) {
		return user, "", "", &constant.PasswordIncorrectErr

	}

	accessToken, refreshToken, err := utils.GenerateTokenPair(user.ID, string(user.Role))

	return user, accessToken, refreshToken, err
}

func (s *AuthService) BindEmail(userID int64, email string) error {
	normalizedEmail, err := normalizeSchoolEmail(email)
	if err != nil {
		return err
	}

	user, err := s.userRepo.FindUserByID(userID)
	if err != nil {
		return err
	}
	if user == nil {
		return &constant.UserNotExistErr
	}
	if user.Email != nil && *user.Email != "" {
		return &constant.EmailIsExistErr
	}

	err = s.userRepo.UpdateEmailByID(userID, normalizedEmail)
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return &constant.EmailHasBeenBoundErr
	}
	if err != nil {
		return err
	}
	return nil
}

func (s *AuthService) Refresh(refreshToken string) (string, string, *BanDecision, error) {
	claims, err := utils.ParseToken(refreshToken)
	if err != nil {
		return "", "", nil, err
	}
	if claims.Type != "refresh" {
		return "", "", nil, &constant.NotRefreshTokenErr
	}
	if claims.ExpiresAt == nil || claims.ExpiresAt.Time.Before(time.Now()) {
		return "", "", nil, &constant.RefreshTokenExpiredErr
	}

	hash := md5.Sum([]byte(refreshToken))
	tokenHash := hex.EncodeToString(hash[:])
	isBlacklisted, err := utils.RDB.Get(utils.Ctx, constant.BlackListPrefix+tokenHash).Result()
	if !errors.Is(err, redis.Nil) && isBlacklisted != "" {
		return "", "", nil, &constant.RefreshTokenExpiredErr
	}
	if err != nil && !errors.Is(err, redis.Nil) {
		return "", "", nil, err
	}

	if s.securitySvc != nil {
		_, banDecision, accessErr := s.securitySvc.EnforceUserAccess(claims.UserID)
		if accessErr != nil {
			if errors.Is(accessErr, ErrSecurityUserBanned) {
				return "", "", banDecision, &constant.UserBannedErr
			}
			return "", "", nil, accessErr
		}
	}

	if s.securitySvc == nil {
		user, findErr := s.userRepo.FindUserByID(claims.UserID)
		if findErr != nil {
			return "", "", nil, findErr
		}
		if user.Status == model.UserStatusBanned {
			return "", "", BanDecisionFromUser(user), &constant.UserBannedErr
		}
	}

	_, err = utils.RDB.Set(utils.Ctx, constant.BlackListPrefix+tokenHash, time.Now().UnixMilli(), 604800*time.Second).Result()
	if err != nil {
		return "", "", nil, err
	}

	accessToken, nextRefreshToken, genErr := utils.GenerateTokenPair(claims.UserID, claims.UserRole)
	return accessToken, nextRefreshToken, nil, genErr
}

func (s *AuthService) Logout(tokenHash string) error {
	_, err := utils.RDB.Set(utils.Ctx, constant.BlackListPrefix+tokenHash, time.Now().UnixMilli(), 604800*time.Second).Result()
	if err != nil {
		return err
	}
	return nil
}

func (s *AuthService) ForgetPwd(email, captcha, password string) error {
	normalizedEmail, err := normalizeSchoolEmail(email)
	if err != nil {
		return err
	}

	if err := s.VerifyCaptcha(normalizedEmail, captcha); err != nil {
		return err
	}

	user, err := s.userRepo.FindUserByEmail(normalizedEmail)
	if user == nil {
		return &constant.UserNotExistErr
	}
	if err != nil {
		return err
	}

	hash, err := utils.HashPassword(password)
	if err != nil {
		return err
	}

	return s.userRepo.UpdatePasswordByID(user.ID, hash)
}

func GetStuNumberByEmail(email string) string {
	normalized := strings.TrimSpace(strings.ToLower(email))
	return strings.TrimSuffix(normalized, constant.SchoolEmailSuffix)
}

func normalizeSchoolEmail(email string) (string, error) {
	normalized := strings.TrimSpace(strings.ToLower(email))
	if !strings.HasSuffix(normalized, constant.SchoolEmailSuffix) {
		return "", &constant.InvalidSchoolEmailErr
	}
	return normalized, nil
}
