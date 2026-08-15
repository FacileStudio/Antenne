package settings

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// NewID mints the identifier a new provider or delivery target carries. It is a
// UUIDv4, which is what the pre-Go dashboard generated, so ids stay
// interchangeable between the two.
//
// There is no error path because crypto/rand.Read has had none since Go 1.24:
// it fills the buffer entirely or crashes the program itself. Handling a
// failure here would be handling a case the standard library guarantees this
// code can never observe.
func NewID() string {
	raw := make([]byte, 16)
	rand.Read(raw)
	raw[6] = (raw[6] & 0x0f) | 0x40
	raw[8] = (raw[8] & 0x3f) | 0x80

	hexed := hex.EncodeToString(raw)
	return hexed[0:8] + "-" + hexed[8:12] + "-" + hexed[12:16] + "-" + hexed[16:20] + "-" + hexed[20:]
}

// MatrixDeviceID is the device a Matrix target logs in as, derived from the
// target id so that it is identical on every boot.
//
// Deriving it rather than storing it is the whole point. A login that does not
// name a device_id makes the homeserver mint a new one, and a device is a slot,
// a set of encryption keys and a row in the account's device list — so a target
// that only remembers its device in memory grows one device per restart, and a
// redeploy loop fills the account to its device limit until it cannot log in at
// all. Naming an existing device makes the login replace it instead, which is
// exactly the idempotent behaviour wanted here.
//
// A target that predates this has no stored deviceId; deriving covers it with
// no migration and no dashboard save.
func MatrixDeviceID(targetID string) string {
	sum := sha256.Sum256([]byte(targetID))
	return "ANTENNE" + strings.ToUpper(hex.EncodeToString(sum[:6]))
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
		deviceID := MatrixDeviceID(target.ID)
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
