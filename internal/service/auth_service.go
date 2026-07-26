package service

import (
	"crypto/md5"
	"crypto/sha256"
	"csu-star-backend/config"
	"csu-star-backend/internal/constant"
	"csu-star-backend/internal/emailpolicy"
	"csu-star-backend/internal/model"
	"csu-star-backend/internal/repo"
	"csu-star-backend/logger"
	"csu-star-backend/pkg/mailer"
	"csu-star-backend/pkg/utils"
	"encoding/hex"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AuthService struct {
	userRepo       repo.UserRepository
	invitationRepo repo.InvitationRepository
	securitySvc    *SecurityService
	miscSvc        *MiscService
}

func NewAuthService(ur repo.UserRepository, ir repo.InvitationRepository) *AuthService {
	return &AuthService{userRepo: ur, invitationRepo: ir}
}

func (s *AuthService) SetSecurityService(securitySvc *SecurityService) {
	s.securitySvc = securitySvc
}

func (s *AuthService) SetMiscService(miscSvc *MiscService) {
	s.miscSvc = miscSvc
}

func (s *AuthService) SendCaptcha(email string, purpose string) (string, error) {
	normalizedEmail, err := emailpolicy.Normalize(email)
	if err != nil {
		return "", err
	}

	switch purpose {
	case "register":
		// 只有会创建账号的路径才校验域名策略；登录/找回密码不校验，
		// 否则将来收紧白名单会把存量用户锁在自己的数据外面。
		if !emailpolicy.DomainAllowed(emailpolicy.Domain(normalizedEmail)) {
			return "", &constant.EmailDomainNotAllowedErr
		}
		userByEmail, err := s.userRepo.FindUserByEmail(normalizedEmail)
		if userByEmail != nil {
			return "", &constant.UserHasRegisteredErr
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return "", err
		}
	case "bind_email":
		// 绑定邮箱同样会写入账号，必须与 register 一样走域名策略；
		// 否则 allow_list 下用户会先收到验证码，再在 BindEmail 时失败。
		if !emailpolicy.DomainAllowed(emailpolicy.Domain(normalizedEmail)) {
			return "", &constant.EmailDomainNotAllowedErr
		}
	case "forget_password", "reset_password":
		// reset_password 与 forget_password 同义：都要求账号已存在，
		// 避免对任意地址滥发验证码（发件通道滥用）。
		userByEmail, err := s.userRepo.FindUserByEmail(normalizedEmail)
		if errors.Is(err, gorm.ErrRecordNotFound) || userByEmail == nil {
			return "", &constant.UserNotExistErr
		}
		if err != nil {
			return "", err
		}
	}

	// 检查是否在60s内重复调用
	id := captchaID(normalizedEmail)
	result, err := utils.RDB.Get(utils.Ctx, constant.CaptchaRepeatPrefix+id).Result()
	if errors.Is(err, redis.Nil) {
		result = ""
	} else if err != nil {
		return "", err
	}
	if result != "" {
		return "", &constant.SendCaptchaRepeatedlyIn60sErr
	}

	// 发送验证码邮件，统一走全局 SMTP provider 池轮询。
	to := []string{normalizedEmail}
	captcha, err := utils.GenerateCaptcha(6)
	if err != nil {
		return "", err
	}
	err = mailer.SendVerificationEmail(to, captcha)
	if err != nil {
		return "", err
	}

	// 存入redis防止60s内重复访问并供后续校验
	if err = utils.RDB.Set(utils.Ctx, constant.CaptchaRepeatPrefix+id, captcha, 60*time.Second).Err(); err != nil {
		return "", err
	}
	// 存验证码，有效期与邮件正文里写的分钟数同源
	if err = utils.RDB.Set(utils.Ctx, constant.CaptchaPrefix+id, captcha, CaptchaTTL()).Err(); err != nil {
		return "", err
	}

	return "验证码发送成功，请注意查收", nil
}

func (s *AuthService) VerifyCaptcha(email string, captcha string) error {
	normalizedEmail, err := emailpolicy.Normalize(email)
	if err != nil {
		return err
	}

	captchaKey := constant.CaptchaPrefix + captchaID(normalizedEmail)
	result, err := utils.RDB.Get(utils.Ctx, captchaKey).Result()
	if errors.Is(err, redis.Nil) {
		return &constant.CaptchaNotMatchErr
	}
	if err != nil {
		return err
	}

	if result == captcha {
		// 先写「已校验」证明再删验证码，供 Register 两步流程消费。
		// 有效期比验证码本身更长，用户补全昵称/邀请码不至于过期。
		verifiedKey := constant.CaptchaVerifiedPrefix + captchaID(normalizedEmail)
		if err = utils.RDB.Set(utils.Ctx, verifiedKey, "1", captchaVerifiedTTL()).Err(); err != nil {
			return err
		}
		if err = utils.RDB.Del(utils.Ctx, captchaKey).Err(); err != nil {
			return err
		}
		return nil
	}

	return &constant.CaptchaNotMatchErr
}

// skipEmailCaptchaVerifiedCheck 仅单测使用：现有 Register 单元测试不启 Redis。
// 生产路径永远走 Redis 证明；切勿在业务代码里置 true。
var skipEmailCaptchaVerifiedCheck bool

func captchaVerifiedTTL() time.Duration {
	return 30 * time.Minute
}

// consumeEmailCaptchaVerified 消费 /verify 写入的邮箱持有证明。
// 成功则删除 key（一次性），失败表示未校验或已过期。
func (s *AuthService) consumeEmailCaptchaVerified(normalizedEmail string) error {
	if skipEmailCaptchaVerifiedCheck {
		return nil
	}
	if utils.RDB == nil {
		return &constant.EmailCaptchaRequiredErr
	}
	key := constant.CaptchaVerifiedPrefix + captchaID(normalizedEmail)
	_, err := utils.RDB.GetDel(utils.Ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return &constant.EmailCaptchaRequiredErr
	}
	if err != nil {
		return err
	}
	return nil
}

func (s *AuthService) Register(email, password, nickName, avatarUrl, inviteCode string) error {
	normalizedEmail, err := emailpolicy.Check(email)
	if err != nil {
		return err
	}

	// 必须先经 /auth/email/verify；否则 allow_all 下可直接调 register 抢注任意邮箱。
	if err := s.consumeEmailCaptchaVerified(normalizedEmail); err != nil {
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
		if s.miscSvc != nil {
			_ = s.miscSvc.RefreshAfterContribution(user.ID)
			_ = s.miscSvc.RefreshAfterContribution(consumedInviterID)
		}
	}

	return nil
}

func (s *AuthService) Login(email, password string) (*model.Users, string, string, error) {
	// 登录只做格式规范化，不查域名策略：收紧白名单不应影响存量用户登录。
	normalizedEmail, err := emailpolicy.Normalize(email)
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
	normalizedEmail, err := emailpolicy.Check(email)
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
	// 同 Login：找回密码不查域名策略，否则用户可能连自己的账号都取不回。
	normalizedEmail, err := emailpolicy.Normalize(email)
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

// captchaID 从已规范化的邮箱派生定长、非 PII 的 Redis key 片段。
//
// 取代旧的「学号」方案（邮箱去掉 @csu.edu.cn）：域名放开后那个方案会退化——
// 非校园邮箱不匹配后缀，TrimSuffix 原样返回完整邮箱，key 变得长度不定、
// 含分隔符，且 12345@qq.com 与 12345@csu.edu.cn 的行为不一致。
func captchaID(normalizedEmail string) string {
	sum := sha256.Sum256([]byte(normalizedEmail))
	return hex.EncodeToString(sum[:16])
}

// CaptchaTTL 是验证码在 Redis 里的有效期，与验证码邮件正文里写的分钟数同源，
// 避免两边各写各的再次漂移（历史上邮件写 5 分钟、实际存 10 分钟）。
func CaptchaTTL() time.Duration {
	const fallback = 10 * time.Minute
	cfg := config.GetConfig()
	if cfg == nil {
		return fallback
	}
	if minutes := cfg.Mail.Verification.CodeTTLMinutes; minutes > 0 {
		return time.Duration(minutes) * time.Minute
	}
	return fallback
}
