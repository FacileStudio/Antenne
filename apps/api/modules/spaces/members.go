package spaces

import (
	"errors"
	"net/http"
	"strings"

	apierrors "github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"
)

type memberRequest struct {
	UserID string `json:"userId"`
	Role   string `json:"role"`
}

func (s *Service) handleListMembers(w http.ResponseWriter, r *http.Request) {
	spaceID, _, ok := s.guard(w, r, RoleOwner, RoleAdmin, RoleMember)
	if !ok {
		return
	}

	members, err := s.Members(spaceID)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, map[string]any{"members": members})
}

func (s *Service) handleAddMember(w http.ResponseWriter, r *http.Request) {
	spaceID, callerRole, ok := s.guard(w, r, RoleOwner, RoleAdmin)
	if !ok {
		return
	}

	var body memberRequest
	if err := httpjson.DecodeJSON(w, r, &body); err != nil {
		httpjson.WriteError(w, apierrors.Invalid("Invalid JSON body."))
		return
	}
	if strings.TrimSpace(body.UserID) == "" {
		httpjson.WriteError(w, apierrors.Invalid("userId is required"))
		return
	}

	role := Role(body.Role)
	if !role.IsValid() {
		role = RoleMember
	}
	if !s.guardOwnerChange(w, spaceID, callerRole, "", role) {
		return
	}

	member, err := s.AddMember(spaceID, strings.TrimSpace(body.UserID), role)
	if err != nil {
		writeSpaceError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusCreated, map[string]any{"member": member})
}

func (s *Service) handleUpdateMember(w http.ResponseWriter, r *http.Request) {
	spaceID, callerRole, ok := s.guard(w, r, RoleOwner)
	if !ok {
		return
	}

	var body memberRequest
	if err := httpjson.DecodeJSON(w, r, &body); err != nil {
		httpjson.WriteError(w, apierrors.Invalid("Invalid JSON body."))
		return
	}

	role := Role(body.Role)
	if !role.IsValid() {
		httpjson.WriteError(w, apierrors.Invalid("valid role is required (owner, admin, member)"))
		return
	}

	userID := param(r, "userId")
	current, err := s.MemberRole(spaceID, userID)
	if err != nil {
		writeMemberError(w, err)
		return
	}
	if !s.guardOwnerChange(w, spaceID, callerRole, current, role) {
		return
	}

	member, err := s.UpdateMemberRole(spaceID, userID, role)
	if err != nil {
		writeMemberError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, map[string]any{"member": member})
}

func (s *Service) handleRemoveMember(w http.ResponseWriter, r *http.Request) {
	spaceID, callerRole, ok := s.guard(w, r, RoleOwner, RoleAdmin)
	if !ok {
		return
	}

	userID := param(r, "userId")
	current, err := s.MemberRole(spaceID, userID)
	if err != nil {
		writeMemberError(w, err)
		return
	}
	if !s.guardOwnerChange(w, spaceID, callerRole, current, "") {
		return
	}

	if err := s.RemoveMember(spaceID, userID); err != nil {
		writeMemberError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func writeMemberError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrNotFound) {
		httpjson.WriteError(w, apierrors.NotFound("membership not found"))
		return
	}
	httpjson.WriteError(w, err)
}
