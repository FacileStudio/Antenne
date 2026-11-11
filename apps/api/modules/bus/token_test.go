package bus

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	antenne "github.com/FacileStudio/antenne-client/go"

	"github.com/FacileStudio/Antenne/apps/api/internal/pgtest"
)

const alphaRegistration = `{"app":"alpha","secret":"wire-secret"}`

// refused fails unless the socket endpoint turns token away.
func (p *wirePool) refused(token string) {
	p.t.Helper()
	response, err := http.Get(p.http.URL + "/antenne/ws?token=" + token)
	if err != nil {
		p.t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusForbidden {
		p.t.Fatalf("ws with a revoked token = %d, want 403", response.StatusCode)
	}
}

// dropped fails unless the server closes app's socket.
func (p *wirePool) dropped(app wireApp) {
	p.t.Helper()
	app.socket.SetReadDeadline(time.Now().Add(3 * time.Second))
	for {
		if _, _, err := app.socket.ReadMessage(); err != nil {
			if isTimeout(err) {
				p.t.Fatal("the server kept a connection whose token was revoked")
			}
			return
		}
	}
}

func isTimeout(err error) bool {
	timeout, ok := err.(interface{ Timeout() bool })
	return ok && timeout.Timeout()
}

// Knowing the secret must never hand out the token another holder is using:
// a second registration under one name issues a new token, keeps the app id,
// revokes the first token and closes the connection that used it.
func TestASecondRegistrationRevokesTheFirstToken(t *testing.T) {
	pool := newWirePool(t)
	first := pool.dial("/antenne/ws", pool.register("/antenne/register", alphaRegistration))

	second := pool.register("/antenne/register", alphaRegistration)
	if second.token == first.token {
		t.Fatal("a second registration returned the existing token")
	}
	if second.id != first.id {
		t.Fatalf("app id changed from %s to %s, which orphans its subscriptions", first.id, second.id)
	}

	pool.dropped(first)
	pool.refused(first.token)
	pool.dial("/antenne/ws", second)
}

// A dashboard save that changes the secret revokes every token issued under
// the old one at once, and the old secret no longer registers.
func TestADashboardRotationRevokesEveryToken(t *testing.T) {
	pool := newWirePool(t)
	live := pool.dial("/antenne/ws", pool.register("/antenne/register", alphaRegistration))
	idle := pool.register("/antenne/register", `{"app":"beta","secret":"wire-secret"}`)

	pool.server.SetSecret("rotated")

	pool.dropped(live)
	pool.refused(live.token)
	pool.refused(idle.token)
	if status, _ := pool.post("/antenne/register", alphaRegistration); status != http.StatusForbidden {
		t.Fatalf("register with the old secret = %d, want 403", status)
	}
	pool.dial("/antenne/ws", pool.register("/antenne/register", `{"app":"alpha","secret":"rotated"}`))
}

// Saving settings without touching the secret calls SetSecret with the same
// value. That must not disconnect anyone.
func TestSavingTheSameSecretKeepsConnections(t *testing.T) {
	pool := newWirePool(t)
	live := pool.dial("/antenne/ws", pool.register("/antenne/register", alphaRegistration))

	pool.server.SetSecret(wireSecret)

	pool.send(live, `{"type":"ping"}`)
	pool.expect(live, `{"type":"pong"}`)
}

// Rotating ANTENNE_SECRET takes effect at the restart: the new process binds
// nothing old to its secret, so every token from before is refused.
func TestAnEnvRotationRevokesEveryToken(t *testing.T) {
	db := pgtest.New(t)
	before := wirePoolOn(t, db, wireSecret)
	old := before.register("/antenne/register", alphaRegistration)

	after := wirePoolOn(t, db, "rotated")
	after.refused(old.token)
	after.dial("/antenne/ws", after.register("/antenne/register", `{"app":"alpha","secret":"rotated"}`))
}

// A token registered before the fingerprint column existed is NULL after the
// migration. The first boot binds it to the current secret, so the deploy
// disconnects nobody.
func TestTokensFromBeforeTheMigrationKeepWorking(t *testing.T) {
	db := pgtest.New(t)
	if err := db.Exec(`INSERT INTO pool_apps (app_id, app, token) VALUES ('legacy-id', 'legacy', 'legacy-token')`).Error; err != nil {
		t.Fatal(err)
	}

	pool := wirePoolOn(t, db, wireSecret)
	pool.dial("/antenne/ws", wireApp{id: "legacy-id", token: "legacy-token"})

	rotated := wirePoolOn(t, db, "rotated")
	rotated.refused("legacy-token")
}

// The suite's own client must survive a revoked token: when its connection is
// closed because someone registered under its name, it registers again by
// itself and reconnects, rather than going silent.
func TestTheClientReregistersAfterItsTokenIsRevoked(t *testing.T) {
	pool := newWirePool(t)
	var connects atomic.Int32
	client := antenne.NewClient(&antenne.Config{App: "alpha", Instance: pool.http.URL, Secret: wireSecret},
		antenne.WithOnConnect(func() { connects.Add(1) }))
	if err := client.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(client.Disconnect)

	pool.register("/antenne/register", alphaRegistration)

	deadline := time.Now().Add(10 * time.Second)
	for connects.Load() < 2 {
		if time.Now().After(deadline) {
			t.Fatalf("client connected %d times, want a reconnect after its token was revoked", connects.Load())
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !client.IsConnected() {
		t.Fatal("client is not connected after re-registering")
	}
}
