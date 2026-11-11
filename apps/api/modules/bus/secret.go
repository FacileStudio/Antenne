package bus

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/gorilla/websocket"
)

// SetSecret swaps the registration secret, after a dashboard save. A changed
// secret revokes every token issued under the old one and closes every live
// connection, so each client has to register again with the new secret.
func (s *Server) SetSecret(secret string) {
	secret = effectiveSecret(secret)
	fingerprint := secretFingerprint(secret)

	s.mu.Lock()
	defer s.mu.Unlock()
	if fingerprint == s.fingerprint {
		return
	}
	s.secret = secret
	s.fingerprint = fingerprint
	for _, conn := range s.connections {
		s.evictLocked(conn, "secret rotated")
	}
	s.log.Info("pool secret rotated, every token issued under the old one is revoked")
}

// effectiveSecret is the secret the pool accepts: the configured one, or the
// insecure default when nothing is configured.
func effectiveSecret(secret string) string {
	if secret == "" {
		return "pool"
	}
	return secret
}

// secretFingerprint is the keyed hash a token is bound to. It identifies a
// secret without storing it, and a database leak does not reveal it.
func secretFingerprint(secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte("antenne pool token binding v1"))
	return hex.EncodeToString(mac.Sum(nil))
}

// evict closes the app's live connection, whose token has just been replaced.
func (s *Server) evict(appID, reason string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if conn, ok := s.connections[appID]; ok {
		s.evictLocked(conn, reason)
	}
}

// evictLocked drops conn from the routing tables and closes its socket after
// telling the peer why. The caller holds s.mu.
func (s *Server) evictLocked(conn *connection, reason string) {
	s.removeLocked(conn)
	go func() {
		conn.socket.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, reason),
			time.Now().Add(time.Second))
		conn.socket.Close()
	}()
}

// currentFingerprint is the fingerprint of the secret the pool accepts now.
func (s *Server) currentFingerprint() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.fingerprint
}

// refuse closes a socket upgraded for a token revoked in the meantime.
func refuse(socket *websocket.Conn) {
	socket.WriteControl(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "token revoked"),
		time.Now().Add(time.Second))
	socket.Close()
}
