package mcp

import (
	"reflect"
	"strings"
	"testing"

	"github.com/jeffdhooton/scry/internal/daemon"
	"github.com/jeffdhooton/scry/internal/index"
)

// TestStatusToolDescriptionNamesTheTrustSignals guards the one place an agent
// can learn that the stale and empty signals exist at all.
//
// callStatus forwards the daemon's reply verbatim, so the fields ship whether
// or not anything mentions them. But an MCP client chooses and interprets a
// tool from its description: an agent told only that scry_status lists
// "document counts, ref counts, and last-indexed timestamps" has no reason to
// look for effective_status, and will read a months-old index as authoritative.
// That is the same silent green this signal was added to remove, one layer up.
func TestStatusToolDescriptionNamesTheTrustSignals(t *testing.T) {
	var desc string
	var found bool
	for _, td := range toolDefinitions {
		if td.Name == "scry_status" {
			desc, found = td.Description, true
			break
		}
	}
	if !found {
		t.Fatal("scry_status missing from toolDefinitions")
	}

	// The wire names are read off the daemon's own struct rather than
	// hardcoded here, so renaming a JSON tag fails this test instead of
	// quietly leaving the description advertising a key that no longer ships.
	entry := reflect.TypeFor[daemon.RepoStatusEntry]()
	for _, field := range []string{"EffectiveStatus", "Stale", "EmptyLanguages"} {
		f, ok := entry.FieldByName(field)
		if !ok {
			t.Fatalf("daemon.RepoStatusEntry has no field %s", field)
		}
		key := strings.Split(f.Tag.Get("json"), ",")[0]
		if key == "" {
			t.Fatalf("daemon.RepoStatusEntry.%s has no json tag", field)
		}
		if !strings.Contains(desc, key) {
			t.Errorf("scry_status description never names %q — an agent reading the tool list can't know the field is there", key)
		}
	}

	// Each state must carry its own explanation, not just appear somewhere in
	// the prose. The trailing colon is what distinguishes "empty:" explaining
	// the state from the incidental "empty" inside "empty_languages" — without
	// it this loop would pass on a description that defines nothing.
	for _, status := range []string{
		index.StatusReady, index.StatusPartial, index.StatusStale, index.StatusEmpty,
	} {
		if !strings.Contains(desc, status+":") {
			t.Errorf("scry_status description never explains what effective_status %q means for query results", status)
		}
	}
}
