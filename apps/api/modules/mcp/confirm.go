package mcp

import (
	"context"
	"encoding/json"

	"github.com/FacileStudio/Antenne/apps/api/modules/delivery"
	"github.com/FacileStudio/Antenne/apps/api/modules/events"
	"github.com/FacileStudio/Antenne/apps/api/modules/settings"
	"github.com/FacileStudio/tronc/errors"
	mcpkit "github.com/FacileStudio/tronc/mcp"
)

// kindReplay is the one sensitive action: sending a logged event again, to a target the
// caller picks, which can carry its content to people the routing never sent it to.
const kindReplay = "replay_event"

var errRef = errors.NotFound("this action names no event and target")

// replayRef is what a replay action targets.
type replayRef struct {
	Event  string `json:"event"`
	Target string `json:"target"`
}

// kinds are the sensitive actions. The fingerprint covers the event and the target as
// they were when the person was asked, so a target edited in between, such as a new
// webhook URL, closes the action instead of sending somewhere the person did not see.
func (t *toolset) kinds() map[string]mcpkit.Kind {
	return map[string]mcpkit.Kind{
		kindReplay: {Tool: toolReplay, Fingerprint: t.fingerprint, Run: t.runReplay},
	}
}

// encodeRef is the pending-action target of a replay.
func encodeRef(ref replayRef) string {
	raw, _ := json.Marshal(ref)
	return string(raw)
}

// refOf reads an action's replay target.
func refOf(a mcpkit.Action) (replayRef, error) {
	var ref replayRef
	if json.Unmarshal([]byte(a.Target), &ref) != nil || ref.Event == "" || ref.Target == "" {
		return replayRef{}, errRef
	}
	return ref, nil
}

// resolve reads the event and the target a replay names, as they are now.
func (t *toolset) resolve(ref replayRef) (events.Entry, settings.DeliveryTarget, error) {
	entry, ok := t.Events.ByID(ref.Event)
	if !ok || entry.Envelope == nil {
		return events.Entry{}, settings.DeliveryTarget{}, errEvent
	}
	target, ok := t.Pipeline.Settings().FindTarget(ref.Target)
	if !ok {
		return events.Entry{}, settings.DeliveryTarget{}, errTarget
	}
	return entry, target, nil
}

// fingerprint digests the event and the whole target, credentials included. The digest
// stays in the pending table and never reaches a client.
func (t *toolset) fingerprint(_ context.Context, a mcpkit.Action) (string, error) {
	ref, err := refOf(a)
	if err != nil {
		return "", err
	}
	entry, target, err := t.resolve(ref)
	if err != nil {
		return "", err
	}
	return digest(entry, target), nil
}

// digest fingerprints a replay.
func digest(entry events.Entry, target settings.DeliveryTarget) string {
	config, _ := json.Marshal(target)
	return mcpkit.Digest(entry.ID, entry.Envelope.ID, text(entry.Message), string(config))
}

// runReplay sends the event to the target.
func (t *toolset) runReplay(ctx context.Context, a mcpkit.Action) error {
	ref, err := refOf(a)
	if err != nil {
		return err
	}
	if err := t.Pipeline.Replay(ctx, ref.Event, ref.Target); err != nil {
		return errors.Failed("the replay was not delivered: " + delivery.RedactError(err))
	}
	return nil
}
