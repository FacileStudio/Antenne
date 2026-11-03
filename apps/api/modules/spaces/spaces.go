// Package spaces holds Antenne's teams and their members.
//
// `space` is the suite-wide name for a team, so the shape here matches Agenda,
// Ardoise and Mycelium rather than inventing a fourth one. Registre owns every
// shared space: Antenne applies the snapshot Registre serves and only reads
// spaces through its own API, so creating, renaming and staffing a space all
// happen in Registre.
package spaces

import (
	"errors"
	"log/slog"
	"time"

	"github.com/FacileStudio/porte/directory"
	"gorm.io/gorm"

	"github.com/FacileStudio/Antenne/apps/api/internal/avatar"
)

// Role is a member's privilege inside one space.
type Role string

// Space roles, most privileged first. They are Registre's three, unchanged.
const (
	RoleOwner  Role = directory.RoleOwner
	RoleAdmin  Role = directory.RoleAdmin
	RoleMember Role = directory.RoleMember
)

// IsValid reports whether the role is one of the three.
func (r Role) IsValid() bool {
	return r == RoleOwner || r == RoleAdmin || r == RoleMember
}

// Space is a team. RegistreSpaceID links it to the Registre space it mirrors;
// nil means it was never linked.
type Space struct {
	ID              string    `gorm:"primaryKey" json:"id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	RegistreSpaceID *string   `gorm:"column:registre_space_id" json:"registreSpaceId"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

// TableName pins the spaces table name.
func (Space) TableName() string { return "spaces" }

// SpaceView is a space as one of its members sees it: the space itself, that
// member's own role, and where the space is managed in Registre.
type SpaceView struct {
	Space
	Role      Role   `json:"role"`
	ManageURL string `gorm:"-" json:"manageUrl,omitempty"`
}

// Member is one person's membership of one space. UserID is the member's
// email address, the key every membership in Antenne is held under.
type Member struct {
	SpaceID   string    `gorm:"primaryKey;column:space_id" json:"spaceId"`
	UserID    string    `gorm:"primaryKey;column:user_id" json:"userId"`
	Role      Role      `json:"role"`
	JoinedAt  time.Time `gorm:"column:joined_at" json:"joinedAt"`
	AvatarURL string    `gorm:"column:avatar_url;->" json:"avatarUrl"`
}

// TableName pins the space_members table name.
func (Member) TableName() string { return "space_members" }

// ErrNotFound is returned when a space or a membership does not exist.
var ErrNotFound = errors.New("spaces: not found")

// Service is the spaces store and the Registre directory sync. A zero
// Client.Issuer leaves the sync off and the spaces read-only as they are.
type Service struct {
	db     *gorm.DB
	issuer string
	logger *slog.Logger
	client directory.Client
	syncer *directory.Syncer
	retry  time.Duration
}

// NewService builds the service. client is Antenne's own OIDC client against
// Registre; with an empty Issuer there is nothing to sync from.
func NewService(db *gorm.DB, client directory.Client, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	s := &Service{
		db:     db,
		issuer: client.Issuer,
		logger: logger,
		client: client,
		retry:  importRetry,
	}
	if client.Issuer != "" {
		s.syncer = &directory.Syncer{Client: client, Apply: s.Apply, Logger: logger}
	}
	return s
}

// members is the one query a membership is read through, so every member a
// response carries has the picture of the user its identifier names.
func (s *Service) members() *gorm.DB {
	return s.db.Model(&Member{}).Select("space_members.*, " + avatar.ByEmail("space_members.user_id") + " AS avatar_url")
}

// ListForMember returns the spaces userID belongs to, oldest first, each
// carrying userID's role in it. The role rides on the join that already scopes
// the query, so it costs nothing extra.
//
// There is deliberately no method returning every space: membership is the only
// way into a space, for an instance administrator as much as for anyone else.
func (s *Service) ListForMember(userID string) ([]SpaceView, error) {
	views := []SpaceView{}
	err := s.db.Model(&Space{}).
		Select("spaces.*, space_members.role").
		Joins("JOIN space_members ON space_members.space_id = spaces.id AND space_members.user_id = ?", userID).
		Order("spaces.created_at asc").
		Find(&views).Error
	for i := range views {
		views[i] = s.view(views[i].Space, views[i].Role)
	}
	return views, err
}

// Get returns one space.
func (s *Service) Get(id string) (Space, error) {
	var space Space
	if err := s.db.Where("id = ?", id).First(&space).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Space{}, ErrNotFound
		}
		return Space{}, err
	}
	return space, nil
}

// Members lists a space's members, oldest first.
func (s *Service) Members(spaceID string) ([]Member, error) {
	members := []Member{}
	err := s.members().Where("space_id = ?", spaceID).Order("joined_at asc").Find(&members).Error
	return members, err
}
