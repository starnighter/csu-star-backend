package service

import (
	"csu-star-backend/config"
	"csu-star-backend/internal/constant"
	"csu-star-backend/internal/model"
	"csu-star-backend/internal/repo"
	"csu-star-backend/logger"
	"csu-star-backend/pkg/utils"
	"errors"
	"strings"
	"time"

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
	if err != nil {
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
		inviterID, err = s.userRepo.FindInviterAndAddPoints(inviteCode)
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

	invitation := &model.Invitations{
		InviteeID: user.ID,
		Status:    model.InvitationStatusInvited,
		UsedAt:    time.Now(),
	}
	err = s.invitationRepo.UpdateInvitationByCode(invitation, inviteCode)
	if err != nil {
		logger.Log.Error("使用邀请码创建新用户后，更新邀请信息失败：", zap.Error(err))
		return err
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

func (s *AuthService) Refresh(userID int64, userRole, refreshToken, tokenHash string) (string, string, error) {
	remainingTime, err := utils.GetTokenRemainingTime(refreshToken)
	if err != nil {
		return "", "", err
	}
	if remainingTime <= 0 {
		return "", "", &constant.RefreshTokenExpiredErr
	}

	_, err = utils.RDB.Set(utils.Ctx, constant.BlackListPrefix+tokenHash, time.Now().UnixMilli(), 604800*time.Second).Result()
	if err != nil {
		return "", "", err
	}

	return utils.GenerateTokenPair(userID, userRole)
}

func (s *AuthService) Logout(tokenHash string) error {
	_, err := utils.RDB.Set(utils.Ctx, constant.BlackListPrefix+tokenHash, time.Now().UnixMilli(), 604800*time.Second).Result()
	if err != nil {
		return err
	}
	return nil
}

func (s *AuthService) ForgetPwd(email, password string) error {
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
