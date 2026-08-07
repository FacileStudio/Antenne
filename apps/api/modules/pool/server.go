package pool

import (
	"crypto/hmac"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"gorm.io/gorm"

	apierrors "github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"
)

const (
	replayBatch    = 100
	writeTimeout   = 10 * time.Second
	readLimitBytes = 4 << 20

	// A pool socket carries nothing at all while the suite is quiet, and every
	// proxy in front of Antenne culls an idle connection — Cloudflare at 100
	// seconds, Traefik at 180. The interval matches the SSE keep-alive for the
	// same reason, and the deadline is three of them so a single dropped ping
	// does not evict a healthy client.
	pingInterval = 20 * time.Second
	pongTimeout  = 60 * time.Second
)

// Activity is a pool lifecycle event, reported to the activity log.
type Activity struct {
	Type   string
	App    string
	Detail string
}

// ActivityLogger receives pool lifecycle events. It may be nil.
type ActivityLogger func(Activity)

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
	c.socket.SetWriteDeadline(time.Now().Add(writeTimeout))
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

// Server is the pool's WebSocket half.
type Server struct {
	queue    *Queue
	log      *slog.Logger
	upgrader websocket.Upgrader

	mu          sync.RWMutex
	secret      string
	connections map[string]*connection
	subscribers map[string]map[string]bool
	activity    ActivityLogger
}

// NewServer builds the pool server.
func NewServer(db *gorm.DB, log *slog.Logger, secret string) *Server {
	if secret == "" {
		secret = "pool"
	}
	if secret == "pool" {
		log.Warn("pool is using the default secret 'pool' — set POOL_SECRET in production")
	}

	return &Server{
		queue:       NewQueue(db, log),
		log:         log,
		secret:      secret,
		connections: map[string]*connection{},
		subscribers: map[string]map[string]bool{},
		// Origin is not checked: pool clients are servers, not browsers, and
		// they authenticate with a token in the query string. A browser origin
		// check would protect nothing here and would break every Go client.
		upgrader: websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }},
	}
}

// SetSecret swaps the registration secret, after a dashboard save.
func (s *Server) SetSecret(secret string) {
	if secret == "" {
		secret = "pool"
	}
	s.mu.Lock()
	s.secret = secret
	s.mu.Unlock()
}

// OnActivity installs the activity logger.
func (s *Server) OnActivity(logger ActivityLogger) {
	s.mu.Lock()
	s.activity = logger
	s.mu.Unlock()
}

// Cleanup applies the retention policy.
func (s *Server) Cleanup() { s.queue.Cleanup() }

// Stats describes the live pool, for the dashboard.
type Stats struct {
	Epoch       string     `json:"epoch"`
	Connections int        `json:"connections"`
	Apps        []StatsApp `json:"apps"`
}

// StatsApp is one connected app.
type StatsApp struct {
	AppID    string   `json:"app_id"`
	App      string   `json:"app"`
	Channels []string `json:"channels"`
}

// Stats returns the live pool state.
func (s *Server) Stats() Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := Stats{Epoch: s.queue.Epoch(), Connections: len(s.connections), Apps: []StatsApp{}}
	for _, conn := range s.connections {
		channels := make([]string, 0, len(conn.channels))
		for channel := range conn.channels {
			channels = append(channels, channel)
		}
		stats.Apps = append(stats.Apps, StatsApp{AppID: conn.app.AppID, App: conn.app.App, Channels: channels})
	}
	return stats
}

type registrationRequest struct {
	App        string `json:"app"`
	InstanceID string `json:"instance_id"`
	Secret     string `json:"secret"`
	Events     struct {
		Emit   []string `json:"emit"`
		Listen []string `json:"listen"`
	} `json:"events"`
}

// HandleRegister exchanges the shared secret for a connection token.
func (s *Server) HandleRegister(w http.ResponseWriter, r *http.Request) {
	var body registrationRequest
	if err := httpjson.DecodeJSON(w, r, &body); err != nil {
		httpjson.WriteError(w, apierrors.Invalid("invalid json"))
		return
	}
	if body.App == "" {
		httpjson.WriteError(w, apierrors.Invalid("app is required"))
		return
	}

	s.mu.RLock()
	secret := s.secret
	s.mu.RUnlock()

	if !hmac.Equal([]byte(body.Secret), []byte(secret)) {
		httpjson.WriteError(w, apierrors.Forbidden("invalid secret"))
		return
	}

	identity := body.App
	if body.InstanceID != "" {
		identity = body.App + ":" + body.InstanceID
	}

	emit := body.Events.Emit
	if emit == nil {
		emit = []string{"*"}
	}
	listen := body.Events.Listen
	if listen == nil {
		listen = []string{"*"}
	}

	registered, err := s.queue.Register(identity, emit, listen)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	s.queue.Touch(registered.AppID)

	s.log.Info("pool app registered",
		slog.String("app", registered.App),
		slog.String("app_id", registered.AppID))

	httpjson.WriteJSON(w, http.StatusOK, map[string]string{
		"token":  registered.Token,
		"app_id": registered.AppID,
	})
}

// HandleWebSocket upgrades a connection and serves the pool protocol on it.
func (s *Server) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		httpjson.WriteError(w, apierrors.Unauthorized("token required"))
		return
	}

	app, ok := s.queue.ByToken(token)
	if !ok {
		httpjson.WriteError(w, apierrors.Forbidden("invalid token"))
		return
	}

	socket, err := s.upgrader.Upgrade(hijackable(w), r, nil)
	if err != nil {
		s.log.Error("pool websocket upgrade failed", slog.Any("error", err))
		return
	}
	socket.SetReadLimit(readLimitBytes)
	socket.SetReadDeadline(time.Now().Add(pongTimeout))
	socket.SetPongHandler(func(string) error {
		return socket.SetReadDeadline(time.Now().Add(pongTimeout))
	})

	conn := &connection{app: app, socket: socket, channels: map[string]bool{}, closed: make(chan struct{})}
	s.attach(conn)

	if err := conn.send(map[string]string{
		"type":   "welcome",
		"epoch":  s.queue.Epoch(),
		"app_id": app.AppID,
	}); err != nil {
		s.detach(conn)
		socket.Close()
		return
	}

	s.log.Info("pool app connected", slog.String("app", app.App), slog.String("app_id", app.AppID))
	s.report(Activity{Type: "pool.connected", App: app.App, Detail: app.App + " joined the pool"})

	go conn.keepAlive(pingInterval)
	go s.readLoop(conn)
}

func (s *Server) attach(conn *connection) {
	s.mu.Lock()
	// One live connection per app: a reconnect that arrives before the old
	// socket's close is noticed would otherwise get events twice.
	if stale, ok := s.connections[conn.app.AppID]; ok && stale != conn {
		s.removeLocked(stale)
		go stale.socket.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, "replaced by new connection"),
			time.Now().Add(time.Second))
	}
	s.connections[conn.app.AppID] = conn
	s.mu.Unlock()
}

func (s *Server) detach(conn *connection) {
	s.mu.Lock()
	if current, ok := s.connections[conn.app.AppID]; ok && current == conn {
		s.removeLocked(conn)
	}
	s.mu.Unlock()
}

func (s *Server) removeLocked(conn *connection) {
	for channel := range conn.channels {
		if subscribers, ok := s.subscribers[channel]; ok {
			delete(subscribers, conn.app.AppID)
			if len(subscribers) == 0 {
				delete(s.subscribers, channel)
			}
		}
	}
	delete(s.connections, conn.app.AppID)
}

func (s *Server) readLoop(conn *connection) {
	defer func() {
		close(conn.closed)
		s.detach(conn)
		conn.socket.Close()
		s.log.Info("pool app disconnected",
			slog.String("app", conn.app.App),
			slog.String("app_id", conn.app.AppID))
		s.report(Activity{Type: "pool.disconnected", App: conn.app.App, Detail: conn.app.App + " left the pool"})
	}()

	for {
		_, raw, err := conn.socket.ReadMessage()
		if err != nil {
			return
		}
		s.handleFrame(conn, raw)
		// Refreshed after the frame is handled, not before: a subscribe replays
		// the backlog inline, and every send in it can block for writeTimeout,
		// so a large replay would otherwise burn the whole budget and the next
		// read would fail the instant the client was finally caught up.
		//
		// Any frame counts, not just a pong — the Go client pings in the
		// protocol rather than in control frames.
		conn.socket.SetReadDeadline(time.Now().Add(pongTimeout))
	}
}

func (s *Server) handleFrame(conn *connection, raw []byte) {
	var frame struct {
		Type string `json:"type"`
	}
	if json.Unmarshal(raw, &frame) != nil {
		s.sendError(conn, "PARSE_ERROR", "invalid json")
		return
	}

	switch frame.Type {
	case "subscribe":
		s.handleSubscribe(conn, raw)
	case "event":
		s.handleEvent(conn, raw)
	case "ack":
		s.handleAck(conn, raw)
	case "ping":
		conn.send(map[string]string{"type": "pong"})
	default:
		s.sendError(conn, "UNKNOWN_TYPE", "unknown message type: "+frame.Type)
	}
}

func (s *Server) handleSubscribe(conn *connection, raw []byte) {
	var frame struct {
		Channels map[string]struct {
			LastOffset int64 `json:"last_offset"`
		} `json:"channels"`
	}
	if json.Unmarshal(raw, &frame) != nil {
		s.sendError(conn, "PARSE_ERROR", "invalid subscribe frame")
		return
	}

	confirmed := map[string]map[string]int64{}

	for channel, options := range frame.Channels {
		s.subscribe(conn, channel)

		// Resume from whichever cursor is further along: the client's own, or
		// the server-side ack cursor for when the client lost its offsets.
		cursor := options.LastOffset
		if acked := s.queue.LastAcked(conn.app.AppID, channel); acked > cursor {
			cursor = acked
		}

		for {
			batch, err := s.queue.After(channel, cursor, replayBatch)
			if err != nil {
				s.log.Error("pool replay failed", slog.String("channel", channel), slog.Any("error", err))
				break
			}
			for _, event := range batch {
				// An app never receives its own emissions back.
				if event.Sender == conn.app.App {
					continue
				}
				conn.send(eventFrame(event))
			}
			if len(batch) > 0 {
				cursor = batch[len(batch)-1].Offset
			}
			if len(batch) < replayBatch {
				break
			}
		}

		confirmed[channel] = map[string]int64{"offset": cursor}
	}

	conn.send(map[string]any{"type": "subscribed", "channels": confirmed})
}

func (s *Server) handleEvent(conn *connection, raw []byte) {
	var frame struct {
		Channel string          `json:"channel"`
		Payload json.RawMessage `json:"payload"`
	}
	if json.Unmarshal(raw, &frame) != nil || frame.Channel == "" {
		s.sendError(conn, "INVALID_EVENT", "channel is required")
		return
	}
	if len(frame.Payload) == 0 {
		frame.Payload = json.RawMessage("null")
	}

	event, err := s.queue.Push(frame.Channel, frame.Payload, conn.app.App)
	if err != nil {
		s.log.Error("pool push failed", slog.String("channel", frame.Channel), slog.Any("error", err))
		s.sendError(conn, "PUSH_FAILED", "could not queue the event")
		return
	}

	recipients := s.fanOut(conn, event)
	conn.send(map[string]any{"type": "ack", "channel": event.Channel, "offset": event.Offset})

	s.report(Activity{
		Type:   "pool.event",
		App:    conn.app.App,
		Detail: conn.app.App + " → " + event.Channel + " (" + pluralRecipients(recipients) + ")",
	})
}

func (s *Server) handleAck(conn *connection, raw []byte) {
	var frame struct {
		Channel string `json:"channel"`
		Offset  int64  `json:"offset"`
	}
	if json.Unmarshal(raw, &frame) != nil || frame.Channel == "" {
		return
	}
	if err := s.queue.Ack(conn.app.AppID, frame.Channel, frame.Offset); err != nil {
		s.log.Error("pool ack failed", slog.Any("error", err))
	}
	s.queue.Touch(conn.app.AppID)
}

func (s *Server) subscribe(conn *connection, channel string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	conn.channels[channel] = true
	subscribers, ok := s.subscribers[channel]
	if !ok {
		subscribers = map[string]bool{}
		s.subscribers[channel] = subscribers
	}
	subscribers[conn.app.AppID] = true
}

func (s *Server) fanOut(sender *connection, event Event) int {
	s.mu.RLock()
	targets := make([]*connection, 0)
	for appID := range s.subscribers[event.Channel] {
		if appID == sender.app.AppID {
			continue
		}
		if conn, ok := s.connections[appID]; ok {
			targets = append(targets, conn)
		}
	}
	s.mu.RUnlock()

	frame := eventFrame(event)
	for _, target := range targets {
		target.send(frame)
	}
	return len(targets)
}

func (s *Server) sendError(conn *connection, code, message string) {
	conn.send(map[string]string{"type": "error", "code": code, "message": message})
}

func (s *Server) report(activity Activity) {
	s.mu.RLock()
	logger := s.activity
	s.mu.RUnlock()
	if logger != nil {
		logger(activity)
	}
}

// hijackable digs the raw ResponseWriter out of the middleware chain.
//
// A WebSocket upgrade needs http.Hijacker, and gorilla type-asserts it on the
// writer it is handed rather than following Unwrap. tronc's request logger
// wraps the writer to record the status and implements Unwrap but not Hijack,
// so passing it straight through fails every upgrade with a 500. Unwrapping
// costs this one request its log line, which a hijacked connection has no
// meaningful status for anyway.
func hijackable(w http.ResponseWriter) http.ResponseWriter {
	for {
		if _, ok := w.(http.Hijacker); ok {
			return w
		}
		unwrapper, ok := w.(interface{ Unwrap() http.ResponseWriter })
		if !ok {
			return w
		}
		w = unwrapper.Unwrap()
	}
}

func eventFrame(event Event) map[string]any {
	return map[string]any{
		"type":      "event",
		"id":        event.ID,
		"channel":   event.Channel,
		"offset":    event.Offset,
		"payload":   event.Payload,
		"timestamp": event.Timestamp,
		"sender":    event.Sender,
	}
}

func pluralRecipients(count int) string {
	if count == 1 {
		return "1 recipient"
	}
	return strconv.Itoa(count) + " recipients"
}
