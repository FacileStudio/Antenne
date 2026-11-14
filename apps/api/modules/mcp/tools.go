package mcp

import (
	"context"
	stderrors "errors"
	"strconv"

	"github.com/FacileStudio/Antenne/apps/api/modules/settings"
	"github.com/FacileStudio/porte/mcpauth"
	"github.com/FacileStudio/tronc/errors"
	mcpkit "github.com/FacileStudio/tronc/mcp"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	toolEvents     = "antenne_list_events"
	toolEvent      = "antenne_get_event"
	toolProviders  = "antenne_list_providers"
	toolTargets    = "antenne_list_targets"
	toolPool       = "antenne_pool_status"
	toolCheck      = "antenne_check_target"
	toolTest       = "antenne_test_target"
	toolTestSource = "antenne_test_provider"
	toolEnabled    = "antenne_set_enabled"
	toolReplay     = "antenne_replay_event"
	kindProvider   = "provider"
	kindTarget     = "target"
	statusOK       = "ok"
	statusFailed   = "failed"
	hintEvent      = "Call antenne_list_events for the ids of logged events."
	hintTarget     = "Call antenne_list_targets for the ids of the delivery targets."
	hintProvider   = "Call antenne_list_providers for the ids of the providers."
)

var (
	errNoCaller = stderrors.New("this request carries no valid access token. Reconnect Antenne")
	errKind     = errors.Invalid(`kind must be "provider" or "target"`)
	errTarget   = errors.NotFound("delivery target not found")
	errProvider = errors.NotFound("provider not found")
	errEvent    = errors.NotFound("event not found")
)

// toolset holds what the tool handlers share.
type toolset struct {
	Deps
	confirm *mcpkit.Confirmations
}

// callerOf checks the verified token names an Antenne account that still exists, the
// way RequireAuth checks a session, and returns its user id.
func (t *toolset) callerOf(ctx context.Context, req *sdk.CallToolRequest) (int64, error) {
	identity, err := mcpauth.AccountExists(ctx, req.Extra.TokenInfo, t.Accounts.IdentityForUser)
	if err != nil {
		return 0, errNoCaller
	}
	return identity.UserID, nil
}

// target returns the delivery target with id, or the refusal a model can act on.
func (t *toolset) target(id string) (settings.DeliveryTarget, error) {
	target, ok := t.Pipeline.Settings().FindTarget(id)
	if !ok {
		return settings.DeliveryTarget{}, mcpkit.Failure(errTarget, hintTarget)
	}
	return target, nil
}

// limitKey is the check-now allowance of one person, apart from any dashboard address.
func limitKey(userID int64) string {
	return "mcp:" + strconv.FormatInt(userID, 10)
}
