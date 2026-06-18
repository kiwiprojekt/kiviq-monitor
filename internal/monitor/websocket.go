package monitor

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/michal/kiviq/internal/shared"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Keepalive timings. The server pings every pingPeriod and drops a client that
// has not produced any frame (pong or otherwise) within pongWait, so a
// half-open connection is detected instead of holding a slot open forever.
// Vars, not consts, so tests can shrink them. pingPeriod must stay below
// pongWait so a healthy client's pong always lands before the deadline.
var (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
)

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan shared.WSMessage
	register   chan *Client
	unregister chan *Client
}

type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan shared.WSMessage, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			log.Printf("WS client connected (%d total)", len(h.clients))

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}

		case msg := <-h.broadcast:
			data, err := json.Marshal(msg)
			if err != nil {
				log.Printf("WS marshal error: %v", err)
				continue
			}
			var dead []*Client
			for client := range h.clients {
				select {
				case client.send <- data:
				default:
					dead = append(dead, client)
				}
			}
			for _, client := range dead {
				close(client.send)
				delete(h.clients, client)
			}
		}
	}
}

// Broadcast queues a message for fan-out to all clients. It never blocks: real-
// time updates are ephemeral, so if the buffer is full it is far better to drop
// this tick than to stall the agent's report-ingestion path on a backed-up
// fan-out. The next report supersedes a dropped one a second later.
func (h *Hub) Broadcast(v shared.WSMessage) {
	select {
	case h.broadcast <- v:
	default:
	}
}

func HandleWS(hub *Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("WS upgrade error: %v", err)
			return
		}

		client := &Client{hub: hub, conn: conn, send: make(chan []byte, 256)}
		hub.register <- client

		go client.writePump()
		go client.readPump()
	}
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	// Every pong (the client's automatic reply to our pings) pushes the read
	// deadline forward. A silent or half-open connection stops extending it and
	// ReadMessage fails once it lapses, ending the pump.
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			break
		}
	}
}

func (c *Client) writePump() {
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
				// The hub closed the channel: tell the peer and stop.
				c.conn.WriteMessage(websocket.CloseMessage, nil)
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
