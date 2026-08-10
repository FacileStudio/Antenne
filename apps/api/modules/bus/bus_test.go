package bus

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
// silent connection — Cloudflare sits in front of antenne.facile.studio and culls
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

// Mycelium emits sessions and subscribes to nothing, which is a healthy
// configuration and used to render as "No channel" — a working app reported as
// a broken one. Stats has to carry what an app produces, not only what it
// consumes.
func TestStatsReportsEmitOnlyApps(t *testing.T) {
	server := &Server{
		queue:       &Queue{},
		connections: map[string]*connection{},
	}
	server.connections["mycelium-1"] = &connection{
		app: RegisteredApp{
			AppID:        "mycelium-1",
			App:          "Mycelium",
			EmitChannels: []string{"usage_alert.created", "agent_session.created"},
		},
		channels: map[string]bool{},
	}
	server.connections["sablier-1"] = &connection{
		app: RegisteredApp{
			AppID:          "sablier-1",
			App:            "Sablier",
			EmitChannels:   []string{"time_entry.created"},
			ListenChannels: []string{"agent_session.created"},
		},
		channels: map[string]bool{"agent_session.created": true},
	}

	byApp := map[string]StatsApp{}
	for _, app := range server.Stats().Apps {
		byApp[app.App] = app
	}

	mycelium := byApp["Mycelium"]
	if len(mycelium.Channels) != 0 {
		t.Fatalf("Mycelium subscribes to %v, want nothing", mycelium.Channels)
	}
	if len(mycelium.Emits) != 2 || mycelium.Emits[0] != "agent_session.created" {
		t.Fatalf("Mycelium emits = %v, want both channels sorted", mycelium.Emits)
	}

	sablier := byApp["Sablier"]
	if len(sablier.Channels) != 1 || len(sablier.Emits) != 1 {
		t.Fatalf("Sablier = %+v, want one of each", sablier)
	}
}

// A nil EmitChannels must serialise as [] and not null, or the client's
// `app.emits.length` throws on the one app that declares nothing.
func TestStatsNeverReportsNilChannels(t *testing.T) {
	server := &Server{
		queue:       &Queue{},
		connections: map[string]*connection{"x": {app: RegisteredApp{AppID: "x", App: "X"}, channels: map[string]bool{}}},
	}

	app := server.Stats().Apps[0]
	if app.Emits == nil || app.Channels == nil {
		t.Fatalf("Stats returned nil slices: %+v", app)
	}
}
