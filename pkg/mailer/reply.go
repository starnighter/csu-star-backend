package mailer

import (
	"context"
	"fmt"
	"strings"

	"csu-star-backend/config"
)

const replyEmailSubject = "CSU Star | 注册结果通知"

const registrationSuccessBody = `尊敬的用户，

恭喜您！您的 CSU Star 账号已注册成功。

邮箱：%s
密码：已安全存储（请妥善保管）

您可以前往 %s 使用邮箱登录。

如有问题，请联系管理员。

CSU Star · 南极星Team`

const registrationAlreadyExistsBody = `尊敬的用户，

您的邮箱 %s 已经注册过 CSU Star 账号，请直接登录。

登录地址：%s

如忘记密码，请使用"忘记密码"功能重置。

CSU Star · 南极星Team`

const registrationInvalidPasswordBody = `尊敬的用户，

您的注册请求未通过密码校验，原因：%s

密码要求：
- 长度为 %d~%d 个字符
- 不能包含空格或换行

请在邮件主题中填写密码，重新发送邮件至本邮箱进行注册。

CSU Star · 南极星Team`

const registrationEmptySubjectBody = `尊敬的用户，

您的注册请求未通过校验，原因：邮件主题为空，无法提取密码。

请在邮件主题中填写您的密码，重新发送邮件至本邮箱进行注册。

CSU Star · 南极星Team`

// SendRegistrationReplyEmail 回复邮件注册的结果。
func SendRegistrationReplyEmail(to string, success bool) error {
	_, loginURL := brand()
	body := fmt.Sprintf(registrationAlreadyExistsBody, to, loginURL)
	if success {
		body = fmt.Sprintf(registrationSuccessBody, to, loginURL)
	}
	return sendReply(to, body)
}

// SendRegistrationInvalidPasswordReplyEmail 说明密码为何被拒。
func SendRegistrationInvalidPasswordReplyEmail(to, reason string, minLen, maxLen int) error {
	return sendReply(to, fmt.Sprintf(registrationInvalidPasswordBody, reason, minLen, maxLen))
}

// SendRegistrationEmptySubjectReplyEmail 说明邮件主题为空无法提取密码。
func SendRegistrationEmptySubjectReplyEmail(to string) error {
	return sendReply(to, registrationEmptySubjectBody)
}

func sendReply(to, body string) error {
	providers := ReplyProviders()
	if len(providers) == 0 {
		return ErrNoSenders
	}

	_, err := newPool(providers).Send(context.Background(), &Message{
		To:       []string{to},
		Subject:  replyEmailSubject,
		TextBody: body,
		// 回信从 noreply 类地址发出时用户无法回复，指回 IMAP 收信箱。
		ReplyTo:      replyToAddress(),
		ExtraHeaders: map[string]string{ReplyHeaderKey: "true"},
	})
	return err
}

func replyToAddress() string {
	cfg := config.GetConfig()
	if cfg == nil {
		return ""
	}
	addr := strings.TrimSpace(cfg.Mail.Imap.Username)
	if addr == "" || !validAddress(addr) {
		return ""
	}
	return addr
}
