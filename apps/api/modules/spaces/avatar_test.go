package spaces

import (
	"encoding/json"
	"net/http"
	"testing"
)

// TestMembersCarryTheirUsersPicture proves the member list resolves a picture
// through users by address, whatever case the identifier was typed in, and that
// an identifier with no user behind it carries none, so the dashboard falls
// back to the initial.
func TestMembersCarryTheirUsersPicture(t *testing.T) {
	owner := user("owner@example.test")
	id := newSpace(t, owner, "Pictures")
	if err := testDB.Exec(
		`INSERT INTO users (email, picture_url) VALUES ('pictured@example.test', 'https://sso.example.test/avatars/abc')`,
	).Error; err != nil {
		t.Fatalf("seeding the user: %v", err)
	}
	t.Cleanup(func() { testDB.Exec("DELETE FROM users WHERE email = 'pictured@example.test'") })
	addMember(t, id, "Pictured@Example.test", RoleMember)

	recorder := call(t, owner, http.MethodGet, "/spaces/"+id+"/members", "")
	var payload struct {
		Members []Member `json:"members"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decoding the members: %v", err)
	}
	got := map[string]string{}
	for _, member := range payload.Members {
		got[member.UserID] = member.AvatarURL
	}
	if got["Pictured@Example.test"] != "https://sso.example.test/avatars/abc" {
		t.Errorf("the pictured member carries %q", got["Pictured@Example.test"])
	}
	if picture, ok := got["owner@example.test"]; !ok || picture != "" {
		t.Errorf("the owner with no user row carries %q (listed %v)", picture, ok)
	}
}
