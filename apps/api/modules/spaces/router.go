package spaces

import (
	"net/http"
	"strings"

	"github.com/FacileStudio/porte/directory"
	apierrors "github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"
	"github.com/go-chi/chi/v5"
)

// RegisterRoutes mounts the read-only spaces API. Every change to a space is
// made in Registre, so there is no route here that writes one.
func (s *Service) RegisterRoutes(router chi.Router) {
	router.Route("/spaces", func(spaces chi.Router) {
		spaces.Get("/", s.handleList)
		spaces.Get("/{id}", s.handleGet)
		spaces.Get("/{id}/members", s.handleListMembers)
	})
}

// RegisterSync mounts Registre's nudge on directory.SyncPath, relative to the
// /api router it is given. It must sit outside every session, CSRF and rate
// limit guard: Registre posts it anonymously, and the nudge only schedules an
// authenticated pull, so a forged one costs nothing.
func (s *Service) RegisterSync(router chi.Router) {
	router.Post(strings.TrimPrefix(directory.SyncPath, "/api"), s.handleSync)
}

// RegisterDeletions mounts Registre's signed user.deleted push on
// directory.DeletionsPath, relative to the /api router it is given. It sits
// outside every session and CSRF guard because Registre posts it with no
// cookie: the receiver authenticates the token itself.
func (s *Service) RegisterDeletions(router chi.Router) {
	router.Post(strings.TrimPrefix(directory.DeletionsPath, "/api"), s.handleDeletions)
}

func (s *Service) handleDeletions(w http.ResponseWriter, r *http.Request) {
	if s.deletions == nil {
		httpjson.WriteError(w, apierrors.NotFound("single sign-on is not configured"))
		return
	}
	s.deletions.Handler().ServeHTTP(w, r)
}

func (s *Service) handleSync(w http.ResponseWriter, r *http.Request) {
	if s.syncer == nil {
		httpjson.WriteError(w, apierrors.NotFound("single sign-on is not configured"))
		return
	}
	s.syncer.Handler().ServeHTTP(w, r)
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
	httpjson.WriteJSON(w, http.StatusOK, map[string]any{"spaces": spaces, "manageUrl": s.manageBase()})
}

func (s *Service) handleGet(w http.ResponseWriter, r *http.Request) {
	spaceID, role, ok := s.guard(w, r)
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
		"space":   s.view(space, role),
		"members": members,
	})
}

func (s *Service) handleListMembers(w http.ResponseWriter, r *http.Request) {
	spaceID, _, ok := s.guard(w, r)
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

// manageBase is Registre's page listing spaces, or "" with no issuer.
func (s *Service) manageBase() string {
	if s.issuer == "" {
		return ""
	}
	return strings.TrimRight(s.issuer, "/") + "/spaces"
}

// view wraps a space with the caller's role and its Registre page.
func (s *Service) view(space Space, role Role) SpaceView {
	manage := s.manageBase()
	if manage != "" && space.RegistreSpaceID != nil {
		manage += "/" + *space.RegistreSpaceID
	}
	return SpaceView{Space: space, Role: role, ManageURL: manage}
}
