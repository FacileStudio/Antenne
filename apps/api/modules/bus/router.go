package bus

import (
	"crypto/hmac"
	"log/slog"
	"net/http"
	"time"

	apierrors "github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"
	"github.com/go-chi/chi/v5"
)

// RegisterRoutes mounts the pool's own endpoints.
//
// They authenticate with the pool secret and a connection token rather than with
// a dashboard session, so they sit outside RequireAuth. They are mounted twice on
// purpose: at the root, where every pool client pins them, and under /api, which
// is what the dashboard's dev proxy reaches.
func (s *Server) RegisterRoutes(router chi.Router) {
	router.Post("/antenne/register", s.HandleRegister)
	router.Get("/antenne/ws", s.HandleWebSocket)
}

// RegisterStats mounts the pool's dashboard endpoint, which does need a session.
func (s *Server) RegisterStats(router chi.Router) {
	router.Get("/antenne/stats", func(w http.ResponseWriter, r *http.Request) {
		httpjson.WriteJSON(w, http.StatusOK, s.Stats())
	})
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

// declaredChannels resolves what a registration payload left out.
//
// Declaring no listeners means "I do not consume", not "send me everything".
// The wildcard default was the opposite reading and it is not free: retention
// here is interest-based, so an app recorded as listening to `*` pins every
// channel at its last acked offset — and an app that never subscribes never
// acks, so it pinned every channel at zero. Jardin, which only emits agent
// sessions, was holding the whole queue that way. A client that genuinely wants
// everything writes ["*"] and says so.
//
// Emit keeps its wildcard: an emitter that declared no channels still has to be
// able to publish, and this registration is not what authorises it.
func declaredChannels(emit, listen []string) ([]string, []string) {
	if emit == nil {
		emit = []string{"*"}
	}
	if listen == nil {
		listen = []string{}
	}
	return emit, listen
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

	emit, listen := declaredChannels(body.Events.Emit, body.Events.Listen)

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
