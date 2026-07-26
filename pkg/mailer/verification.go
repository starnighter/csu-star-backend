package mailer

import (
	"context"
	"errors"
	"fmt"
	"html"
	"strconv"
	"strings"
	"time"

	"csu-star-backend/config"
)

const (
	defaultVerificationSubject = "CSU Star | 南极星邮箱验证码"
	defaultAppName             = "CSU Star"
	defaultLoginURL            = "https://csustar.com/login/"
	defaultCodeTTLMinutes      = 10
)

// sendVerificationEmailFn 是发送验证码邮件的测试 seam。
var sendVerificationEmailFn = sendVerificationEmail

// SendVerificationEmail 发送验证码邮件。
func SendVerificationEmail(to []string, captcha string) error {
	return sendVerificationEmailFn(to, captcha)
}

func sendVerificationEmail(to []string, captcha string) error {
	providers := Providers()
	if len(providers) == 0 {
		return ErrNoSenders
	}

	_, err := newPool(providers).Send(context.Background(), &Message{
		To:       to,
		Subject:  verificationSubject(),
		HTMLBody: renderVerificationEmailHTML(captcha),
	})
	return err
}

// HasUsableProvider 报告是否存在可用的投递通道，供启动自检使用。
func HasUsableProvider() bool { return len(Providers()) > 0 }

func verificationSubject() string {
	if cfg := config.GetConfig(); cfg != nil {
		if s := strings.TrimSpace(cfg.Mail.Verification.Subject); s != "" {
			return s
		}
	}
	return defaultVerificationSubject
}

// CodeTTLMinutes 是验证码有效期（分钟），同时决定 Redis TTL 和邮件正文里的文案。
func CodeTTLMinutes() int {
	if cfg := config.GetConfig(); cfg != nil {
		if m := cfg.Mail.Verification.CodeTTLMinutes; m > 0 {
			return m
		}
	}
	return defaultCodeTTLMinutes
}

func brand() (appName, loginURL string) {
	appName, loginURL = defaultAppName, defaultLoginURL
	if cfg := config.GetConfig(); cfg != nil {
		if v := strings.TrimSpace(cfg.Mail.Brand.AppName); v != "" {
			appName = v
		}
		if v := strings.TrimSpace(cfg.Mail.Brand.LoginURL); v != "" {
			loginURL = v
		}
	}
	return appName, loginURL
}

// renderVerificationEmailHTML 渲染验证码邮件正文。
//
// 用 strings.NewReplacer 而非 html/template：模板通篇是内联 CSS，
// html/template 会在 style 属性和 URL 上下文上做转义而毫无收益。
// 代价是要自己转义，所以每个值都先过 html.EscapeString。
func renderVerificationEmailHTML(captcha string) string {
	appName, loginURL := brand()
	return strings.NewReplacer(
		"{{code}}", html.EscapeString(captcha),
		"{{ttl_minutes}}", strconv.Itoa(CodeTTLMinutes()),
		"{{login_url}}", html.EscapeString(loginURL),
		"{{app_name}}", html.EscapeString(appName),
	).Replace(verificationEmailHTMLTemplate)
}

// ValidAddress 报告字符串是否为合法邮箱，供管理端校验发信地址使用。
func ValidAddress(s string) bool { return validAddress(s) }

// SendTestEmail 用单条通道发送测试邮件，不经过分层池。
// 后台「测试」按钮要验证的是这一条通道，而不是「某一条」通道。
func SendTestEmail(ctx context.Context, p Provider, to string) error {
	if !p.Complete() {
		return errors.New("通道配置不完整")
	}
	if !validAddress(to) {
		return errors.New("收件地址不合法")
	}

	appName, _ := brand()
	perTry := defaultPerTryTimeout
	if cfg := config.GetConfig(); cfg != nil && cfg.Mail.Delivery.PerTryTimeoutSec > 0 {
		perTry = time.Duration(cfg.Mail.Delivery.PerTryTimeoutSec) * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, perTry)
	defer cancel()

	_, err := newSMTPSender(p).Send(ctx, &Message{
		To:      []string{to},
		Subject: appName + " | 邮件通道测试",
		TextBody: "这是一封来自 " + appName + " 后台的邮件通道测试信。\n\n" +
			"通道：" + p.DisplayName() + "\n" +
			"如果你收到了这封邮件，说明该通道配置正确。\n\n" +
			appName + " · 南极星Team",
	})
	return err
}

// RenderForSmoke 渲染但不发送，供 cmd/mail_smoke 的 -dry-run 使用。
// 这是唯一能在不消耗发送额度、不等 DNS 生效的情况下检查报文头的途径。
func RenderForSmoke(p Provider, kind, code, to string) (raw []byte, messageID string, err error) {
	msg := &Message{To: []string{to}}
	if kind == "reply" {
		_, loginURL := brand()
		msg.Subject = replyEmailSubject
		msg.TextBody = fmt.Sprintf(registrationSuccessBody, to, loginURL)
		msg.ReplyTo = replyToAddress()
		msg.ExtraHeaders = map[string]string{ReplyHeaderKey: "true"}
	} else {
		msg.Subject = verificationSubject()
		msg.HTMLBody = renderVerificationEmailHTML(code)
	}
	return buildMIME(p, msg)
}
