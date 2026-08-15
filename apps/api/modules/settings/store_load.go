package settings

import (
	"encoding/json"
	"errors"

	"gorm.io/gorm"
)

func (s *Store) load() (AppSettings, bool, error) {
	var row appSettingsRow
	err := s.db.First(&row, 1).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return AppSettings{}, false, nil
	}
	if err != nil {
		return AppSettings{}, false, err
	}

	loaded := AppSettings{Timeout: row.Timeout}
	loaded.Providers.Port = row.ListenPort
	s.hydrate("system", "app_settings", row.System, &loaded.System)
	loaded.Bus = s.loadBus(row.Bus)

	loaded.Providers.Items, err = s.loadProviders()
	if err != nil {
		return AppSettings{}, false, err
	}
	loaded.Delivery.Items, err = s.loadTargets()
	if err != nil {
		return AppSettings{}, false, err
	}

	return loaded, true, nil
}

// hydrate applies a stored JSON blob onto target, leaving target untouched when
// the blob is absent or no longer parses.
//
// A blob that fails to decode is logged and skipped rather than returned as an
// error, because the row's indexed columns already carry everything needed to
// identify it: one unreadable config costs that provider its options, where
// propagating the failure would leave the whole instance with no configuration
// at all and no way to open the dashboard and repair it.
func (s *Store) hydrate(field string, id string, blob []byte, target any) {
	if len(blob) == 0 {
		return
	}
	if err := json.Unmarshal(blob, target); err != nil {
		s.log.Warn("stored settings blob ignored", "field", field, "id", id, "error", err)
	}
}

func (s *Store) loadBus(blob []byte) *BusSettings {
	if len(blob) == 0 {
		return nil
	}
	pool := BusSettings{}
	if err := json.Unmarshal(blob, &pool); err != nil {
		s.log.Warn("stored settings blob ignored", "field", "bus", "id", "app_settings", "error", err)
		return nil
	}
	return &pool
}

func (s *Store) loadProviders() ([]Provider, error) {
	var rows []providerRow
	if err := s.db.Order("position asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]Provider, 0, len(rows))
	for _, stored := range rows {
		items = append(items, s.providerFrom(stored))
	}
	return items, nil
}

// providerFrom rebuilds a provider from its row. The base columns are applied
// after the config blob rather than before it, so a stale copy of the name,
// type or enabled flag inside the blob can never win over the indexed truth the
// rest of the system queries on.
func (s *Store) providerFrom(stored providerRow) Provider {
	provider := Provider{
		ID:      stored.ID,
		Type:    ProviderType(stored.Type),
		Name:    stored.Name,
		Enabled: stored.Enabled,
		Tags:    []string{},
	}
	s.hydrate("tags", stored.ID, stored.Tags, &provider.Tags)
	s.hydrate("config", stored.ID, stored.Config, &provider)

	provider.ID = stored.ID
	provider.Type = ProviderType(stored.Type)
	provider.Name = stored.Name
	provider.Enabled = stored.Enabled
	if provider.Tags == nil {
		provider.Tags = []string{}
	}
	return provider
}

func (s *Store) loadTargets() ([]DeliveryTarget, error) {
	var rows []targetRow
	if err := s.db.Order("position asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]DeliveryTarget, 0, len(rows))
	for _, stored := range rows {
		items = append(items, s.targetFrom(stored))
	}
	return items, nil
}

// targetFrom rebuilds a delivery target from its row, restoring the base
// columns after the config blob for the same reason providerFrom does.
func (s *Store) targetFrom(stored targetRow) DeliveryTarget {
	target := DeliveryTarget{
		ID:           stored.ID,
		Type:         DeliveryTargetType(stored.Type),
		Name:         stored.Name,
		Enabled:      stored.Enabled,
		ProviderIDs:  []string{},
		ProviderTags: []string{},
	}
	s.hydrate("provider_ids", stored.ID, stored.ProviderIDs, &target.ProviderIDs)
	s.hydrate("provider_tags", stored.ID, stored.ProviderTags, &target.ProviderTags)
	s.hydrate("config", stored.ID, stored.Config, &target)

	target.ID = stored.ID
	target.Type = DeliveryTargetType(stored.Type)
	target.Name = stored.Name
	target.Enabled = stored.Enabled
	if target.ProviderIDs == nil {
		target.ProviderIDs = []string{}
	}
	if target.ProviderTags == nil {
		target.ProviderTags = []string{}
	}
	return target
}
