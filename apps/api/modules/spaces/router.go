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
	key := callerKey(r)
	if key == "" {
		httpjson.WriteError(w, apierrors.Unauthorized("authentication is required"))
		return
	}

	spaces, err := s.ListForMember(key)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, map[string]any{"spaces": spaces})
}

func (s *Service) handleCreate(w http.ResponseWriter, r *http.Request) {
	key := callerKey(r)
	if key == "" {
		httpjson.WriteError(w, apierrors.Unauthorized("authentication is required"))
		return
	}

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

	space, err := s.Create(strings.TrimSpace(*body.Name), description, key)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusCreated, map[string]any{
		"space": SpaceView{Space: space, Role: RoleOwner},
	})
}

func (s *Service) handleGet(w http.ResponseWriter, r *http.Request) {
	spaceID, role, ok := s.guard(w, r, RoleOwner, RoleAdmin, RoleMember)
	if !ok {
		return
	}

	space, err := s.Get(spaceID)
	if err != nil {
		writeSpaceError(w, err)
		return
	}
	members, err := s.Members(space.ID)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, map[string]any{
		"space":   SpaceView{Space: space, Role: role},
		"members": members,
	})
}

func (s *Service) handleUpdate(w http.ResponseWriter, r *http.Request) {
	spaceID, role, ok := s.guard(w, r, RoleOwner, RoleAdmin)
	if !ok {
		return
	}

	var body spaceRequest
	if err := httpjson.DecodeJSON(w, r, &body); err != nil {
		httpjson.WriteError(w, apierrors.Invalid("Invalid JSON body."))
		return
	}

	space, err := s.Update(spaceID, trimPointer(body.Name), trimPointer(body.Description))
	if err != nil {
		writeSpaceError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, map[string]any{
		"space": SpaceView{Space: space, Role: role},
	})
}

func (s *Service) handleDelete(w http.ResponseWriter, r *http.Request) {
	spaceID, _, ok := s.guard(w, r, RoleOwner)
	if !ok {
		return
	}

	if err := s.Delete(spaceID); err != nil {
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
