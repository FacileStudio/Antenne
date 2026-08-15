package bus

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type connection struct {
	app      RegisteredApp
	socket   *websocket.Conn
	channels map[string]bool
	closed   chan struct{}
	mu       sync.Mutex
}

// send serializes writes: gorilla forbids concurrent writers on one socket, and
// a fan-out plus the emitter's own ack race by construction.
func (c *connection) send(message any) error {
	encoded, err := json.Marshal(message)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.socket.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil {
		return err
	}
	return c.socket.WriteMessage(websocket.TextMessage, encoded)
}

// keepAlive pings until the read loop closes the connection.
//
// The ping is a control frame rather than the protocol's own `{"type":"ping"}`
// because every WebSocket peer answers a control frame automatically: the Go
// client's gorilla socket does it while blocked in ReadMessage, and a browser
// or Bun client does it below the API, with no code on either side. The pong
// that comes back is what proves the peer is alive and what keeps a proxy from
// reaping a silent connection.
//
// WriteControl is the one gorilla method safe to call concurrently with a
// writer, so this deliberately does not take the send mutex — waiting on it
// would let a slow fan-out delay a ping past its deadline.
func (c *connection) keepAlive(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-c.closed:
			return
		case <-ticker.C:
			deadline := time.Now().Add(writeTimeout)
			if err := c.socket.WriteControl(websocket.PingMessage, nil, deadline); err != nil {
				return
			}
		}
	}
}
