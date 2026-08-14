package daemon

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/jeffdhooton/scry/internal/index"
)

func TestStatusResultIncludesFullIndexerStderr(t *testing.T) {
	stderr := "compiler context\nfinal complaint\n"
	status := StatusResult{Repos: []*RepoStatusEntry{{
		Repo: "/repo", Status: "partial", Indexers: []index.IndexerResult{{
			Language: "typescript", Status: index.IndexerFailed, Stderr: stderr,
		}},
	}}}
	b, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("marshal status: %v", err)
	}
	if !strings.Contains(string(b), `"stderr":"compiler context\nfinal complaint\n"`) {
		t.Fatalf("status JSON = %s, want full stderr tail", b)
	}
}
