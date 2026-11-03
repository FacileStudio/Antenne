// Package authcontext carries the authenticated caller through the request
// context.
//
// It is deliberately the only thing the rest of the API knows about
// authentication. Handlers read an Identity; they never see a cookie, a token
// or a user row. porte authenticates the credential and hands on a user id;
// modules/auth turns that into the Identity below, so nothing downstream of
// here had to change when the credentials moved.
package authcontext

import (
	"context"
	"net"
	"net/http"
	"strings"
)

type contextKey int

const identityKey contextKey = 0

// Role is the caller's privilege level. Antenne has one tier today; the constants
// exist so the shared package's USER/ADMIN hierarchy maps onto it unchanged.
type Role string

const (
	RoleUser  Role = "USER"
	RoleAdmin Role = "ADMIN"
)

// Identity is the authenticated caller. The zero value is an anonymous caller.
//
// UserID is an int64 to match porte's own, and porte_sessions.user_id, and the
// bigint foreign keys already in this schema. The conversion that used to sit
// between them is gone rather than moved.
type Identity struct {
	UserID    int64
	Email     string
	Name      string
	Role      Role
	AvatarURL string
	ClientIP  string
}

// IsAuthenticated reports whether a caller was identified.
func (i Identity) IsAuthenticated() bool { return i.UserID != 0 }

// IsAdmin reports whether the caller holds the admin role.
func (i Identity) IsAdmin() bool { return i.Role == RoleAdmin }

// With returns a copy of ctx carrying identity.
func With(ctx context.Context, identity Identity) context.Context {
	return context.WithValue(ctx, identityKey, identity)
}

// From returns the identity carried by r, or the anonymous zero value.
func From(r *http.Request) Identity {
	identity, _ := r.Context().Value(identityKey).(Identity)
	return identity
}

// ClientIP resolves the caller's address from the proxy headers, falling back
// to the socket address.
//
// It trusts X-Forwarded-For, which is only safe because every deployment puts
// Traefik in front and Traefik overwrites the header. A directly exposed Antenne
// lets a caller spoof its own rate-limit bucket.
func ClientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		if first, _, found := strings.Cut(forwarded, ","); found {
			return strings.TrimSpace(first)
		}
		return strings.TrimSpace(forwarded)
	}
	for _, header := range []string{"CF-Connecting-IP", "X-Real-IP"} {
		if value := r.Header.Get(header); value != "" {
			return strings.TrimSpace(value)
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
