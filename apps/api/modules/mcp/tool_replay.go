package mcp

import (
	"context"
	"fmt"

	"github.com/FacileStudio/Antenne/apps/api/modules/events"
	"github.com/FacileStudio/Antenne/apps/api/modules/settings"
	mcpkit "github.com/FacileStudio/tronc/mcp"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	statusSent = "sent"
	hintReplay = "Call antenne_list_events and antenne_list_targets for valid ids."
)

// replayEvent is antenne_replay_event: the event goes to the target once the user
// confirms.
func (t *toolset) replayEvent(ctx context.Context, req *sdk.CallToolRequest, in replayInput) (*sdk.CallToolResult, replayOutput, error) {
	if mcpkit.Retrying(req) {
		return t.resumeReplay(ctx, req)
	}
	if _, err := t.callerOf(ctx, req); err != nil {
		return nil, replayOutput{}, err
	}
	ref := replayRef{Event: in.EventID, Target: in.TargetID}
	entry, target, err := t.resolve(ref)
	if err != nil {
		return nil, replayOutput{}, mcpkit.Failure(err, hintReplay)
	}
	result, outcome, err := t.confirm.Request(ctx, req, kindReplay, encodeRef(ref), replayQuestion(entry, target))
	if err != nil {
		return nil, replayOutput{}, mcpkit.Failure(err, hintReplay)
	}
	if result != nil {
		return result, replayOutput{}, nil
	}
	return nil, replayOutput{
		Status: outcome.Status, EventID: in.EventID, TargetID: in.TargetID, ConfirmURL: outcome.ConfirmURL, Message: outcome.Message,
	}, nil
}

// resumeReplay handles the retry that carries the person's answer.
func (t *toolset) resumeReplay(ctx context.Context, req *sdk.CallToolRequest) (*sdk.CallToolResult, replayOutput, error) {
	if _, err := t.callerOf(ctx, req); err != nil {
		return nil, replayOutput{}, err
	}
	outcome, err := t.confirm.Resume(ctx, req, kindReplay)
	if err != nil {
		return nil, replayOutput{}, mcpkit.Failure(err, "Call antenne_replay_event again to ask the user afresh.")
	}
	if outcome.Status == mcpkit.StatusDeclined {
		return nil, replayOutput{Status: mcpkit.StatusDeclined, Message: outcome.Message}, nil
	}
	ref, _ := refOf(outcome.Action)
	return nil, replayOutput{
		Status: statusSent, EventID: ref.Event, TargetID: ref.Target, Message: "The user confirmed and the event was sent to the target.",
	}, nil
}

// replayQuestion is what the person is asked before an event is sent again.
func replayQuestion(entry events.Entry, target settings.DeliveryTarget) string {
	label := entry.Event
	if title := text(entry.Envelope.Event.Title); title != "" {
		label = title
	}
	return fmt.Sprintf("Send the %s event %q from %s again to the %s target %q? Everyone who reads that target will see it.",
		entry.Source, label, entry.ProviderName, settings.TargetLabel(target.Type), target.Name)
}
