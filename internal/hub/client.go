package hub

import (
	"encoding/json"
	"log"
	"time"

	"github.com/fasthttp/websocket"
)

const (
	// writeWait is the maximum time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// pongWait is the maximum time to wait for a pong from the peer.
	pongWait = 60 * time.Second

	// pingPeriod sends pings at this interval. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// maxMessageSize is the maximum message size allowed from peer.
	maxMessageSize = 512
)

// Client wraps a single WebSocket connection for a specific user.
type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	send   chan []byte
	userID uint
}

// NewClient creates a new Client bound to the given hub, connection, and user.
func NewClient(hub *Hub, conn *websocket.Conn, userID uint) *Client {
	return &Client{
		hub:    hub,
		conn:   conn,
		send:   make(chan []byte, 256),
		userID: userID,
	}
}

// ReadPump reads messages from the WebSocket connection.
// It handles pings/pongs and cleans up on disconnect.
// Must be called in its own goroutine.
func (c *Client) ReadPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("websocket read error for user %d: %v", c.userID, err)
			}
			break
		}
		// We don't process inbound messages — the client only receives events.
	}
}

// WritePump writes messages from the send channel to the WebSocket connection.
// It also sends periodic pings to keep the connection alive.
// Must be called in its own goroutine.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// SendEvent marshals an event to JSON and queues it for sending.
func (c *Client) SendEvent(event Event) {
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("failed to marshal event for user %d: %v", c.userID, err)
		return
	}

	select {
	case c.send <- data:
	default:
		// Send buffer full — drop the message to avoid blocking.
		log.Printf("dropping event for user %d: send buffer full", c.userID)
	}
}
