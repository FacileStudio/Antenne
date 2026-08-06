package auth

import (
	"bytes"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

// A long passphrase has to work. bcrypt refuses anything past 72 bytes, and the
// deployed NOOK_ADMIN_PASSWORD is a hyphenated word list well past that — which
// crash-looped the service on the first boot after the move off HMAC.
func TestALongPassphraseCanBeHashedAndVerified(t *testing.T) {
	password := strings.Repeat("correct-horse-battery-staple-", 8)
	if len(password) <= 72 {
		t.Fatalf("the fixture is only %d bytes, which does not exercise the limit", len(password))
	}

	hash, err := bcrypt.GenerateFromPassword(prehash(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hashing a %d-byte password failed: %v", len(password), err)
	}
	if err := bcrypt.CompareHashAndPassword(hash, prehash(password)); err != nil {
		t.Errorf("the password did not verify against its own hash: %v", err)
	}
}

// Two passwords sharing their first 72 bytes must not be interchangeable. This
// is what the prehash buys over passing the password straight to bcrypt.
func TestPasswordsAreNotTruncated(t *testing.T) {
	shared := strings.Repeat("a", 72)
	hash, err := bcrypt.GenerateFromPassword(prehash(shared+"-one"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}

	if bcrypt.CompareHashAndPassword(hash, prehash(shared+"-two")) == nil {
		t.Error("a password differing only past byte 72 was accepted")
	}
}

// The digest is base64-encoded rather than raw because bcrypt stops at the
// first NUL byte, and a raw SHA-256 digest contains one often enough to matter.
func TestThePrehashCarriesNoNulByte(t *testing.T) {
	for _, password := range []string{"", "hunter2", strings.Repeat("x", 500), "é🔑"} {
		digest := prehash(password)
		if bytes.IndexByte(digest, 0) >= 0 {
			t.Errorf("prehash(%q) contains a NUL byte", password)
		}
		if len(digest) > 72 {
			t.Errorf("prehash(%q) is %d bytes, past bcrypt's limit", password, len(digest))
		}
	}
}
