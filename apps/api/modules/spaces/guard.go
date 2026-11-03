package spaces

import (
	"errors"
	"net/http"
	"net/url"

	apierrors "github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"github.com/FacileStudio/Antenne/apps/api/internal/authcontext"
)

// callerKey returns the authenticated caller's membership key, which is their
// email: Member.UserID holds an email everywhere in this module, and the router
// percent-decodes the path parameter for exactly that reason. An empty string
// means nobody was identified.
func callerKey(r *http.Request) string {
	identity := authcontext.From(r)
	if !identity.IsAuthenticated() {
		return ""
	}
	return identity.Email
}

// guard resolves the caller's role in the space named by the request and
// refuses anyone who is not a member of it.
//
// There is no instance-admin bypass, on purpose: an Antenne administrator who
// was never invited to a space may not read it. A stranger and an
// administrator get the same 403 whether or not the space exists, so probing
// an id tells the caller nothing.
func (s *Service) guard(w http.ResponseWriter, r *http.Request) (string, Role, bool) {
	spaceID := param(r, "id")

	key := callerKey(r)
	if key == "" {
		httpjson.WriteError(w, apierrors.Unauthorized("authentication is required"))
		return "", "", false
	}

	role, err := s.MemberRole(spaceID, key)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpjson.WriteError(w, apierrors.Forbidden("not a member of this space"))
			return "", "", false
		}
		httpjson.WriteError(w, err)
		return "", "", false
	}
	return spaceID, role, true
}

// MemberRole returns userID's role in a space, or ErrNotFound when there is no
// such membership.
func (s *Service) MemberRole(spaceID, userID string) (Role, error) {
	var member Member
	err := s.db.Where("space_id = ? AND user_id = ?", spaceID, userID).First(&member).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	return member.Role, nil
}

// param reads a path parameter and undoes the percent-encoding chi leaves in
// place. chi routes on RawPath whenever the request carries any escaping.
func param(r *http.Request, key string) string {
	raw := chi.URLParam(r, key)
	decoded, err := url.PathUnescape(raw)
	if err != nil {
		return raw
	}
	return decoded
}

func writeSpaceError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrNotFound) {
		httpjson.WriteError(w, apierrors.NotFound("space not found"))
		return
	}
	httpjson.WriteError(w, err)
}
