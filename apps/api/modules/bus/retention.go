package bus

import (
	"log/slog"
	"time"

	"gorm.io/gorm"
)

// activeListener is one app still pinning a channel's retention, and how far it
// has acked.
type activeListener struct {
	app   string
	acked int64
}

// Cleanup applies the retention policy.
//
// Events acked by every active listener of a channel are deleted past the 48h
// floor. An app not seen for 14 days stops pinning the log, NATS-style. The
// 30-day ceiling backstops everything; crossing it while a listener is still
// behind is data loss, so it logs loudly rather than silently dropping.
func (q *Queue) Cleanup() {
	now := time.Now()

	apps, channels, ok := q.retentionInputs()
	if !ok {
		return
	}

	appByID := map[string]appRow{}
	for _, app := range apps {
		appByID[app.AppID] = app
	}

	for _, channel := range channels {
		q.cleanupChannel(channel, apps, appByID, now)
	}

	q.report(q.db.Where("emitted_at < ?", now.Add(-retentionCeiling)).Delete(&eventRow{}),
		"pool cleanup failed to enforce the retention ceiling")
}

// retentionInputs reads the apps and the channels the policy is computed over.
// A read that fails aborts the whole pass rather than half of it: deleting
// against a partial list of listeners is exactly how an unacked event goes.
func (q *Queue) retentionInputs() ([]appRow, []string, bool) {
	var apps []appRow
	if err := q.db.Find(&apps).Error; err != nil {
		q.log.Error("pool cleanup failed to read apps", slog.Any("error", err))
		return nil, nil, false
	}

	var channels []string
	if err := q.db.Model(&eventRow{}).Distinct().Pluck("channel", &channels).Error; err != nil {
		q.log.Error("pool cleanup failed to read channels", slog.Any("error", err))
		return nil, nil, false
	}
	return apps, channels, true
}

func (q *Queue) cleanupChannel(channel string, apps []appRow, appByID map[string]appRow, now time.Time) {
	active := q.activeListeners(channel, apps, appByID, now)
	if len(active) == 0 {
		return
	}

	watermark := active[0].acked
	for _, listener := range active {
		if listener.acked < watermark {
			watermark = listener.acked
		}
	}
	if watermark > 0 {
		q.report(q.db.Where("channel = ? AND offset_num <= ? AND emitted_at < ?",
			channel, watermark, now.Add(-retentionFloor)).Delete(&eventRow{}),
			"pool cleanup failed to trim acked events")
	}

	q.warnOnCeilingLoss(channel, active, now)
}

// activeListeners is who still pins this channel: everyone with an interest in
// it, minus anyone unseen for longer than inactiveAfter.
func (q *Queue) activeListeners(channel string, apps []appRow, appByID map[string]appRow, now time.Time) []activeListener {
	interest := q.interestIn(channel, apps)

	active := make([]activeListener, 0, len(interest))
	for appID, acked := range interest {
		app, ok := appByID[appID]
		if !ok {
			continue
		}
		if app.LastSeenAt != nil && now.Sub(*app.LastSeenAt) < inactiveAfter {
			active = append(active, activeListener{app: app.App, acked: acked})
			continue
		}
		if app.LastSeenAt != nil {
			q.log.Warn("pool listener inactive for over 14 days, no longer pinning retention",
				slog.String("app", app.App), slog.String("channel", channel))
		}
	}
	return active
}

// interestIn maps every app with a claim on this channel to how far it has
// acked, whether the claim comes from its declaration or from a live
// subscription. A declaration is not the whole truth: a client may subscribe to
// channels it never declared, and its cursor has to pin retention too.
func (q *Queue) interestIn(channel string, apps []appRow) map[string]int64 {
	interest := map[string]int64{}
	for _, app := range apps {
		for _, candidate := range storedChannels(app.ListenChannels) {
			if candidate == channel || candidate == "*" {
				interest[app.AppID] = q.LastAcked(app.AppID, channel)
				break
			}
		}
	}

	var subscriptions []subscriptionRow
	q.report(q.db.Where("channel = ?", channel).Find(&subscriptions),
		"pool cleanup failed to read subscriptions")
	for _, subscription := range subscriptions {
		if _, ok := interest[subscription.SubscriberID]; !ok {
			interest[subscription.SubscriberID] = subscription.LastAcked
		}
	}
	return interest
}

// warnOnCeilingLoss says out loud which listener is about to lose which
// offsets. Crossing the ceiling with a listener still behind is data loss, and
// the one thing worse than losing those events is losing them quietly.
func (q *Queue) warnOnCeilingLoss(channel string, active []activeListener, now time.Time) {
	var doomed struct{ MaxOffset *int64 }
	q.report(q.db.Model(&eventRow{}).
		Select("max(offset_num) AS max_offset").
		Where("channel = ? AND emitted_at < ?", channel, now.Add(-retentionCeiling)).
		Scan(&doomed), "pool cleanup failed to read the ceiling watermark")
	if doomed.MaxOffset == nil {
		return
	}

	for _, listener := range active {
		if listener.acked < *doomed.MaxOffset {
			q.log.Warn("pool retention ceiling reached, a listener is losing events",
				slog.String("app", listener.app),
				slog.String("channel", channel),
				slog.Int64("from_offset", listener.acked+1),
				slog.Int64("to_offset", *doomed.MaxOffset))
		}
	}
}

// report logs a statement that failed and carries on.
//
// Retention is a periodic best-effort pass, so one failed statement must not
// abort the others — but it used to be dropped on the floor entirely, which
// meant a queue that had quietly stopped trimming looked exactly like a queue
// with nothing to trim.
func (q *Queue) report(result *gorm.DB, message string) {
	if result.Error != nil {
		q.log.Error(message, slog.Any("error", result.Error))
	}
}
