package settings

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// persist writes the whole document in one transaction. Providers and targets
// are replaced wholesale rather than diffed: the dashboard PUTs the entire
// list, ordering is part of the state, and a partial write would leave the
// runtime routing to targets the operator just deleted.
func (s *Store) persist(next AppSettings) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := persistDocument(tx, next); err != nil {
			return err
		}
		if err := persistProviders(tx, next.Providers.Items); err != nil {
			return err
		}
		return persistTargets(tx, next.Delivery.Items)
	})
}

func persistDocument(tx *gorm.DB, next AppSettings) error {
	system, err := json.Marshal(next.System)
	if err != nil {
		return err
	}
	pool, err := json.Marshal(next.Bus)
	if err != nil {
		return err
	}

	row := appSettingsRow{
		ID:         1,
		Timeout:    next.Timeout,
		ListenPort: next.Providers.Port,
		System:     system,
		Bus:        pool,
		UpdatedAt:  time.Now(),
	}
	return upsert(tx, &row, "timeout", "listen_port", "system", "bus", "updated_at")
}

func persistProviders(tx *gorm.DB, items []Provider) error {
	kept := make([]string, 0, len(items))
	for index, provider := range items {
		stored, err := providerRowOf(provider, index)
		if err != nil {
			return err
		}
		if err := upsert(tx, &stored, "type", "name", "enabled", "position", "tags", "config", "updated_at"); err != nil {
			return err
		}
		kept = append(kept, provider.ID)
	}
	return deleteMissing(tx, &providerRow{}, kept)
}

func providerRowOf(provider Provider, index int) (providerRow, error) {
	config, err := json.Marshal(provider)
	if err != nil {
		return providerRow{}, err
	}
	tags, err := json.Marshal(emptySlice(provider.Tags))
	if err != nil {
		return providerRow{}, err
	}
	return providerRow{
		ID:       provider.ID,
		Type:     string(provider.Type),
		Name:     provider.Name,
		Enabled:  provider.Enabled,
		Position: index,
		Tags:     tags,
		Config:   config,
	}, nil
}

func persistTargets(tx *gorm.DB, items []DeliveryTarget) error {
	kept := make([]string, 0, len(items))
	for index, target := range items {
		stored, err := targetRowOf(target, index)
		if err != nil {
			return err
		}
		columns := []string{"type", "name", "enabled", "position", "provider_ids", "provider_tags", "config", "updated_at"}
		if err := upsert(tx, &stored, columns...); err != nil {
			return err
		}
		kept = append(kept, target.ID)
	}
	return deleteMissing(tx, &targetRow{}, kept)
}

func targetRowOf(target DeliveryTarget, index int) (targetRow, error) {
	config, err := json.Marshal(target)
	if err != nil {
		return targetRow{}, err
	}
	providerIDs, err := json.Marshal(emptySlice(target.ProviderIDs))
	if err != nil {
		return targetRow{}, err
	}
	providerTags, err := json.Marshal(emptySlice(target.ProviderTags))
	if err != nil {
		return targetRow{}, err
	}
	return targetRow{
		ID:           target.ID,
		Type:         string(target.Type),
		Name:         target.Name,
		Enabled:      target.Enabled,
		Position:     index,
		ProviderIDs:  providerIDs,
		ProviderTags: providerTags,
		Config:       config,
	}, nil
}

func upsert(tx *gorm.DB, row any, columns ...string) error {
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns(columns),
	}).Create(row).Error
}

func deleteMissing(tx *gorm.DB, model any, kept []string) error {
	if len(kept) == 0 {
		return tx.Where("1 = 1").Delete(model).Error
	}
	return tx.Where("id NOT IN ?", kept).Delete(model).Error
}
