package settings

import "time"

type appSettingsRow struct {
	ID         int `gorm:"primaryKey"`
	Timeout    int
	ListenPort int
	System     []byte `gorm:"type:jsonb"`
	Bus        []byte `gorm:"type:jsonb"`
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
