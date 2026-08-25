package bus

import (
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"gorm.io/gorm"
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
		log.Warn("pool is using the default secret 'pool' — set ANTENNE_SECRET in production")
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
//
// Channels is what the app subscribes to and Emits is what it declared it
// produces. Both are reported because an emit-only client — Mycelium, which
// pushes sessions and consumes nothing — legitimately subscribes to no channel,
// and a view showing only subscriptions renders that healthy state as an empty
// one. "No channel" is true and reads like a fault.
type StatsApp struct {
	AppID    string   `json:"app_id"`
	App      string   `json:"app"`
	Channels []string `json:"channels"`
	Emits    []string `json:"emits"`
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
		emits := conn.app.EmitChannels
		if emits == nil {
			emits = []string{}
		}
		sort.Strings(channels)
		sort.Strings(emits)
		stats.Apps = append(stats.Apps, StatsApp{AppID: conn.app.AppID, App: conn.app.App, Channels: channels, Emits: emits})
	}
	return stats
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
