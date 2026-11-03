package auth

import (
	"context"
	"strings"

	"github.com/FacileStudio/Antenne/apps/api/internal/avatar"
	apierrors "github.com/FacileStudio/tronc/errors"
)

// BackfillPictures gives every federated account without a picture the URL its
// identity provider serves one at, and returns how many rows it filled.
//
// Users who last signed in before Antenne stored the picture claim would
// otherwise wait for their next sign-in. The URL is built the way Registre
// builds its own claim, <issuer>/avatars/<subject>, from the porte identity the
// account signed in with, so it needs no network call and no sign-in. Rows that
// already hold a URL are left alone, which makes it safe on every boot, and an
// issuer that is not https fills nothing.
func (s *Store) BackfillPictures(ctx context.Context, issuer string) (int64, error) {
	base := avatar.PhotoURL(strings.TrimRight(issuer, "/"))
	if base == "" {
		return 0, nil
	}
	result := s.orm.WithContext(ctx).Exec(`
		UPDATE users SET picture_url = ? || porte_identities.subject
		  FROM porte_identities
		 WHERE porte_identities.user_id = users.id
		   AND porte_identities.provider = ?
		   AND users.picture_url = ''`,
		base+"/avatars/", issuer)
	if result.Error != nil {
		return 0, apierrors.Internal("failed to backfill the pictures", result.Error)
	}
	return result.RowsAffected, nil
}
