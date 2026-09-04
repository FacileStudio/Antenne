package providers_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/FacileStudio/Antenne/apps/api/modules/providers"
	"github.com/FacileStudio/Antenne/apps/api/modules/settings"
)

func TestNewIMAP_NoEmit(t *testing.T) {
	log := slog.New(slog.DiscardHandler)
	imap := providers.NewIMAP(log, func(alert providers.Alert) {})

	// Empty settings should not panic.
	imap.Apply(context.Background(), settings.AppSettings{})
	imap.Stop()
}

func TestNewMonitor_NoEmit(t *testing.T) {
	log := slog.New(slog.DiscardHandler)
	monitor := providers.NewMonitor(log, func(alert providers.Alert) {})

	monitor.Apply(context.Background(), settings.AppSettings{})
	monitor.Stop()
}

func TestNewRSS_NoEmit(t *testing.T) {
	log := slog.New(slog.DiscardHandler)
	rss := providers.NewRSS(log, func(alert providers.Alert) {})

	rss.Apply(context.Background(), settings.AppSettings{})
	rss.Stop()
}
