package mailer

import (
	"encoding/base64"
	"io"
	"mime"
	"net/mail"
	"strings"
	"testing"
	"time"
)

func testProvider() Provider {
	return Provider{
		Name:          "test",
		Kind:          KindAliyunDM,
		Host:          "smtpdm.aliyun.com",
		Port:          465,
		Username:      "noreply@mail.csustar.wiki",
		Password:      "secret",
		FromEmailAddr: "noreply@mail.csustar.wiki",
		FromName:      "CSU Star",
	}
}

func parseBuilt(t *testing.T, raw []byte) (*mail.Message, string) {
	t.Helper()
	msg, err := mail.ReadMessage(strings.NewReader(string(raw)))
	if err != nil {
		t.Fatalf("ReadMessage() error = %v", err)
	}
	body, err := io.ReadAll(msg.Body)
	if err != nil {
		t.Fatalf("read body error = %v", err)
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(strings.ReplaceAll(string(body), "\r\n", ""), "\n", ""))
	if err != nil {
		t.Fatalf("base64 decode error = %v", err)
	}
	return msg, string(decoded)
}

func TestBuildMIMEEncodesChineseSubject(t *testing.T) {
	subject := "CSU Star | 南极星邮箱验证码"
	raw, _, err := buildMIME(testProvider(), &Message{To: []string{"a@qq.com"}, Subject: subject, HTMLBody: "<p>hi</p>"})
	if err != nil {
		t.Fatalf("buildMIME() error = %v", err)
	}

	msg, _ := parseBuilt(t, raw)
	got := msg.Header.Get("Subject")
	// RFC 2047 的编码标记大小写不敏感，Go 输出小写 b。
	if !strings.HasPrefix(strings.ToUpper(got), "=?UTF-8?B?") {
		t.Fatalf("expected RFC 2047 B-encoded subject, got %q", got)
	}

	decoded, err := new(mime.WordDecoder).DecodeHeader(got)
	if err != nil {
		t.Fatalf("DecodeHeader() error = %v", err)
	}
	if decoded != subject {
		t.Fatalf("round-trip = %q, want %q", decoded, subject)
	}
}

// 纯 ASCII 主题不应被编码，否则线路上的报文平白变得不可读。
func TestBuildMIMELeavesASCIISubjectUnencoded(t *testing.T) {
	raw, _, err := buildMIME(testProvider(), &Message{To: []string{"a@qq.com"}, Subject: "Verification Code", TextBody: "hi"})
	if err != nil {
		t.Fatalf("buildMIME() error = %v", err)
	}
	msg, _ := parseBuilt(t, raw)
	if got := msg.Header.Get("Subject"); got != "Verification Code" {
		t.Fatalf("Subject = %q, want unencoded", got)
	}
}

func TestBuildMIMEHasParseableDateAndMessageID(t *testing.T) {
	raw, messageID, err := buildMIME(testProvider(), &Message{To: []string{"a@qq.com"}, Subject: "s", TextBody: "b"})
	if err != nil {
		t.Fatalf("buildMIME() error = %v", err)
	}

	msg, _ := parseBuilt(t, raw)
	if _, err := msg.Header.Date(); err != nil {
		t.Fatalf("Date header unparseable: %v", err)
	}
	if got := msg.Header.Get("Message-ID"); got != messageID {
		t.Fatalf("Message-ID header = %q, want %q", got, messageID)
	}
	if !strings.HasSuffix(messageID, "@mail.csustar.wiki>") {
		t.Fatalf("Message-ID should use the sending domain, got %q", messageID)
	}
}

func TestBuildMIMEBodyBase64RoundTrips(t *testing.T) {
	// 刻意用一段远超 998 字节的单行内容，模拟手维护的 HTML 模板。
	body := "<p>" + strings.Repeat("南极星", 500) + "</p>"
	raw, _, err := buildMIME(testProvider(), &Message{To: []string{"a@qq.com"}, Subject: "s", HTMLBody: body})
	if err != nil {
		t.Fatalf("buildMIME() error = %v", err)
	}

	msg, decoded := parseBuilt(t, raw)
	if decoded != body {
		t.Fatal("body did not round-trip through base64")
	}
	if got := msg.Header.Get("Content-Transfer-Encoding"); got != "base64" {
		t.Fatalf("Content-Transfer-Encoding = %q", got)
	}
	for _, line := range strings.Split(string(raw), "\r\n") {
		if len(line) > 998 {
			t.Fatalf("line exceeds RFC 5321 limit: %d bytes", len(line))
		}
	}
}

// X-CSU-Star-Reply 是 IMAP 轮询器识别自己回信的唯一依据。
// 这个头一旦丢失，poller 会对自己的回信无限回复。
func TestBuildMIMEIncludesReplyToAndExtraHeaders(t *testing.T) {
	raw, _, err := buildMIME(testProvider(), &Message{
		To:           []string{"a@qq.com"},
		Subject:      "s",
		TextBody:     "b",
		ReplyTo:      "logincsu@foxmail.com",
		ExtraHeaders: map[string]string{ReplyHeaderKey: "true"},
	})
	if err != nil {
		t.Fatalf("buildMIME() error = %v", err)
	}

	msg, _ := parseBuilt(t, raw)
	if got := msg.Header.Get("Reply-To"); got != "logincsu@foxmail.com" {
		t.Fatalf("Reply-To = %q", got)
	}
	if got := msg.Header.Get(ReplyHeaderKey); got != "true" {
		t.Fatalf("%s = %q", ReplyHeaderKey, got)
	}
}

// 头部值里的换行会造成头注入。
func TestBuildMIMEStripsHeaderInjection(t *testing.T) {
	raw, _, err := buildMIME(testProvider(), &Message{
		To:       []string{"a@qq.com"},
		Subject:  "ok\r\nBcc: attacker@evil.com",
		TextBody: "b",
	})
	if err != nil {
		t.Fatalf("buildMIME() error = %v", err)
	}
	msg, _ := parseBuilt(t, raw)
	if msg.Header.Get("Bcc") != "" {
		t.Fatal("header injection was not stripped")
	}
}

func TestBuildMIMERejectsEmptyRecipients(t *testing.T) {
	if _, _, err := buildMIME(testProvider(), &Message{Subject: "s", TextBody: "b"}); err == nil {
		t.Fatal("expected error for empty recipients")
	}
}

func TestFormatMailAddressEncodesNonASCIIName(t *testing.T) {
	got := formatMailAddress("南极星", "a@b.com")
	if !strings.Contains(strings.ToUpper(got), "=?UTF-8?B?") {
		t.Fatalf("expected encoded display name, got %q", got)
	}
	if formatMailAddress("CSU Star", "a@b.com") != "CSU Star <a@b.com>" {
		t.Fatal("ASCII display name should pass through unchanged")
	}
	if formatMailAddress("  ", "a@b.com") != "a@b.com" {
		t.Fatal("empty display name should yield bare address")
	}
}

func TestNowFnSeamIsUsed(t *testing.T) {
	fixed := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	nowFn = func() time.Time { return fixed }
	t.Cleanup(func() { nowFn = time.Now })

	raw, _, err := buildMIME(testProvider(), &Message{To: []string{"a@qq.com"}, Subject: "s", TextBody: "b"})
	if err != nil {
		t.Fatalf("buildMIME() error = %v", err)
	}
	msg, _ := parseBuilt(t, raw)
	date, err := msg.Header.Date()
	if err != nil {
		t.Fatalf("Date() error = %v", err)
	}
	if !date.Equal(fixed) {
		t.Fatalf("Date = %v, want %v", date, fixed)
	}
}
