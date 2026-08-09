package doctor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jeffdhooton/scry/internal/index"
)

// checkIDLanguages maps each indexer environment check to the manifest
// language it gates. When one of those checks warns, this check answers the
// question it leaves open: how many indexed repos does that actually degrade?
var checkIDLanguages = map[string]string{
	"indexers.scip_python":     "python",
	"indexers.scip_typescript": "typescript",
	"indexers.scip_go":         "go",
	"indexers.php":             "php",
}

// checkIndexerImpact cross-references the indexer checks already run against
// every manifest on disk. Without it, doctor reports "scip-python not on
// PATH" as a mild environment note while 17 indexed repos silently serve
// degraded results.
func checkIndexerImpact(scryHome string, prior []Check) Check {
	base := Check{
		ID:       "indexers.impact",
		Category: CategoryIndexers,
		Name:     "indexer impact",
	}

	// Which languages are currently unavailable, and how to fix each.
	missing := map[string]string{} // language -> remedy
	for _, c := range prior {
		lang, ok := checkIDLanguages[c.ID]
		if !ok || (c.Status != StatusWarn && c.Status != StatusFail) {
			continue
		}
		missing[lang] = c.Remedy
	}
	if len(missing) == 0 {
		base.Status = StatusPass
		base.Detail = "all indexers available"
		return base
	}

	// Count repos where a missing language is primary.
	affected := map[string]int{}
	reposDir := filepath.Join(scryHome, "repos")
	entries, _ := os.ReadDir(reposDir)
	for _, ent := range entries {
		b, err := os.ReadFile(filepath.Join(reposDir, ent.Name(), "manifest.json"))
		if err != nil {
			continue
		}
		var m index.Manifest
		if err := json.Unmarshal(b, &m); err != nil {
			continue
		}
		for _, r := range m.Indexers {
			if r.Tier != index.TierPrimary {
				continue
			}
			if _, ok := missing[r.Language]; ok {
				affected[r.Language]++
			}
		}
	}
	if len(affected) == 0 {
		base.Status = StatusPass
		base.Detail = "missing indexers affect no indexed repo"
		return base
	}

	langs := make([]string, 0, len(affected))
	for l := range affected {
		langs = append(langs, l)
	}
	sort.Strings(langs)

	var parts []string
	var remedies []string
	for _, l := range langs {
		parts = append(parts, fmt.Sprintf("%s missing — affects %d indexed repo(s)", l, affected[l]))
		if r := missing[l]; r != "" {
			remedies = append(remedies, r)
		}
	}
	base.Status = StatusWarn
	base.Detail = strings.Join(parts, "; ")
	base.Remedy = strings.Join(remedies, " && ")
	return base
}
