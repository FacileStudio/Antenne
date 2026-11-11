package mcp

import (
	"context"
	"database/sql"
	"strconv"

	mcpkit "github.com/FacileStudio/tronc/mcp"
)

// EraseAccount deletes every MCP action of an erased person, open or resolved, in the
// transaction that deletes their account.
func EraseAccount(ctx context.Context, tx *sql.Tx, userID int64) error {
	return mcpkit.ErasePending(ctx, tx, strconv.FormatInt(userID, 10))
}
