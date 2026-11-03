package spaces

import "github.com/FacileStudio/tronc/apiref"

// SpaceListResponse documents GET /api/spaces.
type SpaceListResponse struct {
	Spaces    []SpaceView `json:"spaces"`
	ManageURL string      `json:"manageUrl"`
}

// SpaceDetailResponse documents GET /api/spaces/{id}.
type SpaceDetailResponse struct {
	Space   SpaceView `json:"space"`
	Members []Member  `json:"members"`
}

// MemberListResponse documents GET /api/spaces/{id}/members.
type MemberListResponse struct {
	Members []Member `json:"members"`
}

// Documentation is the spaces module's entry in the API reference.
func Documentation() apiref.Module {
	return apiref.Module{
		Name:        "Spaces",
		Description: "Teams and their members, owned by Registre and read-only here: Antenne applies Registre's snapshot, and every change is made at manageUrl. Every route is membership-scoped, with no instance-admin bypass.",
		Routes:      append(readRoutes(), syncRoute()),
	}
}

// readRoutes are the three membership-scoped reads.
func readRoutes() []apiref.Route {
	return []apiref.Route{
		{
			Method:       "GET",
			Path:         "/api/spaces",
			Summary:      "List spaces",
			Description:  "Only the spaces the caller is a member of. Each carries the caller's own role in it and its Registre page.",
			Auth:         "session",
			ResponseBody: SpaceListResponse{},
		},
		{
			Method:       "GET",
			Path:         "/api/spaces/{id}",
			Summary:      "Read a space",
			Description:  "Returns the space, the caller's own role in it, and its members in one call.",
			Auth:         "session",
			PathParams:   []apiref.Field{{Name: "id", Type: "string", Description: "the space id"}},
			ResponseBody: SpaceDetailResponse{},
			Errors: []apiref.Error{
				{Status: 403, Code: "permission_denied", Description: "the caller is not a member of this space"},
			},
		},
		{
			Method:       "GET",
			Path:         "/api/spaces/{id}/members",
			Summary:      "List members",
			Auth:         "session",
			PathParams:   []apiref.Field{{Name: "id", Type: "string", Description: "the space id"}},
			ResponseBody: MemberListResponse{},
			Errors: []apiref.Error{
				{Status: 403, Code: "permission_denied", Description: "the caller is not a member of this space"},
			},
		},
	}
}

// syncRoute is the nudge Registre posts after any change.
func syncRoute() apiref.Route {
	return apiref.Route{
		Method:      "POST",
		Path:        "/api/registre/sync",
		Summary:     "Registre nudge",
		Description: "Registre posts here after any change to a space. The body is ignored and the caller is not trusted: Antenne only schedules an authenticated pull of the snapshot two seconds later.",
		Auth:        "none",
		Status:      202,
		Errors: []apiref.Error{
			{Status: 404, Code: "not_found", Description: "single sign-on is not configured"},
		},
	}
}
