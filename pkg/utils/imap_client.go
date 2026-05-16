package utils

import (
	"crypto/tls"
	"fmt"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

type ImapClient struct {
	client *client.Client
}

type ImapMessage struct {
	UID     uint32
	From    string
	Subject string
}

func NewImapClient(host string, port int, username, password string) (*ImapClient, error) {
	addr := fmt.Sprintf("%s:%d", host, port)
	c, err := client.DialTLS(addr, &tls.Config{
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
	return &ImapClient{client: c}, nil
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
	uids, err := ic.client.Search(criteria)
	if err != nil {
		return nil, fmt.Errorf("search unseen: %w", err)
	}
	if len(uids) == 0 {
		return nil, nil
	}

	// Fetch in batches of 50
	const batchSize = 50
	var messages []*ImapMessage

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

		items := []imap.FetchItem{imap.FetchEnvelope, imap.FetchFlags}
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

			messages = append(messages, imapMsg)
		}

		if err := <-done; err != nil {
			return messages, fmt.Errorf("fetch messages: %w", err)
		}
	}

	return messages, nil
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

	_ = ic.client.Expunge(nil)

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

func (ic *ImapClient) Close() error {
	return ic.client.Logout()
}

