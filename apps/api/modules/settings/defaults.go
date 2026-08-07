package settings

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
)

// NewID mints the identifier a new provider or delivery target carries. It is a
// UUIDv4, which is what the pre-Go dashboard generated, so ids stay
// interchangeable between the two.
func NewID() string {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		// crypto/rand only fails when the OS entropy source is gone, which is
		// not a condition a settings edit can recover from.
		panic(err)
	}
	raw[6] = (raw[6] & 0x0f) | 0x40
	raw[8] = (raw[8] & 0x3f) | 0x80

	hexed := hex.EncodeToString(raw)
	return hexed[0:8] + "-" + hexed[8:12] + "-" + hexed[12:16] + "-" + hexed[16:20] + "-" + hexed[20:]
}

func newMatrixDeviceID() string {
	compact := strings.ToUpper(strings.ReplaceAll(NewID(), "-", ""))
	return "ANTENNE" + compact[:12]
}

// DefaultProvider builds a provider of this type with the field defaults the
// dashboard expects.
func DefaultProvider(providerType ProviderType) Provider {
	provider := Provider{
		ID:      NewID(),
		Type:    providerType,
		Name:    ProviderLabel(providerType),
		Enabled: true,
		Tags:    []string{},
	}

	switch providerType {
	case ProviderWebsite:
		provider.URL = "https://"
		provider.Interval = 60
	case ProviderRSS:
		provider.Name = "RSS Feed"
		provider.PollInterval = 300
	case ProviderIMAP:
		provider.Name = "IMAP Inbox"
		provider.Port = 993
		provider.Secure = true
		provider.PollInterval = 60
	default:
		provider.Path = WebhookDefaultPath(providerType)
		header := WebhookDefaultSecretHeader(providerType)
		provider.SecretHeader = &header
	}

	return provider
}

// DefaultTarget builds a delivery target of this type with the field defaults
// the dashboard expects.
func DefaultTarget(targetType DeliveryTargetType) DeliveryTarget {
	target := DeliveryTarget{
		ID:           NewID(),
		Type:         targetType,
		Name:         TargetLabel(targetType),
		Enabled:      true,
		ProviderIDs:  []string{},
		ProviderTags: []string{},
	}

	switch targetType {
	case TargetMatrix:
		target.Homeserver = "https://matrix.org"
		deviceID := newMatrixDeviceID()
		target.DeviceID = &deviceID
		target.RoomIDs = []string{}
	case TargetSMTP:
		target.Port = 587
		target.Secure = true
		target.ToAddresses = []string{}
		target.SubjectPrefix = "[Antenne]"
	case TargetNtfy:
		target.ServerURL = "https://ntfy.sh"
		priority := 3
		target.Priority = &priority
		target.Tags = []string{"rotating_light"}
	}

	return target
}

// Defaults is the settings document a first boot starts from.
func Defaults(port, timeout int) AppSettings {
	return AppSettings{
		Delivery: DeliveryConfig{Items: []DeliveryTarget{}},
		System: SystemSettings{Events: SystemEvents{
			Startup: true,
		}},
		Timeout: timeout,
		Providers: ProvidersConfig{
			Port:  port,
			Items: []Provider{},
		},
		Bus: &BusSettings{},
	}
}
