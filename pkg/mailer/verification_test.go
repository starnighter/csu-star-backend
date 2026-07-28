package mailer

import (
	"strings"
	"testing"

	"csu-star-backend/config"
)

func TestRenderVerificationEmailSubstitutesAllPlaceholders(t *testing.T) {
	config.SetConfig(&config.Config{Mail: config.MailConfig{
		Brand:        config.MailBrandConfig{AppName: "南极星", LoginURL: "https://csustar.wiki/login/"},
		Verification: config.VerificationMailConfig{CodeTTLMinutes: 7},
	}})
	t.Cleanup(func() { config.SetConfig(nil) })

	html := renderVerificationEmailHTML("123456")

	if strings.Contains(html, "{{") {
		t.Fatal("template still contains unsubstituted placeholders")
	}
	if !strings.Contains(html, "123456") {
		t.Fatal("captcha not injected")
	}
	if !strings.Contains(html, "https://csustar.wiki/login/") {
		t.Fatal("login URL not injected")
	}
	if !strings.Contains(html, "> 7 <") {
		t.Fatal("TTL minutes not injected")
	}
	if !strings.Contains(html, "南极星") {
		t.Fatal("app name not injected")
	}
}

// 邮件正文里写的分钟数必须与 Redis TTL 同源。
// 历史上正文写「5 分钟」而实际存 10 分钟，这个测试防止漂移复发。
func TestCodeTTLMinutesDefaultsToTen(t *testing.T) {
	config.SetConfig(nil)
	if got := CodeTTLMinutes(); got != 10 {
		t.Fatalf("nil config: CodeTTLMinutes() = %d, want 10", got)
	}

	config.SetConfig(&config.Config{})
	t.Cleanup(func() { config.SetConfig(nil) })
	if got := CodeTTLMinutes(); got != 10 {
		t.Fatalf("zero value: CodeTTLMinutes() = %d, want 10", got)
	}
}

func TestRenderVerificationEmailUsesDefaultsWithNilConfig(t *testing.T) {
	config.SetConfig(nil)

	html := renderVerificationEmailHTML("000000")
	if strings.Contains(html, "{{") {
		t.Fatal("template still contains unsubstituted placeholders")
	}
	if !strings.Contains(html, "> 10 <") {
		t.Fatal("expected default TTL of 10 minutes")
	}
	if !strings.Contains(html, defaultLoginURL) {
		t.Fatal("expected default login URL")
	}
}

func TestVerificationSubjectFallsBackToDefault(t *testing.T) {
	config.SetConfig(&config.Config{Mail: config.MailConfig{
		Verification: config.VerificationMailConfig{Subject: "   "},
	}})
	t.Cleanup(func() { config.SetConfig(nil) })

	if got := verificationSubject(); got != defaultVerificationSubject {
		t.Fatalf("verificationSubject() = %q, want default", got)
	}
}

// 验证码是纯数字，但渲染路径必须仍然转义——模板占位符的替换是
// strings.Replacer 而非 html/template，转义是唯一的防线。
func TestRenderVerificationEmailEscapesInjectedValues(t *testing.T) {
	config.SetConfig(&config.Config{Mail: config.MailConfig{
		Brand: config.MailBrandConfig{LoginURL: `https://x/"><script>alert(1)</script>`},
	}})
	t.Cleanup(func() { config.SetConfig(nil) })

	if strings.Contains(renderVerificationEmailHTML("1"), "<script>") {
		t.Fatal("config value was not escaped into the template")
	}
}
