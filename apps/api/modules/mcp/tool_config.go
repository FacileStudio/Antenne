package mcp

import (
	"context"
	"net/url"
	"strconv"
	"time"

	"github.com/FacileStudio/Antenne/apps/api/internal/server"
	"github.com/FacileStudio/Antenne/apps/api/modules/delivery"
	"github.com/FacileStudio/Antenne/apps/api/modules/settings"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// listProviders is antenne_list_providers: every provider, without its credentials.
func (t *toolset) listProviders(ctx context.Context, req *sdk.CallToolRequest, _ emptyInput) (*sdk.CallToolResult, providersOutput, error) {
	if _, err := t.callerOf(ctx, req); err != nil {
		return nil, providersOutput{}, err
	}
	items := t.Pipeline.Settings().Providers.Items
	out := providersOutput{Providers: make([]providerOut, 0, len(items))}
	for _, p := range items {
		tags := p.Tags
		if tags == nil {
			tags = []string{}
		}
		out.Providers = append(out.Providers, providerOut{
			ID: p.ID, Type: string(p.Type), Name: p.Name, Enabled: p.Enabled, Tags: tags, Watches: watches(p),
		})
	}
	return nil, out, nil
}

// watches is what a provider listens to, reduced to what carries no credential: the
// intake path of a webhook, the host of a website, a feed or a mailbox.
func watches(p settings.Provider) string {
	switch {
	case p.Type.IsWebhook():
		return "/webhook" + p.Path
	case p.Type == settings.ProviderWebsite:
		return hostOf(p.URL)
	case p.Type == settings.ProviderRSS:
		return hostOf(p.FeedURL)
	case p.Type == settings.ProviderIMAP:
		return p.Host + ":" + strconv.Itoa(p.Port)
	}
	return ""
}

// hostOf is the scheme and host of a URL, or nothing when it does not parse.
func hostOf(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return ""
	}
	return parsed.Scheme + "://" + parsed.Host
}

// listTargets is antenne_list_targets: every delivery target with its routing and its
// health, without its credentials or address.
func (t *toolset) listTargets(ctx context.Context, req *sdk.CallToolRequest, _ emptyInput) (*sdk.CallToolResult, targetsOutput, error) {
	if _, err := t.callerOf(ctx, req); err != nil {
		return nil, targetsOutput{}, err
	}
	healthPage := t.Health.HealthPage()
	rows := map[string]server.TargetHealth{}
	for _, row := range healthPage.Targets {
		rows[row.TargetID] = row
	}
	items := t.Pipeline.Settings().Delivery.Items
	out := targetsOutput{
		Targets: make([]targetOut, 0, len(items)), AnyDeliverable: healthPage.AnyDeliverable,
		StandingAlerts: make([]standingOut, 0, len(healthPage.StandingAlerts)),
	}
	for _, target := range items {
		out.Targets = append(out.Targets, targetOf(target, rows[target.ID]))
	}
	for _, alert := range healthPage.StandingAlerts {
		out.StandingAlerts = append(out.StandingAlerts, standingOut{
			TargetID: alert.TargetID, Target: alert.TargetName, Title: alert.Title, Detail: delivery.ScrubURLs(alert.Detail),
			Since: alert.Since.UTC().Format(time.RFC3339),
		})
	}
	return nil, out, nil
}

// targetOf renders one target and its health row.
func targetOf(target settings.DeliveryTarget, row server.TargetHealth) targetOut {
	out := targetOut{
		ID: target.ID, Type: string(target.Type), Name: target.Name, Enabled: target.Enabled,
		ProviderIDs: orEmpty(target.ProviderIDs), ProviderTags: orEmpty(target.ProviderTags),
		State: row.State, Deliverable: row.Deliverable, Detail: delivery.ScrubURLs(row.Detail),
		DownSince: stamp(row.DownSince), LastSuccess: stamp(row.LastSuccessAt),
	}
	if out.State == "" {
		out.State = "unknown"
	}
	return out
}

// poolStatus is antenne_pool_status: the suite apps connected to the bus. Tokens and the
// pool secret are never part of it.
func (t *toolset) poolStatus(ctx context.Context, req *sdk.CallToolRequest, _ emptyInput) (*sdk.CallToolResult, poolOutput, error) {
	if _, err := t.callerOf(ctx, req); err != nil {
		return nil, poolOutput{}, err
	}
	stats := t.Pool.Stats()
	out := poolOutput{Connections: stats.Connections, Apps: make([]poolApp, 0, len(stats.Apps))}
	for _, app := range stats.Apps {
		out.Apps = append(out.Apps, poolApp{App: app.App, AppID: app.AppID, Channels: orEmpty(app.Channels), Emits: orEmpty(app.Emits)})
	}
	return nil, out, nil
}

// orEmpty keeps a nil list from encoding as null.
func orEmpty(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

// stamp renders an optional instant in UTC.
func stamp(at *time.Time) string {
	if at == nil {
		return ""
	}
	return at.UTC().Format(time.RFC3339)
}
