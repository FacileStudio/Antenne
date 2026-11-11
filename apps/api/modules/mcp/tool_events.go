package mcp

import (
	"context"
	"encoding/json"

	"github.com/FacileStudio/Antenne/apps/api/internal/envelope"
	"github.com/FacileStudio/Antenne/apps/api/modules/delivery"
	"github.com/FacileStudio/Antenne/apps/api/modules/events"
	mcpkit "github.com/FacileStudio/tronc/mcp"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	defaultLimit  = 20
	maxLimit      = 50
	conciseText   = 400
	detailedText  = 4000
	rawPayloadCap = 20_000
)

// listEvents is antenne_list_events: a page of the activity log, newest first.
func (t *toolset) listEvents(ctx context.Context, req *sdk.CallToolRequest, in eventsInput) (*sdk.CallToolResult, eventsOutput, error) {
	if _, err := t.callerOf(ctx, req); err != nil {
		return nil, eventsOutput{}, err
	}
	detailed, err := mcpkit.Detailed(in.ResponseFormat)
	if err != nil {
		return nil, eventsOutput{}, err
	}
	offset := 0
	if err := mcpkit.DecodeCursor(in.Cursor, &offset); err != nil {
		return nil, eventsOutput{}, err
	}
	limit := in.Limit
	if limit <= 0 || limit > maxLimit {
		limit = defaultLimit
	}
	query := events.Query{
		Search: in.Query, Source: in.Source, ProviderID: in.ProviderID, TargetID: in.TargetID, Limit: limit, Offset: offset,
	}
	entries, total, err := t.Events.List(query)
	if err != nil {
		return nil, eventsOutput{}, mcpkit.ErrUnavailable
	}
	stats, err := t.Events.StatsFor(events.Query{ProviderID: in.ProviderID, TargetID: in.TargetID})
	if err != nil {
		return nil, eventsOutput{}, mcpkit.ErrUnavailable
	}
	return nil, page(entries, total, stats, offset, detailed), nil
}

// page renders the entries that fit the result budget and the cursor past them.
func page(entries []events.Entry, total int64, stats events.Stats, offset int, detailed bool) eventsOutput {
	out := eventsOutput{
		Events: make([]eventOut, 0, len(entries)), Total: total,
		Stats: statsOut{
			Events: stats.TotalEvents, LastEventAt: stats.LastEventAt,
			Delivered: stats.DeliverySuccessCount, Failed: stats.DeliveryErrorCount,
		},
	}
	for _, entry := range entries {
		out.Events = append(out.Events, render(entry, detailed))
	}
	all := out.Events
	fit := mcpkit.FitCount(len(all), func(n int) int {
		out.Events = all[:n]
		return mcpkit.EncodedSize(out)
	})
	out.Events = all[:fit]
	if next := offset + fit; int64(next) < total {
		out.NextCursor = mcpkit.EncodeCursor(next)
	}
	return out
}

// getEvent is antenne_get_event: one event with its normalized payload and raw body.
func (t *toolset) getEvent(ctx context.Context, req *sdk.CallToolRequest, in eventInput) (*sdk.CallToolResult, eventFull, error) {
	if _, err := t.callerOf(ctx, req); err != nil {
		return nil, eventFull{}, err
	}
	entry, ok := t.Events.ByID(in.EventID)
	if !ok {
		return nil, eventFull{}, mcpkit.Failure(errEvent, hintEvent)
	}
	out := eventFull{eventOut: render(entry, true)}
	if entry.Envelope == nil {
		return nil, out, nil
	}
	out.Normalized = entry.Envelope.Payload.Normalized
	if entry.Envelope.Payload.Raw != nil {
		raw, _ := json.Marshal(entry.Envelope.Payload.Raw)
		out.Raw, out.Truncated = mcpkit.Clip(string(raw), rawPayloadCap)
	}
	if mcpkit.EncodedSize(out) > mcpkit.ResultBudget {
		out.Normalized, out.Truncated = nil, true
	}
	return nil, out, nil
}

// render turns one log entry into its tool shape.
func render(entry events.Entry, detailed bool) eventOut {
	out := eventOut{
		ID: entry.ID, At: entry.Timestamp, Source: entry.Source, Provider: entry.ProviderName,
		ProviderType: entry.ProviderType, Event: entry.Event, Deliveries: attempts(entry),
	}
	if entry.ProviderID != nil {
		out.ProviderID = *entry.ProviderID
	}
	limit := conciseText
	if detailed {
		limit = detailedText
	}
	if entry.Message != nil {
		out.Message, _ = mcpkit.Clip(envelope.HTMLToPlainText(*entry.Message), limit)
	}
	if entry.Envelope == nil {
		return out
	}
	out.Title, out.URL = text(entry.Envelope.Event.Title), text(entry.Envelope.Event.URL)
	if detailed {
		out.Detail = detailOf(*entry.Envelope)
	}
	return out
}

// detailOf is the type, subject and actor of an envelope.
func detailOf(e envelope.Envelope) *eventDetail {
	detail := &eventDetail{Type: e.Event.Type, Subject: string(e.Subject.Type)}
	if name := text(e.Subject.Name); name != "" {
		detail.Subject += ": " + name
	}
	if e.Actor != nil {
		detail.Actor = text(e.Actor.Name)
		if detail.Actor == "" {
			detail.Actor = text(e.Actor.Email)
		}
	}
	return detail
}

// attempts are the deliveries of an entry, with URLs reduced to their host.
func attempts(entry events.Entry) []attemptOut {
	out := make([]attemptOut, 0, len(entry.Deliveries))
	for _, a := range entry.Deliveries {
		out = append(out, attemptOut{TargetID: a.TargetID, Target: a.TargetName, Status: a.Status, Error: delivery.ScrubURLs(a.Error)})
	}
	return out
}

// text reads an optional string.
func text(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
