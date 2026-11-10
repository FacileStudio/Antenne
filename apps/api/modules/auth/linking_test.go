package auth

import (
	"context"
	"testing"

	"gorm.io/gorm"

	"github.com/FacileStudio/Antenne/apps/api/internal/pgtest"
	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/porte/kit"
)

const issuer = "https://sso.example"

func claims(subject, email string, verified bool) porte.Claims {
	return porte.Claims{Provider: issuer, Subject: subject, Email: email, EmailVerified: verified, Locale: "de"}
}

func link(t *testing.T, db *gorm.DB, userID int64, subject string) {
	t.Helper()
	err := db.Exec(`INSERT INTO porte_identities (user_id, provider, subject) VALUES (?, ?, ?)`,
		userID, issuer, subject).Error
	if err != nil {
		t.Fatalf("link: %v", err)
	}
}

func accounts(t *testing.T) (*gorm.DB, *kit.Store) {
	t.Helper()
	db := pgtest.New(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	return db, Accounts(sqlDB)
}

// TestLinkingFollowsPorteRules covers the three branches the email fallback
// has over Antenne's own users table: an unverified address never links, a
// verified one links an account with no identity from the issuer, and an
// account that already holds another subject of the issuer is refused.
func TestLinkingFollowsPorteRules(t *testing.T) {
	db, store := accounts(t)
	ctx := context.Background()

	local := User{Email: "ada@example.com", Role: "ADMIN"}
	if err := db.Create(&local).Error; err != nil {
		t.Fatal(err)
	}

	if _, err := store.UpsertFromOIDC(ctx, claims("sub-1", "ada@example.com", false)); err == nil {
		t.Fatal("an unverified address linked into an existing account")
	}

	id, err := store.UpsertFromOIDC(ctx, claims("sub-1", "ada@example.com", true))
	if err != nil || id != local.ID {
		t.Fatalf("verified link: id=%d err=%v, want %d", id, err, local.ID)
	}
	link(t, db, id, "sub-1")

	var locale string
	db.Raw(`SELECT locale FROM users WHERE id = ?`, id).Scan(&locale)
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

// TestEveryNewAccountIsAnAdministrator proves the kit's insert fits Antenne's
// schema and that the account it creates holds the one role Antenne has, with
// the picture claim in picture_url.
func TestEveryNewAccountIsAnAdministrator(t *testing.T) {
	db, store := accounts(t)
	ctx := context.Background()

	for _, email := range []string{"first@example.com", "second@example.com"} {
		person := claims("sub-"+email, email, true)
		person.Picture = "https://sso.example/avatars/" + email
		id, err := store.UpsertFromOIDC(ctx, person)
		if err != nil {
			t.Fatal(err)
		}
		var user User
		if err := db.First(&user, id).Error; err != nil {
			t.Fatal(err)
		}
		if user.Role != "ADMIN" || user.PictureURL != person.Picture || user.Locale != "de" {
			t.Errorf("%s = role %q picture %q locale %q", email, user.Role, user.PictureURL, user.Locale)
		}
	}
}

// TestBackfillFillsOnlyEmptyPictures checks the boot backfill against the
// picture_url column: an account linked to the issuer without a picture gets
// <issuer>/avatars/<subject>, and one that has a picture keeps it.
func TestBackfillFillsOnlyEmptyPictures(t *testing.T) {
	db, store := accounts(t)
	bare, kept := User{Email: "bare@example.com"}, User{Email: "kept@example.com", PictureURL: "https://x.example/p"}
	for _, user := range []*User{&bare, &kept} {
		if err := db.Create(user).Error; err != nil {
			t.Fatal(err)
		}
		link(t, db, user.ID, "sub-"+user.Email)
	}

	filled, err := store.BackfillPictures(context.Background(), issuer)
	if err != nil || filled != 1 {
		t.Fatalf("filled = %d, %v; want 1", filled, err)
	}
	var pictures []string
	db.Raw(`SELECT picture_url FROM users ORDER BY id`).Scan(&pictures)
	if want := issuer + "/avatars/sub-bare@example.com"; len(pictures) != 2 || pictures[0] != want || pictures[1] != kept.PictureURL {
		t.Errorf("pictures = %v", pictures)
	}
}
