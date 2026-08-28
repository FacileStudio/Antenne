package spaces

import (
	"errors"
	"net/http"

	apierrors "github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"

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
// refuses anyone the allowed roles do not cover.
//
// There is no instance-admin bypass, on purpose: an Antenne administrator who
// was never invited to a space may not read it, rename it or delete it. A
// stranger and an administrator get the same 403 whether or not the space
// exists, so probing an id tells the caller nothing.
func (s *Service) guard(w http.ResponseWriter, r *http.Request, allowed ...Role) (string, Role, bool) {
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
	if !covers(role, allowed) {
		httpjson.WriteError(w, apierrors.Forbidden("this role may not perform that action"))
		return "", "", false
	}
	return spaceID, role, true
}

func covers(role Role, allowed []Role) bool {
	for _, candidate := range allowed {
		if role == candidate {
			return true
		}
	}
	return false
}

// guardOwnerChange refuses the two ways a role change can leave a space without
// an owner or hand ownership out sideways: an admin touching an owner, and the
// last owner stepping down.
func (s *Service) guardOwnerChange(w http.ResponseWriter, spaceID string, callerRole, target, next Role) bool {
	if callerRole != RoleOwner && (target == RoleOwner || next == RoleOwner) {
		httpjson.WriteError(w, apierrors.Forbidden("only an owner may grant or remove ownership"))
		return false
	}
	if target != RoleOwner || next == RoleOwner {
		return true
	}

	owners, err := s.CountOwners(spaceID)
	if err != nil {
		httpjson.WriteError(w, err)
		return false
	}
	if owners <= 1 {
		httpjson.WriteError(w, apierrors.Failed("a space must keep at least one owner"))
		return false
	}
	return true
}
