package webhooks

import (
	"strings"
	"testing"

	"github.com/FacileStudio/Antenne/apps/api/modules/settings"
)

func TestSondeWebhookOutage(t *testing.T) {
	payload := `{
		"event": "monitor.down",
		"monitor_id": 42,
		"name": "Production Web",
		"slug": "prod-web",
		"status": "down",
		"url": "https://facile.studio",
		"error": "connection refused",
		"actor_email": "admin@facile.studio",
		"occurred_at": "2026-08-28T18:00:00Z"
	}`

	def, ok := Lookup(settings.ProviderSonde)
	if !ok {
		t.Fatal("sonde provider not registered")
	}

	result, err := def.Handle([]byte(payload), Context{
		ProviderID:   "p-1",
		ProviderName: "Sonde Alerts",
		ProviderType: settings.ProviderSonde,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.EventType != "sonde.monitor.down" {
		t.Errorf("expected EventType sonde.monitor.down, got %s", result.EventType)
	}
	if !strings.Contains(result.MessageHTML, "Production Web") || !strings.Contains(result.MessageHTML, "is DOWN") {
		t.Errorf("unexpected MessageHTML: %s", result.MessageHTML)
	}
	if !strings.Contains(result.MessageHTML, "connection refused") {
		t.Errorf("expected error in MessageHTML: %s", result.MessageHTML)
	}
	if result.Actor == nil || result.Actor.Email == nil || *result.Actor.Email != "admin@facile.studio" {
		t.Errorf("expected actor email admin@facile.studio, got %v", result.Actor)
	}
}

func TestSondeWebhookRecovery(t *testing.T) {
	payload := `{
		"event": "monitor.up",
		"monitor_id": 42,
		"name": "Production Web",
		"slug": "prod-web",
		"status": "up",
		"latency_ms": 35,
		"occurred_at": "2026-08-28T18:05:00Z"
	}`

	def, ok := Lookup(settings.ProviderSonde)
	if !ok {
		t.Fatal("sonde provider not registered")
	}

	result, err := def.Handle([]byte(payload), Context{
		ProviderID:   "p-1",
		ProviderName: "Sonde Alerts",
		ProviderType: settings.ProviderSonde,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.EventType != "sonde.monitor.up" {
		t.Errorf("expected EventType sonde.monitor.up, got %s", result.EventType)
	}
	if !strings.Contains(result.MessageHTML, "Production Web") || !strings.Contains(result.MessageHTML, "is back UP") {
		t.Errorf("unexpected MessageHTML: %s", result.MessageHTML)
	}
	if !strings.Contains(result.MessageHTML, "35ms") {
		t.Errorf("expected latency in MessageHTML: %s", result.MessageHTML)
	}
}

func TestSondeEnvelopeFormat(t *testing.T) {
	payload := `{
		"version": 1,
		"app": "sonde",
		"object": "monitor",
		"action": "updated",
		"facile_id": "api-service",
		"timestamp": "2026-08-28T18:00:00Z",
		"payload": {
			"monitor_id": "7",
			"name": "API Service",
			"slug": "api-service",
			"status": "down",
			"latency_ms": 0,
			"error": "HTTP 502 Bad Gateway",
			"actor_email": "ops@facile.studio"
		}
	}`

	def, ok := Lookup(settings.ProviderSonde)
	if !ok {
		t.Fatal("sonde provider not registered")
	}

	result, err := def.Handle([]byte(payload), Context{
		ProviderID:   "p-1",
		ProviderName: "Sonde Alerts",
		ProviderType: settings.ProviderSonde,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.EventType != "sonde.monitor.down" {
		t.Errorf("expected EventType sonde.monitor.down, got %s", result.EventType)
	}
	if !strings.Contains(result.MessageHTML, "API Service") || !strings.Contains(result.MessageHTML, "is DOWN") {
		t.Errorf("unexpected MessageHTML: %s", result.MessageHTML)
	}
	if !strings.Contains(result.MessageHTML, "HTTP 502 Bad Gateway") {
		t.Errorf("expected error in MessageHTML: %s", result.MessageHTML)
	}
}
