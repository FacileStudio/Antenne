package events

import (
	"fmt"
	"strings"
	"time"
)

// List returns a filtered page of the log, newest first, with the total that
// matched the filter.
func (s *Service) List(query Query) ([]Entry, int64, error) {
	base := s.db.Model(&row{})

	if query.ProviderID != "" {
		base = base.Where("provider_id = ?", query.ProviderID)
	}
	if query.Source != "" {
		base = base.Where("source = ?", query.Source)
	}
	if query.TargetID != "" {
		// The attempts live in a jsonb array; @> asks Postgres whether one of
		// them names this target, which the GIN-less index still answers fast
		// at this table size.
		base = base.Where("deliveries @> ?", fmt.Sprintf(`[{"targetId":%q}]`, query.TargetID))
	}
	for _, term := range strings.Fields(strings.ToLower(query.Search)) {
		base = base.Where("search_text LIKE ?", "%"+term+"%")
	}

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	limit := query.Limit
	if limit <= 0 {
		limit = 50
	}

	var stored []row
	if err := base.Order("occurred_at desc, id desc").
		Limit(limit).Offset(query.Offset).Find(&stored).Error; err != nil {
		return nil, 0, err
	}

	entries := make([]Entry, 0, len(stored))
	for _, item := range stored {
		entries = append(entries, decode(item))
	}
	return entries, total, nil
}

// StatsFor summarizes the log under the same filter List takes.
func (s *Service) StatsFor(query Query) (Stats, error) {
	base := s.db.Model(&row{})
	if query.ProviderID != "" {
		base = base.Where("provider_id = ?", query.ProviderID)
	}
	if query.TargetID != "" {
		base = base.Where("deliveries @> ?", fmt.Sprintf(`[{"targetId":%q}]`, query.TargetID))
	}

	var aggregate struct {
		Total       int64
		LastEventAt *time.Time
		Successes   int64
		Errors      int64
	}

	err := base.Select(`
		count(*) AS total,
		max(occurred_at) AS last_event_at,
		coalesce(sum((SELECT count(*) FROM jsonb_array_elements(deliveries) AS d WHERE d->>'status' = 'success')), 0) AS successes,
		coalesce(sum((SELECT count(*) FROM jsonb_array_elements(deliveries) AS d WHERE d->>'status' = 'error')), 0) AS errors
	`).Scan(&aggregate).Error
	if err != nil {
		return Stats{}, err
	}

	stats := Stats{
		TotalEvents:          int(aggregate.Total),
		DeliverySuccessCount: int(aggregate.Successes),
		DeliveryErrorCount:   int(aggregate.Errors),
	}
	if aggregate.LastEventAt != nil {
		formatted := aggregate.LastEventAt.UTC().Format(time.RFC3339Nano)
		stats.LastEventAt = &formatted
	}
	return stats, nil
}
