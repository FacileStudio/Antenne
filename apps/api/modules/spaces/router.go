package spaces

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	apierrors "github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"
	"github.com/go-chi/chi/v5"
)

type spaceRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	OwnerUserID string  `json:"ownerUserId"`
}

type memberRequest struct {
	UserID string `json:"userId"`
	Role   string `json:"role"`
}

// RegisterRoutes mounts the spaces API.
func (s *Service) RegisterRoutes(router chi.Router) {
	router.Route("/spaces", func(spaces chi.Router) {
		spaces.Get("/", s.handleList)
		spaces.Post("/", s.handleCreate)
		spaces.Get("/{id}", s.handleGet)
		spaces.Put("/{id}", s.handleUpdate)
		spaces.Delete("/{id}", s.handleDelete)
		spaces.Get("/{id}/members", s.handleListMembers)
		spaces.Post("/{id}/members", s.handleAddMember)
		spaces.Put("/{id}/members/{userId}", s.handleUpdateMember)
		spaces.Delete("/{id}/members/{userId}", s.handleRemoveMember)
	})
}

func (s *Service) handleList(w http.ResponseWriter, r *http.Request) {
	spaces, err := s.List()
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, map[string]any{"spaces": spaces})
}

func (s *Service) handleCreate(w http.ResponseWriter, r *http.Request) {
	var body spaceRequest
	if err := httpjson.DecodeJSON(w, r, &body); err != nil {
		httpjson.WriteError(w, apierrors.Invalid("Invalid JSON body."))
		return
	}
	if body.Name == nil || strings.TrimSpace(*body.Name) == "" {
		httpjson.WriteError(w, apierrors.Invalid("name is required"))
		return
	}

	description := ""
	if body.Description != nil {
		description = strings.TrimSpace(*body.Description)
	}

	space, err := s.Create(strings.TrimSpace(*body.Name), description, body.OwnerUserID)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusCreated, map[string]any{"space": space})
}

func (s *Service) handleGet(w http.ResponseWriter, r *http.Request) {
	space, err := s.Get(param(r, "id"))
	if err != nil {
		writeSpaceError(w, err)
		return
	}
	members, err := s.Members(space.ID)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, map[string]any{"space": space, "members": members})
}

func (s *Service) handleUpdate(w http.ResponseWriter, r *http.Request) {
	var body spaceRequest
	if err := httpjson.DecodeJSON(w, r, &body); err != nil {
		httpjson.WriteError(w, apierrors.Invalid("Invalid JSON body."))
		return
	}

	space, err := s.Update(param(r, "id"), trimPointer(body.Name), trimPointer(body.Description))
	if err != nil {
		writeSpaceError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, map[string]any{"space": space})
}

func (s *Service) handleDelete(w http.ResponseWriter, r *http.Request) {
	if err := s.Delete(param(r, "id")); err != nil {
		writeSpaceError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Service) handleListMembers(w http.ResponseWriter, r *http.Request) {
	spaceID := param(r, "id")
	if _, err := s.Get(spaceID); err != nil {
		writeSpaceError(w, err)
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

	member, err := s.AddMember(param(r, "id"), strings.TrimSpace(body.UserID), role)
	if err != nil {
		writeSpaceError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusCreated, map[string]any{"member": member})
}

func (s *Service) handleUpdateMember(w http.ResponseWriter, r *http.Request) {
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

	member, err := s.UpdateMemberRole(param(r, "id"), param(r, "userId"), role)
	if err != nil {
		writeSpaceError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, map[string]any{"member": member})
}

func (s *Service) handleRemoveMember(w http.ResponseWriter, r *http.Request) {
	if err := s.RemoveMember(param(r, "id"), param(r, "userId")); err != nil {
		writeSpaceError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// param reads a path parameter and undoes the percent-encoding chi leaves in
// place. chi routes on RawPath whenever the request carries any escaping, so a
// client sending encodeURIComponent(email) would otherwise never match a stored
// user id.
func param(r *http.Request, key string) string {
	raw := chi.URLParam(r, key)
	decoded, err := url.PathUnescape(raw)
	if err != nil {
		return raw
	}
	return decoded
}

func trimPointer(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	return &trimmed
}

func writeSpaceError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrNotFound) {
		httpjson.WriteError(w, apierrors.NotFound("space not found"))
		return
	}
	httpjson.WriteError(w, err)
}
