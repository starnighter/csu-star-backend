package utils

import (
	"csu-star-backend/config"
	"errors"
	"strings"
	"testing"
)

func TestRenderVerificationEmailHTMLInjectsCaptcha(t *testing.T) {
	body := renderVerificationEmailHTML("123456")

	if !strings.Contains(body, "123456") {
		t.Fatal("expected rendered body to include captcha")
	}
	if strings.Contains(body, "{{code}}") {
		t.Fatal("expected template placeholder to be replaced")
	}
}

func TestSendVerificationEmailFallsBackInOrder(t *testing.T) {
	originalCfg := config.GlobalConfig
	originalFn := sendVerificationEmailWithFallbackFn
	originalTencent := tencentVerificationEmailSender
	originalSMTP := smtpVerificationEmailSender
	t.Cleanup(func() {
		config.GlobalConfig = originalCfg
		sendVerificationEmailWithFallbackFn = originalFn
		tencentVerificationEmailSender = originalTencent
		smtpVerificationEmailSender = originalSMTP
	})

	config.GlobalConfig = &config.Config{
		Tencent: config.TencentConfig{
			Ses: config.SesConfig{
				FromEmailAddr: "noreply@csustar.wiki",
				Subject:       "CSU Star | 南极星邮箱验证码",
			},
		},
		Mail: config.MailConfig{
			Verification: config.VerificationMailConfig{
				Subject: "CSU Star | 南极星邮箱验证码",
				Aliyun: config.SMTPConfig{
					Host:          "smtpdm.aliyun.com",
					Port:          465,
					Username:      "noreply@csustar.wiki",
					Password:      "secret",
					FromEmailAddr: "noreply@csustar.wiki",
					FromName:      "CSU Star",
				},
				QQ: config.SMTPConfig{
					Host:          "smtp.qq.com",
					Port:          465,
					Username:      "csustar@foxmail.com",
					Password:      "secret",
					FromEmailAddr: "csustar@foxmail.com",
					FromName:      "CSU Star",
				},
			},
		},
	}

	var attempts []string
	tencentVerificationEmailSender = func(from string, to []string, captcha string) error {
		attempts = append(attempts, "tencent")
		return errors.New("tencent failed")
	}
	smtpVerificationEmailSender = func(cfg config.SMTPConfig, to []string, captcha string) error {
		attempts = append(attempts, cfg.Host)
		if cfg.Host == "smtpdm.aliyun.com" {
			return nil
		}
		return errors.New("unexpected provider call")
	}

	err := sendVerificationEmailWithFallback([]string{"test@csu.edu.cn"}, "123456")
	if err != nil {
		t.Fatalf("expected fallback success, got %v", err)
	}

	got := strings.Join(attempts, ",")
	if got != "tencent,smtpdm.aliyun.com" {
		t.Fatalf("expected ordered fallback attempts, got %s", got)
	}
}

func TestBuildHTMLMessageUsesSubjectAndHTMLHeaders(t *testing.T) {
	message := string(buildHTMLMessage("CSU Star", "noreply@csustar.wiki", []string{"test@csu.edu.cn"}, "CSU Star | 南极星邮箱验证码", "<b>123456</b>"))

	for _, want := range []string{
		"From: CSU Star <noreply@csustar.wiki>",
		"To: test@csu.edu.cn",
		"Subject: CSU Star | 南极星邮箱验证码",
		"Content-Type: text/html; charset=UTF-8",
		"<b>123456</b>",
	} {
		if !strings.Contains(message, want) {
			t.Fatalf("expected message to contain %q", want)
		}
	}
}

func TestSendVerificationEmailReturnsJoinedErrorWhenAllProvidersFail(t *testing.T) {
	originalCfg := config.GlobalConfig
	originalTencent := tencentVerificationEmailSender
	originalSMTP := smtpVerificationEmailSender
	t.Cleanup(func() {
		config.GlobalConfig = originalCfg
		tencentVerificationEmailSender = originalTencent
		smtpVerificationEmailSender = originalSMTP
	})

	config.GlobalConfig = &config.Config{
		Tencent: config.TencentConfig{
			Ses: config.SesConfig{FromEmailAddr: "noreply@csustar.wiki"},
		},
		Mail: config.MailConfig{
			Verification: config.VerificationMailConfig{
				Aliyun: config.SMTPConfig{
					Host:          "smtpdm.aliyun.com",
					Port:          465,
					Username:      "noreply@csustar.wiki",
					Password:      "secret",
					FromEmailAddr: "noreply@csustar.wiki",
				},
				QQ: config.SMTPConfig{
					Host:          "smtp.qq.com",
					Port:          465,
					Username:      "csustar@foxmail.com",
					Password:      "secret",
					FromEmailAddr: "csustar@foxmail.com",
				},
			},
		},
	}

	tencentVerificationEmailSender = func(from string, to []string, captcha string) error {
		return errors.New("tencent failed")
	}
	smtpVerificationEmailSender = func(cfg config.SMTPConfig, to []string, captcha string) error {
		return errors.New(cfg.Host + " failed")
	}

	err := sendVerificationEmailWithFallback([]string{"test@csu.edu.cn"}, "123456")
	if err == nil {
		t.Fatal("expected joined error when all providers fail")
	}

	for _, want := range []string{"tencent_ses", "aliyun_directmail_smtp", "qq_smtp"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected joined error to contain %q, got %v", want, err)
		}
	}
}

func TestHasVerificationEmailFallbackProvider(t *testing.T) {
	originalCfg := config.GlobalConfig
	t.Cleanup(func() {
		config.GlobalConfig = originalCfg
	})

	config.GlobalConfig = &config.Config{
		Mail: config.MailConfig{
			Verification: config.VerificationMailConfig{},
		},
	}
	if HasVerificationEmailFallbackProvider() {
		t.Fatal("expected false when no smtp provider is configured")
	}

	config.GlobalConfig.Mail.Verification.QQ = config.SMTPConfig{
		Host:          "smtp.qq.com",
		Port:          465,
		Username:      "csustar@foxmail.com",
		Password:      "secret",
		FromEmailAddr: "csustar@foxmail.com",
	}
	if !HasVerificationEmailFallbackProvider() {
		t.Fatal("expected true when one smtp provider is configured")
	}
}
