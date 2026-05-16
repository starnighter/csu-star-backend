package utils

import (
	"csu-star-backend/config"
	"csu-star-backend/logger"
	"errors"
	"fmt"
	"net/smtp"
	"strings"
	"sync/atomic"

	"go.uber.org/zap"
)

const replyEmailSubject = "CSU Star | 注册结果通知"

const registrationSuccessBody = `尊敬的用户，

恭喜您！您的 CSU Star 账号已注册成功。

邮箱：%s
密码：已安全存储（请妥善保管）

您可以前往 https://csustar.wiki/login 使用邮箱登录。

如有问题，请联系管理员。

CSU Star · 南极星Team`

const registrationAlreadyExistsBody = `尊敬的用户，

您的邮箱 %s 已经注册过 CSU Star 账号，请直接登录。

登录地址：https://csustar.wiki/login

如忘记密码，请使用"忘记密码"功能重置。

CSU Star · 南极星Team`

const registrationInvalidPasswordBody = `尊敬的用户，

您的注册请求未通过密码校验，原因：%s

密码要求：
- 长度为 %d~%d 个字符
- 不能包含空格或换行

请修正后重新发送邮件至本邮箱进行注册。

CSU Star · 南极星Team`

const registrationMismatchBody = `尊敬的用户，

您的注册请求未通过校验，原因：邮件主题与正文中的密码不一致。

请确保邮件主题和正文填写相同的密码，然后重新发送邮件至本邮箱进行注册。

CSU Star · 南极星Team`

var replyEmailProviderCursor atomic.Uint64

func SendRegistrationReplyEmail(to string, success bool) error {
	providers := replySMTPProviders()
	if len(providers) == 0 {
		return errors.New("no SMTP providers available for reply emails")
	}

	var body string
	if success {
		body = fmt.Sprintf(registrationSuccessBody, to)
	} else {
		body = fmt.Sprintf(registrationAlreadyExistsBody, to)
	}

	start := int(replyEmailProviderCursor.Add(1)-1) % len(providers)

	var errs []error
	for offset := range providers {
		provider := providers[(start+offset)%len(providers)]
		if err := sendReplyEmailViaSMTP(provider, to, body); err != nil {
			logger.Log.Warn(
				"回复邮件发送失败，尝试下一个SMTP通道",
				zap.String("provider", providerDisplayName(provider)),
				zap.Error(err),
			)
			errs = append(errs, fmt.Errorf("%s: %w", providerDisplayName(provider), err))
			continue
		}

		logger.Log.Info(
			"回复邮件发送成功",
			zap.String("provider", providerDisplayName(provider)),
			zap.String("to", to),
			zap.Bool("success", success),
		)
		return nil
	}

	return errors.Join(errs...)
}

// SendRegistrationInvalidPasswordReplyEmail sends a reply explaining why the password was rejected.
func SendRegistrationInvalidPasswordReplyEmail(to, reason string, minLen, maxLen int) error {
	providers := replySMTPProviders()
	if len(providers) == 0 {
		return errors.New("no SMTP providers available for reply emails")
	}

	body := fmt.Sprintf(registrationInvalidPasswordBody, reason, minLen, maxLen)

	start := int(replyEmailProviderCursor.Add(1)-1) % len(providers)

	var errs []error
	for offset := range providers {
		provider := providers[(start+offset)%len(providers)]
		if err := sendReplyEmailViaSMTP(provider, to, body); err != nil {
			logger.Log.Warn(
				"密码校验回复邮件发送失败，尝试下一个SMTP通道",
				zap.String("provider", providerDisplayName(provider)),
				zap.Error(err),
			)
			errs = append(errs, fmt.Errorf("%s: %w", providerDisplayName(provider), err))
			continue
		}

		logger.Log.Info(
			"密码校验回复邮件发送成功",
			zap.String("provider", providerDisplayName(provider)),
			zap.String("to", to),
		)
		return nil
	}

	return errors.Join(errs...)
}

// SendRegistrationMismatchReplyEmail sends a reply when subject and body passwords don't match.
func SendRegistrationMismatchReplyEmail(to string) error {
	return sendSimpleReply(to, registrationMismatchBody)
}

func sendSimpleReply(to, body string) error {
	providers := replySMTPProviders()
	if len(providers) == 0 {
		return errors.New("no SMTP providers available for reply emails")
	}

	start := int(replyEmailProviderCursor.Add(1)-1) % len(providers)

	var errs []error
	for offset := range providers {
		provider := providers[(start+offset)%len(providers)]
		if err := sendReplyEmailViaSMTP(provider, to, body); err != nil {
			logger.Log.Warn(
				"回复邮件发送失败，尝试下一个SMTP通道",
				zap.String("provider", providerDisplayName(provider)),
				zap.Error(err),
			)
			errs = append(errs, fmt.Errorf("%s: %w", providerDisplayName(provider), err))
			continue
		}

		logger.Log.Info(
			"回复邮件发送成功",
			zap.String("provider", providerDisplayName(provider)),
			zap.String("to", to),
		)
		return nil
	}

	return errors.Join(errs...)
}

func sendReplyEmailViaSMTP(cfg config.SMTPConfig, to string, body string) error {
	if !isCompleteSMTPConfig(cfg) {
		return errors.New("smtp config is incomplete")
	}

	subject := replyEmailSubject
	message := buildPlainTextMessage(cfg.FromName, cfg.FromEmailAddr, []string{to}, subject, body)
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
	return sendMailUsingTLS(addr, auth, cfg.FromEmailAddr, []string{to}, message)
}

func buildPlainTextMessage(fromName, fromEmail string, to []string, subject, body string) []byte {
	headers := map[string]string{
		"From":              formatMailAddress(fromName, fromEmail),
		"To":                strings.Join(to, ","),
		"Subject":           subject,
		"MIME-Version":      "1.0",
		"Content-Type":      "text/plain; charset=UTF-8",
		replyHeaderKey:      "true",
	}

	var builder strings.Builder
	for _, key := range []string{"From", "To", "Subject", "MIME-Version", "Content-Type", replyHeaderKey} {
		builder.WriteString(fmt.Sprintf("%s: %s\r\n", key, headers[key]))
	}
	builder.WriteString("\r\n")
	builder.WriteString(body)
	return []byte(builder.String())
}

// replySMTPProviders returns all complete SMTP providers except the IMAP mailbox.
func replySMTPProviders() []config.SMTPConfig {
	if config.GetConfig() == nil {
		return nil
	}

	imapUser := strings.TrimSpace(strings.ToLower(config.GetConfig().Mail.Imap.Username))

	providers := make([]config.SMTPConfig, 0, len(config.GetConfig().Mail.Verification.Providers))
	for _, provider := range config.GetConfig().Mail.Verification.Providers {
		if !isCompleteSMTPConfig(provider) {
			continue
		}
		// Exclude the IMAP mailbox from reply senders
		if imapUser != "" && strings.EqualFold(provider.Username, imapUser) {
			continue
		}
		providers = append(providers, provider)
	}
	return providers
}
