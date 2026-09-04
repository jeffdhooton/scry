package resolve

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jeffdhooton/scry/internal/memory/distill"
	"github.com/jeffdhooton/scry/internal/memory/extract"
	"github.com/jeffdhooton/scry/internal/memory/store"
)

// TestValueBoundaryEndToEndLive carries provider output through ParseResult
// and Apply into a temporary Badger store. It complements the extraction-only
// probe by proving that correct model labels produce the intended graph shape.
func TestValueBoundaryEndToEndLive(t *testing.T) {
	if os.Getenv("SCRY_LIVE_VALUE_TEST") != "1" {
		t.Skip("set SCRY_LIVE_VALUE_TEST=1 to run the end-to-end value-boundary experiment")
	}
	model := os.Getenv("SCRY_VALUE_PROBE_MODEL")
	keyEnv := os.Getenv("SCRY_VALUE_PROBE_KEY_ENV")
	if model == "" || keyEnv == "" || os.Getenv(keyEnv) == "" {
		t.Fatal("SCRY_VALUE_PROBE_MODEL and a populated SCRY_VALUE_PROBE_KEY_ENV are required")
	}
	haiku := extract.NewHaiku(extract.Provider{
		APIKey: os.Getenv(keyEnv), Model: model, BaseURL: os.Getenv("SCRY_VALUE_PROBE_BASE_URL"), KeyEnv: keyEnv,
	})
	cases := []struct {
		name string
		text string
		want map[string]bool // exact name -> should persist as an identity
	}{
		{
			name: "pinned values",
			text: "Scry is the project. The CI validation status is validation_failed and the review verdict is DONE_WITH_CONCERNS. dirty_working_tree is the repository status. Release readiness is READY-AFTER-FIXES. The migration outcome is DID_NOT_START. The workflow status is pause-resume-completed. Each is the current state of Scry, not a named component.",
			want: map[string]bool{"validation_failed": false, "DONE_WITH_CONCERNS": false, "dirty_working_tree": false, "READY-AFTER-FIXES": false, "DID_NOT_START": false, "pause-resume-completed": false},
		},
		{
			name: "pinned and external identities",
			text: "We implement the durable auth event identifier user_login_failed and the argcomplete protocol marker PYTHON_ARGCOMPLETE_OK. Done For Now is our journaling product. Ready Player One is a cataloged novel. dirty-working-tree-check is our command-line tool. validation-failed-handler is our event consumer service. These are durable named things, not outcomes of this run.",
			want: map[string]bool{"user_login_failed": true, "PYTHON_ARGCOMPLETE_OK": true, "Done For Now": true, "Ready Player One": true, "dirty-working-tree-check": true, "validation-failed-handler": true},
		},
	}
	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
			defer cancel()
			at := time.Date(2026, 9, 4, 12+i, 0, 0, 0, time.UTC)
			result, err := haiku.Extract(ctx, distill.RawEpisode{Source: "manual", Text: tc.text, OccurredAt: at}, nil)
			if err != nil {
				t.Fatal(err)
			}
			byName := map[string]extract.Ent{}
			for _, entity := range result.Entities {
				byName[strings.ToLower(entity.Name)] = entity
			}
			st, err := store.Open(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			defer st.Close()
			ep := store.Episode{ID: "live-value-" + store.Slugify(tc.name), Source: "manual", SourceRef: tc.name, OccurredAt: at, IngestedAt: at}
			if _, err := Apply(st, ep, "", result, nil); err != nil {
				t.Fatal(err)
			}
			for name, wantIdentity := range tc.want {
				entity, extracted := byName[strings.ToLower(name)]
				if !extracted {
					t.Errorf("model=%s omitted %q", model, name)
					continue
				}
				_, stored, err := st.ResolveAlias(name)
				if err != nil {
					t.Error(err)
				}
				if stored != wantIdentity {
					t.Errorf("model=%s name=%q type=%s fallback=%v stored=%v want_identity=%v", model, name, entity.Type, entity.TypeFallback, stored, wantIdentity)
				}
			}
		})
	}
}
