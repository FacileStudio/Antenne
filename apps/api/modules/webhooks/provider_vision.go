package webhooks

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/FacileStudio/Antenne/apps/api/internal/envelope"
	"github.com/FacileStudio/Antenne/apps/api/modules/settings"
)

type visionEntry struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type visionPayload struct {
	EventType string `json:"event_type"`
	Site      *struct {
		ID     int    `json:"id"`
		Name   string `json:"name"`
		Domain string `json:"domain"`
	} `json:"site"`
	Period struct {
		Type string `json:"type"`
		From string `json:"from"`
		To   string `json:"to"`
	} `json:"period"`
	Metrics *struct {
		TotalPageviews     int     `json:"total_pageviews"`
		UniqueVisitors     int     `json:"unique_visitors"`
		ViewsPerVisitor    float64 `json:"views_per_visitor"`
		PageviewsChangePct float64 `json:"pageviews_change_pct"`
		VisitorsChangePct  float64 `json:"visitors_change_pct"`
	} `json:"metrics"`
	TopPages     []visionEntry `json:"top_pages"`
	TopReferrers []visionEntry `json:"top_referrers"`
}

var vision = Definition{
	Type:         settings.ProviderVision,
	Label:        "Vision",
	Authenticate: signedWith("sha256="),
	Handle: func(body []byte, _ Context) (Result, error) {
		var payload visionPayload
		if err := decode(body, &payload); err != nil {
			return Result{}, err
		}

		if payload.EventType == "" || payload.Site == nil || payload.Metrics == nil {
			return Result{Description: "unknown vision event"}, nil
		}

		site := payload.Site
		metrics := payload.Metrics
		period := capitalize(payload.Period.Type)

		description := fmt.Sprintf("%s analytics report for %s: %s pageviews, %s visitors",
			period, site.Domain, thousands(metrics.TotalPageviews), thousands(metrics.UniqueVisitors))

		lines := []string{
			fmt.Sprintf("<p><strong>📊 Vision Analytics — %s</strong></p>", escape(site.Domain)),
			fmt.Sprintf("<p>%s report (%s → %s)</p>", escape(period), escape(payload.Period.From), escape(payload.Period.To)),
			fmt.Sprintf("<p><strong>Pageviews:</strong> %s (%s)</p>", thousands(metrics.TotalPageviews), changeMarkup(metrics.PageviewsChangePct)),
			fmt.Sprintf("<p><strong>Visitors:</strong> %s (%s)</p>", thousands(metrics.UniqueVisitors), changeMarkup(metrics.VisitorsChangePct)),
			fmt.Sprintf("<p><strong>Views/Visitor:</strong> %.1f</p>", metrics.ViewsPerVisitor),
		}
		if len(payload.TopPages) > 0 {
			lines = append(lines, fmt.Sprintf("<p><strong>Top Pages:</strong> %s</p>", topList(payload.TopPages)))
		}
		if len(payload.TopReferrers) > 0 {
			lines = append(lines, fmt.Sprintf("<p><strong>Top Sources:</strong> %s</p>", topList(payload.TopReferrers)))
		}

		return Result{
			Description: description,
			MessageHTML: strings.Join(lines, "\n"),
			EventType:   "vision." + payload.EventType,
			EventTitle:  fmt.Sprintf("%s Analytics — %s", period, site.Domain),
			Subject: &envelope.Subject{
				Type: envelope.SubjectService,
				ID:   str(strconv.Itoa(site.ID)),
				Name: str(site.Domain),
			},
			Normalized: map[string]any{
				"period":             payload.Period.Type,
				"siteDomain":         site.Domain,
				"totalPageviews":     metrics.TotalPageviews,
				"uniqueVisitors":     metrics.UniqueVisitors,
				"viewsPerVisitor":    metrics.ViewsPerVisitor,
				"pageviewsChangePct": metrics.PageviewsChangePct,
				"visitorsChangePct":  metrics.VisitorsChangePct,
				"status":             "delivered",
			},
			// The analytics digest carries no personal data, so the raw payload
			// is the one Antenne forwards intact — a receiver can chart it.
			RawPayloadAllowed: true,
		}, nil
	},
}

func changeMarkup(percent float64) string {
	switch {
	case percent > 0:
		return fmt.Sprintf(`<span style="color:green">↑ +%.1f%%</span>`, percent)
	case percent < 0:
		return fmt.Sprintf(`<span style="color:red">↓ %.1f%%</span>`, percent)
	}
	return "<span>0%</span>"
}

func topList(entries []visionEntry) string {
	if len(entries) > 3 {
		entries = entries[:3]
	}
	parts := make([]string, 0, len(entries))
	for _, entry := range entries {
		parts = append(parts, fmt.Sprintf("%s (%s)", escape(entry.Name), thousands(entry.Count)))
	}
	return strings.Join(parts, ", ")
}

func capitalize(value string) string {
	if value == "" {
		return value
	}
	return strings.ToUpper(value[:1]) + value[1:]
}

// thousands groups a count with commas, matching the toLocaleString() the old
// dashboard produced.
func thousands(value int) string {
	digits := strconv.Itoa(value)
	sign := ""
	if strings.HasPrefix(digits, "-") {
		sign, digits = "-", digits[1:]
	}

	var grouped strings.Builder
	for index, digit := range digits {
		if index > 0 && (len(digits)-index)%3 == 0 {
			grouped.WriteByte(',')
		}
		grouped.WriteRune(digit)
	}
	return sign + grouped.String()
}
