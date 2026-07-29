package daemon

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jeffdhooton/scry/internal/memory/extract"
	memstore "github.com/jeffdhooton/scry/internal/memory/store"
)

// TestMemoryUIIndexServesRenderedGraph covers the daemon-hosted live UI:
// GET / must succeed, and the rendered page must contain both the page
// title and a committed entity's slug (proof the export data actually made
// it into the injected JSON, not just a blank template).
func TestMemoryUIIndexServesRenderedGraph(t *testing.T) {
	d := newTestMemoryDaemon(t)
	ctx := context.Background()
	now := time.Now()

	commitParams := MemoryCommitParams{
		Episode: memstore.Episode{ID: "ep-ui-1", Source: "manual", SourceRef: "manual", OccurredAt: now, IngestedAt: now},
		Result: extract.Result{
			EpisodeSummary: "seed",
			Entities: []extract.Ent{
				{Name: "book-system", Type: "service"},
			},
		},
	}
	if _, err := d.handleMemoryCommit(ctx, mustJSON(t, commitParams)); err != nil {
		t.Fatalf("handleMemoryCommit: %v", err)
	}

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	d.handleMemoryUIIndex(w, req)

	resp := w.Result()
	if resp.StatusCode != 200 {
		t.Fatalf("GET / status = %d, want 200", resp.StatusCode)
	}
	body := w.Body.String()
	if !strings.Contains(body, "scry memory") {
		t.Errorf("body does not contain %q", "scry memory")
	}
	if !strings.Contains(body, "book-system") {
		t.Errorf("body does not contain committed entity slug %q", "book-system")
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("Content-Type = %q, want text/html prefix", ct)
	}
	if cc := resp.Header.Get("Cache-Control"); cc != "no-store" {
		t.Errorf("Cache-Control = %q, want %q", cc, "no-store")
	}
}

// TestMemoryUIDataServesValidJSON covers GET /data.json: it must return the
// export as valid JSON, including the entities array.
func TestMemoryUIDataServesValidJSON(t *testing.T) {
	d := newTestMemoryDaemon(t)
	ctx := context.Background()
	now := time.Now()

	commitParams := MemoryCommitParams{
		Episode: memstore.Episode{ID: "ep-ui-2", Source: "manual", SourceRef: "manual", OccurredAt: now, IngestedAt: now},
		Result: extract.Result{
			EpisodeSummary: "seed",
			Entities: []extract.Ent{
				{Name: "hermes-mini", Type: "machine"},
			},
		},
	}
	if _, err := d.handleMemoryCommit(ctx, mustJSON(t, commitParams)); err != nil {
		t.Fatalf("handleMemoryCommit: %v", err)
	}

	req := httptest.NewRequest("GET", "/data.json", nil)
	w := httptest.NewRecorder()
	d.handleMemoryUIData(w, req)

	resp := w.Result()
	if resp.StatusCode != 200 {
		t.Fatalf("GET /data.json status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var export MemoryExportResult
	if err := json.Unmarshal(w.Body.Bytes(), &export); err != nil {
		t.Fatalf("unmarshal /data.json body: %v", err)
	}
	found := false
	for _, e := range export.Entities {
		if e.Slug == "hermes-mini" {
			found = true
		}
	}
	if !found {
		t.Errorf("export.Entities = %+v, want to contain hermes-mini", export.Entities)
	}
}

// TestMemoryUIUnknownPathReturns404 covers "anything else: 404" for both
// mux-registered handlers.
func TestMemoryUIUnknownPathReturns404(t *testing.T) {
	d := newTestMemoryDaemon(t)

	req := httptest.NewRequest("GET", "/nope", nil)
	w := httptest.NewRecorder()
	d.handleMemoryUIIndex(w, req)
	if w.Result().StatusCode != 404 {
		t.Errorf("GET /nope via index handler status = %d, want 404", w.Result().StatusCode)
	}
}

// TestForceMemoryUILoopback covers the non-loopback-host guard: any host
// other than 127.0.0.1/localhost must be rewritten to 127.0.0.1, and an
// unparsable addr must fall back to the default entirely.
func TestForceMemoryUILoopback(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"loopback ip unchanged", "127.0.0.1:7279", "127.0.0.1:7279"},
		{"localhost unchanged", "localhost:9000", "localhost:9000"},
		{"non-loopback host forced to loopback", "0.0.0.0:7279", "127.0.0.1:7279"},
		{"external host forced to loopback", "10.0.0.5:7279", "127.0.0.1:7279"},
		{"unparsable falls back to default", "not-a-valid-addr", defaultMemoryUIAddr},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := forceMemoryUILoopback(tc.in)
			if got != tc.want {
				t.Errorf("forceMemoryUILoopback(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
