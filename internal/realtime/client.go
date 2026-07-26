package realtime

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

// NewClient builds a hub client for an authenticated user connection.
func NewClient(hub *Hub, userID int64, conn Conn) *Client {
	return &Client{
		hub:    hub,
		userID: userID,
		conn:   conn,
		send:   make(chan []byte, sendBufferSize),
	}
}

// Serve pumps messages to/from the peer.
func (c *Client) Serve() {
	go c.writePump()
	c.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.hub.Unregister(c)
		_ = c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		var envelope Envelope
		if err := json.Unmarshal(data, &envelope); err != nil {
			continue
		}
		switch envelope.Type {
		case "ping":
			message, err := NewEnvelope(EventPong, map[string]any{})
			if err == nil {
				c.enqueue(message)
			}
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) enqueue(message []byte) {
	select {
	case c.send <- message:
	default:
	}
}

// SendReady writes the initial ready event into the client queue.
func (c *Client) SendReady(unreadCount int64) {
	message, err := NewEnvelope(EventReady, ReadyPayload{
		UserID:      strconv.FormatInt(c.userID, 10),
		UnreadCount: unreadCount,
	})
	if err != nil {
		return
	}
	c.enqueue(message)
}
