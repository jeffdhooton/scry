package distill

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"
)

// CuratedSource is deliberately separate from seed and automatic sweep sources.
// One short authored constraint is retained as its own episode summary, only
// after the normal queue worker and transactional resolver complete.
const CuratedSource = "curated-constraint-v1"

const MaxCuratedBytes = 600

// CuratedRef binds the exact file bytes to an absolute source path. Repository
// provenance travels separately in the existing Cwd/CwdIsRepo fields.
func CuratedRef(path, text string) string {
	h := sha256.Sum256([]byte(text))
	return path + "#sha256=" + hex.EncodeToString(h[:])
}

func ParseCuratedRef(ref string) (path, revision string, ok bool) {
	i := strings.LastIndex(ref, "#sha256=")
	if i < 0 {
		return "", "", false
	}
	path, revision = ref[:i], ref[i+8:]
	h, err := hex.DecodeString(revision)
	return path, revision, err == nil && len(h) == sha256.Size && revision == strings.ToLower(revision) && validCuratedPath(path)
}

func validCuratedPath(path string) bool {
	return filepath.IsAbs(path) && filepath.Clean(path) == path && len(path) <= 400 &&
		utf8.ValidString(path) && !strings.ContainsFunc(path, unicode.IsControl)
}

// ValidateCurated checks the transport without statting client-side paths on
// the daemon. It also makes unsupported older workers' paraphrases ineligible
// for exact-source presentation when reopening a store.
func ValidateCurated(ep RawEpisode) error {
	path, _, ok := ParseCuratedRef(ep.SourceRef)
	text := strings.TrimSpace(ep.Text)
	if ep.Source != CuratedSource || !ok || !ep.CwdIsRepo || !validCuratedPath(ep.Cwd) {
		return fmt.Errorf("curated constraint requires an exact source revision and explicit repository attestation")
	}
	rel, err := filepath.Rel(ep.Cwd, path)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("curated constraint must be inside its explicitly selected repository")
	}
	if text == "" || len(ep.Text) > MaxCuratedBytes || !utf8.ValidString(ep.Text) || strings.ContainsFunc(text, unicode.IsControl) {
		return fmt.Errorf("curated constraint must be one nonempty line, at most %d UTF-8 bytes", MaxCuratedBytes)
	}
	// Leave room for the ordinary orientation header, labels, hash and footer.
	// Count quoted paths, since escaping can be longer than the source bytes.
	if len(fmt.Sprintf("%q%q%s%s", path, ep.Cwd, text, filepath.Base(ep.Cwd))) > 1600 {
		return fmt.Errorf("curated constraint and attribution exceed the normal orientation budget")
	}
	if CuratedRef(path, ep.Text) != ep.SourceRef || Redact(ep.Text) != ep.Text {
		return fmt.Errorf("curated constraint content does not match its revision or requires secret redaction")
	}
	return nil
}
