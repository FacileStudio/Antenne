// Package spaces holds Antenne's teams and their members.
//
// `space` is the suite-wide name for a team, so the shape here matches Agenda,
// Ardoise and Jardin rather than inventing a fourth one.
package spaces

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/FacileStudio/Antenne/apps/api/modules/settings"
)

// Role is a member's privilege inside one space.
type Role string

// Space roles, most privileged first.
const (
	RoleOwner  Role = "owner"
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
)

// IsValid reports whether the role is one of the three.
func (r Role) IsValid() bool {
	return r == RoleOwner || r == RoleAdmin || r == RoleMember
}

// Space is a team.
type Space struct {
	ID          string    `gorm:"primaryKey" json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// TableName pins the spaces table name.
func (Space) TableName() string { return "spaces" }

// Member is one person's membership of one space.
type Member struct {
	SpaceID  string    `gorm:"primaryKey;column:space_id" json:"spaceId"`
	UserID   string    `gorm:"primaryKey;column:user_id" json:"userId"`
	Role     Role      `json:"role"`
	JoinedAt time.Time `gorm:"column:joined_at" json:"joinedAt"`
}

// TableName pins the space_members table name.
func (Member) TableName() string { return "space_members" }

// ErrNotFound is returned when a space or a membership does not exist.
var ErrNotFound = errors.New("spaces: not found")

// Service is the spaces store.
type Service struct {
	db *gorm.DB
}

// NewService builds the service.
func NewService(db *gorm.DB) *Service { return &Service{db: db} }

// List returns every space, oldest first.
func (s *Service) List() ([]Space, error) {
	spaces := []Space{}
	err := s.db.Order("created_at asc").Find(&spaces).Error
	return spaces, err
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

// Create adds a space, optionally seeding its owner.
func (s *Service) Create(name, description, ownerUserID string) (Space, error) {
	space := Space{ID: settings.NewID(), Name: name, Description: description}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&space).Error; err != nil {
			return err
		}
		if ownerUserID == "" {
			return nil
		}
		return tx.Create(&Member{
			SpaceID:  space.ID,
			UserID:   ownerUserID,
			Role:     RoleOwner,
			JoinedAt: time.Now(),
		}).Error
	})
	if err != nil {
		return Space{}, err
	}

	return s.Get(space.ID)
}

// Update renames or re-describes a space. A nil field is left alone.
func (s *Service) Update(id string, name, description *string) (Space, error) {
	space, err := s.Get(id)
	if err != nil {
		return Space{}, err
	}

	updates := map[string]any{"updated_at": time.Now()}
	if name != nil {
		updates["name"] = *name
	}
	if description != nil {
		updates["description"] = *description
	}
	if err := s.db.Model(&space).Updates(updates).Error; err != nil {
		return Space{}, err
	}
	return s.Get(id)
}

// Delete removes a space and, by cascade, its memberships.
func (s *Service) Delete(id string) error {
	result := s.db.Where("id = ?", id).Delete(&Space{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// Members lists a space's members, oldest first.
func (s *Service) Members(spaceID string) ([]Member, error) {
	members := []Member{}
	err := s.db.Where("space_id = ?", spaceID).Order("joined_at asc").Find(&members).Error
	return members, err
}

// AddMember adds a member, returning the existing membership unchanged if there
// already is one — adding someone twice is not an error worth surfacing.
func (s *Service) AddMember(spaceID, userID string, role Role) (Member, error) {
	if _, err := s.Get(spaceID); err != nil {
		return Member{}, err
	}

	var existing Member
	err := s.db.Where("space_id = ? AND user_id = ?", spaceID, userID).First(&existing).Error
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return Member{}, err
	}

	member := Member{SpaceID: spaceID, UserID: userID, Role: role, JoinedAt: time.Now()}
	if err := s.db.Create(&member).Error; err != nil {
		return Member{}, err
	}
	return member, nil
}

// UpdateMemberRole changes one member's role.
func (s *Service) UpdateMemberRole(spaceID, userID string, role Role) (Member, error) {
	result := s.db.Model(&Member{}).
		Where("space_id = ? AND user_id = ?", spaceID, userID).
		Update("role", role)
	if result.Error != nil {
		return Member{}, result.Error
	}
	if result.RowsAffected == 0 {
		return Member{}, ErrNotFound
	}

	var member Member
	if err := s.db.Where("space_id = ? AND user_id = ?", spaceID, userID).First(&member).Error; err != nil {
		return Member{}, err
	}
	return member, nil
}

// RemoveMember drops one membership.
func (s *Service) RemoveMember(spaceID, userID string) error {
	result := s.db.Where("space_id = ? AND user_id = ?", spaceID, userID).Delete(&Member{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
