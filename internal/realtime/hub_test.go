package realtime

import (
	"csu-star-backend/internal/model"
	"encoding/json"
	"sync"
	"testing"
	"time"
)

type mockConn struct {
	mu       sync.Mutex
	messages [][]byte
	closed   bool
	readCh   chan struct{}
}

func (m *mockConn) SetReadLimit(limit int64)                    {}
func (m *mockConn) SetReadDeadline(t time.Time) error           { return nil }
func (m *mockConn) SetWriteDeadline(t time.Time) error          { return nil }
func (m *mockConn) SetPongHandler(h func(appData string) error) {}
func (m *mockConn) ReadMessage() (messageType int, p []byte, err error) {
	<-m.readCh
	return 0, nil, errClosed
}
func (m *mockConn) WriteMessage(messageType int, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return errClosed
	}
	copied := make([]byte, len(data))
	copy(copied, data)
	m.messages = append(m.messages, copied)
	return nil
}
func (m *mockConn) WriteControl(messageType int, data []byte, deadline time.Time) error {
	return nil
}
func (m *mockConn) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

var errClosed = errString("closed")

type errString string

func (e errString) Error() string { return string(e) }

func TestHubSendToUser(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Stop()
	defaultHub = hub
	defer func() { defaultHub = nil }()

	conn := &mockConn{readCh: make(chan struct{})}
	client := NewClient(hub, 42, conn)
	hub.Register(client)
	time.Sleep(20 * time.Millisecond)

	go client.writePump()

	PublishNewNotification(&model.Notifications{
		ID:       1001,
		UserID:   42,
		Type:     model.NotificationTypeLiked,
		Category: model.NotificationCategoryInteraction,
		Result:   model.NotificationResultInform,
		Title:    "收到新的点赞",
		Content:  "你的资源收到了新的点赞。",
	})

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		conn.mu.Lock()
		n := len(conn.messages)
		conn.mu.Unlock()
		if n > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	conn.mu.Lock()
	defer conn.mu.Unlock()
	if len(conn.messages) == 0 {
		t.Fatal("expected at least one pushed message")
	}
	var envelope Envelope
	if err := json.Unmarshal(conn.messages[0], &envelope); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}
	if envelope.Type != EventNotificationNew {
		t.Fatalf("expected %s, got %s", EventNotificationNew, envelope.Type)
	}
}
