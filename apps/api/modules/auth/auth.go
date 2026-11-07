// Package auth resolves a dashboard caller to an Antenne identity.
//
// The credentials belong to porte now: one hashed session row, one cookie, one
// logout, and the same session whether a human arrived through single sign-on
// or through the bootstrap password. What is left here is the part porte has no
// opinion about — what an Antenne user is, and how the instance seeds its first
// account from the environment.
//
// Handlers still read authcontext.Identity and still never see a cookie, a
// token or a user row.
package auth

import (
	"context"
	stderrors "errors"
	"log/slog"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/FacileStudio/Antenne/apps/api/internal/authcontext"
	"github.com/FacileStudio/Antenne/apps/api/internal/avatar"
	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/porte/local"
	"github.com/FacileStudio/porte/session"
	apierrors "github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"
)

// registrationLockKey is the advisory lock the account rules are taken under.
// porte cannot take it — it does not own this database — which is why the
// account rules live in this package's stores.
const registrationLockKey = 0x616E74656E6E65

// bootstrapDomain is the domain the bootstrap account's address is built on
// when ANTENNE_ADMIN_USERNAME is a bare label rather than an address. It is not
// a routable domain and is not meant to be: the account is a local credential
// for an appliance, and nothing ever sends mail to it.
const bootstrapDomain = "antenne.local"

// User is a dashboard account.
//
// password_hash is gone from this struct and stays in the table: porte holds
// the credential now, and the column carries pre-porte bcrypt digests that
// nothing can verify any more. It is dropped in a later migration rather than
// in the one that stopped reading it, so a rollback still has somewhere to land.
type User struct {
	ID         int64  `gorm:"primaryKey"`
	Email      string `gorm:"uniqueIndex"`
	Name       string
	Role       string
	PictureURL string `gorm:"column:picture_url"`
	Locale     string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// TableName pins the table so a future GORM naming change cannot rename it.
func (User) TableName() string { return "users" }

// Store resolves porte's callbacks to rows in Antenne's own users table.
//
// It is porte's escape hatch, taken deliberately: porte/pg ships a porte_users
// table and Antenne could have moved onto it, but users is referenced by name
// across this codebase and carries the role, which porte transports and never
// assigns meaning to.
type Store struct {
	orm *gorm.DB
}

// NewStore returns a store over the app's users table.
func NewStore(orm *gorm.DB) *Store {
	return &Store{orm: orm}
}

var (
	_ porte.UserStore         = (*Store)(nil)
	_ porte.PasswordUserStore = (*Store)(nil)
)

// profile is what the provider asserts about a person and the dashboard stores.
type profile struct {
	email    string
	name     string
	picture  string
	locale   string
	verified bool
}

// UpsertFromOIDC matches on (provider, subject) first and creates the account
// when nothing is found.
//
// Every account Antenne creates this way is an administrator, because Antenne
// has one tier: the dashboard holds the provider credentials, the delivery
// secrets and the pool secret, and there is no view of it that is safe for
// someone who should not have all three. Who may reach it at all is therefore
// the issuer's decision, expressed as an access policy on the application, not
// something this callback can soften.
//
// The email fallback follows porte/pg's linking rules: it links only on an
// address the provider verified, and it refuses an account that already holds
// another subject of the same issuer, so a recycled address is never handed
// over.
func (s *Store) UpsertFromOIDC(ctx context.Context, claims porte.Claims) (int64, error) {
	person, err := profileFrom(claims)
	if err != nil {
		return 0, err
	}

	var userID int64
	txErr := s.orm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", int64(registrationLockKey)).Error; err != nil {
			return apierrors.Internal("failed to acquire the registration lock", err)
		}
		id, err := resolveUser(tx, claims, person)
		userID = id
		return err
	})
	if txErr != nil {
		return 0, txErr
	}
	return userID, nil
}

// profileFrom normalises the claims, refusing an address that cannot be one.
func profileFrom(claims porte.Claims) (profile, error) {
	email := strings.ToLower(strings.TrimSpace(claims.Email))
	if _, err := mail.ParseAddress(email); err != nil {
		return profile{}, apierrors.Invalid("the identity provider returned no usable email")
	}
	return profile{
		email:    email,
		name:     claims.DisplayName(),
		picture:  avatar.PhotoURL(claims.Picture),
		locale:   claims.Locale,
		verified: claims.EmailVerified,
	}, nil
}

// resolveUser finds the account the claims belong to, or creates it.
func resolveUser(tx *gorm.DB, claims porte.Claims, person profile) (int64, error) {
	var linked int64
	err := tx.Raw(
		`SELECT user_id FROM porte_identities WHERE provider = ? AND subject = ?`,
		claims.Provider, claims.Subject,
	).Scan(&linked).Error
	if err != nil {
		return 0, apierrors.Internal("failed to resolve the identity", err)
	}
	if linked != 0 {
		return linked, refreshProfile(tx, linked, person)
	}

	var existing User
	err = tx.Where("email = ?", person.email).First(&existing).Error
	switch {
	case err == nil:
		if err := mayLink(tx, existing.ID, claims); err != nil {
			return 0, err
		}
		return existing.ID, refreshProfile(tx, existing.ID, person)
	case !stderrors.Is(err, gorm.ErrRecordNotFound):
		return 0, apierrors.Internal("failed to look up the account", err)
	}

	user := User{
		Email: person.email, Name: person.name, Role: string(authcontext.RoleAdmin),
		PictureURL: person.picture, Locale: localeOrDefault(person.locale),
	}
	if err := tx.Create(&user).Error; err != nil {
		return 0, apierrors.Internal("failed to create the account", err)
	}
	return user.ID, nil
}

// mayLink decides whether a sign-in may attach to the account that already
// holds its address.
//
// Matching on the address alone is an account takeover primitive wherever a
// provider lets a user claim an address without proving it, so the provider
// must say it verified the address. An account that already holds a different
// subject of the same issuer belongs to someone else: the address was
// recycled, and linking would hand that person's data over.
func mayLink(tx *gorm.DB, userID int64, claims porte.Claims) error {
	if !claims.EmailVerified {
		return apierrors.Conflict("an account with this email already exists and the identity provider did not verify the address")
	}
	var other int64
	err := tx.Raw(
		`SELECT count(*) FROM porte_identities WHERE user_id = ? AND provider = ? AND subject <> ?`,
		userID, claims.Provider, claims.Subject,
	).Scan(&other).Error
	if err != nil {
		return apierrors.Internal("failed to check the account's identities", err)
	}
	if other > 0 {
		return apierrors.Conflict("this account is already linked to another identity from the same provider")
	}
	return nil
}

// localeOrDefault gives a new account the default language when the provider
// sent no locale claim.
func localeOrDefault(locale string) string {
	if locale == "" {
		return porte.DefaultLocale
	}
	return locale
}

// refreshProfile keeps the name, the language and, when the provider verified
// it, the address the provider asserts, which is what makes the identity provider the source of truth for
// them. An empty value never overwrites one the user already has: a provider
// that stops sending a claim should not blank the dashboard.
//
// The picture is the exception and is always written: the provider sends one
// for every user it has a picture for, so a missing or non-https claim means
// there is none and the dashboard should draw the initial instead.
func refreshProfile(tx *gorm.DB, userID int64, person profile) error {
	updates := map[string]any{"picture_url": person.picture}
	if person.verified {
		updates["email"] = person.email
	}
	if person.name != "" {
		updates["name"] = person.name
	}
	if person.locale != "" {
		updates["locale"] = person.locale
	}
	if err := tx.Model(&User{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
		return apierrors.Internal("failed to update the account", err)
	}
	return nil
}

// CreateFromPassword creates a user row for a local registration.
//
// Antenne never opens registration — the bootstrap account is seeded from the
// environment and local.Config carries AllowRegistration false — so this is
// reached only through that seed. It implements the interface honestly all the
// same, under the same lock, because a store that is correct only for its
// current caller is a trap for the next one.
func (s *Store) CreateFromPassword(ctx context.Context, email, name string) (int64, error) {
	var userID int64
	txErr := s.orm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", int64(registrationLockKey)).Error; err != nil {
			return apierrors.Internal("failed to acquire the registration lock", err)
		}
		var existing int64
		if err := tx.Model(&User{}).Where("email = ?", email).Count(&existing).Error; err != nil {
			return apierrors.Internal("failed to check the address", err)
		}
		if existing > 0 {
			return apierrors.Conflict("an account with this email already exists")
		}
		user := User{Email: email, Name: name, Role: string(authcontext.RoleAdmin)}
		if err := tx.Create(&user).Error; err != nil {
			return apierrors.Internal("failed to create the account", err)
		}
		userID = user.ID
		return nil
	})
	if txErr != nil {
		return 0, txErr
	}
	return userID, nil
}

// FindByEmail returns the user id for an address, or porte.ErrNotFound.
func (s *Store) FindByEmail(ctx context.Context, email string) (int64, error) {
	var user User
	err := s.orm.WithContext(ctx).Select("id").Where("email = ?", email).First(&user).Error
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return 0, porte.ErrNotFound
	}
	if err != nil {
		return 0, apierrors.Internal("failed to look up the account", err)
	}
	return user.ID, nil
}

// CountUsers backs porte/local's registration gate.
func (s *Store) CountUsers(ctx context.Context) (int64, error) {
	var count int64
	if err := s.orm.WithContext(ctx).Model(&User{}).Count(&count).Error; err != nil {
		return 0, apierrors.Internal("failed to count accounts", err)
	}
	return count, nil
}

// Config is what Service needs from the environment.
type Config struct {
	AdminUsername string
	AdminPassword string

	// Issuer is the OIDC issuer. With one configured, sign-in goes through it
	// alone: porte mounts no password route, so the bootstrap account is not
	// seeded and the dashboard draws no password form.
	Issuer string

	// JournalBrowserURL and JournalBrowserKey are relayed to the dashboard
	// through ConfigExtra. They are not auth's business, but /auth/config is the
	// one endpoint the client already calls before it renders anything, and the
	// browser SDK has to be configured before the first error rather than after.
	JournalBrowserURL string
	JournalBrowserKey string
}

// Service seeds the bootstrap account and turns the user id porte
// authenticated into the identity the rest of Antenne reads.
type Service struct {
	orm        *gorm.DB
	log        *slog.Logger
	cfg        Config
	store      *Store
	identities porte.IdentityStore
	sessions   *session.Manager
}

// NewService wires the bootstrap seed and the identity lookup.
func NewService(orm *gorm.DB, log *slog.Logger, cfg Config, store *Store, identities porte.IdentityStore, sessions *session.Manager) *Service {
	return &Service{orm: orm, log: log, cfg: cfg, store: store, identities: identities, sessions: sessions}
}

// PasswordRequired reports whether the bootstrap password login is in use. It
// never is once an OIDC issuer is configured.
func (s *Service) PasswordRequired() bool { return s.cfg.AdminPassword != "" && s.cfg.Issuer == "" }

// Username is the login label the dashboard shows.
func (s *Service) Username() string { return s.cfg.AdminUsername }

// AdminEmail is the address the bootstrap account signs in with.
func (s *Service) AdminEmail() string { return bootstrapEmail(s.cfg.AdminUsername) }

// bootstrapEmail turns a bare login label into an address, because porte's
// local login resolves an account by address and Antenne's has always been
// configured as a username.
func bootstrapEmail(username string) string {
	if strings.Contains(username, "@") {
		return strings.ToLower(strings.TrimSpace(username))
	}
	return strings.ToLower(strings.TrimSpace(username)) + "@" + bootstrapDomain
}

// EnsureAdmin seeds or re-seeds the bootstrap account from the environment.
//
// ANTENNE_ADMIN_PASSWORD stays the source of truth, exactly as before porte:
// changing it and restarting re-hashes the credential and revokes that
// account's sessions, which is the recovery path and the only one an operator
// has without database access.
//
// What changed is where the hash lands. It used to be bcrypt over a
// base64(sha256(password)) prehash in users.password_hash; it is argon2id in
// porte_identities now, keyed on the account id rather than on the address. The
// old digests are not migrated because they cannot be — no argon2 verifier
// accepts them — and nothing is lost, because this function has always been
// able to rebuild the credential from the environment.
func (s *Service) EnsureAdmin(ctx context.Context) error {
	if !s.PasswordRequired() {
		return nil
	}

	email := s.AdminEmail()
	userID, err := s.store.FindByEmail(ctx, email)
	switch {
	case stderrors.Is(err, porte.ErrNotFound):
		if userID, err = s.store.CreateFromPassword(ctx, email, s.cfg.AdminUsername); err != nil {
			return err
		}
	case err != nil:
		return err
	}

	subject := porte.LocalSubject(userID)
	existing, findErr := s.identities.Find(ctx, porte.ProviderLocal, subject)
	switch {
	case findErr == nil:
		if local.VerifyPassword(s.cfg.AdminPassword, existing.PasswordHash) {
			return nil
		}
	case !stderrors.Is(findErr, porte.ErrNotFound):
		return findErr
	}

	hash, err := local.HashPassword(s.cfg.AdminPassword)
	if err != nil {
		return err
	}
	if err := s.identities.Save(ctx, porte.StoredIdentity{
		UserID:       userID,
		Provider:     porte.ProviderLocal,
		Subject:      subject,
		PasswordHash: hash,
	}); err != nil {
		return err
	}

	// Only a *replaced* credential revokes sessions. Seeding the first one
	// must not, or every restart of a fresh instance would end the session of
	// whoever just signed in — findErr is read rather than err because the
	// hash and the save have both reassigned err by now, and reusing it here
	// is how this check silently inverts.
	if stderrors.Is(findErr, porte.ErrNotFound) {
		return nil
	}
	s.log.Info("admin password changed in the environment, re-seeding the credential and revoking that account's sessions")
	_, err = s.sessions.RevokeUser(ctx, userID)
	return err
}

// IdentityForUser turns the user id porte authenticated into Antenne's own
// identity.
//
// porte hands the middleware a session and a user id and stops there: it holds
// no address and no role, because a library that decided what an admin may do
// would be routed around by the second app that adopted it. The profile is read
// here, from the table this app owns.
func (s *Service) IdentityForUser(ctx context.Context, userID int64) (authcontext.Identity, error) {
	var user User
	if err := s.orm.WithContext(ctx).First(&user, userID).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			// The session points at a user that no longer exists. The
			// foreign key cascades a delete, so this is a race rather
			// than a leak, and it is still not an authenticated request.
			return authcontext.Identity{}, apierrors.Unauthorized("unauthorized")
		}
		return authcontext.Identity{}, apierrors.Internal("failed to load the account", err)
	}
	return authcontext.Identity{
		UserID:    user.ID,
		Email:     user.Email,
		Name:      user.Name,
		Role:      authcontext.Role(user.Role),
		AvatarURL: user.PictureURL,
		Locale:    user.Locale,
	}, nil
}

// PurgeExpiredSessions deletes the sessions that have expired.
func (s *Service) PurgeExpiredSessions(ctx context.Context) {
	if _, err := s.sessions.Sweep(ctx); err != nil {
		s.log.Error("failed to purge expired sessions", slog.Any("error", err))
	}
}

// IdentityResolver turns an authenticated user id into an Antenne identity.
type IdentityResolver interface {
	IdentityForUser(ctx context.Context, userID int64) (authcontext.Identity, error)
}

// authenticator is the one method this package needs from porte's session
// manager, named as an interface so the middleware stays testable.
type authenticator interface {
	RequireAuth(http.Handler) http.Handler
}

// RequireAuth runs behind porte's own middleware and hydrates what porte
// deliberately does not carry.
//
// porte authenticates the credential — cookie or bearer, one hashed session
// row, one expiry, one idle window — and hands on a user id. Everything
// downstream still reads authcontext.Identity and is unchanged.
func RequireAuth(sessions authenticator, resolver IdentityResolver) func(http.Handler) http.Handler {
	hydrate := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authenticated, ok := porte.From(r.Context())
			if !ok {
				httpjson.WriteError(w, apierrors.Unauthorized("unauthorized"))
				return
			}
			identity, err := resolver.IdentityForUser(r.Context(), authenticated.UserID)
			if err != nil {
				httpjson.WriteError(w, err)
				return
			}
			identity.ClientIP = authcontext.ClientIP(r)
			next.ServeHTTP(w, r.WithContext(authcontext.With(r.Context(), identity)))
		})
	}
	return func(next http.Handler) http.Handler {
		return sessions.RequireAuth(hydrate(next))
	}
}
