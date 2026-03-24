package service

import (
	"crypto/md5"
	"csu-star-backend/config"
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
}

func NewAuthService(ur repo.UserRepository, ir repo.InvitationRepository) *AuthService {
	return &AuthService{userRepo: ur, invitationRepo: ir}
}

func (s *AuthService) SendCaptcha(email string) error {
	// 检查是否在60s内重复调用
	stuNumber := GetStuNumberByEmail(email)
	result, err := utils.RDB.Get(utils.Ctx, constant.CaptchaPrefix+stuNumber).Result()
	if errors.Is(err, redis.Nil) {
		result = ""
	} else if err != nil {
		return err
	}
	if result != "" {
		return &constant.SendCaptchaRepeatedlyIn60sErr
	}

	// 调用腾讯云SES SDK发送验证码到指定邮箱
	to := []string{email}
	captcha, err := utils.GenerateCaptcha(6)
	if err != nil {
		return err
	}
	err = utils.TencentSesSendEmail(config.GlobalConfig.Tencent.Ses.FromEmailAddr, to, captcha)
	if err != nil {
		return err
	}
	// 存入redis防止60s内重复访问并供后续校验
	if err = utils.RDB.Set(utils.Ctx, constant.CaptchaPrefix+stuNumber, captcha, 60*time.Second).Err(); err != nil {
		return err
	}

	return nil
}

func (s *AuthService) VerifyCaptcha(email string, captcha string) error {
	stuNumber := GetStuNumberByEmail(email)
	result, err := utils.RDB.GetDel(utils.Ctx, constant.CaptchaPrefix+stuNumber).Result()
	if errors.Is(err, redis.Nil) {
		return &constant.CaptchaNotMatchErr
	}
	if err != nil {
		return err
	}

	if result == captcha {
		return nil
	}

	return &constant.CaptchaNotMatchErr
}

func (s *AuthService) Register(email, password, nickName, avatarUrl, inviteCode string) error {
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

	var inviterID int64
	if inviteCode != "" {
		inviterID, err = s.invitationRepo.FindInviterIDByCode(inviteCode)
		if err != nil {
			return err
		}
	}

	user := &model.Users{
		Email:         email,
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
		if consumedInviterID != inviterID {
			logger.Log.Error("邀请码邀请人不一致", zap.Int64("expected_inviter_id", inviterID), zap.Int64("actual_inviter_id", consumedInviterID))
			return &constant.InviteCodeNotExistErr
		}
		if err := s.userRepo.RewardInviter(consumedInviterID); err != nil {
			logger.Log.Error("使用邀请码创建新用户后，发放邀请奖励失败：", zap.Error(err))
			return err
		}
	}

	return nil
}

func (s *AuthService) Login(email, password string) (*model.Users, string, string, error) {
	user, err := s.userRepo.FindUserByEmail(email)
	if user == nil {
		return user, "", "", &constant.UserNotExistErr
	}
	if err != nil {
		return user, "", "", err
	}
	if user.Status == model.UserStatusBanned {
		return user, "", "", &constant.UserBannedErr
	}
	if !utils.CheckPasswordHash(password, user.Password) {
		return user, "", "", &constant.PasswordIncorrectErr

	}

	accessToken, refreshToken, err := utils.GenerateTokenPair(user.ID, string(user.Role))

	return user, accessToken, refreshToken, err
}

func (s *AuthService) BindEmail(userID int64, email string) error {
	user, err := s.userRepo.FindUserByID(userID)
	if err != nil {
		return err
	}
	if user == nil {
		return &constant.UserNotExistErr
	}
	if user.Email != "" {
		return &constant.EmailIsExistErr
	}

	err = s.userRepo.UpdateEmailByID(userID, email)
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return &constant.EmailHasBeenBoundErr
	}
	if err != nil {
		return err
	}
	return nil
}

func (s *AuthService) Refresh(refreshToken string) (string, string, error) {
	claims, err := utils.ParseToken(refreshToken)
	if err != nil {
		return "", "", err
	}
	if claims.Type != "refresh" {
		return "", "", &constant.NotRefreshTokenErr
	}
	if claims.ExpiresAt == nil || claims.ExpiresAt.Time.Before(time.Now()) {
		return "", "", &constant.RefreshTokenExpiredErr
	}

	hash := md5.Sum([]byte(refreshToken))
	tokenHash := hex.EncodeToString(hash[:])
	isBlacklisted, err := utils.RDB.Get(utils.Ctx, constant.BlackListPrefix+tokenHash).Result()
	if !errors.Is(err, redis.Nil) && isBlacklisted != "" {
		return "", "", &constant.RefreshTokenExpiredErr
	}
	if err != nil && !errors.Is(err, redis.Nil) {
		return "", "", err
	}

	_, err = utils.RDB.Set(utils.Ctx, constant.BlackListPrefix+tokenHash, time.Now().UnixMilli(), 604800*time.Second).Result()
	if err != nil {
		return "", "", err
	}

	return utils.GenerateTokenPair(claims.UserID, claims.UserRole)
}

func (s *AuthService) Logout(tokenHash string) error {
	_, err := utils.RDB.Set(utils.Ctx, constant.BlackListPrefix+tokenHash, time.Now().UnixMilli(), 604800*time.Second).Result()
	if err != nil {
		return err
	}
	return nil
}

func (s *AuthService) ForgetPwd(email, captcha, password string) error {
	if err := s.VerifyCaptcha(email, captcha); err != nil {
		return err
	}

	user, err := s.userRepo.FindUserByEmail(email)
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
	return strings.TrimSuffix(email, constant.SchoolEmailSuffix)
}
