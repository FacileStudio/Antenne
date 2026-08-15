package events

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// TestNewEntrySerialisesDeliveriesAsAnArray pins the one field on this struct
// whose zero value is a client-side crash rather than a missing value. The
// dashboard calls .some() and .length on deliveries, and the read path cannot
// produce nil — decode() always returns a slice — so a freshly pushed event is
// the only way null reaches the stream, and only a live event shows it.
func TestNewEntrySerialisesDeliveriesAsAnArray(t *testing.T) {
	entry := newEntry(New{Source: "webhook", Event: "test"}, time.Now().UTC(), 1)

	if entry.Deliveries == nil {
		t.Fatal("Deliveries is nil, which marshals to null")
	}

	encoded, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(encoded), `"deliveries":[]`) {
		t.Errorf(`missing "deliveries":[] in %s`, encoded)
	}
}
