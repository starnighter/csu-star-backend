package realtime

import (
	"csu-star-backend/internal/model"
	"csu-star-backend/logger"
	"strconv"
	"sync"
	"time"

	"go.uber.org/zap"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 4096
	sendBufferSize = 32
)

// Hub maintains active WebSocket clients keyed by user ID.
type Hub struct {
	mu         sync.RWMutex
	clients    map[int64]map[*Client]struct{}
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
	stop       chan struct{}
}

// Client is a single authenticated WebSocket connection.
type Client struct {
	hub    *Hub
	userID int64
	conn   Conn
	send   chan []byte
}

// Conn abstracts the websocket connection for tests.
type Conn interface {
	SetReadLimit(limit int64)
	SetReadDeadline(t time.Time) error
	SetWriteDeadline(t time.Time) error
	SetPongHandler(h func(appData string) error)
	ReadMessage() (messageType int, p []byte, err error)
	WriteMessage(messageType int, data []byte) error
	WriteControl(messageType int, data []byte, deadline time.Time) error
	Close() error
}

var defaultHub *Hub

// Default returns the process-wide hub (may be nil before Init).
func Default() *Hub {
	return defaultHub
}

// Init creates and starts the process-wide hub.
func Init() *Hub {
	hub := NewHub()
	go hub.Run()
	defaultHub = hub
	return hub
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[int64]map[*Client]struct{}),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte, 64),
		stop:       make(chan struct{}),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if h.clients[client.userID] == nil {
				h.clients[client.userID] = make(map[*Client]struct{})
			}
			h.clients[client.userID][client] = struct{}{}
			h.mu.Unlock()
		case client := <-h.unregister:
			h.removeClient(client)
		case message := <-h.broadcast:
			h.mu.RLock()
			for _, set := range h.clients {
				for client := range set {
					h.enqueue(client, message)
				}
			}
			h.mu.RUnlock()
		case <-h.stop:
			return
		}
	}
}

func (h *Hub) Stop() {
	select {
	case <-h.stop:
	default:
		close(h.stop)
	}
}

func (h *Hub) Register(client *Client) {
	h.register <- client
}

func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

func (h *Hub) removeClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	set, ok := h.clients[client.userID]
	if !ok {
		return
	}
	if _, exists := set[client]; !exists {
		return
	}
	delete(set, client)
	close(client.send)
	if len(set) == 0 {
		delete(h.clients, client.userID)
	}
}

func (h *Hub) enqueue(client *Client, message []byte) {
	select {
	case client.send <- message:
	default:
		// Slow consumer: drop connection to protect the hub.
		go h.Unregister(client)
	}
}

// SendToUser pushes a raw envelope to every connection of the user.
func (h *Hub) SendToUser(userID int64, message []byte) {
	if h == nil || userID <= 0 || len(message) == 0 {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for client := range h.clients[userID] {
		h.enqueue(client, message)
	}
}

// Broadcast pushes a raw envelope to every connected client.
func (h *Hub) Broadcast(message []byte) {
	if h == nil || len(message) == 0 {
		return
	}
	select {
	case h.broadcast <- message:
	default:
		logger.Log.Warn("realtime hub broadcast queue full, dropping message")
	}
}

// OnlineUserCount returns how many distinct users are connected.
func (h *Hub) OnlineUserCount() int {
	if h == nil {
		return 0
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// PublishNewNotification fans out a newly created notification.
func PublishNewNotification(n *model.Notifications) {
	hub := Default()
	if hub == nil || n == nil {
		return
	}
	payload := NotificationFromModel(n)
	if payload == nil {
		return
	}
	message, err := NewEnvelope(EventNotificationNew, payload)
	if err != nil {
		logger.Log.Warn("encode notification.new failed", zap.Error(err))
		return
	}
	if n.IsGlobal {
		hub.Broadcast(message)
		return
	}
	hub.SendToUser(n.UserID, message)
}

// PublishUnreadCount pushes the latest unread counter for a user.
func PublishUnreadCount(userID int64, count int64) {
	hub := Default()
	if hub == nil || userID <= 0 {
		return
	}
	message, err := NewEnvelope(EventNotificationUnread, UnreadCountPayload{Count: count})
	if err != nil {
		return
	}
	hub.SendToUser(userID, message)
}

// PublishRead pushes a single-notification read event.
func PublishRead(userID, notificationID int64) {
	hub := Default()
	if hub == nil || userID <= 0 || notificationID <= 0 {
		return
	}
	message, err := NewEnvelope(EventNotificationRead, ReadPayload{
		ID: strconv.FormatInt(notificationID, 10),
	})
	if err != nil {
		return
	}
	hub.SendToUser(userID, message)
}

// PublishReadAll pushes a read-all event.
func PublishReadAll(userID int64) {
	hub := Default()
	if hub == nil || userID <= 0 {
		return
	}
	message, err := NewEnvelope(EventNotificationReadAll, map[string]any{})
	if err != nil {
		return
	}
	hub.SendToUser(userID, message)
}
