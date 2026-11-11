// Package pool is the Antenne Pool: a WebSocket bus with an offset-based queue
// that other Facile apps register with and sync through.
package bus

import (
	"encoding/json"
	"log/slog"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/FacileStudio/Antenne/apps/api/modules/settings"
)

// Retention policy. Interest-based rather than time-based: events stay until
// every active listener has acked them, past a floor kept for replay and
// debugging, and a ceiling backstops everything.
const (
	retentionCeiling = 30 * 24 * time.Hour
	retentionFloor   = 48 * time.Hour
	inactiveAfter    = 14 * 24 * time.Hour
)

type appRow struct {
	AppID          string `gorm:"primaryKey;column:app_id"`
	App            string
	Token          string
	Fingerprint    *string    `gorm:"column:secret_fingerprint"`
	EmitChannels   []byte     `gorm:"column:emit_channels;type:jsonb"`
	ListenChannels []byte     `gorm:"column:listen_channels;type:jsonb"`
	RegisteredAt   time.Time  `gorm:"column:registered_at"`
	LastSeenAt     *time.Time `gorm:"column:last_seen_at"`
}

func (appRow) TableName() string { return "pool_apps" }

type eventRow struct {
	ID        string `gorm:"primaryKey"`
	Channel   string
	OffsetNum int64  `gorm:"column:offset_num"`
	Payload   []byte `gorm:"type:jsonb"`
	Sender    string
	EmittedAt time.Time `gorm:"column:emitted_at"`
}

func (eventRow) TableName() string { return "pool_events" }

type offsetRow struct {
	Channel    string `gorm:"primaryKey"`
	NextOffset int64  `gorm:"column:next_offset"`
}

func (offsetRow) TableName() string { return "pool_channel_offsets" }

type subscriptionRow struct {
	SubscriberID string `gorm:"primaryKey;column:subscriber_id"`
	Channel      string `gorm:"primaryKey"`
	LastAcked    int64  `gorm:"column:last_acked"`
}

func (subscriptionRow) TableName() string { return "pool_subscriptions" }

// RegisteredApp is a pool client's identity and its declared channels.
type RegisteredApp struct {
	AppID          string   `json:"app_id"`
	App            string   `json:"app"`
	Token          string   `json:"token"`
	EmitChannels   []string `json:"emit_channels"`
	ListenChannels []string `json:"listen_channels"`
	RegisteredAt   string   `json:"registered_at"`
	fingerprint    string
}

// Event is one queued message.
type Event struct {
	ID        string          `json:"id"`
	Channel   string          `json:"channel"`
	Offset    int64           `json:"offset"`
	Payload   json.RawMessage `json:"payload"`
	Sender    string          `json:"sender"`
	Timestamp int64           `json:"timestamp"`
}

// Queue is the durable half of the pool.
type Queue struct {
	db    *gorm.DB
	log   *slog.Logger
	epoch string
}

// NewQueue builds the queue with a fresh epoch.
//
// The epoch changes on every boot and is sent in the welcome frame, so a client
// can tell a restart from a reconnect and resync from its own cursor rather
// than trusting offsets across a gap.
func NewQueue(db *gorm.DB, log *slog.Logger) *Queue {
	return &Queue{db: db, log: log, epoch: settings.NewID()}
}

// Epoch is this boot's identifier.
func (q *Queue) Epoch() string { return q.epoch }

// Push appends an event to a channel and returns it with its assigned offset.
func (q *Queue) Push(channel string, payload json.RawMessage, sender string) (Event, error) {
	event := Event{
		ID:        settings.NewID(),
		Channel:   channel,
		Payload:   payload,
		Sender:    sender,
		Timestamp: time.Now().UnixMilli(),
	}

	if q.db == nil {
		event.Offset = 1
		return event, nil
	}

	err := q.db.Transaction(func(tx *gorm.DB) error {
		// The row lock is what keeps two concurrent emitters on one channel
		// from being handed the same offset.
		var current offsetRow
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("channel = ?", channel).First(&current).Error

		switch {
		case err == nil:
			event.Offset = current.NextOffset
			if err := tx.Model(&current).Update("next_offset", current.NextOffset+1).Error; err != nil {
				return err
			}
		case err == gorm.ErrRecordNotFound:
			event.Offset = 1
			if err := tx.Create(&offsetRow{Channel: channel, NextOffset: 2}).Error; err != nil {
				return err
			}
		default:
			return err
		}

		return tx.Create(&eventRow{
			ID:        event.ID,
			Channel:   channel,
			OffsetNum: event.Offset,
			Payload:   payload,
			Sender:    sender,
			EmittedAt: time.UnixMilli(event.Timestamp).UTC(),
		}).Error
	})
	if err != nil {
		return Event{}, err
	}

	return event, nil
}

// After returns up to limit events on a channel past an offset.
func (q *Queue) After(channel string, afterOffset int64, limit int) ([]Event, error) {
	var rows []eventRow
	err := q.db.Where("channel = ? AND offset_num > ?", channel, afterOffset).
		Order("offset_num asc").Limit(limit).Find(&rows).Error
	if err != nil {
		return nil, err
	}

	events := make([]Event, 0, len(rows))
	for _, row := range rows {
		events = append(events, Event{
			ID:        row.ID,
			Channel:   row.Channel,
			Offset:    row.OffsetNum,
			Payload:   row.Payload,
			Sender:    row.Sender,
			Timestamp: row.EmittedAt.UnixMilli(),
		})
	}
	return events, nil
}

// Ack records how far a subscriber has consumed a channel.
func (q *Queue) Ack(subscriberID, channel string, offset int64) error {
	return q.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "subscriber_id"}, {Name: "channel"}},
		DoUpdates: clause.AssignmentColumns([]string{"last_acked"}),
	}).Create(&subscriptionRow{SubscriberID: subscriberID, Channel: channel, LastAcked: offset}).Error
}

// LastAcked returns a subscriber's cursor on a channel.
func (q *Queue) LastAcked(subscriberID, channel string) int64 {
	var row subscriptionRow
	if err := q.db.Where("subscriber_id = ? AND channel = ?", subscriberID, channel).First(&row).Error; err != nil {
		return 0
	}
	return row.LastAcked
}

// Touch records that an app is alive, which is what pins retention to it.
func (q *Queue) Touch(appID string) {
	now := time.Now()
	q.db.Model(&appRow{}).Where("app_id = ?", appID).Update("last_seen_at", now)
}

// Register issues a fresh token for an app and records its declared channels.
//
// Every registration replaces the app's previous token, so knowing the secret
// never hands out a token someone else is using. The app keeps its id, which
// is what its subscriptions and retention are keyed by. The token is bound to
// fingerprint, the keyed hash of the secret it was issued under.
func (q *Queue) Register(app string, emitChannels, listenChannels []string, fingerprint string) (RegisteredApp, error) {
	emit, err := json.Marshal(emitChannels)
	if err != nil {
		return RegisteredApp{}, err
	}
	listen, err := json.Marshal(listenChannels)
	if err != nil {
		return RegisteredApp{}, err
	}

	row := appRow{
		AppID:          settings.NewID(),
		App:            app,
		Token:          settings.NewID(),
		Fingerprint:    &fingerprint,
		EmitChannels:   emit,
		ListenChannels: listen,
		RegisteredAt:   time.Now().UTC(),
	}
	err = q.db.Clauses(
		clause.OnConflict{
			Columns:   []clause.Column{{Name: "app"}},
			DoUpdates: clause.AssignmentColumns([]string{"token", "secret_fingerprint", "emit_channels", "listen_channels"}),
		},
		clause.Returning{Columns: []clause.Column{{Name: "app_id"}, {Name: "registered_at"}}},
	).Create(&row).Error
	if err != nil {
		return RegisteredApp{}, err
	}

	return RegisteredApp{
		AppID:          row.AppID,
		App:            row.App,
		Token:          row.Token,
		EmitChannels:   emitChannels,
		ListenChannels: listenChannels,
		RegisteredAt:   row.RegisteredAt.UTC().Format(time.RFC3339),
		fingerprint:    fingerprint,
	}, nil
}

// ByToken resolves a connection token to its app. A token issued under any
// secret other than the one fingerprint names is refused.
func (q *Queue) ByToken(token, fingerprint string) (RegisteredApp, bool) {
	var row appRow
	if err := q.db.Where("token = ? AND secret_fingerprint = ?", token, fingerprint).First(&row).Error; err != nil {
		return RegisteredApp{}, false
	}

	app := RegisteredApp{
		AppID:        row.AppID,
		App:          row.App,
		Token:        row.Token,
		RegisteredAt: row.RegisteredAt.UTC().Format(time.RFC3339),
		fingerprint:  fingerprint,
	}
	app.EmitChannels = storedChannels(row.EmitChannels)
	app.ListenChannels = storedChannels(row.ListenChannels)
	return app, true
}

// adoptUnbound binds every token that predates fingerprints to the current
// secret. It runs at boot, so the deploy that adds the column disconnects
// nobody; a token that is still unbound is refused.
func (q *Queue) adoptUnbound(fingerprint string) (int64, error) {
	result := q.db.Model(&appRow{}).Where("secret_fingerprint IS NULL").Update("secret_fingerprint", fingerprint)
	return result.RowsAffected, result.Error
}

// storedChannels reads a stored channel list. A list that no longer parses is
// read as declaring nothing rather than everything: an app that appears to
// declare no channels pins no retention and is fanned out to by nobody, where
// guessing "all of them" would push events at a client that never asked.
func storedChannels(blob []byte) []string {
	var channels []string
	if json.Unmarshal(blob, &channels) != nil {
		return nil
	}
	return channels
}
