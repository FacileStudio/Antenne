// Package settings owns Antenne's configuration: the provider list, the delivery
// target list, the system event switches and the pool settings.
//
// The JSON these types marshal to is the dashboard's contract and is unchanged
// from the pre-Go service, so a config exported from the old dashboard imports
// here untouched.
package settings

import "fmt"

// SystemProviderID is the reserved provider id Antenne's own events carry. A
// delivery target must list it explicitly to receive them; there is no
// catch-all.
const SystemProviderID = "__antenne_system__"

// Redacted is what a secret reads as over the wire. A PUT echoing it back means
// "keep the stored value".
const Redacted = "••••••••"

// SystemEvents are the switches for Antenne's own events.
type SystemEvents struct {
	Startup          bool `json:"startup"`
	ProviderEnabled  bool `json:"providerEnabled"`
	ProviderDisabled bool `json:"providerDisabled"`
	ProviderAdded    bool `json:"providerAdded"`
	ProviderRemoved  bool `json:"providerRemoved"`
	DeliveryFailure  bool `json:"deliveryFailure"`
}

// SystemSettings wraps the system event switches.
type SystemSettings struct {
	Events SystemEvents `json:"events"`
}

// BusSettings configures the Antenne Pool.
type BusSettings struct {
	Secret      string `json:"secret,omitempty"`
	LogActivity *bool  `json:"logActivity,omitempty"`
}

// LogActivityEnabled reports whether pool traffic reaches the activity feed.
func (p *BusSettings) LogActivityEnabled() bool {
	return p == nil || p.LogActivity == nil || *p.LogActivity
}

// DeliveryConfig is the ordered delivery target list.
type DeliveryConfig struct {
	Items []DeliveryTarget `json:"items"`
}

// ProvidersConfig is the ordered provider list plus the listen port.
type ProvidersConfig struct {
	Port  int        `json:"port"`
	Items []Provider `json:"items"`
}

// AppSettings is the whole configuration document.
type AppSettings struct {
	Delivery  DeliveryConfig  `json:"delivery"`
	System    SystemSettings  `json:"system"`
	Timeout   int             `json:"timeout"`
	Providers ProvidersConfig `json:"providers"`
	Bus       *BusSettings    `json:"bus,omitempty"`
}

// FindProvider returns the provider with this id, if any.
func (s AppSettings) FindProvider(id string) (Provider, bool) {
	for _, provider := range s.Providers.Items {
		if provider.ID == id {
			return provider, true
		}
	}
	return Provider{}, false
}

// FindTarget returns the delivery target with this id, if any.
func (s AppSettings) FindTarget(id string) (DeliveryTarget, bool) {
	for _, target := range s.Delivery.Items {
		if target.ID == id {
			return target, true
		}
	}
	return DeliveryTarget{}, false
}

// ProviderTags returns the tags of the named provider, or nothing for the
// system pseudo-provider.
func (s AppSettings) ProviderTags(providerID string) []string {
	if providerID == SystemProviderID {
		return nil
	}
	if provider, ok := s.FindProvider(providerID); ok {
		return provider.Tags
	}
	return nil
}

// RuntimeStatus is what the dashboard shows about the running process.
type RuntimeStatus struct {
	ServerPort             int    `json:"serverPort"`
	SettingsPath           string `json:"settingsPath"`
	UsingPersistedSettings bool   `json:"usingPersistedSettings"`
	DashboardBuilt         bool   `json:"dashboardBuilt"`
	LastReloadedAt         string `json:"lastReloadedAt"`
}

// emptySlice keeps a nil slice from marshalling as null. The dashboard iterates
// these without a null guard, exactly as it did against the old service.
func emptySlice(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func nilIfEmpty(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func describeCollision(path string) error {
	return fmt.Errorf("provider path collision detected for `%s`", path)
}
