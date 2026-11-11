package mcp

import (
	"context"
	"net/http"
	"strconv"

	"github.com/FacileStudio/Antenne/apps/api/internal/authcontext"
	"github.com/FacileStudio/Antenne/apps/api/modules/settings"
	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/tronc/errors"
	mcpkit "github.com/FacileStudio/tronc/mcp"
)

const webActor = "antenne-web"

// pendingUser is the signed-in person of a confirm page request. Only the person's own
// session passes: a delegated token must never confirm what an agent asked for.
func pendingUser(r *http.Request) (string, error) {
	authenticated, ok := porte.From(r.Context())
	identity := authcontext.From(r)
	if !ok || !identity.IsAuthenticated() {
		return "", errors.Unauthorized("missing auth")
	}
	if authenticated.SessionID == 0 || authenticated.Actor != "" {
		return "", mcpkit.ErrDelegated
	}
	return strconv.FormatInt(identity.UserID, 10), nil
}

// describe adds the event and the target a replay names, as they are now, to a confirm
// page item.
func (t *toolset) describe(_ context.Context, item mcpkit.PendingItem, a mcpkit.Action) PendingItem {
	out := PendingItem{PendingItem: item}
	ref, err := refOf(a)
	if err != nil {
		return out
	}
	if entry, ok := t.Events.ByID(ref.Event); ok {
		event := render(entry, false)
		out.Event = &PendingEvent{
			ID: event.ID, At: event.At, Source: event.Source, Provider: event.Provider, Event: event.Event, Title: event.Title,
		}
	}
	if target, ok := t.Pipeline.Settings().FindTarget(ref.Target); ok {
		out.Target = &PendingTarget{ID: target.ID, Name: target.Name, Type: string(target.Type), Label: settings.TargetLabel(target.Type)}
	}
	return out
}

// done is the confirm page's answer once an action ran: the event is sent.
func done(context.Context, mcpkit.Outcome) mcpkit.PendingResult {
	return mcpkit.PendingResult{Status: statusSent}
}
