package auth

import (
	"net/http"
	"time"

	"github.com/FacileStudio/Antenne/apps/api/internal/authcontext"
	apierrors "github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"
	"github.com/go-chi/chi/v5"
)

type loginRequest struct {
	Password string `json:"password"`
}

type sessionResponse struct {
	Authenticated    bool   `json:"authenticated"`
	PasswordRequired bool   `json:"passwordRequired"`
	Username         string `json:"username"`
}

// RegisterPublicRoutes mounts the routes that must answer before authentication.
func (s *Service) RegisterPublicRoutes(router chi.Router) {
	router.Post("/login", s.handleLogin)
	router.Post("/logout", s.handleLogout)
	router.Get("/session/public", s.handlePublicSession)
}

// RegisterRoutes mounts the routes that require a caller.
func (s *Service) RegisterRoutes(router chi.Router) {
	router.Get("/session", s.handleSession)
}

func (s *Service) handleLogin(w http.ResponseWriter, r *http.Request) {
	if !s.PasswordRequired() {
		httpjson.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
		return
	}

	var body loginRequest
	if err := httpjson.DecodeJSON(w, r, &body); err != nil {
		httpjson.WriteError(w, apierrors.Invalid("Invalid JSON body."))
		return
	}

	token, expires, err := s.Login(authcontext.ClientIP(r), body.Password)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}

	http.SetCookie(w, s.buildCookie(r, token, expires))
	httpjson.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Service) handleLogout(w http.ResponseWriter, r *http.Request) {
	s.Logout(r)
	http.SetCookie(w, s.buildCookie(r, "", time.Unix(0, 0)))
	httpjson.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// handlePublicSession lets the login page ask whether a password is needed at
// all without holding one. It never reports an authenticated caller.
func (s *Service) handlePublicSession(w http.ResponseWriter, r *http.Request) {
	identity, err := s.Authenticate(r)
	httpjson.WriteJSON(w, http.StatusOK, sessionResponse{
		Authenticated:    err == nil && identity.IsAuthenticated(),
		PasswordRequired: s.PasswordRequired(),
		Username:         s.cfg.AdminUsername,
	})
}

func (s *Service) handleSession(w http.ResponseWriter, r *http.Request) {
	httpjson.WriteJSON(w, http.StatusOK, sessionResponse{
		Authenticated:    true,
		PasswordRequired: s.PasswordRequired(),
		Username:         s.cfg.AdminUsername,
	})
}

// buildCookie marks the cookie Secure whenever the request reached us over TLS,
// including through Traefik, which terminates it and forwards plain HTTP.
func (s *Service) buildCookie(r *http.Request, value string, expires time.Time) *http.Cookie {
	secure := r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"

	cookie := &http.Cookie{
		Name:     SessionCookie,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		Expires:  expires,
	}
	if value == "" {
		cookie.MaxAge = -1
	} else {
		cookie.MaxAge = int(time.Until(expires).Seconds())
	}
	return cookie
}
