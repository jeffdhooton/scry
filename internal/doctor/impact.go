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
// PATH" as a mild environment note while 19 indexed repos silently serve
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

	// Count repos where an unavailable language is primary. Manifests
	// written before the Indexers field existed (m.Indexers == nil) have no
	// per-language tier data at all, so fall back to m.Languages — the old
	// 1%-threshold list is precisely the set of languages that invoked an
	// indexer at the time. Track that fallback count separately so the
	// detail string can flag it as a pre-upgrade estimate rather than
	// silently passing it off as confirmed against current manifests.
	affected := map[string]int{}
	legacyAffected := map[string]int{}
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
		if m.Indexers == nil {
			for _, lang := range m.Languages {
				if _, ok := missing[lang]; ok {
					legacyAffected[lang]++
				}
			}
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
	if len(affected) == 0 && len(legacyAffected) == 0 {
		base.Status = StatusPass
		base.Detail = "missing indexers affect no indexed repo"
		return base
	}

	langSet := map[string]bool{}
	for l := range affected {
		langSet[l] = true
	}
	for l := range legacyAffected {
		langSet[l] = true
	}
	langs := make([]string, 0, len(langSet))
	for l := range langSet {
		langs = append(langs, l)
	}
	sort.Strings(langs)

	var parts []string
	var remedies []string
	for _, l := range langs {
		total := affected[l] + legacyAffected[l]
		part := fmt.Sprintf("%s unavailable — affects %d indexed repo(s)", l, total)
		if legacyAffected[l] > 0 {
			// Some (or all) of this count came from a legacy manifest with
			// no per-indexer data — that portion is an estimate derived
			// from the old file-share language list, not a confirmed hit.
			part += fmt.Sprintf(" (%d from pre-upgrade manifests; reindex to confirm)", legacyAffected[l])
		}
		parts = append(parts, part)
		if r := missing[l]; r != "" {
			remedies = append(remedies, r)
		}
	}
	base.Status = StatusWarn
	base.Detail = strings.Join(parts, "; ")
	base.Remedy = strings.Join(remedies, " && ")
	return base
}
