package health

import (
	"context"
	"log/slog"
	"time"

	"github.com/FacileStudio/Antenne/apps/api/modules/settings"
)

func (m *Monitor) loop(ctx context.Context, targets []settings.DeliveryTarget) {
	defer m.group.Done()

	m.sweep(ctx, targets)
	for {
		// A small uniform per-round jitter on top of the deterministic per-target
		// phase, so two Antenne instances restarted by the same deploy do not
		// sweep in lockstep.
		wait := sweepInterval + time.Duration(m.random()*float64(sweepInterval/4))
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			m.sweep(ctx, targets)
		}
	}
}

func (m *Monitor) sweep(ctx context.Context, targets []settings.DeliveryTarget) {
	m.mu.Lock()
	startedAt := m.startedAt
	m.mu.Unlock()

	for _, target := range targets {
		if ctx.Err() != nil {
			return
		}

		record, err := m.records.Load(ctx, target.ID)
		if err != nil {
			m.log.Warn("could not read target health",
				slog.String("target", target.Name), slog.Any("error", err))
			continue
		}

		if !ShouldProbe(record, target.ID, ProbeInterval(target.Type), startedAt, m.now()) {
			continue
		}

		m.log.Debug("probing delivery target",
			slog.String("target", target.Name), slog.String("target_type", string(target.Type)))
		m.Observe(ctx, target, m.probe(ctx, target))
	}

	m.mu.Lock()
	m.lastSweep = m.now()
	m.mu.Unlock()
}

// prune drops health for targets that no longer exist, so a deleted target does
// not sit in the dashboard forever as a red row nobody can fix.
func (m *Monitor) prune(ctx context.Context, targets []settings.DeliveryTarget) {
	records, err := m.records.LoadAll(ctx)
	if err != nil {
		// Logged rather than dropped: a prune that has quietly stopped working
		// is indistinguishable from a prune with nothing left to do.
		m.log.Warn("could not read target health to prune it", slog.Any("error", err))
		return
	}
	live := make(map[string]struct{}, len(targets))
	for _, target := range targets {
		live[target.ID] = struct{}{}
	}
	for _, record := range records {
		if _, ok := live[record.TargetID]; ok {
			continue
		}
		if err := m.records.Delete(ctx, record.TargetID); err != nil {
			m.log.Warn("could not drop health for a removed target",
				slog.String("target_id", record.TargetID), slog.Any("error", err))
		}
	}
}
