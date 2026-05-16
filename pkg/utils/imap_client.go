package utils

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net"
	"net/mail"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	idle "github.com/emersion/go-imap-idle"
)

const replyHeaderKey = "X-CSU-Star-Reply"

type ImapClient struct {
	client *client.Client
	idler  *idle.Client
}

type ImapMessage struct {
	UID     uint32
	From    string
	Subject string
	Body    string
	IsReply bool
}

func NewImapClient(host string, port int, username, password string) (*ImapClient, error) {
	addr := fmt.Sprintf("%s:%d", host, port)

	// 使用带超时的拨号器，避免 TCP 连接无限等待
	dialer := &net.Dialer{Timeout: 15 * time.Second}
	c, err := client.DialWithDialerTLS(dialer, addr, &tls.Config{
		ServerName: host,
		MinVersion: tls.VersionTLS12,
	})
	if err != nil {
		return nil, fmt.Errorf("IMAP dial: %w", err)
	}
	c.Timeout = 30 * time.Second

	if err := c.Login(username, password); err != nil {
		_ = c.Logout()
		return nil, fmt.Errorf("IMAP login: %w", err)
	}

	ic := &ImapClient{client: c}
	if supported, err := idle.NewClient(c).SupportIdle(); err == nil && supported {
		ic.idler = idle.NewClient(c)
	}
	return ic, nil
}

func (ic *ImapClient) FetchUnseenMessages() ([]*ImapMessage, error) {
	mbox, err := ic.client.Select("INBOX", false)
	if err != nil {
		return nil, fmt.Errorf("select INBOX: %w", err)
	}
	if mbox.Messages == 0 {
		return nil, nil
	}

	criteria := imap.NewSearchCriteria()
	criteria.WithoutFlags = []string{imap.SeenFlag}
	uids, err := ic.client.UidSearch(criteria)
	if err != nil {
		return nil, fmt.Errorf("search unseen: %w", err)
	}
	if len(uids) == 0 {
		return nil, nil
	}

	// Fetch in batches of 50
	const batchSize = 50
	var messages []*ImapMessage
	section := &imap.BodySectionName{}

	for i := 0; i < len(uids); i += batchSize {
		end := i + batchSize
		if end > len(uids) {
			end = len(uids)
		}
		batch := uids[i:end]

		seqSet := new(imap.SeqSet)
		for _, uid := range batch {
			seqSet.AddNum(uid)
		}

		items := []imap.FetchItem{imap.FetchEnvelope, imap.FetchFlags, imap.FetchUid, section.FetchItem()}
		msgCh := make(chan *imap.Message, len(batch))
		done := make(chan error, 1)

		go func() {
			done <- ic.client.UidFetch(seqSet, items, msgCh)
		}()

		for msg := range msgCh {
			if msg == nil {
				continue
			}

			imapMsg := &ImapMessage{
				UID:     msg.Uid,
				Subject: msg.Envelope.Subject,
			}

			// Extract sender email
			if len(msg.Envelope.From) > 0 {
				addr := msg.Envelope.From[0]
				imapMsg.From = fmt.Sprintf("%s@%s", addr.MailboxName, addr.HostName)
			}

			// Parse body section for reply header and plain text body
			if r := msg.GetBody(section); r != nil {
				if raw, err := io.ReadAll(r); err == nil {
					imapMsg.IsReply = hasReplyHeader(raw)
					imapMsg.Body = extractPlainTextBody(raw)
				}
			}

			messages = append(messages, imapMsg)
		}

		if err := <-done; err != nil {
			return nil, fmt.Errorf("fetch messages: %w", err)
		}
	}

	return messages, nil
}

// hasReplyHeader checks if the raw RFC822 message contains the X-CSU-Star-Reply header.
func hasReplyHeader(raw []byte) bool {
	// Only scan the header portion (before the first blank line)
	idx := bytes.Index(raw, []byte("\r\n\r\n"))
	if idx == -1 {
		idx = bytes.Index(raw, []byte("\n\n"))
		if idx == -1 {
			return false
		}
	}
	header := raw[:idx]
	return bytes.Contains(header, []byte(replyHeaderKey+":"))
}

// extractPlainTextBody parses the raw RFC822 message and returns the plain text body.
func extractPlainTextBody(raw []byte) string {
	msg, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		return ""
	}

	mediaType, params, err := mime.ParseMediaType(msg.Header.Get("Content-Type"))
	if err != nil {
		// Default: treat as text/plain
		body, _ := io.ReadAll(msg.Body)
		return strings.TrimSpace(string(body))
	}

	if strings.HasPrefix(mediaType, "text/plain") {
		body, _ := io.ReadAll(msg.Body)
		return strings.TrimSpace(string(body))
	}

	if strings.HasPrefix(mediaType, "multipart/") {
		boundary := params["boundary"]
		if boundary == "" {
			return ""
		}
		mr := multipart.NewReader(msg.Body, boundary)
		for {
			part, err := mr.NextPart()
			if err != nil {
				break
			}
			partType, _, _ := mime.ParseMediaType(part.Header.Get("Content-Type"))
			if strings.HasPrefix(partType, "text/plain") {
				body, _ := io.ReadAll(part)
				return strings.TrimSpace(string(body))
			}
		}
	}

	return ""
}

// ReadMessageHeader reads a specific header from the raw RFC822 message.
func ReadMessageHeader(raw []byte, key string) string {
	idx := bytes.Index(raw, []byte("\r\n\r\n"))
	if idx == -1 {
		idx = bytes.Index(raw, []byte("\n\n"))
		if idx == -1 {
			return ""
		}
	}
	headerBlock := raw[:idx]
	scanner := bufio.NewScanner(bytes.NewReader(headerBlock))
	prefix := key + ":"
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSpace(line[len(prefix):])
		}
	}
	return ""
}

func (ic *ImapClient) DeleteMessages(uids []uint32) error {
	if len(uids) == 0 {
		return nil
	}

	uidSet := new(imap.SeqSet)
	for _, uid := range uids {
		uidSet.AddNum(uid)
	}

	item := imap.FormatFlagsOp(imap.AddFlags, true)
	flags := []interface{}{imap.DeletedFlag}
	if err := ic.client.UidStore(uidSet, item, flags, nil); err != nil {
		return fmt.Errorf("mark deleted: %w", err)
	}

	if err := ic.client.Expunge(nil); err != nil {
		return fmt.Errorf("expunge: %w", err)
	}

	return nil
}

// MarkMessagesSeen flags the given messages as \Seen so they won't be fetched again.
func (ic *ImapClient) MarkMessagesSeen(uids []uint32) error {
	if len(uids) == 0 {
		return nil
	}

	uidSet := new(imap.SeqSet)
	for _, uid := range uids {
		uidSet.AddNum(uid)
	}

	item := imap.FormatFlagsOp(imap.AddFlags, true)
	flags := []interface{}{imap.SeenFlag}
	if err := ic.client.UidStore(uidSet, item, flags, nil); err != nil {
		return fmt.Errorf("mark seen: %w", err)
	}

	return nil
}

// IsIdleSupported returns true if the server advertised IDLE capability.
func (ic *ImapClient) IsIdleSupported() bool {
	return ic.idler != nil
}

// Idle enters IMAP IDLE mode. It blocks until either:
//   - the caller closes stop (sends DONE to the server, returns nil)
//   - the server sends a notification (returns nil)
//   - the connection drops (returns error)
//
// The client.Timeout is temporarily raised to 30 minutes during IDLE,
// because RFC 3501 allows servers to remain silent for up to 29 minutes.
func (ic *ImapClient) Idle(stop <-chan struct{}) error {
	if ic.idler == nil {
		return fmt.Errorf("IDLE not supported by server")
	}
	originalTimeout := ic.client.Timeout
	ic.client.Timeout = 30 * time.Minute
	defer func() { ic.client.Timeout = originalTimeout }()

	return ic.idler.Idle(stop)
}

func (ic *ImapClient) Close() error {
	return ic.client.Logout()
}
