package spaces

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/FacileStudio/Antenne/apps/api/internal/authcontext"
)

// TestAStrangerIsRefusedEveryRouteOnASpace pins the hole this guard closes: an
// account that is simply signed in used to be able to read, rename, delete and
// staff any space on the instance.
func TestAStrangerIsRefusedEveryRouteOnASpace(t *testing.T) {
	owner := user("owner@example.test")
	id := newSpace(t, owner, "Private")

	for _, attempt := range refusedProbes(id) {
		recorder := call(t, user("stranger@example.test"), attempt.method, attempt.path, "")
		if recorder.Code != http.StatusForbidden {
			t.Errorf("%s %s: a stranger got %d, want 403 (%s)", attempt.method, attempt.path, recorder.Code, recorder.Body)
		}
	}
}

// TestAnInstanceAdminIsRefusedTheSameRoutes states the suite rule out loud:
// there is no admin bypass, so an uninvited administrator is a stranger.
func TestAnInstanceAdminIsRefusedTheSameRoutes(t *testing.T) {
	owner := user("owner2@example.test")
	id := newSpace(t, owner, "Private")

	for _, attempt := range refusedProbes(id) {
		recorder := call(t, admin("root@example.test"), attempt.method, attempt.path, "")
		if recorder.Code != http.StatusForbidden {
			t.Errorf("%s %s: an admin got %d, want 403 (%s)", attempt.method, attempt.path, recorder.Code, recorder.Body)
		}
	}
}

// TestTheListOnlyShowsSpacesTheCallerIsIn covers the enumeration half of the
// hole, which no per-space guard would catch.
func TestTheListOnlyShowsSpacesTheCallerIsIn(t *testing.T) {
	owner := user("owner3@example.test")
	id := newSpace(t, owner, "Private")

	mine := call(t, owner, http.MethodGet, "/spaces", "")
	if !strings.Contains(mine.Body.String(), id) {
		t.Fatalf("the owner cannot see their own space: %s", mine.Body)
	}

	for _, caller := range []string{"stranger", "admin"} {
		identity := user("stranger3@example.test")
		if caller == "admin" {
			identity = admin("root3@example.test")
		}
		recorder := call(t, identity, http.MethodGet, "/spaces", "")
		if strings.Contains(recorder.Body.String(), id) {
			t.Errorf("%s can enumerate a space they are not in: %s", caller, recorder.Body)
		}
	}
}

// TestNoRouteWritesASpace pins that Registre is the only place a space
// changes: every write the dashboard used to make is no longer served.
func TestNoRouteWritesASpace(t *testing.T) {
	owner := user("owner4@example.test")
	id := newSpace(t, owner, "Owned")

	for _, attempt := range []probe{
		{http.MethodPost, "/spaces"},
		{http.MethodPut, "/spaces/" + id},
		{http.MethodDelete, "/spaces/" + id},
		{http.MethodPost, "/spaces/" + id + "/members"},
		{http.MethodPut, "/spaces/" + id + "/members/" + owner.Email},
		{http.MethodDelete, "/spaces/" + id + "/members/" + owner.Email},
	} {
		recorder := call(t, owner, attempt.method, attempt.path, `{"name":"x","userId":"x@example.test","role":"owner"}`)
		if recorder.Code < 400 {
			t.Errorf("%s %s: the owner got %d, want it refused", attempt.method, attempt.path, recorder.Code)
		}
	}
}

// TestTheResponsesCarryTheCallersOwnRole covers what the dashboard gates its
// controls on. A fixed "owner", or the role of whoever created the space, would
// pass a single-caller check and still hand a plain member the full set of
// controls, so an owner and a member read the same space here.
func TestTheResponsesCarryTheCallersOwnRole(t *testing.T) {
	owner := user("owner8@example.test")
	member := user("member8@example.test")
	id := newSpace(t, owner, "Roled")
	addMember(t, id, member.Email, RoleMember)

	for _, want := range []struct {
		caller authcontext.Identity
		role   Role
	}{{owner, RoleOwner}, {member, RoleMember}} {
		if got := roleFromGet(t, want.caller, id); got != want.role {
			t.Errorf("GET /spaces/%s as %s: role %q, want %q", id, want.caller.Email, got, want.role)
		}
		if got := roleFromList(t, want.caller, id); got != want.role {
			t.Errorf("GET /spaces as %s: role %q, want %q", want.caller.Email, got, want.role)
		}
	}
}

func roleFromGet(t *testing.T, caller authcontext.Identity, id string) Role {
	t.Helper()
	recorder := call(t, caller, http.MethodGet, "/spaces/"+id, "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("reading the space as %s: %d %s", caller.Email, recorder.Code, recorder.Body)
	}

	var payload struct {
		Space SpaceView `json:"space"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decoding the space: %v", err)
	}
	return payload.Space.Role
}

func roleFromList(t *testing.T, caller authcontext.Identity, id string) Role {
	t.Helper()
	recorder := call(t, caller, http.MethodGet, "/spaces", "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("listing spaces as %s: %d %s", caller.Email, recorder.Code, recorder.Body)
	}

	var payload struct {
		Spaces []SpaceView `json:"spaces"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decoding the list: %v", err)
	}
	for _, space := range payload.Spaces {
		if space.ID == id {
			return space.Role
		}
	}
	t.Fatalf("%s cannot see space %s in their own list: %s", caller.Email, id, recorder.Body)
	return ""
}
