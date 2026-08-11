package auth

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/FacileStudio/Antenne/apps/api/internal/authcontext"
	"github.com/FacileStudio/Antenne/apps/api/internal/ratelimit"
	apierrors "github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"
)

type sessionResponse struct {
	Authenticated    bool   `json:"authenticated"`
	PasswordRequired bool   `json:"passwordRequired"`
	Username         string `json:"username"`
	Email            string `json:"email"`
}

// RegisterRoutes mounts the routes porte does not own.
//
// The login, the logout and the whole OIDC flow come from the kits. What is
// left is the session read the dashboard polls, which answers from Antenne's
// own users table.
func (s *Service) RegisterRoutes(router chi.Router) {
	router.Get("/session", s.handleSession)
}

// ConfigExtra adds Antenne's own keys to porte's GET /auth/config.
//
// porte serves sso_only and oidc_enabled there and owns the route, so an app
// with a third key either loses it or registers the path twice and chi panics
// at boot. The login page draws its password form on password_required, which
// is false on an instance that only federates.
func (s *Service) ConfigExtra() func() map[string]any {
	return func() map[string]any {
		return map[string]any{
			"password_required": s.PasswordRequired(),
			"username":          s.cfg.AdminUsername,
			"email":             s.AdminEmail(),
		}
	}
}

func (s *Service) handleSession(w http.ResponseWriter, r *http.Request) {
	identity := authcontext.From(r)
	httpjson.WriteJSON(w, http.StatusOK, sessionResponse{
		Authenticated:    identity.IsAuthenticated(),
		PasswordRequired: s.PasswordRequired(),
		Username:         s.cfg.AdminUsername,
		Email:            identity.Email,
	})
}

// ThrottleLogins keeps the per-address login throttle porte does not carry.
//
// porte equalises the timing of a wrong password and an unknown address, which
// closes the enumeration oracle but does nothing about volume: without this a
// single-account appliance with one guessable address is an offline dictionary
// attack served over HTTP. Five failures per address per ten minutes is what
// Antenne enforced before porte and it is unchanged.
//
// The bucket is keyed on the client address, and the outcome is read off the
// status code because the kit owns the handler: anything but a 2xx from a login
// attempt counts against it, and a success clears it.
func ThrottleLogins() func(http.Handler) http.Handler {
	limiter := ratelimit.New(5, 10*time.Minute, 10*time.Minute)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clientIP := authcontext.ClientIP(r)
			if limiter.Blocked(clientIP) {
				httpjson.WriteError(w, apierrors.RateLimited("too many attempts"))
				return
			}
			recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(recorder, r)
			if recorder.status >= http.StatusBadRequest {
				limiter.Fail(clientIP)
				return
			}
			limiter.Reset(clientIP)
		})
	}
}

// statusRecorder remembers the status a handler wrote, so the throttle can tell
// a refused login from an accepted one without duplicating the kit's logic.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}
