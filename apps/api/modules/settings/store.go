package settings

import (
	"encoding/json"
	"log/slog"
	"sync"

	"gorm.io/gorm"
)

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
// on first boot.
func NewStore(db *gorm.DB, log *slog.Logger, defaults AppSettings) (*Store, error) {
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

	if err := store.persist(defaults); err != nil {
		return nil, err
	}
	store.current = defaults
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
	if source.Bus != nil {
		pool := *source.Bus
		clone.Bus = &pool
	}
	return clone
}
