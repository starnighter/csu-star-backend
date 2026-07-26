package realtime

import (
	"csu-star-backend/internal/model"
	"encoding/json"
	"strconv"
	"time"
)

const (
	EventReady               = "ready"
	EventPong                = "pong"
	EventNotificationNew     = "notification.new"
	EventNotificationUnread  = "notification.unread_count"
	EventNotificationRead    = "notification.read"
	EventNotificationReadAll = "notification.read_all"
)

// Envelope is the WebSocket JSON frame shared by client and server.
type Envelope struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

// NotificationPayload mirrors the REST notification item shape used by the frontend.
type NotificationPayload struct {
	ID         string          `json:"id"`
	Type       string          `json:"type"`
	Category   string          `json:"category,omitempty"`
	Result     string          `json:"result,omitempty"`
	Title      string          `json:"title"`
	Content    string          `json:"content"`
	RelatedID  string          `json:"related_id,omitempty"`
	SourceType string          `json:"source_type,omitempty"`
	SourceID   string          `json:"source_id,omitempty"`
	IsRead     bool            `json:"is_read"`
	IsGlobal   bool            `json:"is_global"`
	IsPinned   bool            `json:"is_pinned"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
}

type ReadyPayload struct {
	UserID      string `json:"user_id"`
	UnreadCount int64  `json:"unread_count"`
}

type UnreadCountPayload struct {
	Count int64 `json:"count"`
}

type ReadPayload struct {
	ID string `json:"id"`
}

func NewEnvelope(eventType string, data any) ([]byte, error) {
	var raw json.RawMessage
	if data != nil {
		encoded, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}
		raw = encoded
	}
	return json.Marshal(Envelope{Type: eventType, Data: raw})
}

func NotificationFromModel(n *model.Notifications) *NotificationPayload {
	if n == nil {
		return nil
	}

	payload := &NotificationPayload{
		ID:        strconv.FormatInt(n.ID, 10),
		Type:      string(n.Type),
		Category:  string(n.Category),
		Result:    string(n.Result),
		Title:     n.Title,
		Content:   n.Content,
		IsRead:    n.IsRead,
		IsGlobal:  n.IsGlobal,
		CreatedAt: n.CreatedAt,
	}
	if n.RelatedID != 0 {
		payload.RelatedID = strconv.FormatInt(n.RelatedID, 10)
	}
	if len(n.Metadata) > 0 {
		payload.Metadata = json.RawMessage(n.Metadata)
		var meta map[string]any
		if err := json.Unmarshal(n.Metadata, &meta); err == nil {
			if sourceType, ok := meta["source_type"].(string); ok {
				payload.SourceType = sourceType
			}
			switch sourceID := meta["source_id"].(type) {
			case string:
				payload.SourceID = sourceID
			case float64:
				payload.SourceID = strconv.FormatInt(int64(sourceID), 10)
			case json.Number:
				payload.SourceID = sourceID.String()
			}
		}
	}
	if payload.CreatedAt.IsZero() {
		payload.CreatedAt = time.Now()
	}
	return payload
}
