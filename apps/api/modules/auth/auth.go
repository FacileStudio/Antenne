// Package auth authenticates dashboard callers.
//
// The implementation is deliberately thin and sits behind one interface. Antenne
// owns a users table, an opaque session token hashed at rest, and a password
// login — no OIDC, no shared identity engine. When the suite's unified auth
// package arrives, it replaces Service as the Authenticator and supplies its own
// login routes; RequireAuth, authcontext.Identity and every handler that reads
// one stay exactly as they are.
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/FacileStudio/Antenne/apps/api/internal/authcontext"
	"github.com/FacileStudio/Antenne/apps/api/internal/ratelimit"
	apierrors "github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"
)

// SessionCookie is the cookie the dashboard session travels in.
const SessionCookie = "antenne_session"

const sessionLifetime = 30 * 24 * time.Hour

// ErrUnauthenticated is returned when a request carries no usable credential.
var ErrUnauthenticated = errors.New("auth: unauthenticated")

// Authenticator resolves a request to an identity. It is the seam the shared
// auth package will slot into.
type Authenticator interface {
	Authenticate(r *http.Request) (authcontext.Identity, error)
	// Optional reports whether an unauthenticated caller is still allowed
	// through. Antenne leaves the dashboard open when no admin password is set,
	// which is how a fresh local install is usable at all.
	Optional() bool
}

// User is a dashboard account.
type User struct {
	ID           uint   `gorm:"primaryKey"`
	Email        string `gorm:"uniqueIndex"`
	Name         string
	PasswordHash string
	Role         string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// TableName pins the table so a future GORM naming change cannot rename it.
func (User) TableName() string { return "users" }

// Session is a live login. Only the hash of the token is stored, so a database
// leak does not hand over live sessions.
type Session struct {
	TokenHash string `gorm:"primaryKey"`
	UserID    uint
	CreatedAt time.Time
	ExpiresAt time.Time
	LastSeen  time.Time
}

// TableName pins the sessions table name.
func (Session) TableName() string { return "sessions" }

// Config is what Service needs from the environment.
type Config struct {
	AdminUsername string
	AdminPassword string
}

// Service is Antenne's own Authenticator.
type Service struct {
	db     *gorm.DB
	log    *slog.Logger
	cfg    Config
	logins *ratelimit.Limiter
}

// NewService builds the service and ensures the bootstrap admin exists.
func NewService(db *gorm.DB, log *slog.Logger, cfg Config) (*Service, error) {
	service := &Service{
		db:     db,
		log:    log,
		cfg:    cfg,
		logins: ratelimit.New(5, 10*time.Minute, 10*time.Minute),
	}
	if err := service.ensureAdmin(); err != nil {
		return nil, err
	}
	return service, nil
}

// PasswordRequired reports whether login is enforced at all.
func (s *Service) PasswordRequired() bool { return s.cfg.AdminPassword != "" }

// Optional implements Authenticator: with no admin password configured, every
// caller is served as the admin.
func (s *Service) Optional() bool { return !s.PasswordRequired() }

// Username is the login label the dashboard shows.
func (s *Service) Username() string { return s.cfg.AdminUsername }

// ensureAdmin seeds or re-syncs the bootstrap account from the environment.
//
// ANTENNE_ADMIN_PASSWORD stays the source of truth so an operator can recover
// access by changing one variable and restarting, exactly as before the users
// table existed.
func (s *Service) ensureAdmin() error {
	if !s.PasswordRequired() {
		return nil
	}

	email := s.cfg.AdminUsername
	if !strings.Contains(email, "@") {
		email = email + "@antenne.local"
	}

	hash, err := bcrypt.GenerateFromPassword(prehash(s.cfg.AdminPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	var existing User
	err = s.db.Where("email = ?", email).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.db.Create(&User{
			Email:        email,
			Name:         s.cfg.AdminUsername,
			PasswordHash: string(hash),
			Role:         string(authcontext.RoleAdmin),
		}).Error
	}
	if err != nil {
		return err
	}

	if bcrypt.CompareHashAndPassword([]byte(existing.PasswordHash), prehash(s.cfg.AdminPassword)) == nil {
		return nil
	}

	s.log.Info("admin password changed in the environment, re-seeding the account and revoking its sessions")
	if err := s.db.Model(&existing).Updates(map[string]any{"password_hash": string(hash)}).Error; err != nil {
		return err
	}
	return s.db.Where("user_id = ?", existing.ID).Delete(&Session{}).Error
}

// Authenticate implements Authenticator.
func (s *Service) Authenticate(r *http.Request) (authcontext.Identity, error) {
	clientIP := authcontext.ClientIP(r)

	if !s.PasswordRequired() {
		return authcontext.Identity{
			UserID:   1,
			Email:    s.cfg.AdminUsername,
			Name:     s.cfg.AdminUsername,
			Role:     authcontext.RoleAdmin,
			ClientIP: clientIP,
		}, nil
	}

	cookie, err := r.Cookie(SessionCookie)
	if err != nil || cookie.Value == "" {
		return authcontext.Identity{}, ErrUnauthenticated
	}

	var session Session
	if err := s.db.Where("token_hash = ?", hashToken(cookie.Value)).First(&session).Error; err != nil {
		return authcontext.Identity{}, ErrUnauthenticated
	}
	if session.ExpiresAt.Before(time.Now()) {
		s.db.Delete(&session)
		return authcontext.Identity{}, ErrUnauthenticated
	}

	var user User
	if err := s.db.First(&user, session.UserID).Error; err != nil {
		return authcontext.Identity{}, ErrUnauthenticated
	}

	// One write per minute at most: touching last_seen on every request would
	// turn a read-only dashboard poll into a write.
	if time.Since(session.LastSeen) > time.Minute {
		s.db.Model(&session).Update("last_seen", time.Now())
	}

	return authcontext.Identity{
		UserID:   user.ID,
		Email:    user.Email,
		Name:     user.Name,
		Role:     authcontext.Role(user.Role),
		ClientIP: clientIP,
	}, nil
}

// Login verifies a password and issues a session token.
func (s *Service) Login(clientIP, password string) (string, time.Time, error) {
	if !s.PasswordRequired() {
		return "", time.Time{}, apierrors.Failed("no admin password is configured")
	}
	if s.logins.Blocked(clientIP) {
		return "", time.Time{}, apierrors.RateLimited("too many attempts")
	}

	email := s.cfg.AdminUsername
	if !strings.Contains(email, "@") {
		email = email + "@antenne.local"
	}

	var user User
	if err := s.db.Where("email = ?", email).First(&user).Error; err != nil {
		s.logins.Fail(clientIP)
		return "", time.Time{}, apierrors.Unauthorized("invalid password")
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), prehash(password)) != nil {
		s.logins.Fail(clientIP)
		s.log.Warn("login failed", slog.String("client_ip", clientIP))
		return "", time.Time{}, apierrors.Unauthorized("invalid password")
	}

	s.logins.Reset(clientIP)

	token, err := newToken()
	if err != nil {
		return "", time.Time{}, err
	}
	expires := time.Now().Add(sessionLifetime)

	if err := s.db.Create(&Session{
		TokenHash: hashToken(token),
		UserID:    user.ID,
		ExpiresAt: expires,
		LastSeen:  time.Now(),
	}).Error; err != nil {
		return "", time.Time{}, err
	}

	s.log.Info("login succeeded", slog.String("client_ip", clientIP), slog.String("email", user.Email))
	return token, expires, nil
}

// Logout revokes the session the request carries.
func (s *Service) Logout(r *http.Request) {
	cookie, err := r.Cookie(SessionCookie)
	if err != nil || cookie.Value == "" {
		return
	}
	s.db.Where("token_hash = ?", hashToken(cookie.Value)).Delete(&Session{})
}

// PurgeExpiredSessions deletes sessions past their expiry.
func (s *Service) PurgeExpiredSessions() {
	if err := s.db.Where("expires_at < ?", time.Now()).Delete(&Session{}).Error; err != nil {
		s.log.Error("failed to purge expired sessions", slog.Any("error", err))
	}
}

// RequireAuth rejects unauthenticated callers and puts the identity on the
// context for everything downstream.
func RequireAuth(authenticator Authenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			identity, err := authenticator.Authenticate(r)
			if err != nil {
				httpjson.WriteError(w, apierrors.Unauthorized("unauthorized"))
				return
			}
			next.ServeHTTP(w, r.WithContext(authcontext.With(r.Context(), identity)))
		})
	}
}

func newToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// prehash folds a password to a fixed 44-character string before bcrypt sees it.
//
// bcrypt silently ignores everything past 72 bytes, and golang.org/x/crypto
// refuses outright — which is how a deployment running a long passphrase
// crash-looped on the first boot after the move off HMAC. The digest is
// base64-encoded rather than passed raw because bcrypt truncates at the first
// NUL byte, and a raw SHA-256 digest contains one about 40% of the time.
func prehash(password string) []byte {
	sum := sha256.Sum256([]byte(password))
	encoded := base64.StdEncoding.EncodeToString(sum[:])
	return []byte(encoded)
}
