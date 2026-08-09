package index

import (
	"errors"

	"github.com/jeffdhooton/scry/internal/sources/golang"
	"github.com/jeffdhooton/scry/internal/sources/php"
	"github.com/jeffdhooton/scry/internal/sources/python"
	"github.com/jeffdhooton/scry/internal/sources/typescript"
)

// Indexer outcome values recorded in IndexerResult.Status.
const (
	IndexerOK      = "ok"      // ran, output parsed
	IndexerMissing = "missing" // binary not installed
	IndexerFailed  = "failed"  // ran and errored, or output failed to parse
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
	"go":         "check network access; scry auto-downloads scip-go into ~/.scry/bin",
	"php":        "install PHP 8.3+ and ensure `php` is on PATH",
	"python":     "npm i -g @sourcegraph/scip-python",
}

// classify converts one indexer's error into a (status, error, remedy)
// triple. A nil error is ok; a wrapped not-found sentinel is missing and
// carries a remedy; anything else is a genuine failure.
func classify(language string, err error) (status, errMsg, remedy string) {
	if err == nil {
		return IndexerOK, "", ""
	}
	if sentinel, ok := notFoundSentinels[language]; ok && errors.Is(err, sentinel) {
		return IndexerMissing, err.Error(), indexerRemedies[language]
	}
	return IndexerFailed, err.Error(), ""
}

// deriveStatus computes the manifest status from the full result set. Only
// primary languages can degrade a repo: an incidental language whose indexer
// is missing is a fact worth recording, not a reason to call an otherwise
// complete index degraded.
//
// "broken" is never returned. When no indexer produces output at all,
// buildAtLayout returns an error and writes no manifest, so that value in
// Manifest.Status is vestigial.
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
