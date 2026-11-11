package bus

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"

	"github.com/FacileStudio/Antenne/apps/api/internal/pgtest"
)

const wireSecret = "wire-secret"

type wirePool struct {
	t      *testing.T
	server *Server
	http   *httptest.Server
}

type wireApp struct {
	id     string
	token  string
	socket *websocket.Conn
}

// newWirePool serves the pool routes at the root and under /api, the way
// BuildRouter mounts them, over a migrated Postgres schema.
func newWirePool(t *testing.T) *wirePool {
	t.Helper()
	return wirePoolOn(t, pgtest.New(t), wireSecret)
}

// wirePoolOn boots a pool over db with secret, the way a restart does.
func wirePoolOn(t *testing.T, db *gorm.DB, secret string) *wirePool {
	t.Helper()
	server := NewServer(db, discardLogger(), secret)
	router := chi.NewRouter()
	server.RegisterRoutes(router)
	router.Route("/api", server.RegisterRoutes)
	httpServer := httptest.NewServer(router)
	t.Cleanup(httpServer.Close)
	return &wirePool{t: t, server: server, http: httpServer}
}

// post sends body to path and returns the status and the raw response body.
func (p *wirePool) post(path, body string) (int, string) {
	p.t.Helper()
	response, err := http.Post(p.http.URL+path, "application/json", strings.NewReader(body))
	if err != nil {
		p.t.Fatal(err)
	}
	defer response.Body.Close()
	raw, _ := io.ReadAll(response.Body)
	return response.StatusCode, string(raw)
}

// register exchanges the secret for a token and pins the response's shape.
func (p *wirePool) register(path, body string) wireApp {
	p.t.Helper()
	status, raw := p.post(path, body)
	var app struct {
		AppID string `json:"app_id"`
		Token string `json:"token"`
	}
	if err := json.Unmarshal([]byte(raw), &app); err != nil || status != http.StatusOK {
		p.t.Fatalf("register = %d %s", status, raw)
	}
	want := fmt.Sprintf(`{"app_id":%q,"token":%q}`+"\n", app.AppID, app.Token)
	if raw != want {
		p.t.Fatalf("register body = %q, want %q", raw, want)
	}
	return wireApp{id: app.AppID, token: app.Token}
}

// dial opens the socket at path and pins the welcome frame.
func (p *wirePool) dial(path string, app wireApp) wireApp {
	p.t.Helper()
	url := "ws" + strings.TrimPrefix(p.http.URL, "http") + path + "?token=" + app.token
	socket, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		p.t.Fatal(err)
	}
	p.t.Cleanup(func() { socket.Close() })
	app.socket = socket
	p.expect(app, fmt.Sprintf(`{"app_id":%q,"epoch":%q,"type":"welcome"}`, app.id, p.server.queue.Epoch()))
	return app
}

// read returns the next frame's bytes exactly as the server wrote them.
func (p *wirePool) read(app wireApp) string {
	p.t.Helper()
	app.socket.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, raw, err := app.socket.ReadMessage()
	if err != nil {
		p.t.Fatal(err)
	}
	return string(raw)
}

// expect fails unless the next frame is byte for byte want.
func (p *wirePool) expect(app wireApp, want string) {
	p.t.Helper()
	if got := p.read(app); got != want {
		p.t.Fatalf("frame = %s\nwant    %s", got, want)
	}
}

// send writes one text frame.
func (p *wirePool) send(app wireApp, frame string) {
	p.t.Helper()
	if err := app.socket.WriteMessage(websocket.TextMessage, []byte(frame)); err != nil {
		p.t.Fatal(err)
	}
}

// expectEvent pins an event frame, reading the id and timestamp the server
// assigned and checking every other byte.
func (p *wirePool) expectEvent(app wireApp, channel string, offset int64, payload, sender string) {
	p.t.Helper()
	raw := p.read(app)
	var frame struct {
		ID        string `json:"id"`
		Timestamp int64  `json:"timestamp"`
	}
	if err := json.Unmarshal([]byte(raw), &frame); err != nil {
		p.t.Fatal(err)
	}
	want := fmt.Sprintf(`{"channel":%q,"id":%q,"offset":%d,"payload":%s,"sender":%q,"timestamp":%d,"type":"event"}`,
		channel, frame.ID, offset, payload, sender, frame.Timestamp)
	if raw != want {
		p.t.Fatalf("event = %s\nwant    %s", raw, want)
	}
}

// The registration endpoint is pinned by every pool client in the suite. Its
// refusals and its answer must not change by a byte, at either mount.
func TestWireRegistration(t *testing.T) {
	pool := newWirePool(t)
	for _, prefix := range []string{"", "/api"} {
		refusals := map[string]string{
			`{"app":"alpha","secret":"wrong"}`: `403 {"error":{"code":"permission_denied","message":"invalid secret"}}`,
			`{"secret":"wire-secret"}`:         `400 {"error":{"code":"invalid_argument","message":"app is required"}}`,
			`not json`:                         `400 {"error":{"code":"invalid_argument","message":"invalid json"}}`,
		}
		for body, want := range refusals {
			status, raw := pool.post(prefix+"/antenne/register", body)
			if got := fmt.Sprintf("%d %s", status, strings.TrimSuffix(raw, "\n")); got != want {
				t.Errorf("%s register %s = %s, want %s", prefix, body, got, want)
			}
		}
		pool.register(prefix+"/antenne/register", `{"app":"alpha","secret":"wire-secret"}`)
	}
}

// The socket refuses a missing or unknown token with the same bodies as
// before, at either mount.
func TestWireSocketRefusals(t *testing.T) {
	pool := newWirePool(t)
	for _, prefix := range []string{"", "/api"} {
		cases := map[string]string{
			"":         `401 {"error":{"code":"unauthenticated","message":"token required"}}`,
			"?token=x": `403 {"error":{"code":"permission_denied","message":"invalid token"}}`,
		}
		for query, want := range cases {
			response, err := http.Get(pool.http.URL + prefix + "/antenne/ws" + query)
			if err != nil {
				t.Fatal(err)
			}
			raw, _ := io.ReadAll(response.Body)
			response.Body.Close()
			if got := fmt.Sprintf("%d %s", response.StatusCode, strings.TrimSuffix(string(raw), "\n")); got != want {
				t.Errorf("%s ws%s = %s, want %s", prefix, query, got, want)
			}
		}
	}
}

// Every frame the server writes, pinned byte for byte: welcome with the
// epoch, subscribed with the cursor, pong, the three error codes, the ack an
// emitter gets, the event a listener gets live, the cursor after an ack, and
// an event Antenne publishes itself.
func TestWireFrames(t *testing.T) {
	pool := newWirePool(t)
	listener := pool.dial("/antenne/ws", pool.register("/antenne/register",
		`{"app":"alpha","secret":"wire-secret","events":{"listen":["git.push"]}}`))
	emitter := pool.dial("/api/antenne/ws", pool.register("/api/antenne/register",
		`{"app":"beta","instance_id":"one","secret":"wire-secret","events":{"emit":["git.push"]}}`))

	pool.send(listener, `{"type":"subscribe","channels":{"git.push":{"last_offset":0}}}`)
	pool.expect(listener, `{"channels":{"git.push":{"offset":0}},"type":"subscribed"}`)

	pool.send(listener, `{"type":"ping"}`)
	pool.expect(listener, `{"type":"pong"}`)
	pool.send(listener, `nope`)
	pool.expect(listener, `{"code":"PARSE_ERROR","message":"invalid json","type":"error"}`)
	pool.send(listener, `{"type":"shout"}`)
	pool.expect(listener, `{"code":"UNKNOWN_TYPE","message":"unknown message type: shout","type":"error"}`)
	pool.send(listener, `{"type":"event","payload":{}}`)
	pool.expect(listener, `{"code":"INVALID_EVENT","message":"channel is required","type":"error"}`)

	pool.send(emitter, `{"type":"event","channel":"git.push","payload":{"ref":"main"}}`)
	pool.expect(emitter, `{"channel":"git.push","offset":1,"type":"ack"}`)
	pool.expectEvent(listener, "git.push", 1, `{"ref":"main"}`, "beta:one")

	pool.send(listener, `{"type":"ack","channel":"git.push","offset":1}`)
	pool.send(listener, `{"type":"subscribe","channels":{"git.push":{"last_offset":0}}}`)
	pool.expect(listener, `{"channels":{"git.push":{"offset":1}},"type":"subscribed"}`)

	if _, err := pool.server.Publish("antenne", "git.push", json.RawMessage(`{"ref":"dev"}`)); err != nil {
		t.Fatal(err)
	}
	pool.expectEvent(listener, "git.push", 2, `{"ref":"dev"}`, "antenne")
}

// A listener that reconnects with an old cursor is replayed what it missed,
// in the same event frame, before the subscribed confirmation.
func TestWireReplay(t *testing.T) {
	pool := newWirePool(t)
	registration := pool.register("/antenne/register", `{"app":"alpha","secret":"wire-secret","events":{"listen":["*"]}}`)
	for _, ref := range []string{"a", "b"} {
		if _, err := pool.server.Publish("sonde", "monitor.down", json.RawMessage(`{"ref":"`+ref+`"}`)); err != nil {
			t.Fatal(err)
		}
	}
	listener := pool.dial("/antenne/ws", registration)
	pool.send(listener, `{"type":"subscribe","channels":{"monitor.down":{"last_offset":1}}}`)
	pool.expectEvent(listener, "monitor.down", 2, `{"ref":"b"}`, "sonde")
	pool.expect(listener, `{"channels":{"monitor.down":{"offset":2}},"type":"subscribed"}`)
}
