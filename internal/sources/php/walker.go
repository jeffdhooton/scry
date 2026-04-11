// Laravel non-PSR-4 file walker.
//
// scip-php only indexes files under the project's PSR-4 autoload roots
// (typically app/, database/factories/, tests/). It skips routes/, config/,
// database/migrations/, and bootstrap/ entirely. For Laravel projects this
// is the single biggest indexing gap — routes/web.php on a real codebase
// (hoopless_crm) holds 522 `::class` controller references that the SCIP
// index never sees, so `scry refs UserController` returns nothing even
// though every route handler is named there.
//
// This walker plugs the hole. It runs after scip-php has populated the
// store, walks the four directories above, lexes each .php file for
// `use` statements and `::class` references, resolves names against the
// file's use list, and writes synthetic ref occurrences into the same
// store. Refs are joined to scip-php's existing symbol records via a
// descriptor lookup so that the walker's output and scip-php's output
// share symbol IDs.
//
// What we deliberately don't try to do:
//
//   - parse PHP perfectly. We need ::class refs and `use` aliases. A
//     full parser would buy us nothing for that target and add the kind
//     of dependency mass that pushed us off the PHAR path.
//   - resolve dynamic constructions like `[$controller, 'index']` where
//     $controller is a runtime variable. The post-processor only resolves
//     what static analysis can.
//   - emit definition occurrences. Routes contain references, never
//     definitions; the source classes are already defined under app/.
package php

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jeffdhooton/scry/internal/store"
)

// laravelNonPSR4Dirs are the four directories Laravel projects use for code
// that lives outside autoloaded namespaces. The walker probes each one;
// missing dirs are silently skipped (not every project has all four).
var laravelNonPSR4Dirs = []string{
	"routes",
	"database/migrations",
	"config",
	"bootstrap",
}

// WalkerStats is what RunWalker returns to the caller for logging.
type WalkerStats struct {
	FilesScanned   int
	ClassRefsTotal int
	ClassRefsBound int // resolved to a known scip-php symbol
}

// RunWalker scans the Laravel non-PSR-4 directories under repoRoot and adds
// synthetic ref occurrences to st. Safe to call after scip.Parse has flushed
// its writer; opens its own writer.
//
// Errors from individual files are skipped (logged via the returned err only
// if no file scanned successfully). Returning a partial result is the right
// default for a best-effort post-processor.
func RunWalker(repoRoot string, st *store.Store) (WalkerStats, error) {
	stats := WalkerStats{}

	pkgName, pkgVersion, err := readComposerIdentity(repoRoot)
	if err != nil {
		return stats, fmt.Errorf("read composer identity: %w", err)
	}

	// Build a name → []symbolID lookup once per walk. The store has
	// LookupSymbolsByName but iterating it for every ref would dominate
	// runtime; cache the descriptor table here.
	resolver := newSymbolResolver(st)

	w := st.NewWriter()
	defer w.Flush()

	for _, sub := range laravelNonPSR4Dirs {
		dir := filepath.Join(repoRoot, sub)
		info, err := os.Stat(dir)
		if err != nil || !info.IsDir() {
			continue
		}
		_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				return nil
			}
			if filepath.Ext(d.Name()) != ".php" {
				return nil
			}
			rel, err := filepath.Rel(repoRoot, path)
			if err != nil {
				return nil
			}
			if scanned, err := scanFile(path, rel, pkgName, pkgVersion, resolver, w, &stats); err == nil && scanned {
				stats.FilesScanned++
			}
			return nil
		})
	}
	return stats, nil
}

// scanFile lexes one PHP file, parses use statements + ::class refs, and
// emits ref occurrences for each resolved class. Returns true if the file
// was lexed at all (false on read error).
func scanFile(
	absPath, relPath, pkgName, pkgVersion string,
	resolver *symbolResolver,
	w *store.Writer,
	stats *WalkerStats,
) (bool, error) {
	src, err := os.ReadFile(absPath)
	if err != nil {
		return false, err
	}
	scan := newPhpScanner(src)
	res := scan.collect()
	uses := res.uses
	classRefs := res.classRefs

	// Cache the source lines for context strings — re-reading per occurrence
	// would dominate runtime on routes/web.php with 1000+ refs.
	var lines []string
	if len(classRefs) > 0 {
		lines = strings.Split(string(src), "\n")
	}

	for _, ref := range classRefs {
		stats.ClassRefsTotal++
		fqn := resolveName(ref.name, uses)
		if fqn == "" {
			continue
		}
		descriptor := strings.ReplaceAll(fqn, "\\", "/") + "#"

		symbolID := resolver.find(descriptor)
		if symbolID == "" {
			// Class not in the store at all — synthesize a project-pkg id and
			// register it. This handles the case where a project file references
			// a vendor class that scip-php DID see (joined via resolver above)
			// vs one it didn't (synthesized below). The synthesized side will
			// not join with anything, but at least `scry refs <Name>` returns
			// the route hit.
			symbolID = fmt.Sprintf("scip-php composer %s %s %s", pkgName, pkgVersion, descriptor)
			displayName := lastSegment(fqn)
			_ = w.PutSymbol(&store.SymbolRecord{
				Symbol:      symbolID,
				DisplayName: displayName,
				Kind:        "Class",
			})
			resolver.remember(descriptor, symbolID, displayName)
		} else {
			stats.ClassRefsBound++
		}

		occ := &store.OccurrenceRecord{
			Symbol:    symbolID,
			File:      relPath,
			Line:      ref.line,
			Column:    ref.col,
			EndLine:   ref.line,
			EndColumn: ref.col + len(ref.name),
			Context:   contextForLine(lines, ref.line),
		}
		_ = w.PutOccurrence(occ)
		_ = w.PutFileSymbol(relPath, symbolID)
	}
	return true, nil
}

func contextForLine(lines []string, oneIndexedLine int) string {
	idx := oneIndexedLine - 1
	if idx < 0 || idx >= len(lines) {
		return ""
	}
	return strings.TrimSpace(lines[idx])
}

func resolveName(name string, uses map[string]string) string {
	if name == "" {
		return ""
	}
	// Absolute name: leading backslash means "from global namespace, ignore
	// uses". Strip the backslash; FQN is what's left.
	if strings.HasPrefix(name, "\\") {
		return strings.TrimPrefix(name, "\\")
	}
	// Qualified name (Foo\Bar): the first segment is the alias to look up,
	// the remainder is appended.
	if i := strings.Index(name, "\\"); i >= 0 {
		alias := name[:i]
		rest := name[i:]
		if base, ok := uses[alias]; ok {
			return base + rest
		}
		// No matching use — assume it's already a fully-qualified name in
		// the current (root) namespace. routes/, config/, migrations/ files
		// have no namespace declaration, so this is the common case.
		return name
	}
	// Unqualified name: must match a use alias to be meaningful.
	if base, ok := uses[name]; ok {
		return base
	}
	// No use match. routes/ files reference root-namespace classes by their
	// short name only when there's a use; without a use, this is most likely
	// a stdlib class (e.g. \Closure) and we can't resolve it.
	return ""
}

func lastSegment(fqn string) string {
	if i := strings.LastIndex(fqn, "\\"); i >= 0 {
		return fqn[i+1:]
	}
	return fqn
}

// readComposerIdentity returns the project's composer package name and the
// version string scip-php uses (the root reference from composer.lock if
// available, falling back to the root version, falling back to "dev").
func readComposerIdentity(repoRoot string) (name, version string, err error) {
	jsonBytes, err := os.ReadFile(filepath.Join(repoRoot, "composer.json"))
	if err != nil {
		return "", "", err
	}
	var jsonMeta struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(jsonBytes, &jsonMeta); err != nil {
		return "", "", fmt.Errorf("decode composer.json: %w", err)
	}
	name = jsonMeta.Name
	if name == "" {
		name = "root"
	}

	lockBytes, err := os.ReadFile(filepath.Join(repoRoot, "composer.lock"))
	if err == nil {
		// scip-php pulls root reference/version from vendor/composer/installed.php,
		// which mirrors composer.lock's `_metadata` section in modern Composer.
		// We approximate via composer.lock's content-hash for the version
		// (matches scip-php behavior closely enough that synthesized refs
		// merge with scip-php-emitted ones during the same indexing run).
		var lock struct {
			ContentHash string `json:"content-hash"`
		}
		if err := json.Unmarshal(lockBytes, &lock); err == nil && lock.ContentHash != "" {
			version = lock.ContentHash
		}
	}
	if version == "" {
		version = "dev"
	}
	return name, version, nil
}

// symbolResolver caches a descriptor → symbolID map built lazily from the
// store's name index. We don't preload everything because most files
// reference only a handful of distinct class names.
type symbolResolver struct {
	st    *store.Store
	cache map[string]string // descriptor -> symbol id
}

func newSymbolResolver(st *store.Store) *symbolResolver {
	return &symbolResolver{st: st, cache: map[string]string{}}
}

// find returns a symbol ID whose descriptor (the trailing token of the SCIP
// id, e.g. "App/Models/User#") matches the requested descriptor, or "" if
// nothing in the store matches.
func (r *symbolResolver) find(descriptor string) string {
	if id, ok := r.cache[descriptor]; ok {
		return id
	}
	// Pull the leaf class name from the descriptor for the by-name lookup.
	// "App/Models/User#" → "User"
	trimmed := strings.TrimSuffix(descriptor, "#")
	short := trimmed
	if i := strings.LastIndex(trimmed, "/"); i >= 0 {
		short = trimmed[i+1:]
	}
	if short == "" {
		r.cache[descriptor] = ""
		return ""
	}
	ids, err := r.st.LookupSymbolsByName(short)
	if err != nil {
		r.cache[descriptor] = ""
		return ""
	}
	for _, id := range ids {
		// SCIP id is "scip-php composer <pkg> <version> <descriptor>". The
		// descriptor is the last space-separated token. We compare suffixes
		// rather than parsing the whole thing.
		if strings.HasSuffix(id, " "+descriptor) {
			r.cache[descriptor] = id
			return id
		}
	}
	r.cache[descriptor] = ""
	return ""
}

// remember caches a descriptor we just synthesized so subsequent lookups in
// the same walk join to it.
func (r *symbolResolver) remember(descriptor, symbolID, _ string) {
	r.cache[descriptor] = symbolID
}
