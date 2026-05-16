package service

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"csu-star-backend/config"
	"csu-star-backend/internal/constant"
	"csu-star-backend/internal/model"
	"csu-star-backend/internal/repo"
	"csu-star-backend/logger"
	"csu-star-backend/pkg/utils"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type EmailRegisterService struct {
	userRepo repo.UserRepository
}

func NewEmailRegisterService(ur repo.UserRepository) *EmailRegisterService {
	return &EmailRegisterService{userRepo: ur}
}

// ProcessRegistrationEmail processes a single email message for registration.
// The password is extracted from the email subject only.
// Returns (registered bool, err error).
func (s *EmailRegisterService) ProcessRegistrationEmail(senderEmail, subject, body string) (bool, error) {
	// 1. Normalize and validate sender
	sender := strings.TrimSpace(strings.ToLower(senderEmail))
	if !strings.HasSuffix(sender, constant.SchoolEmailSuffix) {
		logger.Log.Info("忽略非校域邮箱", zap.String("sender", sender))
		return false, nil
	}

	// 2. Extract password from subject
	password := strings.TrimSpace(subject)

	if password == "" {
		logger.Log.Warn("邮件主题为空", zap.String("sender", sender))
		if replyErr := utils.SendRegistrationEmptySubjectReplyEmail(sender); replyErr != nil {
			logger.Log.Error("发送校验失败回复邮件失败", zap.String("sender", sender), zap.Error(replyErr))
		}
		return false, nil
	}

	// 3. Validate password format
	cfg := config.GetConfig().Mail.EmailRegister
	if reason, ok := validateEmailPassword(password, cfg); !ok {
		logger.Log.Warn("密码校验不通过", zap.String("sender", sender), zap.String("reason", reason))
		if replyErr := utils.SendRegistrationInvalidPasswordReplyEmail(sender, reason, cfg.MinPasswordLen, cfg.MaxPasswordLen); replyErr != nil {
			logger.Log.Error("发送密码校验失败回复邮件失败", zap.String("sender", sender), zap.Error(replyErr))
		}
		return false, nil
	}

	// 4. Rate limit by sender email (max 3 per day, atomic INCR + EXPIRE)
	rateLimitKey := constant.EmailRegisterRateLimitPrefix + sender
	count, err := incrWithExpiry(rateLimitKey, 24*time.Hour)
	if err != nil {
		return false, err
	}
	if count > 3 {
		logger.Log.Warn("邮箱注册频率超限", zap.String("sender", sender))
		return false, nil
	}

	// 5. Check if user already exists
	existingUser, err := s.userRepo.FindUserByEmail(sender)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return false, err
	}
	if existingUser != nil {
		// Send "already exists" reply
		if replyErr := utils.SendRegistrationReplyEmail(sender, false); replyErr != nil {
			logger.Log.Error("发送'已注册'回复邮件失败", zap.String("sender", sender), zap.Error(replyErr))
		}
		return false, nil
	}

	// 6. Hash password: SHA256 first (matching frontend behavior), then bcrypt
	sha256Hash := sha256.Sum256([]byte(password))
	sha256Hex := hex.EncodeToString(sha256Hash[:])
	bcryptHash, err := utils.HashPassword(sha256Hex)
	if err != nil {
		return false, err
	}

	// 7. Generate random nickname
	nickname, err := utils.GenerateNickname()
	if err != nil {
		return false, err
	}

	// 8. Create user
	user := &model.Users{
		Email:         &sender,
		Password:      bcryptHash,
		Nickname:      nickname,
		EmailVerified: true,
	}
	if err := s.userRepo.CreateUser(user); err != nil {
		return false, err
	}

	logger.Log.Info("邮箱注册成功", zap.String("email", sender), zap.Int64("user_id", user.ID))

	// 9. Send success reply
	if replyErr := utils.SendRegistrationReplyEmail(sender, true); replyErr != nil {
		logger.Log.Error("发送注册成功回复邮件失败", zap.String("sender", sender), zap.Error(replyErr))
		// Don't return error -- user was already created successfully
	}

	return true, nil
}

// incrWithExpiry atomically increments a Redis key and sets expiry on first increment.
func incrWithExpiry(key string, ttl time.Duration) (int64, error) {
	pipe := utils.RDB.Pipeline()
	incrCmd := pipe.Incr(utils.Ctx, key)
	pipe.Expire(utils.Ctx, key, ttl)
	if _, err := pipe.Exec(utils.Ctx); err != nil {
		return 0, err
	}
	return incrCmd.Val(), nil
}

// validateEmailPassword checks the password meets all requirements.
// Returns (reason, false) if invalid, ("", true) if valid.
func validateEmailPassword(password string, cfg config.EmailRegisterConfig) (string, bool) {
	if cfg.MinPasswordLen > 0 && len(password) < cfg.MinPasswordLen {
		return "密码长度过短", false
	}
	if cfg.MaxPasswordLen > 0 && len(password) > cfg.MaxPasswordLen {
		return "密码长度过长", false
	}
	if strings.ContainsAny(password, " \t\n\r") {
		return "密码中不能包含空格或换行符", false
	}
	return "", true
}
