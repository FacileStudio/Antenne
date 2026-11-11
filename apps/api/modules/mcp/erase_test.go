package mcp_test

import (
	"context"
	"testing"
	"time"

	"github.com/FacileStudio/Antenne/apps/api/internal/pgtest"
	"github.com/FacileStudio/Antenne/apps/api/modules/mcp"
	mcpkit "github.com/FacileStudio/tronc/mcp"
)

// TestErasureDeletesOnlyThePersonsActions runs the migrations, stores an open and a
// resolved action for one person and an open one for another, erases the first inside a
// transaction, and checks only the second person's action is left.
func TestErasureDeletesOnlyThePersonsActions(t *testing.T) {
	db := pgtest.New(t)
	sqlDB, err := db.DB()
	must(t, err)
	ctx := context.Background()
	store := mcpkit.PostgresPending{DB: sqlDB}
	now := time.Now().UTC()
	for i, user := range []string{"1", "1", "2"} {
		must(t, store.Create(ctx, mcpkit.Action{
			ID: string(rune('a' + i)), UserID: user, Kind: "replay_event", Target: "{}", Digest: "d",
			CreatedAt: now, ExpiresAt: now.Add(time.Hour),
		}))
	}
	must(t, store.Resolve(ctx, "1", "b", mcpkit.OutcomeDismissed))

	tx, err := sqlDB.BeginTx(ctx, nil)
	must(t, err)
	must(t, mcp.EraseAccount(ctx, tx, 1))
	must(t, tx.Commit())

	var left []string
	must(t, db.Raw(`SELECT user_id FROM mcp_actions ORDER BY id`).Scan(&left).Error)
	if len(left) != 1 || left[0] != "2" {
		t.Errorf("want only person 2's action left, got %v", left)
	}
}
