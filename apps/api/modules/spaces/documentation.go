package spaces

import "github.com/FacileStudio/tronc/apiref"

// Documentation returns this module's entry in the reference at /docs.
func Documentation() apiref.Module {
	return apiref.Module{
		Name:        "Spaces",
		Description: "Teams and their members. Providers and delivery targets are instance-wide; spaces group people.",
		Routes:      append(spaceRoutes(), memberRoutes()...),
	}
}

// spaceRoutes cover the spaces themselves.
func spaceRoutes() []apiref.Route {
	return []apiref.Route{
		{
			Method:       "GET",
			Path:         "/api/spaces",
			Summary:      "List spaces",
			Auth:         "session",
			ResponseBody: "SpaceListResponse",
		},
		{
			Method:       "POST",
			Path:         "/api/spaces",
			Summary:      "Create a space",
			Description:  "An ownerUserId, when given, is seeded as the space's owner.",
			Auth:         "session",
			RequestBody:  "SpaceRequest",
			ResponseBody: "SpaceResponse",
			Errors: []apiref.Error{
				{Status: 400, Code: "invalid_argument", Description: "name is required"},
			},
		},
		{
			Method:       "GET",
			Path:         "/api/spaces/{id}",
			Summary:      "Read a space",
			Description:  "Returns the space and its members in one call.",
			Auth:         "session",
			PathParams:   []apiref.Field{{Name: "id", Type: "string", Description: "the space id"}},
			ResponseBody: "SpaceDetailResponse",
			Errors:       []apiref.Error{{Status: 404, Code: "not_found", Description: "no such space"}},
		},
		{
			Method:       "PUT",
			Path:         "/api/spaces/{id}",
			Summary:      "Update a space",
			Description:  "An absent field is left alone.",
			Auth:         "session",
			PathParams:   []apiref.Field{{Name: "id", Type: "string", Description: "the space id"}},
			RequestBody:  "SpaceRequest",
			ResponseBody: "SpaceResponse",
			Errors:       []apiref.Error{{Status: 404, Code: "not_found", Description: "no such space"}},
		},
		{
			Method:       "DELETE",
			Path:         "/api/spaces/{id}",
			Summary:      "Delete a space",
			Description:  "Its memberships go with it.",
			Auth:         "session",
			PathParams:   []apiref.Field{{Name: "id", Type: "string", Description: "the space id"}},
			ResponseBody: "OkResponse",
			Errors:       []apiref.Error{{Status: 404, Code: "not_found", Description: "no such space"}},
		},
	}
}

// memberRoutes cover who is in a space and what they may do there.
func memberRoutes() []apiref.Route {
	return []apiref.Route{
		{
			Method:       "GET",
			Path:         "/api/spaces/{id}/members",
			Summary:      "List members",
			Auth:         "session",
			PathParams:   []apiref.Field{{Name: "id", Type: "string", Description: "the space id"}},
			ResponseBody: "MemberListResponse",
			Errors:       []apiref.Error{{Status: 404, Code: "not_found", Description: "no such space"}},
		},
		{
			Method:       "POST",
			Path:         "/api/spaces/{id}/members",
			Summary:      "Add a member",
			Description:  "Adding someone already in the space returns that membership unchanged rather than erroring. An unknown role falls back to member.",
			Auth:         "session",
			PathParams:   []apiref.Field{{Name: "id", Type: "string", Description: "the space id"}},
			RequestBody:  "MemberRequest",
			ResponseBody: "MemberResponse",
			Errors: []apiref.Error{
				{Status: 400, Code: "invalid_argument", Description: "userId is required"},
				{Status: 404, Code: "not_found", Description: "no such space"},
			},
		},
		{
			Method:      "PUT",
			Path:        "/api/spaces/{id}/members/{userId}",
			Summary:     "Change a member's role",
			Description: "userId is percent-decoded server-side, so a client may send encodeURIComponent(email).",
			Auth:        "session",
			PathParams: []apiref.Field{
				{Name: "id", Type: "string", Description: "the space id"},
				{Name: "userId", Type: "string", Description: "the member's identifier, percent-encoded"},
			},
			RequestBody:  "MemberRequest",
			ResponseBody: "MemberResponse",
			Errors: []apiref.Error{
				{Status: 400, Code: "invalid_argument", Description: "role must be owner, admin or member"},
				{Status: 404, Code: "not_found", Description: "no such membership"},
			},
		},
		{
			Method:  "DELETE",
			Path:    "/api/spaces/{id}/members/{userId}",
			Summary: "Remove a member",
			Auth:    "session",
			PathParams: []apiref.Field{
				{Name: "id", Type: "string", Description: "the space id"},
				{Name: "userId", Type: "string", Description: "the member's identifier, percent-encoded"},
			},
			ResponseBody: "OkResponse",
			Errors:       []apiref.Error{{Status: 404, Code: "not_found", Description: "no such membership"}},
		},
	}
}
