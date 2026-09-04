package extract

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jeffdhooton/scry/internal/memory/distill"
)

// TestValueBoundaryLive is a deliberately gated provider experiment. Build
// it with `go test -c`, run it where the configured key already lives, and
// pass model/base URL/key-env names without ever moving or printing a secret.
func TestValueBoundaryLive(t *testing.T) {
	if os.Getenv("SCRY_LIVE_VALUE_TEST") != "1" {
		t.Skip("set SCRY_LIVE_VALUE_TEST=1 to run the provider value-boundary experiment")
	}
	model := os.Getenv("SCRY_VALUE_PROBE_MODEL")
	keyEnv := os.Getenv("SCRY_VALUE_PROBE_KEY_ENV")
	if model == "" || keyEnv == "" || os.Getenv(keyEnv) == "" {
		t.Fatal("SCRY_VALUE_PROBE_MODEL and a populated SCRY_VALUE_PROBE_KEY_ENV are required")
	}
	provider := Provider{APIKey: os.Getenv(keyEnv), Model: model, BaseURL: os.Getenv("SCRY_VALUE_PROBE_BASE_URL"), KeyEnv: keyEnv}
	h := NewHaiku(provider)

	cases := []struct {
		name string
		text string
		want map[string]bool // exact entity name -> should be value
	}{
		{
			name: "ambiguous statuses",
			text: "The Scry CI run finished. Its validation step recorded status validation_failed. The review run's final verdict was DONE_WITH_CONCERNS. These are the two current outcomes of this run. Scry is the project being tested.",
			want: map[string]bool{"validation_failed": true, "DONE_WITH_CONCERNS": true},
		},
		{
			name: "ambiguous identifiers",
			text: "In auth/events.ts we added the durable event identifier user_login_failed. In argcomplete.py we parse the protocol environment-marker identifier PYTHON_ARGCOMPLETE_OK. Both identifiers are named code interfaces we are implementing, not the outcome of this run.",
			want: map[string]bool{"user_login_failed": false, "PYTHON_ARGCOMPLETE_OK": false},
		},
		{
			name: "additional statuses",
			text: "The repository check returned status dirty_working_tree. Release readiness is READY-AFTER-FIXES. The migration run outcome is DID_NOT_START. The workflow's current status is pause-resume-completed. Each phrase describes the present state of its subject.",
			want: map[string]bool{"dirty_working_tree": true, "READY-AFTER-FIXES": true, "DID_NOT_START": true, "pause-resume-completed": true},
		},
		{
			name: "real names",
			text: "Done For Now is the product name of our journaling app. Ready Player One is a novel in the catalog. dirty-working-tree-check is the command-line tool we maintain. validation-failed-handler is the service that consumes validation events. These are durable named things.",
			want: map[string]bool{"Done For Now": false, "Ready Player One": false, "dirty-working-tree-check": false, "validation-failed-handler": false},
		},
	}

	correct, total := 0, 0
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
			defer cancel()
			result, err := h.Extract(ctx, distill.RawEpisode{Source: "manual", Text: tc.text, OccurredAt: time.Now()}, nil)
			if err != nil {
				t.Fatal(err)
			}
			encoded, _ := json.Marshal(result.Entities)
			t.Logf("model=%s entities=%s", model, encoded)
			byName := map[string]Ent{}
			for _, entity := range result.Entities {
				byName[strings.ToLower(entity.Name)] = entity
			}
			for name, wantValue := range tc.want {
				total++
				entity, ok := byName[strings.ToLower(name)]
				if !ok {
					t.Errorf("model=%s omitted %q", model, name)
					continue
				}
				gotValue := entity.Type == "value"
				if gotValue != wantValue {
					t.Errorf("model=%s classified %q as %s; want value=%v", model, name, entity.Type, wantValue)
					continue
				}
				correct++
			}
		})
	}
	t.Logf("VALUE_BOUNDARY model=%s correct=%d total=%d", model, correct, total)
}
