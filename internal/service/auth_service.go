package service

import (
	"csu-star-backend/config"
	"csu-star-backend/internal/constant"
	"csu-star-backend/internal/model"
	"csu-star-backend/internal/repo"
	"csu-star-backend/logger"
	"csu-star-backend/pkg/utils"
	"strings"
	"time"

	"go.uber.org/zap"
)

type AuthService struct {
	UserRepo       repo.UserRepository
	InvitationRepo repo.InvitationRepository
}

func NewAuthService(ur repo.UserRepository, ir repo.InvitationRepository) *AuthService {
	return &AuthService{UserRepo: ur, InvitationRepo: ir}
}

func (s *AuthService) SendCaptcha(email string) error {
	// 检查是否在60s内重复调用
	stuNumber := GetStuNumberByEmail(email)
	result, err := utils.RDB.Get(utils.Ctx, constant.CaptchaPrefix+stuNumber).Result()
	if err != nil {
		return err
	}
	if result != "" {
		err = &constant.SendCaptchaRepeatedlyIn60sErr
		return err
	}

	// 调用腾讯云SES SDK发送验证码到指定邮箱
	to := []string{email}
	captcha, err := utils.GenerateCaptcha(6)
	if err != nil {
		return err
	}
	err = utils.TencentSesSendEmail(config.GlobalConfig.Tencent.FromEmailAddr, to, captcha)
	if err != nil {
		return err
	}
	// 存入redis防止60s内重复访问并供后续校验
	if err = utils.RDB.Set(utils.Ctx, constant.CaptchaPrefix+stuNumber, captcha, 60*time.Second).Err(); err != nil {
		return err
	}

	return nil
}

func (s *AuthService) Register(email, password, nickName, avatarUrl, inviteCode string) error {
	hashPassword, err := utils.HashPassword(password)
	if err != nil {
		return err
	}

	if nickName == "" {
		nickName = utils.GenerateNickname()
	}

	var inviterID int64
	if inviteCode != "" {
		inviterID, err = s.UserRepo.FindInviterAndAddPoints(inviteCode)
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
		Metadata:      nil,
		InviterID:     inviterID,
	}

	if err := s.UserRepo.CreateUser(user); err != nil {
		logger.Log.Error("使用邮箱创建新用户失败：", zap.Error(err))
		return err
	}

	invitation := &model.Invitations{
		InviteeID: user.ID,
		Status:    model.InvitationStatusInvited,
		UsedAt:    time.Now(),
	}
	err = s.InvitationRepo.UpdateInvitationByCode(invitation, inviteCode)
	if err != nil {
		logger.Log.Error("使用邀请码创建新用户后，更新邀请信息失败：", zap.Error(err))
		return err
	}

	return nil
}

func GetStuNumberByEmail(email string) string {
	return strings.TrimSuffix(email, constant.SchoolEmailSuffix)
}
