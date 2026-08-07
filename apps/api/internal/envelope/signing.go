package envelope

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strconv"
	"time"
)

// SignedHeaders builds the headers Antenne sends with an outbound event.
//
// The signature covers `timestamp.eventId.body`, so a receiver that only checks
// the body cannot be fooled by a replay of an older event under a new
// timestamp — it has to bind all three, and the timestamp is what lets it
// bound how old a request it accepts.
func SignedHeaders(secret string, eventID string, body []byte) http.Header {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	headers := http.Header{}
	headers.Set("Content-Type", "application/json")
	headers.Set("User-Agent", "Antenne/1.0 uptime-bot")
	headers.Set("x-antenne-event-id", eventID)
	headers.Set("x-antenne-timestamp", timestamp)
	headers.Set("x-antenne-schema-version", "1")

	if secret != "" {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write([]byte(timestamp))
		mac.Write([]byte("."))
		mac.Write([]byte(eventID))
		mac.Write([]byte("."))
		mac.Write(body)
		headers.Set("x-antenne-signature-256", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	}

	return headers
}

// VerifyHexSignature reports whether signature is the HMAC-SHA256 of body under
// secret, with an optional prefix such as `sha256=`.
func VerifyHexSignature(secret, prefix, signature string, body []byte) bool {
	if signature == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := prefix + hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}
