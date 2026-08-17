package env

import "testing"

// TestJournalBrowserURLMustEndInAPI pins the one validation the browser SDK
// cannot perform for itself. Journal's dashboard answers an unmatched path with
// 200 and HTML, and the SDK never throws, so a base URL missing /api produces a
// page that reports every error successfully into nothing.
func TestJournalBrowserURLMustEndInAPI(t *testing.T) {
	cases := []struct {
		url     string
		wantErr bool
	}{
		{"", false},
		{"https://journal.facile.studio/api", false},
		{"https://journal.facile.studio", true},
		{"https://journal.facile.studio/apis", true},
	}

	for _, tc := range cases {
		cfg := Config{AdminPassword: "set", JournalBrowserURL: tc.url}
		err := cfg.validate()
		if tc.wantErr && err == nil {
			t.Errorf("JOURNAL_BROWSER_URL=%q was accepted, want a refusal", tc.url)
		}
		if !tc.wantErr && err != nil {
			t.Errorf("JOURNAL_BROWSER_URL=%q was refused: %v", tc.url, err)
		}
	}
}
