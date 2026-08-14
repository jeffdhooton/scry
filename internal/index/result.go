package index

import (
	"errors"

	sourceexec "github.com/jeffdhooton/scry/internal/sources/exec"
	"github.com/jeffdhooton/scry/internal/sources/golang"
	"github.com/jeffdhooton/scry/internal/sources/php"
	"github.com/jeffdhooton/scry/internal/sources/python"
	"github.com/jeffdhooton/scry/internal/sources/typescript"
)

// Indexer outcome values recorded in IndexerResult.Status.
const (
	IndexerOK      = "ok"      // ran, output parsed
	IndexerMissing = "missing" // binary not installed
	IndexerFailed  = "failed"  // ran and returned a non-sentinel error
	IndexerSkipped = "skipped" // incidental language, deliberately not invoked
)

// Language tiers recorded in IndexerResult.Tier. Only primary languages can
// degrade a repo's status — see deriveStatus.
const (
	TierPrimary    = "primary"
	TierIncidental = "incidental"
)

// IndexerResult records the outcome of one language indexer for one build.
// Persisted in manifest.json so a degraded index can be diagnosed after the
// fact instead of only from the stderr of the build that produced it.
type IndexerResult struct {
	Language  string  `json:"language"`
	Status    string  `json:"status"`
	Tier      string  `json:"tier"`
	FileCount int     `json:"file_count"`
	Share     float64 `json:"share"`
	Error     string  `json:"error,omitempty"`
	Remedy    string  `json:"remedy,omitempty"`
	Stderr    string  `json:"stderr,omitempty"`

	// Ingest counts for this language alone, taken from the scip.Stats its
	// own .scip dump parsed into, before they are summed into
	// Manifest.Stats. They exist so a consumer can tell "this language
	// indexed and found nothing" from "this language never ran" — the
	// aggregate stats can't distinguish those, and the difference is the
	// whole diagnosis.
	//
	// All four are zero for every status other than IndexerOK: a missing or
	// skipped indexer never ran, and a failed one either never ran or
	// produced output that wouldn't parse. So Status == IndexerOK with
	// SymbolCount == 0 and FileCount > 0 means exactly "claimed success,
	// produced nothing".
	DocumentCount   int `json:"document_count,omitempty"`
	SymbolCount     int `json:"symbol_count,omitempty"`
	DefinitionCount int `json:"definition_count,omitempty"`
	ReferenceCount  int `json:"reference_count,omitempty"`
}

// notFoundSentinels maps a language to the sentinel its source package
// returns when the indexer binary is absent. Classification keys off these
// because "you never installed scip-python" is a one-line fix for the
// operator while "scip-go crashed" is a bug worth reporting — and today both
// surface identically.
var notFoundSentinels = map[string]error{
	"typescript": typescript.ErrIndexerNotFound,
	"go":         golang.ErrIndexerNotFound,
	"php":        php.ErrPhpNotFound,
	"python":     python.ErrIndexerNotFound,
}

// indexerRemedies is the operator-facing fix for a missing indexer. These
// mirror the remedy strings scry doctor already prints for the same tools,
// deliberately: one wording for one problem.
var indexerRemedies = map[string]string{
	"typescript": "npm i -g @sourcegraph/scip-typescript",
	"go":         "install scip-go manually: go install github.com/sourcegraph/scip-go/cmd/scip-go@latest",
	"php":        "install PHP 8.3+ — Laravel Herd, brew install php, or https://www.php.net",
	"python":     "npm i -g @sourcegraph/scip-python",
}

// parseFailureRemedy is the fix for a language whose indexer ran cleanly but
// whose SCIP dump would not parse. Deliberately not one of indexerRemedies:
// the tool is installed, so telling the operator to install it is wrong
// advice. A corrupt or truncated dump is almost always a killed or
// out-of-disk indexer run, which a clean rebuild fixes.
const parseFailureRemedy = "the indexer ran but its SCIP output would not parse — re-run `scry init --force` on this repo; if it recurs, the dump is corrupt and the error above is worth reporting"

// indexerFailureRemedy is the fix for a language whose indexer is installed
// but exited with an error. Like parseFailureRemedy, deliberately not one of
// indexerRemedies: the binary is present, so "install it" is wrong advice.
// A failure recorded with no remedy is half a diagnosis, and this is the
// half the manifest exists to carry — the operator reads it after the build,
// with the build's stderr long gone.
const indexerFailureRemedy = "the indexer is installed but exited with an error — the message above is its own; run it directly against this repo to see its full output, and check it supports this project's toolchain version"

// classify converts one indexer's error into a status, display error, remedy,
// and captured stderr tail. A nil error is ok; a wrapped not-found sentinel is
// missing and carries the install command; anything else is a genuine failure
// and carries indexerFailureRemedy. Every non-ok status carries a remedy — a
// manifest that records a failure without one leaves the operator exactly
// where the bare error string did.
//
// "ok" here means only that the indexer binary exited cleanly. The ingest
// step can still fail afterwards, in which case buildAtLayout's ingest loop
// downgrades that one language's result to IndexerFailed and swaps in
// parseFailureRemedy.
func classify(language string, err error) (status, errMsg, remedy, stderr string) {
	if err == nil {
		return IndexerOK, "", "", ""
	}
	var exitErr *sourceexec.ExitError
	if errors.As(err, &exitErr) {
		stderr = exitErr.Stderr
	}
	if sentinel, ok := notFoundSentinels[language]; ok && errors.Is(err, sentinel) {
		return IndexerMissing, err.Error(), indexerRemedies[language], stderr
	}
	return IndexerFailed, err.Error(), indexerFailureRemedy, stderr
}

// deriveStatus computes the manifest status from the full result set. Only
// primary languages can degrade a repo: an incidental language whose indexer
// is missing is a fact worth recording, not a reason to call an otherwise
// complete index degraded.
//
// "broken" is never returned, including when no language produced usable
// output at all: that build still writes a manifest, and every primary
// language in it is missing or failed, so it derives to "partial". A repo
// whose every indexer failed and one that lost a single language differ in
// the per-language results, not in the status word.
func deriveStatus(results []IndexerResult) string {
	for _, r := range results {
		if r.Tier != TierPrimary {
			continue
		}
		if r.Status == IndexerMissing || r.Status == IndexerFailed {
			return "partial"
		}
	}
	return "ready"
}
