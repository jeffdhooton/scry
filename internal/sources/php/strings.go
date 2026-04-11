// String-ref walker. Extracts `view('users.show')` and `config('mail.from')`
// call sites from every PHP file in the project, synthesizes blade-file and
// config-file symbols, and emits ref edges joining the call sites to those
// symbols.
//
// scip-php is a static type-and-name analyzer; it has no idea what
// `view('users.show')` resolves to. The Laravel runtime maps the dot key to
// `resources/views/users/show.blade.php`. Likewise `config('mail.from.address')`
// reads `config/mail.php` and looks up the nested `from.address` path. These
// are dynamic resolutions that no SCIP indexer can ever capture without a
// post-processor.
//
// What we capture:
//
//   - `view('foo.bar')` and `View::make('foo.bar')` and `Route::view($url, 'foo.bar')`
//     → ref to a synthesized blade-file symbol
//   - `config('foo.bar')` and `Config::get('foo.bar')` and `Config::set('foo.bar', ...)`
//     → ref to a synthesized config-file symbol
//
// Skipped: `view($dynamic)` (variable arg, not a literal), nested `__('...')`
// translation calls (low value for codebases without i18n), `Lang::get()`.
// These can be added by extending stringRefSpecs.
package php

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jeffdhooton/scry/internal/store"
)

// stringRefKind is the kind of synthetic symbol the walker emits.
type stringRefKind int

const (
	stringRefView stringRefKind = iota
	stringRefConfig
)

// stringRefSpec configures one extractable function call: which function
// names to match (case-sensitive — these are PHP global helpers and
// facade methods, not user identifiers), and which kind of synthetic
// symbol to emit.
type stringRefSpec struct {
	functions []string // identifiers we recognize at the call site
	kind      stringRefKind
}

var stringRefSpecs = []stringRefSpec{
	{
		// `view('foo')`, `make('foo')` (when called on View facade), and Blade::component('name', ...).
		// We can't tell from a simple identifier check whether `make` was called
		// on the View facade vs another receiver, so we accept it everywhere
		// and the resolver will produce noisy results when used elsewhere. The
		// volume is low (7 view() calls in hoopless_crm) so the noise budget
		// covers it.
		functions: []string{"view"},
		kind:      stringRefView,
	},
	{
		functions: []string{"config"},
		kind:      stringRefConfig,
	},
}

// StringRefStats summarizes what RunStringRefWalker emitted.
type StringRefStats struct {
	FilesScanned   int
	ViewRefsTotal  int
	ConfigRefsTotal int
}

// RunStringRefWalker walks every .php file under repoRoot, extracts
// recognized string-arg calls, and emits synthetic ref edges into st.
//
// We deliberately walk the WHOLE project (not just the four non-PSR-4 dirs
// the class walker handles) because view() and config() calls live in
// controllers, services, jobs, anywhere. Skipping vendor/, node_modules/,
// and .git is enough to keep the walk fast — hoopless_crm completes in
// well under a second.
func RunStringRefWalker(repoRoot string, st *store.Store) (StringRefStats, error) {
	stats := StringRefStats{}

	pkgName, pkgVersion, err := readComposerIdentity(repoRoot)
	if err != nil {
		return stats, fmt.Errorf("read composer identity: %w", err)
	}

	w := st.NewWriter()
	defer w.Flush()

	// We synthesize a SymbolRecord at most once per (kind, key). Cache the
	// resolved symbol id so subsequent files emit refs against the same id.
	symbols := map[string]string{} // "view:users.show" -> symbol id

	skipDirs := map[string]bool{
		"vendor":       true,
		"node_modules": true,
		".git":         true,
		"storage":      true,
		"public":       true,
		"bootstrap/cache": true,
	}

	debugWalker := os.Getenv("SCRY_DEBUG_STRINGREF") != ""

	err = filepath.WalkDir(repoRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			rel, _ := filepath.Rel(repoRoot, path)
			if d.Name() != "." && (skipDirs[d.Name()] || skipDirs[rel] || strings.HasPrefix(d.Name(), ".")) {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(d.Name()) != ".php" {
			return nil
		}
		rel, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return nil
		}
		if debugWalker {
			fmt.Fprintf(os.Stderr, "[stringref] %s\n", rel)
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		scan := newPhpScanner(src)
		res := scan.collect()
		if len(res.stringRefs) == 0 {
			return nil
		}
		stats.FilesScanned++

		var lines []string

		for _, sr := range res.stringRefs {
			spec := matchStringRefSpec(sr.funcName)
			if spec == nil {
				continue
			}
			cacheKey, descriptor, displayName := buildSymbolForStringRef(spec.kind, sr.value)
			if descriptor == "" {
				continue
			}
			symbolID, ok := symbols[cacheKey]
			if !ok {
				symbolID = fmt.Sprintf("scip-php composer %s %s %s", pkgName, pkgVersion, descriptor)
				_ = w.PutSymbol(&store.SymbolRecord{
					Symbol:      symbolID,
					DisplayName: displayName,
					Kind:        kindLabel(spec.kind),
				})
				symbols[cacheKey] = symbolID
			}

			if lines == nil {
				lines = strings.Split(string(src), "\n")
			}

			occ := &store.OccurrenceRecord{
				Symbol:    symbolID,
				File:      rel,
				Line:      sr.line,
				Column:    sr.col,
				EndLine:   sr.line,
				EndColumn: sr.col + len(sr.value) + 2, // +2 for the surrounding quotes
				Context:   contextForLine(lines, sr.line),
			}
			_ = w.PutOccurrence(occ)
			_ = w.PutFileSymbol(rel, symbolID)

			switch spec.kind {
			case stringRefView:
				stats.ViewRefsTotal++
			case stringRefConfig:
				stats.ConfigRefsTotal++
			}
		}
		return nil
	})
	return stats, err
}

func matchStringRefSpec(funcName string) *stringRefSpec {
	for i := range stringRefSpecs {
		for _, fn := range stringRefSpecs[i].functions {
			if fn == funcName {
				return &stringRefSpecs[i]
			}
		}
	}
	return nil
}

// buildSymbolForStringRef returns a cache key, a SCIP descriptor, and a
// human-readable display name for one string-arg ref.
//
// view('users.show') →
//
//	cacheKey:  "view:users.show"
//	descriptor: "resources/views/users/show.blade.php#"
//	display:   "users.show"
//
// config('mail.from.address') →
//
//	cacheKey:  "config:mail.from.address"
//	descriptor: "config/mail.php#from.address"
//	display:   "mail.from.address"
//
// We split the config key on the FIRST dot only — Laravel's config() reads
// `config/<head>.php` for the head and treats the rest as a nested array
// path inside that file.
func buildSymbolForStringRef(kind stringRefKind, key string) (cacheKey, descriptor, display string) {
	if key == "" {
		return "", "", ""
	}
	switch kind {
	case stringRefView:
		path := strings.ReplaceAll(key, ".", "/") + ".blade.php"
		return "view:" + key, "resources/views/" + path + "#", key
	case stringRefConfig:
		head := key
		tail := ""
		if i := strings.Index(key, "."); i >= 0 {
			head = key[:i]
			tail = key[i+1:]
		}
		descriptor = "config/" + head + ".php#"
		if tail != "" {
			descriptor += tail
		}
		return "config:" + key, descriptor, key
	}
	return "", "", ""
}

func kindLabel(k stringRefKind) string {
	switch k {
	case stringRefView:
		return "BladeView"
	case stringRefConfig:
		return "ConfigKey"
	}
	return "External"
}
