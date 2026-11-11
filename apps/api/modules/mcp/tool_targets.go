package mcp

import (
	"context"
	"encoding/json"

	"github.com/FacileStudio/Antenne/apps/api/modules/delivery"
	"github.com/FacileStudio/Antenne/apps/api/modules/settings"
	"github.com/FacileStudio/tronc/errors"
	mcpkit "github.com/FacileStudio/tronc/mcp"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

var errNotIMAP = errors.Invalid("only IMAP providers have a connection to test")

// checkTarget is antenne_check_target: the dashboard's check-now probe, under the same
// allowance per person.
func (t *toolset) checkTarget(ctx context.Context, req *sdk.CallToolRequest, in targetInput) (*sdk.CallToolResult, checkOutput, error) {
	userID, err := t.callerOf(ctx, req)
	if err != nil {
		return nil, checkOutput{}, err
	}
	result, _, err := t.Health.CheckTarget(ctx, limitKey(userID), in.TargetID)
	if err != nil {
		return nil, checkOutput{}, mcpkit.Failure(err, hintTarget)
	}
	return nil, checkOutput{
		TargetID: result.TargetID, OK: result.OK, Kind: result.Kind, Detail: delivery.ScrubURLs(result.Detail),
		State: result.State, RetryAfterSeconds: result.RetryAfterSeconds,
	}, nil
}

// testTarget is antenne_test_target: one fixed test message straight to one target,
// past the routing rules. A send the far side refused is a result, not an error.
func (t *toolset) testTarget(ctx context.Context, req *sdk.CallToolRequest, in targetInput) (*sdk.CallToolResult, sendOutput, error) {
	if _, err := t.callerOf(ctx, req); err != nil {
		return nil, sendOutput{}, err
	}
	if _, err := t.target(in.TargetID); err != nil {
		return nil, sendOutput{}, err
	}
	if err := t.Pipeline.TestTarget(ctx, in.TargetID); err != nil {
		return nil, sendOutput{Status: statusFailed, ID: in.TargetID, Error: delivery.RedactError(err)}, nil
	}
	return nil, sendOutput{Status: statusOK, ID: in.TargetID}, nil
}

// testProvider is antenne_test_provider: an IMAP provider's login, tried once.
func (t *toolset) testProvider(ctx context.Context, req *sdk.CallToolRequest, in providerInput) (*sdk.CallToolResult, sendOutput, error) {
	if _, err := t.callerOf(ctx, req); err != nil {
		return nil, sendOutput{}, err
	}
	provider, ok := t.Pipeline.Settings().FindProvider(in.ProviderID)
	switch {
	case !ok:
		return nil, sendOutput{}, mcpkit.Failure(errProvider, hintProvider)
	case provider.Type != settings.ProviderIMAP:
		return nil, sendOutput{}, mcpkit.Failure(errNotIMAP, "")
	}
	if err := t.Pipeline.TestProvider(in.ProviderID); err != nil {
		return nil, sendOutput{Status: statusFailed, ID: in.ProviderID, Error: delivery.RedactError(err)}, nil
	}
	return nil, sendOutput{Status: statusOK, ID: in.ProviderID}, nil
}

// setEnabled is antenne_set_enabled: the dashboard's on/off switch of a provider or a
// target, saved through the same settings update.
func (t *toolset) setEnabled(ctx context.Context, req *sdk.CallToolRequest, in enabledInput) (*sdk.CallToolResult, enabledOutput, error) {
	if _, err := t.callerOf(ctx, req); err != nil {
		return nil, enabledOutput{}, err
	}
	next, name, err := switched(t.Pipeline.Settings(), in)
	if err != nil {
		return nil, enabledOutput{}, err
	}
	raw, err := json.Marshal(next)
	if err != nil {
		return nil, enabledOutput{}, mcpkit.ErrUnavailable
	}
	if _, err := t.Pipeline.UpdateSettings(ctx, raw); err != nil {
		return nil, enabledOutput{}, mcpkit.Failure(errors.Invalid(err.Error()), "")
	}
	return nil, enabledOutput{Kind: in.Kind, ID: in.ID, Name: name, Enabled: in.Enabled}, nil
}

// switched is current with the named provider or target switched, and its name.
func switched(current settings.AppSettings, in enabledInput) (settings.AppSettings, string, error) {
	switch in.Kind {
	case kindProvider:
		items, name, ok := switchProvider(current.Providers.Items, in)
		if !ok {
			return current, "", mcpkit.Failure(errProvider, hintProvider)
		}
		current.Providers.Items = items
		return current, name, nil
	case kindTarget:
		items, name, ok := switchTarget(current.Delivery.Items, in)
		if !ok {
			return current, "", mcpkit.Failure(errTarget, hintTarget)
		}
		current.Delivery.Items = items
		return current, name, nil
	}
	return current, "", mcpkit.Failure(errKind, "")
}

// switchProvider is a copy of providers with the named one switched, and its name.
func switchProvider(providers []settings.Provider, in enabledInput) ([]settings.Provider, string, bool) {
	items := append([]settings.Provider(nil), providers...)
	for i := range items {
		if items[i].ID == in.ID {
			items[i].Enabled = in.Enabled
			return items, items[i].Name, true
		}
	}
	return nil, "", false
}

// switchTarget is a copy of targets with the named one switched, and its name.
func switchTarget(targets []settings.DeliveryTarget, in enabledInput) ([]settings.DeliveryTarget, string, bool) {
	items := append([]settings.DeliveryTarget(nil), targets...)
	for i := range items {
		if items[i].ID == in.ID {
			items[i].Enabled = in.Enabled
			return items, items[i].Name, true
		}
	}
	return nil, "", false
}
