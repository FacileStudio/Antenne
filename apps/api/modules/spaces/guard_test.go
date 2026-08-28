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
		recorder := call(t, user("stranger@example.test"), attempt.method, attempt.path, attempt.body)
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
		recorder := call(t, admin("root@example.test"), attempt.method, attempt.path, attempt.body)
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

// TestCreateSeedsTheCallerAsOwner pins that ownership comes from the session,
// and that a body still naming an owner is refused outright rather than
// silently ignored.
func TestCreateSeedsTheCallerAsOwner(t *testing.T) {
	owner := user("owner4@example.test")

	smuggled := call(t, owner, http.MethodPost, "/spaces",
		`{"name":"Seeded","ownerUserId":"attacker@example.test"}`)
	if smuggled.Code != http.StatusBadRequest {
		t.Fatalf("a body naming an owner was accepted: %d %s", smuggled.Code, smuggled.Body)
	}

	id := newSpaceWithBody(t, owner, `{"name":"Seeded"}`)
	recorder := call(t, owner, http.MethodGet, "/spaces/"+id+"/members", "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("the creator cannot read their own members: %d %s", recorder.Code, recorder.Body)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, "owner4@example.test") || strings.Contains(body, "attacker@example.test") {
		t.Fatalf("ownership was taken from the body: %s", body)
	}
}

// TestAMemberMayReadButNotWrite covers the middle role: reads pass, every
// mutation is refused.
func TestAMemberMayReadButNotWrite(t *testing.T) {
	owner := user("owner5@example.test")
	member := user("member5@example.test")
	id := newSpace(t, owner, "Shared")
	addMember(t, owner, id, member.Email, RoleMember)

	if recorder := call(t, member, http.MethodGet, "/spaces/"+id, ""); recorder.Code != http.StatusOK {
		t.Fatalf("a member cannot read the space: %d %s", recorder.Code, recorder.Body)
	}
	writes := []probe{
		{http.MethodPut, "/spaces/" + id, `{"name":"Renamed"}`},
		{http.MethodDelete, "/spaces/" + id, ""},
		{http.MethodPost, "/spaces/" + id + "/members", `{"userId":"x@example.test","role":"member"}`},
	}
	for _, write := range writes {
		recorder := call(t, member, write.method, write.path, write.body)
		if recorder.Code != http.StatusForbidden {
			t.Errorf("%s %s: a member got %d, want 403", write.method, write.path, recorder.Code)
		}
	}
}

// TestAnAdminMayNotTouchOwnership keeps role escalation off the admin role: an
// admin staffs a space but cannot mint or unseat an owner.
func TestAnAdminMayNotTouchOwnership(t *testing.T) {
	owner := user("owner6@example.test")
	spaceAdmin := user("admin6@example.test")
	id := newSpace(t, owner, "Staffed")
	addMember(t, owner, id, spaceAdmin.Email, RoleAdmin)

	granting := call(t, spaceAdmin, http.MethodPost, "/spaces/"+id+"/members",
		`{"userId":"new@example.test","role":"owner"}`)
	if granting.Code != http.StatusForbidden {
		t.Errorf("an admin granted ownership: %d %s", granting.Code, granting.Body)
	}

	removing := call(t, spaceAdmin, http.MethodDelete, "/spaces/"+id+"/members/"+owner.Email, "")
	if removing.Code != http.StatusForbidden {
		t.Errorf("an admin removed the owner: %d %s", removing.Code, removing.Body)
	}

	demoting := call(t, spaceAdmin, http.MethodPut, "/spaces/"+id+"/members/"+owner.Email, `{"role":"member"}`)
	if demoting.Code != http.StatusForbidden {
		t.Errorf("an admin demoted the owner: %d %s", demoting.Code, demoting.Body)
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
	addMember(t, owner, id, member.Email, RoleMember)

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

// TestTheLastOwnerCannotStepDown keeps a space from becoming unadministrable,
// which nothing in the API could then repair.
func TestTheLastOwnerCannotStepDown(t *testing.T) {
	owner := user("owner7@example.test")
	id := newSpace(t, owner, "Sole")

	demoting := call(t, owner, http.MethodPut, "/spaces/"+id+"/members/"+owner.Email, `{"role":"admin"}`)
	if demoting.Code != http.StatusPreconditionFailed {
		t.Errorf("the last owner demoted themselves: %d %s", demoting.Code, demoting.Body)
	}

	removing := call(t, owner, http.MethodDelete, "/spaces/"+id+"/members/"+owner.Email, "")
	if removing.Code != http.StatusPreconditionFailed {
		t.Errorf("the last owner removed themselves: %d %s", removing.Code, removing.Body)
	}

	addMember(t, owner, id, "second@example.test", RoleOwner)
	stepping := call(t, owner, http.MethodDelete, "/spaces/"+id+"/members/"+owner.Email, "")
	if stepping.Code != http.StatusOK {
		t.Errorf("an owner with a co-owner could not leave: %d %s", stepping.Code, stepping.Body)
	}
}
