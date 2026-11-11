package bus

import (
	"log/slog"
	"strings"
	"testing"

	"github.com/FacileStudio/Antenne/apps/api/internal/pgtest"
)

// TestCleanupEnforcesTheCeilingAndTheAckedFloor seeds a channel one listener has
// acked to offset 2: offset 1 is acked and past the 48 h floor, offset 2 is acked but
// inside it, offset 3 is unacked and past the 30-day ceiling, offset 4 is unacked
// and fresh. Cleanup keeps exactly offsets 2 and 4.
func TestCleanupEnforcesTheCeilingAndTheAckedFloor(t *testing.T) {
	db := pgtest.New(t)
	statements := []string{
		`INSERT INTO pool_apps (app_id, app, token, listen_channels, last_seen_at) VALUES ('l1', 'listener', 't1', '["c"]', now())`,
		`INSERT INTO pool_subscriptions (subscriber_id, channel, last_acked) VALUES ('l1', 'c', 2)`,
		`INSERT INTO pool_events (id, channel, offset_num, payload, sender, emitted_at) VALUES
			('acked-old', 'c', 1, '{}', 's', now() - interval '3 days'),
			('acked-fresh', 'c', 2, '{}', 's', now() - interval '1 hour'),
			('past-ceiling', 'c', 3, '{}', 's', now() - interval '31 days'),
			('fresh', 'c', 4, '{}', 's', now())`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("%s: %v", statement, err)
		}
	}

	NewQueue(db, slog.New(slog.DiscardHandler)).Cleanup()

	var left []string
	if err := db.Raw(`SELECT id FROM pool_events ORDER BY offset_num`).Scan(&left).Error; err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(left, ","); got != "acked-fresh,fresh" {
		t.Fatalf("pool_events left = %s, want acked-fresh,fresh", got)
	}
}
