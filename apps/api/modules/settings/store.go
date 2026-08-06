package settings

import (
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"sync"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type appSettingsRow struct {
	ID         int `gorm:"primaryKey"`
	Timeout    int
	ListenPort int
	System     []byte `gorm:"type:jsonb"`
	Pool       []byte `gorm:"type:jsonb"`
	UpdatedAt  time.Time
}

func (appSettingsRow) TableName() string { return "app_settings" }

type providerRow struct {
	ID        string `gorm:"primaryKey"`
	Type      string
	Name      string
	Enabled   bool
	Position  int
	Tags      []byte `gorm:"type:jsonb"`
	Config    []byte `gorm:"type:jsonb"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (providerRow) TableName() string { return "providers" }

type targetRow struct {
	ID           string `gorm:"primaryKey"`
	Type         string
	Name         string
	Enabled      bool
	Position     int
	ProviderIDs  []byte `gorm:"column:provider_ids;type:jsonb"`
	ProviderTags []byte `gorm:"column:provider_tags;type:jsonb"`
	Config       []byte `gorm:"type:jsonb"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (targetRow) TableName() string { return "delivery_targets" }

// Store holds the settings document in memory and writes every change through
// to Postgres. Reads are served from memory because the monitor, IMAP and RSS
// managers ask for the current settings on every tick.
type Store struct {
	db  *gorm.DB
	log *slog.Logger

	mu        sync.RWMutex
	current   AppSettings
	persisted bool
}

// NewStore loads the settings from the database, seeding them from the defaults
// — or from a legacy nook.json, if one is still on the volume — on first boot.
func NewStore(db *gorm.DB, log *slog.Logger, defaults AppSettings, legacyPath string) (*Store, error) {
	store := &Store{db: db, log: log, current: defaults}

	loaded, found, err := store.load()
	if err != nil {
		return nil, err
	}

	if found {
		store.current = loaded
		store.persisted = true
		return store, nil
	}

	seed := defaults
	imported := false
	if legacyPath != "" {
		if legacy, ok := store.importLegacy(legacyPath, defaults); ok {
			seed, imported = legacy, true
		}
	}

	if err := store.persist(seed); err != nil {
		return nil, err
	}
	store.current = seed
	// An import is a real configuration, not the defaults, and the dashboard
	// says so. Without this the boot that imports 16 providers still reports
	// "Defaults" until the next restart.
	store.persisted = imported
	return store, nil
}

// Get returns a deep copy of the current settings, so a caller mutating a slice
// cannot corrupt the shared document.
func (s *Store) Get() AppSettings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneSettings(s.current)
}

// Persisted reports whether the settings were ever written through the
// dashboard, as opposed to seeded from the defaults.
func (s *Store) Persisted() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.persisted
}

// Update normalizes an incoming document against the current one and writes it.
// It returns the previous and next settings so the caller can diff them for
// system events.
func (s *Store) Update(raw json.RawMessage) (previous AppSettings, next AppSettings, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	previous = cloneSettings(s.current)

	next, err = Normalize(raw, previous)
	if err != nil {
		return AppSettings{}, AppSettings{}, err
	}

	if err := s.persist(next); err != nil {
		return AppSettings{}, AppSettings{}, err
	}

	s.current = cloneSettings(next)
	s.persisted = true
	return previous, cloneSettings(next), nil
}

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
	if len(row.System) > 0 {
		_ = json.Unmarshal(row.System, &loaded.System)
	}
	if len(row.Pool) > 0 {
		pool := PoolSettings{}
		if err := json.Unmarshal(row.Pool, &pool); err == nil {
			loaded.Pool = &pool
		}
	}

	var providerRows []providerRow
	if err := s.db.Order("position asc").Find(&providerRows).Error; err != nil {
		return AppSettings{}, false, err
	}
	loaded.Providers.Items = make([]Provider, 0, len(providerRows))
	for _, stored := range providerRows {
		provider := Provider{
			ID:      stored.ID,
			Type:    ProviderType(stored.Type),
			Name:    stored.Name,
			Enabled: stored.Enabled,
			Tags:    []string{},
		}
		_ = json.Unmarshal(stored.Tags, &provider.Tags)
		_ = json.Unmarshal(stored.Config, &provider)
		// The config blob does not carry the base columns; restore them so a
		// stale copy inside config can never win over the indexed truth.
		provider.ID = stored.ID
		provider.Type = ProviderType(stored.Type)
		provider.Name = stored.Name
		provider.Enabled = stored.Enabled
		if provider.Tags == nil {
			provider.Tags = []string{}
		}
		loaded.Providers.Items = append(loaded.Providers.Items, provider)
	}

	var targetRows []targetRow
	if err := s.db.Order("position asc").Find(&targetRows).Error; err != nil {
		return AppSettings{}, false, err
	}
	loaded.Delivery.Items = make([]DeliveryTarget, 0, len(targetRows))
	for _, stored := range targetRows {
		target := DeliveryTarget{
			ID:           stored.ID,
			Type:         DeliveryTargetType(stored.Type),
			Name:         stored.Name,
			Enabled:      stored.Enabled,
			ProviderIDs:  []string{},
			ProviderTags: []string{},
		}
		_ = json.Unmarshal(stored.ProviderIDs, &target.ProviderIDs)
		_ = json.Unmarshal(stored.ProviderTags, &target.ProviderTags)
		_ = json.Unmarshal(stored.Config, &target)
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
		loaded.Delivery.Items = append(loaded.Delivery.Items, target)
	}

	return loaded, true, nil
}

// persist writes the whole document in one transaction. Providers and targets
// are replaced wholesale rather than diffed: the dashboard PUTs the entire
// list, ordering is part of the state, and a partial write would leave the
// runtime routing to targets the operator just deleted.
func (s *Store) persist(next AppSettings) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		system, err := json.Marshal(next.System)
		if err != nil {
			return err
		}
		pool, err := json.Marshal(next.Pool)
		if err != nil {
			return err
		}

		row := appSettingsRow{
			ID:         1,
			Timeout:    next.Timeout,
			ListenPort: next.Providers.Port,
			System:     system,
			Pool:       pool,
			UpdatedAt:  time.Now(),
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{"timeout", "listen_port", "system", "pool", "updated_at"}),
		}).Create(&row).Error; err != nil {
			return err
		}

		keptProviders := make([]string, 0, len(next.Providers.Items))
		for index, provider := range next.Providers.Items {
			config, err := json.Marshal(provider)
			if err != nil {
				return err
			}
			tags, err := json.Marshal(emptySlice(provider.Tags))
			if err != nil {
				return err
			}
			stored := providerRow{
				ID:       provider.ID,
				Type:     string(provider.Type),
				Name:     provider.Name,
				Enabled:  provider.Enabled,
				Position: index,
				Tags:     tags,
				Config:   config,
			}
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "id"}},
				DoUpdates: clause.AssignmentColumns([]string{"type", "name", "enabled", "position", "tags", "config", "updated_at"}),
			}).Create(&stored).Error; err != nil {
				return err
			}
			keptProviders = append(keptProviders, provider.ID)
		}
		if err := deleteMissing(tx, &providerRow{}, keptProviders); err != nil {
			return err
		}

		keptTargets := make([]string, 0, len(next.Delivery.Items))
		for index, target := range next.Delivery.Items {
			config, err := json.Marshal(target)
			if err != nil {
				return err
			}
			providerIDs, err := json.Marshal(emptySlice(target.ProviderIDs))
			if err != nil {
				return err
			}
			providerTags, err := json.Marshal(emptySlice(target.ProviderTags))
			if err != nil {
				return err
			}
			stored := targetRow{
				ID:           target.ID,
				Type:         string(target.Type),
				Name:         target.Name,
				Enabled:      target.Enabled,
				Position:     index,
				ProviderIDs:  providerIDs,
				ProviderTags: providerTags,
				Config:       config,
			}
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "id"}},
				DoUpdates: clause.AssignmentColumns([]string{"type", "name", "enabled", "position", "provider_ids", "provider_tags", "config", "updated_at"}),
			}).Create(&stored).Error; err != nil {
				return err
			}
			keptTargets = append(keptTargets, target.ID)
		}
		return deleteMissing(tx, &targetRow{}, keptTargets)
	})
}

func deleteMissing(tx *gorm.DB, model any, kept []string) error {
	if len(kept) == 0 {
		return tx.Where("1 = 1").Delete(model).Error
	}
	return tx.Where("id NOT IN ?", kept).Delete(model).Error
}

// importLegacy reads a pre-Postgres nook.json so an existing deployment keeps
// its providers, targets and secrets across the migration. It runs once: after
// the first write the database is the only source, and the file is left on disk
// untouched as a rollback copy.
func (s *Store) importLegacy(path string, defaults AppSettings) (AppSettings, bool) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			s.log.Warn("could not read the legacy settings file", slog.String("path", path), slog.Any("error", err))
		}
		return AppSettings{}, false
	}

	imported, err := Normalize(raw, defaults)
	if err != nil {
		s.log.Error("the legacy settings file could not be imported, starting from defaults",
			slog.String("path", path), slog.Any("error", err))
		return AppSettings{}, false
	}

	s.log.Info("imported the legacy settings file into Postgres",
		slog.String("path", path),
		slog.Int("providers", len(imported.Providers.Items)),
		slog.Int("delivery_targets", len(imported.Delivery.Items)))
	return imported, true
}

func cloneSettings(source AppSettings) AppSettings {
	clone := source
	clone.Providers.Items = make([]Provider, len(source.Providers.Items))
	for index, provider := range source.Providers.Items {
		copied := provider
		copied.Tags = append([]string{}, provider.Tags...)
		clone.Providers.Items[index] = copied
	}
	clone.Delivery.Items = make([]DeliveryTarget, len(source.Delivery.Items))
	for index, target := range source.Delivery.Items {
		copied := target
		copied.ProviderIDs = append([]string{}, target.ProviderIDs...)
		copied.ProviderTags = append([]string{}, target.ProviderTags...)
		copied.RoomIDs = append([]string{}, target.RoomIDs...)
		copied.ToAddresses = append([]string{}, target.ToAddresses...)
		copied.Tags = append([]string{}, target.Tags...)
		clone.Delivery.Items[index] = copied
	}
	if source.Pool != nil {
		pool := *source.Pool
		clone.Pool = &pool
	}
	return clone
}
