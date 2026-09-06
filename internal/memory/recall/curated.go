package recall

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jeffdhooton/scry/internal/memory/distill"
	"github.com/jeffdhooton/scry/internal/memory/store"
)

// curatedForOrient presents completed authored source observations, not a new
// temporal fact policy. Graph facts still use the unchanged resolver; in
// particular a same-triple paraphrase can retain its old wording. Such facts
// must not reintroduce obsolete or foreign curated text through generic bullets.
func curatedForOrient(st *store.Store, cwd string) ([]string, map[string]bool, error) {
	episodes, err := st.AllEpisodes()
	if err != nil {
		return nil, nil, err
	}
	excluded := map[string]bool{}
	latest := map[string]store.Episode{}
	for _, ep := range episodes {
		if ep.Source != distill.CuratedSource {
			continue
		}
		excluded[ep.ID] = true
		if err := distill.ValidateCurated(distill.RawEpisode{Source: ep.Source, SourceRef: ep.SourceRef, Text: ep.Summary, Cwd: ep.Cwd, CwdIsRepo: ep.CwdIsRepo}); err != nil {
			continue // fail closed for malformed/unattested/unsupported evidence
		}
		// Exact explicitly mapped root only. A nested or sibling repository
		// must never inherit a rule via a guessed ancestor or shared basename.
		if filepath.Clean(cwd) != ep.Cwd {
			continue
		}
		path, _, _ := distill.ParseCuratedRef(ep.SourceRef)
		old, ok := latest[path]
		if !ok || ep.OccurredAt.After(old.OccurredAt) || (ep.OccurredAt.Equal(old.OccurredAt) && ep.ID > old.ID) {
			latest[path] = ep
		}
	}
	paths := make([]string, 0, len(latest))
	for path := range latest {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	var bullets []string
	for _, path := range paths {
		ep := latest[path]
		_, hash, _ := distill.ParseCuratedRef(ep.SourceRef)
		bullets = append(bullets, fmt.Sprintf("- Curated constraint: %s\n  Source: %q; sha256:%s; repository: %q (attested).", strings.TrimSpace(ep.Summary), path, hash, ep.Cwd))
	}
	return bullets, excluded, nil
}

func withoutCuratedFacts(facts []store.Fact, excluded map[string]bool) []store.Fact {
	var out []store.Fact
	for _, f := range facts {
		curated := false
		for _, id := range f.Episodes {
			if excluded[id] {
				curated = true
				break
			}
		}
		if !curated {
			out = append(out, f)
		}
	}
	return out
}
