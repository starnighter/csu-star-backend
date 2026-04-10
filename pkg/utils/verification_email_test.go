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
	originalSMTP := smtpVerificationEmailSender
	t.Cleanup(func() {
		config.GlobalConfig = originalCfg
		sendVerificationEmailWithFallbackFn = originalFn
		smtpVerificationEmailSender = originalSMTP
		verificationEmailProviderCursor.Store(0)
	})

	config.GlobalConfig = &config.Config{
		Mail: config.MailConfig{
			Verification: config.VerificationMailConfig{
				Subject: "CSU Star | 南极星邮箱验证码",
				Providers: []config.SMTPConfig{
					{
						Name:          "provider-a",
						Host:          "smtp.163.com",
						Port:          465,
						Username:      "a@example.com",
						Password:      "secret",
						FromEmailAddr: "a@example.com",
						FromName:      "CSU Star",
					},
					{
						Name:          "provider-b",
						Host:          "smtp.126.com",
						Port:          465,
						Username:      "b@example.com",
						Password:      "secret",
						FromEmailAddr: "b@example.com",
						FromName:      "CSU Star",
					},
				},
			},
		},
	}

	var attempts []string
	smtpVerificationEmailSender = func(cfg config.SMTPConfig, to []string, captcha string) error {
		attempts = append(attempts, cfg.Name)
		if cfg.Name == "provider-a" {
			return nil
		}
		return errors.New("unexpected provider call")
	}

	err := sendVerificationEmailWithFallback([]string{"test@csu.edu.cn"}, "123456")
	if err != nil {
		t.Fatalf("expected fallback success, got %v", err)
	}

	got := strings.Join(attempts, ",")
	if got != "provider-a" {
		t.Fatalf("expected first provider to be used, got %s", got)
	}
}

func TestSendVerificationEmailRoundRobinAcrossProviders(t *testing.T) {
	originalCfg := config.GlobalConfig
	originalSMTP := smtpVerificationEmailSender
	t.Cleanup(func() {
		config.GlobalConfig = originalCfg
		smtpVerificationEmailSender = originalSMTP
		verificationEmailProviderCursor.Store(0)
	})

	config.GlobalConfig = &config.Config{
		Mail: config.MailConfig{
			Verification: config.VerificationMailConfig{
				Providers: []config.SMTPConfig{
					{Name: "provider-a", Host: "smtp.163.com", Port: 465, Username: "a@example.com", Password: "secret", FromEmailAddr: "a@example.com"},
					{Name: "provider-b", Host: "smtp.126.com", Port: 465, Username: "b@example.com", Password: "secret", FromEmailAddr: "b@example.com"},
					{Name: "provider-c", Host: "smtp.qq.com", Port: 465, Username: "c@example.com", Password: "secret", FromEmailAddr: "c@example.com"},
				},
			},
		},
	}

	var starts []string
	smtpVerificationEmailSender = func(cfg config.SMTPConfig, to []string, captcha string) error {
		starts = append(starts, cfg.Name)
		return nil
	}

	for range 4 {
		if err := sendVerificationEmailWithFallback([]string{"test@csu.edu.cn"}, "123456"); err != nil {
			t.Fatalf("expected send success, got %v", err)
		}
	}

	got := strings.Join(starts, ",")
	if got != "provider-a,provider-b,provider-c,provider-a" {
		t.Fatalf("expected round robin order, got %s", got)
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
	originalSMTP := smtpVerificationEmailSender
	t.Cleanup(func() {
		config.GlobalConfig = originalCfg
		smtpVerificationEmailSender = originalSMTP
		verificationEmailProviderCursor.Store(0)
	})

	config.GlobalConfig = &config.Config{
		Mail: config.MailConfig{
			Verification: config.VerificationMailConfig{
				Providers: []config.SMTPConfig{
					{Name: "provider-a", Host: "smtp.163.com", Port: 465, Username: "a@example.com", Password: "secret", FromEmailAddr: "a@example.com"},
					{Name: "provider-b", Host: "smtp.126.com", Port: 465, Username: "b@example.com", Password: "secret", FromEmailAddr: "b@example.com"},
				},
			},
		},
	}

	smtpVerificationEmailSender = func(cfg config.SMTPConfig, to []string, captcha string) error {
		return errors.New(cfg.Name + " failed")
	}

	err := sendVerificationEmailWithFallback([]string{"test@csu.edu.cn"}, "123456")
	if err == nil {
		t.Fatal("expected joined error when all providers fail")
	}

	for _, want := range []string{"provider-a", "provider-b"} {
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

	config.GlobalConfig.Mail.Verification.Providers = []config.SMTPConfig{
		{
			Name:          "provider-a",
			Host:          "smtp.qq.com",
			Port:          465,
			Username:      "csustar@foxmail.com",
			Password:      "secret",
			FromEmailAddr: "csustar@foxmail.com",
		},
	}
	if !HasVerificationEmailFallbackProvider() {
		t.Fatal("expected true when one smtp provider is configured")
	}
}

func TestSendVerificationEmailFallsBackToNextProviderOnFailure(t *testing.T) {
	originalCfg := config.GlobalConfig
	originalSMTP := smtpVerificationEmailSender
	t.Cleanup(func() {
		config.GlobalConfig = originalCfg
		smtpVerificationEmailSender = originalSMTP
		verificationEmailProviderCursor.Store(0)
	})

	config.GlobalConfig = &config.Config{
		Mail: config.MailConfig{
			Verification: config.VerificationMailConfig{
				Providers: []config.SMTPConfig{
					{Name: "provider-a", Host: "smtp.163.com", Port: 465, Username: "a@example.com", Password: "secret", FromEmailAddr: "a@example.com"},
					{Name: "provider-b", Host: "smtp.126.com", Port: 465, Username: "b@example.com", Password: "secret", FromEmailAddr: "b@example.com"},
				},
			},
		},
	}

	var attempts []string
	smtpVerificationEmailSender = func(cfg config.SMTPConfig, to []string, captcha string) error {
		attempts = append(attempts, cfg.Name)
		if cfg.Name == "provider-a" {
			return errors.New("provider-a failed")
		}
		return nil
	}

	if err := sendVerificationEmailWithFallback([]string{"test@csu.edu.cn"}, "123456"); err != nil {
		t.Fatalf("expected fallback success, got %v", err)
	}

	got := strings.Join(attempts, ",")
	if got != "provider-a,provider-b" {
		t.Fatalf("expected fallback order, got %s", got)
	}
}

func TestSendVerificationEmailUsesDefaultSubjectWhenConfigBlank(t *testing.T) {
	originalCfg := config.GlobalConfig
	t.Cleanup(func() {
		config.GlobalConfig = originalCfg
	})

	config.GlobalConfig = &config.Config{
		Mail: config.MailConfig{
			Verification: config.VerificationMailConfig{},
		},
	}

	message := string(buildHTMLMessage("CSU Star", "noreply@csustar.wiki", []string{"test@csu.edu.cn"}, defaultVerificationEmailSubject(), "<b>123456</b>"))
	if !strings.Contains(message, "Subject: CSU Star | 南极星邮箱验证码") {
		t.Fatalf("expected default subject in message, got %s", message)
	}
}
