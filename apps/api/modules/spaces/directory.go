package spaces

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/FacileStudio/porte/directory"
	"gorm.io/gorm"

	"github.com/FacileStudio/Antenne/apps/api/modules/settings"
)

// Apply makes Antenne's spaces match a Registre snapshot, in one transaction.
//
// Each Registre space is linked by its id, else to the unlinked local space of
// the same name in any case, else created; its name follows Registre's. Each
// member is resolved to a local address by porte identity, then by an existing
// user with the same email, and is otherwise held by address alone: the import
// never creates an account, because every Antenne account is an admin and only a
// sign-in may create one. Membership is keyed by address, so a first sign-in
// lands on it. A member of a linked space who signs in through Registre, or
// whom the import added, and is absent from the snapshot is removed; a
// local-only member, such as the bootstrap account, is never touched, and
// neither is an unlinked space. A linked space missing from the
// snapshot is kept and logged, because Registre does not delete spaces.
//
// Applying the same snapshot twice changes nothing.
func (s *Service) Apply(ctx context.Context, snapshot directory.Snapshot) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		linked := []string{}
		for _, remote := range snapshot.Spaces {
			spaceID, err := linkSpace(tx, remote)
			if err != nil {
				return err
			}
			if err := s.applyMembers(tx, spaceID, remote.Members); err != nil {
				return err
			}
			linked = append(linked, remote.ID)
		}
		return s.logOrphans(tx, linked)
	})
}

// linkSpace returns the local space mirroring remote, linking or creating it
// and keeping its name equal to Registre's.
func linkSpace(tx *gorm.DB, remote directory.Space) (string, error) {
	var space Space
	err := tx.Where("registre_space_id = ?", remote.ID).Take(&space).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = tx.Where("registre_space_id IS NULL AND lower(name) = lower(?)", remote.Name).
			Order("created_at asc").Take(&space).Error
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		space = Space{ID: settings.NewID(), Name: remote.Name, RegistreSpaceID: &remote.ID}
		return space.ID, tx.Create(&space).Error
	}
	if err != nil {
		return "", err
	}
	return space.ID, tx.Model(&Space{}).
		Where("id = ? AND (name <> ? OR registre_space_id IS NULL)", space.ID, remote.Name).
		Updates(map[string]any{"name": remote.Name, "registre_space_id": remote.ID, "updated_at": time.Now()}).Error
}

// applyMembers sets a linked space's memberships to members.
func (s *Service) applyMembers(tx *gorm.DB, spaceID string, members []directory.Member) error {
	kept := []string{}
	for _, member := range members {
		email, err := s.resolve(tx, member)
		if err != nil {
			return err
		}
		if email == "" {
			continue
		}
		role := Role(member.Role)
		if !role.IsValid() {
			role = RoleMember
		}
		if err := setMembership(tx, spaceID, email, role); err != nil {
			return err
		}
		kept = append(kept, email)
	}

	federated := tx.Table("users").Select("lower(users.email)").
		Joins("JOIN porte_identities ON porte_identities.user_id = users.id").
		Where("porte_identities.provider = ?", s.issuer)
	stale := tx.Where("space_id = ? AND (from_registre OR lower(user_id) IN (?))", spaceID, federated)
	if len(kept) > 0 {
		stale = stale.Where("lower(user_id) NOT IN ?", kept)
	}
	return stale.Delete(&Member{}).Error
}

// setMembership adds email to a space at role, marked as written by the import,
// or moves an existing membership to role. The address is matched in any case.
func setMembership(tx *gorm.DB, spaceID, email string, role Role) error {
	if err := tx.Exec(`
		INSERT INTO space_members (space_id, user_id, role, from_registre)
		SELECT ?, ?, ?, true
		 WHERE NOT EXISTS (SELECT 1 FROM space_members WHERE space_id = ? AND lower(user_id) = ?)`,
		spaceID, email, role, spaceID, email).Error; err != nil {
		return err
	}
	return tx.Model(&Member{}).
		Where("space_id = ? AND lower(user_id) = ? AND role <> ?", spaceID, email, role).
		Update("role", role).Error
}

// resolve returns the local address a Registre member is held under, giving an
// existing user with that address the member's porte identity. It creates no
// user. "" means the member has neither a known subject nor an address, or is a
// person Antenne already erased, and is skipped.
func (s *Service) resolve(tx *gorm.DB, member directory.Member) (string, error) {
	if erased, err := s.erased(tx.Statement.Context, member.Subject); err != nil || erased {
		return "", err
	}
	if member.Subject != "" {
		var email string
		err := tx.Raw(`
			SELECT users.email FROM users
			  JOIN porte_identities ON porte_identities.user_id = users.id
			 WHERE porte_identities.provider = ? AND porte_identities.subject = ?`,
			s.issuer, member.Subject).Scan(&email).Error
		if err != nil || email != "" {
			return email, err
		}
	}

	email := strings.ToLower(strings.TrimSpace(member.Email))
	if email == "" {
		return "", nil
	}
	var userID int64
	if err := tx.Raw(`SELECT id FROM users WHERE lower(email) = ?`, email).Scan(&userID).Error; err != nil {
		return "", err
	}
	if userID == 0 || member.Subject == "" {
		return email, nil
	}
	return email, tx.Exec(`
		INSERT INTO porte_identities (user_id, provider, subject) VALUES (?, ?, ?)
		ON CONFLICT (provider, subject) DO NOTHING`,
		userID, s.issuer, member.Subject).Error
}

// erased reports whether the deletion receiver already handled subject, so a
// snapshot taken before Registre's erasure cannot link that identity again.
func (s *Service) erased(ctx context.Context, subject string) (bool, error) {
	if subject == "" || s.deletions == nil {
		return false, nil
	}
	return s.deletions.Erased(ctx, s.issuer, subject)
}

// logOrphans reports linked spaces the snapshot no longer carries. They are
// kept as they are: Registre has no deletion, so a missing space is an anomaly
// to look at, not an instruction.
func (s *Service) logOrphans(tx *gorm.DB, linked []string) error {
	query := tx.Model(&Space{}).Where("registre_space_id IS NOT NULL")
	if len(linked) > 0 {
		query = query.Where("registre_space_id NOT IN ?", linked)
	}
	var names []string
	if err := query.Pluck("name", &names).Error; err != nil {
		return err
	}
	if len(names) > 0 {
		s.logger.Warn("directory: linked spaces are missing from the registre snapshot, kept", slog.Any("spaces", names))
	}
	return nil
}
