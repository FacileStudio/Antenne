package auth

import (
	"context"
	"testing"

	"github.com/FacileStudio/Antenne/apps/api/internal/pgtest"
	"github.com/FacileStudio/porte"
)

const issuer = "https://sso.example"

func claims(subject, email string, verified bool) porte.Claims {
	return porte.Claims{Provider: issuer, Subject: subject, Email: email, EmailVerified: verified, Locale: "de"}
}

func link(t *testing.T, store *Store, userID int64, subject string) {
	t.Helper()
	err := store.orm.Exec(`INSERT INTO porte_identities (user_id, provider, subject) VALUES (?, ?, ?)`,
		userID, issuer, subject).Error
	if err != nil {
		t.Fatalf("link: %v", err)
	}
}

// TestLinkingFollowsPorteRules covers the three branches the email fallback
// has: an unverified address never links, a verified one links an account
// with no identity from the issuer, and an account that already holds another
// subject of the issuer is refused.
func TestLinkingFollowsPorteRules(t *testing.T) {
	store := NewStore(pgtest.New(t))
	ctx := context.Background()

	local := User{Email: "ada@example.com", Role: "ADMIN"}
	if err := store.orm.Create(&local).Error; err != nil {
		t.Fatal(err)
	}

	if _, err := store.UpsertFromOIDC(ctx, claims("sub-1", "ada@example.com", false)); err == nil {
		t.Fatal("an unverified address linked into an existing account")
	}

	id, err := store.UpsertFromOIDC(ctx, claims("sub-1", "ada@example.com", true))
	if err != nil || id != local.ID {
		t.Fatalf("verified link: id=%d err=%v, want %d", id, err, local.ID)
	}
	link(t, store, id, "sub-1")

	var locale string
	store.orm.Raw(`SELECT locale FROM users WHERE id = ?`, id).Scan(&locale)
	if locale != "de" {
		t.Errorf("locale = %q, want de", locale)
	}

	if _, err := store.UpsertFromOIDC(ctx, claims("sub-2", "ada@example.com", true)); err == nil {
		t.Fatal("a second subject of the same issuer took over the account")
	}

	again, err := store.UpsertFromOIDC(ctx, claims("sub-1", "ada@example.com", true))
	if err != nil || again != id {
		t.Fatalf("known subject: id=%d err=%v, want %d", again, err, id)
	}
}
