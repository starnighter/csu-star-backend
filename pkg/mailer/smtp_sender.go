package mailer

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"strconv"
	"strings"
)

// TLS 传输方式。
const (
	TLSModeImplicit = "implicit" // 465，连接即 TLS
	TLSModeStartTLS = "starttls" // 25/587，先明文再 STARTTLS 升级
)

// dialFn 是网络层的测试 seam：替换它即可在完全不联网的情况下测试上层逻辑。
var dialFn = dialSMTP

type smtpSender struct{ p Provider }

func newSMTPSender(p Provider) Sender { return &smtpSender{p: p} }

func (s *smtpSender) Name() string { return s.p.DisplayName() }

func (s *smtpSender) Send(ctx context.Context, msg *Message) (*SendResult, error) {
	if !s.p.Complete() {
		return nil, errors.New("smtp 配置不完整")
	}

	raw, messageID, err := buildMIME(s.p, msg)
	if err != nil {
		return nil, err
	}

	addr := net.JoinHostPort(s.p.Host, strconv.Itoa(s.p.Port))
	auth := smtp.PlainAuth("", s.p.Username, s.p.Password, s.p.Host)
	if err := deliver(ctx, s.p, addr, auth, msg.To, raw); err != nil {
		return nil, err
	}

	return &SendResult{Channel: s.Name(), MessageID: messageID}, nil
}

func deliver(ctx context.Context, p Provider, addr string, auth smtp.Auth, to []string, raw []byte) (err error) {
	client, err := dialFn(ctx, p, addr)
	if err != nil {
		return err
	}
	defer client.Close()

	if auth != nil {
		if ok, _ := client.Extension("AUTH"); ok {
			if err = client.Auth(auth); err != nil {
				return err
			}
		}
	}

	if err = client.Mail(p.FromEmailAddr); err != nil {
		return err
	}
	for _, recipient := range to {
		if err = client.Rcpt(recipient); err != nil {
			return err
		}
	}

	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err = writer.Write(raw); err != nil {
		_ = writer.Close()
		return err
	}
	if err = writer.Close(); err != nil {
		return err
	}
	return client.Quit()
}

func dialSMTP(ctx context.Context, p Provider, addr string) (*smtp.Client, error) {
	mode := strings.ToLower(strings.TrimSpace(p.TLSMode))
	if mode == "" {
		// 空值等于 implicit，保证所有存量配置行为逐字节不变。
		mode = TLSModeImplicit
	}

	tlsCfg := &tls.Config{ServerName: p.Host, MinVersion: tls.VersionTLS12}
	dialer := &net.Dialer{}

	switch mode {
	case TLSModeImplicit:
		conn, err := (&tls.Dialer{NetDialer: dialer, Config: tlsCfg}).DialContext(ctx, "tcp", addr)
		if err != nil {
			return nil, err
		}
		applyDeadline(ctx, conn)
		client, err := smtp.NewClient(conn, p.Host)
		if err != nil {
			_ = conn.Close()
			return nil, err
		}
		return client, nil

	case TLSModeStartTLS:
		conn, err := dialer.DialContext(ctx, "tcp", addr)
		if err != nil {
			return nil, err
		}
		applyDeadline(ctx, conn)
		client, err := smtp.NewClient(conn, p.Host)
		if err != nil {
			_ = conn.Close()
			return nil, err
		}
		if ok, _ := client.Extension("STARTTLS"); !ok {
			_ = client.Close()
			return nil, fmt.Errorf("服务器不支持 STARTTLS: %s", addr)
		}
		if err := client.StartTLS(tlsCfg); err != nil {
			_ = client.Close()
			return nil, err
		}
		return client, nil

	default:
		return nil, fmt.Errorf("未知的 tls_mode: %q", p.TLSMode)
	}
}

// applyDeadline 把 context 的截止时间落到 socket 上。
// 这是让 Pool 的 per-try 预算真正生效的地方——否则 context 取消了，
// 底层 SMTP 交互仍会一直阻塞到自己的超时。
func applyDeadline(ctx context.Context, conn net.Conn) {
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}
}
