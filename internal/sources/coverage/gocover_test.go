package coverage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseGocover(t *testing.T) {
	dir := t.TempDir()

	// Write a fake go.mod so module detection works.
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/foo\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Write a minimal cover.out.
	covData := `mode: set
example.com/foo/pkg/handler.go:10.2,15.30 3 1
example.com/foo/pkg/handler.go:20.2,25.30 2 0
example.com/foo/internal/db.go:5.1,8.20 1 5
`
	covPath := filepath.Join(dir, "cover.out")
	if err := os.WriteFile(covPath, []byte(covData), 0o644); err != nil {
		t.Fatal(err)
	}

	ranges, err := parseGocover(covPath, dir)
	if err != nil {
		t.Fatalf("parseGocover: %v", err)
	}

	if len(ranges) != 3 {
		t.Fatalf("expected 3 ranges, got %d", len(ranges))
	}

	// First range: covered (count=1).
	r := ranges[0]
	if r.File != "pkg/handler.go" {
		t.Errorf("ranges[0].File = %q, want %q", r.File, "pkg/handler.go")
	}
	if r.Line != 10 || r.EndLine != 15 {
		t.Errorf("ranges[0] lines = %d-%d, want 10-15", r.Line, r.EndLine)
	}
	if r.Count != 1 {
		t.Errorf("ranges[0].Count = %d, want 1", r.Count)
	}

	// Second range: not covered (count=0).
	if ranges[1].Count != 0 {
		t.Errorf("ranges[1].Count = %d, want 0", ranges[1].Count)
	}

	// Third range: covered 5 times.
	r = ranges[2]
	if r.File != "internal/db.go" {
		t.Errorf("ranges[2].File = %q, want %q", r.File, "internal/db.go")
	}
	if r.Count != 5 {
		t.Errorf("ranges[2].Count = %d, want 5", r.Count)
	}
}

func TestParseGocoverEmpty(t *testing.T) {
	dir := t.TempDir()
	covPath := filepath.Join(dir, "cover.out")
	if err := os.WriteFile(covPath, []byte("mode: set\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ranges, err := parseGocover(covPath, dir)
	if err != nil {
		t.Fatalf("parseGocover: %v", err)
	}
	if len(ranges) != 0 {
		t.Errorf("expected 0 ranges, got %d", len(ranges))
	}
}
