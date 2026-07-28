package mailer

import (
	"encoding/base64"
	"fmt"
	"mime"
	"net/mail"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// base64 正文的折行宽度。RFC 5321 限制单行不超过 998 字节，
// 而 HTML 模板是手维护的字符串常量，没人会记得保持折行——
// 用 base64 让超长行在结构上不可能发生。
const base64LineWidth = 76

// nowFn 是时间源的测试 seam。
var nowFn = time.Now

// newMessageIDFn 是 Message-ID 生成的测试 seam。
var newMessageIDFn = uuid.NewString

// buildMIME 构造完整的 RFC 5322 报文，返回报文字节和生成的 Message-ID。
//
// 相比旧的 buildHTMLMessage/buildPlainTextMessage，这里补齐了三个会影响
// 送达率的头：RFC 2047 编码的 Subject、Date、Message-ID。缺 Date 是明确的
// 垃圾邮件评分项，而高校邮件网关正是最保守的那类过滤器。
func buildMIME(p Provider, msg *Message) (raw []byte, messageID string, err error) {
	if len(msg.To) == 0 {
		return nil, "", fmt.Errorf("mailer: 收件人为空")
	}

	domain := "localhost"
	if at := strings.LastIndex(p.FromEmailAddr, "@"); at >= 0 && at < len(p.FromEmailAddr)-1 {
		domain = p.FromEmailAddr[at+1:]
	}
	messageID = fmt.Sprintf("<%s@%s>", newMessageIDFn(), domain)

	contentType := "text/plain; charset=UTF-8"
	body := msg.TextBody
	if msg.HTMLBody != "" {
		contentType = "text/html; charset=UTF-8"
		body = msg.HTMLBody
	}

	type header struct{ key, value string }
	headers := []header{
		{"From", formatMailAddress(p.FromName, p.FromEmailAddr)},
		{"To", strings.Join(msg.To, ", ")},
		{"Subject", encodeHeaderWord(msg.Subject)},
		{"Date", nowFn().Format(time.RFC1123Z)},
		{"Message-ID", messageID},
		{"MIME-Version", "1.0"},
		{"Content-Type", contentType},
		{"Content-Transfer-Encoding", "base64"},
	}
	if replyTo := strings.TrimSpace(msg.ReplyTo); replyTo != "" {
		headers = append(headers, header{"Reply-To", replyTo})
	}

	// 排序保证同样的输入产生同样的报文，便于测试与调试。
	extraKeys := make([]string, 0, len(msg.ExtraHeaders))
	for k := range msg.ExtraHeaders {
		extraKeys = append(extraKeys, k)
	}
	sort.Strings(extraKeys)
	for _, k := range extraKeys {
		headers = append(headers, header{k, msg.ExtraHeaders[k]})
	}

	var b strings.Builder
	for _, h := range headers {
		// 头部值里的 CR/LF 会造成头注入，直接剔除。
		v := strings.NewReplacer("\r", "", "\n", "").Replace(h.value)
		fmt.Fprintf(&b, "%s: %s\r\n", h.key, v)
	}
	b.WriteString("\r\n")
	b.WriteString(wrapBase64(body))

	return []byte(b.String()), messageID, nil
}

// encodeHeaderWord 对含非 ASCII 的头做 RFC 2047 B 编码。
// 纯 ASCII 会原样返回，所以英文主题在线路上仍然可读。
// 用 B 而非 Q：主题以中文为主，Q 编码会把每个汉字膨胀成三个 =XX。
func encodeHeaderWord(s string) string {
	return mime.BEncoding.Encode("UTF-8", s)
}

func formatMailAddress(name, email string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return email
	}
	return fmt.Sprintf("%s <%s>", encodeHeaderWord(trimmed), email)
}

func wrapBase64(body string) string {
	encoded := base64.StdEncoding.EncodeToString([]byte(body))

	var b strings.Builder
	for i := 0; i < len(encoded); i += base64LineWidth {
		end := i + base64LineWidth
		if end > len(encoded) {
			end = len(encoded)
		}
		b.WriteString(encoded[i:end])
		b.WriteString("\r\n")
	}
	return b.String()
}

// validAddress 报告字符串是否为可用作信封地址的邮箱。
func validAddress(s string) bool {
	addr, err := mail.ParseAddress(strings.TrimSpace(s))
	return err == nil && addr.Address != ""
}
