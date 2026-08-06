package pool

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// serve runs handler over a real WebSocket pair and returns the client half.
// The handler never calls t.Error: httptest stops tracking a connection once it
// is hijacked, so Close does not wait for the handler, and a failed assertion
// there could land after the test finished.
func serve(t *testing.T, handler func(*websocket.Conn)) *websocket.Conn {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		socket, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer socket.Close()
		handler(socket)
	}))
	t.Cleanup(server.Close)

	client, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { client.Close() })

	return client
}

// A pool socket carries nothing while the suite is quiet, and a proxy reaps a
// silent connection — Cloudflare sits in front of nook.facile.studio and culls
// an idle WebSocket at around a hundred seconds, which is what put Opus in a
// reconnect loop all day. The server is what has to keep the connection warm:
// the TypeScript client sends nothing at all, and a ping control frame is the
// only keep-alive every client answers without any code of its own.
func TestAnIdleConnectionIsPinged(t *testing.T) {
	client := serve(t, func(socket *websocket.Conn) {
		conn := &connection{socket: socket, closed: make(chan struct{})}
		defer close(conn.closed)
		go conn.keepAlive(10 * time.Millisecond)
		socket.ReadMessage()
	})

	pinged := make(chan struct{}, 1)
	client.SetPingHandler(func(string) error {
		select {
		case pinged <- struct{}{}:
		default:
		}
		return nil
	})

	// The client never writes, so only the server's ping can end this read.
	go client.ReadMessage()

	select {
	case <-pinged:
	case <-time.After(2 * time.Second):
		t.Fatal("an idle connection was never pinged, so a proxy will reap it")
	}
}

// The ping loop has to end with the connection. A loop that outlives its socket
// leaks a goroutine and a ticker per reconnect, and a client that reconnects
// every couple of minutes turns that into a leak nobody notices until the
// process is weeks old.
//
// The interval is long enough that the ticker never fires, so this needs no
// socket at all.
func TestTheKeepAliveStopsWithTheConnection(t *testing.T) {
	conn := &connection{closed: make(chan struct{})}

	stopped := make(chan struct{})
	go func() {
		conn.keepAlive(time.Hour)
		close(stopped)
	}()

	close(conn.closed)

	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("keepAlive outlived the connection it was pinging")
	}
}

// The protocol's own ping is what the Go client sends, in a text frame rather
// than a control frame, and it is answered by the same switch that routes every
// other frame — which is why it is pinned here.
func TestTheProtocolPingIsStillAnswered(t *testing.T) {
	server := &Server{
		queue:       NewQueue(nil, discardLogger()),
		log:         discardLogger(),
		connections: map[string]*connection{},
		subscribers: map[string]map[string]bool{},
	}

	client := serve(t, func(socket *websocket.Conn) {
		conn := &connection{socket: socket, closed: make(chan struct{})}
		server.readLoop(conn)
	})

	if err := client.WriteJSON(map[string]string{"type": "ping"}); err != nil {
		t.Fatal(err)
	}

	client.SetReadDeadline(time.Now().Add(2 * time.Second))
	var reply struct {
		Type string `json:"type"`
	}
	if err := client.ReadJSON(&reply); err != nil {
		t.Fatal(err)
	}
	if reply.Type != "pong" {
		t.Errorf("a protocol ping was answered with %q, not a pong", reply.Type)
	}
}
