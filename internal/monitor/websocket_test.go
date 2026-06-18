package monitor

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/michal/kiviq/internal/shared"
)

// dialTestWS starts a server around HandleWS and returns a connected client.
func dialTestWS(t *testing.T, hub *Hub) *websocket.Conn {
	t.Helper()
	srv := httptest.NewServer(HandleWS(hub))
	t.Cleanup(srv.Close)

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

// withShortPingTimings shrinks the keepalive timings for the duration of a test
// so ping/pong behavior is observable in milliseconds rather than minutes.
func withShortPingTimings(t *testing.T) {
	t.Helper()
	oldPing, oldPong := pingPeriod, pongWait
	pingPeriod = 20 * time.Millisecond
	pongWait = 100 * time.Millisecond
	t.Cleanup(func() { pingPeriod, pongWait = oldPing, oldPong })
}

// The server must ping idle clients so a healthy connection stays alive and a
// dead one becomes detectable. Without pings, a half-open TCP connection would
// linger forever.
func TestClientSendsPings(t *testing.T) {
	withShortPingTimings(t)

	hub := NewHub()
	go hub.Run()

	conn := dialTestWS(t, hub)

	gotPing := make(chan struct{}, 1)
	conn.SetPingHandler(func(string) error {
		select {
		case gotPing <- struct{}{}:
		default:
		}
		return nil
	})
	// Reads drive the control-frame handlers (the ping handler fires inside ReadMessage).
	go func() {
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	select {
	case <-gotPing:
	case <-time.After(2 * time.Second):
		t.Fatal("server never sent a ping; a half-open client would linger undetected")
	}
}

// A client that stops responding to pings must be disconnected once the pong
// deadline lapses, instead of holding a slot open indefinitely.
func TestClientDroppedWhenPongStops(t *testing.T) {
	withShortPingTimings(t)

	hub := NewHub()
	go hub.Run()

	conn := dialTestWS(t, hub)

	// Suppress the default pong response so the connection goes silent.
	conn.SetPingHandler(func(string) error { return nil })

	closed := make(chan struct{})
	go func() {
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				close(closed)
				return
			}
		}
	}()

	select {
	case <-closed:
	case <-time.After(2 * time.Second):
		t.Fatal("server never disconnected a client that stopped ponging")
	}
}

func TestHubBroadcast(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	client := &Client{
		hub:  hub,
		send: make(chan []byte, 256),
	}
	hub.register <- client
	time.Sleep(10 * time.Millisecond)

	msg := shared.WSMessage{
		Type: "snapshot",
		Data: &shared.AgentSnapshot{
			AgentID: "a1",
		},
	}
	hub.Broadcast(msg)

	select {
	case data := <-client.send:
		var received shared.WSMessage
		if err := json.Unmarshal(data, &received); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if received.Type != "snapshot" {
			t.Errorf("type = %q, want %q", received.Type, "snapshot")
		}
		if received.Data == nil || received.Data.AgentID != "a1" {
			t.Errorf("data agent_id = %v, want a1", received.Data)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for broadcast")
	}
}

func TestHubUnregister(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	client := &Client{
		hub:  hub,
		send: make(chan []byte, 256),
	}
	hub.register <- client
	time.Sleep(10 * time.Millisecond)

	hub.unregister <- client

	// Unregistering closes the client's send channel — observable proof the hub
	// dropped it, without peeking at internal state.
	select {
	case _, ok := <-client.send:
		if ok {
			t.Error("expected send channel to be closed after unregister")
		}
	case <-time.After(time.Second):
		t.Fatal("send channel was not closed after unregister")
	}
}

// Report ingestion must never block on the realtime fan-out. Even with the
// hub loop not draining and the buffer full, Broadcast returns immediately —
// an overflowed realtime tick is dropped rather than stalling the reporter.
func TestHubBroadcastNeverBlocks(t *testing.T) {
	hub := NewHub() // Run is not started, so nothing drains h.broadcast.

	done := make(chan struct{})
	go func() {
		for i := 0; i < cap(hub.broadcast)+10; i++ {
			hub.Broadcast(shared.WSMessage{Type: "snapshot"})
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Broadcast blocked when the broadcast buffer was full")
	}
}

func TestHubBroadcastToMultiple(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	var clients []*Client
	for i := 0; i < 3; i++ {
		c := &Client{
			hub:  hub,
			send: make(chan []byte, 256),
		}
		hub.register <- c
		clients = append(clients, c)
	}
	time.Sleep(10 * time.Millisecond)

	hub.Broadcast(shared.WSMessage{Type: "snapshot", Data: &shared.AgentSnapshot{}})

	for i, c := range clients {
		select {
		case <-c.send:
		case <-time.After(time.Second):
			t.Fatalf("client %d timed out", i)
		}
	}
}

func TestHubSlowClient(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	slow := &Client{
		hub:  hub,
		send: make(chan []byte, 1),
	}
	hub.register <- slow
	time.Sleep(10 * time.Millisecond)

	hub.Broadcast(shared.WSMessage{Type: "first"})
	time.Sleep(10 * time.Millisecond)

	done := make(chan bool, 1)
	go func() {
		hub.Broadcast(shared.WSMessage{Type: "second"})
		done <- true
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("broadcast blocked on slow client")
	}

	// Let the hub process the second broadcast before reading: draining the
	// buffer too early would free a slot and let the delivery succeed, masking
	// the drop we're testing for.
	time.Sleep(50 * time.Millisecond)

	// A client whose buffer stayed full gets dropped: the hub closes its send
	// channel. Drain the buffered message, then observe the close.
	deadline := time.After(time.Second)
	for {
		select {
		case _, ok := <-slow.send:
			if !ok {
				return // channel closed — slow client was dropped
			}
		case <-deadline:
			t.Fatal("slow client's send channel was never closed (not dropped)")
		}
	}
}
