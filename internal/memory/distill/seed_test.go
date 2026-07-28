package distill

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const seedFixturePath = "testdata/seed_memory.md"

func TestSeedMarkdownDistills(t *testing.T) {
	info, err := os.Stat(seedFixturePath)
	if err != nil {
		t.Fatalf("stat fixture: %v", err)
	}

	ep, err := SeedMarkdown(seedFixturePath)
	if err != nil {
		t.Fatalf("SeedMarkdown error: %v", err)
	}

	if ep.Source != seedSource {
		t.Errorf("Source = %q, want %q", ep.Source, seedSource)
	}
	if ep.SourceRef != seedFixturePath {
		t.Errorf("SourceRef = %q, want %q", ep.SourceRef, seedFixturePath)
	}
	if ep.ID == "" {
		t.Errorf("empty ID")
	}
	if !ep.OccurredAt.Equal(info.ModTime()) {
		t.Errorf("OccurredAt = %v, want file mtime %v", ep.OccurredAt, info.ModTime())
	}

	for _, want := range []string{
		"on-call rotation escalation", // from the description field
		"On-call escalation",          // body heading
		"Acknowledge in PagerDuty",    // body content
	} {
		if !strings.Contains(ep.Text, want) {
			t.Errorf("episode text missing %q; got: %s", want, ep.Text)
		}
	}

	if strings.Contains(ep.Text, "fixture-seed-memory") {
		t.Errorf("episode text should not leak the frontmatter name field; got: %s", ep.Text)
	}
	if strings.Contains(ep.Text, "---") {
		t.Errorf("episode text should not contain frontmatter delimiters; got: %s", ep.Text)
	}
}

func TestSeedMarkdownMissingFileErrors(t *testing.T) {
	if _, err := SeedMarkdown("testdata/does_not_exist.md"); err == nil {
		t.Errorf("expected error for a missing file, got nil")
	}
}

// TestSeedMarkdownNoFrontmatterUsesWholeFileAsBody guards the tolerant
// fallback: a markdown file with no "---" frontmatter block at all is not
// an error, it's just treated as having no name/description and the whole
// file as body - distillation should never hard-fail on a seed file that
// doesn't happen to follow the frontmatter convention.
func TestSeedMarkdownNoFrontmatterUsesWholeFileAsBody(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "plain.md")
	content := "# Just a plain note\n\nNo frontmatter here at all.\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	ep, err := SeedMarkdown(path)
	if err != nil {
		t.Fatalf("SeedMarkdown error: %v", err)
	}
	if !strings.Contains(ep.Text, "Just a plain note") {
		t.Errorf("expected whole-file fallback body; got: %s", ep.Text)
	}
}

// TestSeedMarkdownIDChangesWithContent covers F2: the episode ID must be
// content-sensitive (derived from the file's mtime), not just the path, so
// an edited seed file produces a distinct episode on re-distill instead of
// being extracted again at LLM cost and then silently dropped by HasEpisode
// as a duplicate of the stale content.
func TestSeedMarkdownIDChangesWithContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "seed.md")
	if err := os.WriteFile(path, []byte("# v1\n\nfirst version\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	t1 := time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)
	if err := os.Chtimes(path, t1, t1); err != nil {
		t.Fatalf("chtimes: %v", err)
	}

	first, err := SeedMarkdown(path)
	if err != nil {
		t.Fatalf("SeedMarkdown (first): %v", err)
	}

	if err := os.WriteFile(path, []byte("# v2\n\nsecond version\n"), 0o644); err != nil {
		t.Fatalf("rewrite fixture: %v", err)
	}
	t2 := t1.Add(5 * time.Minute)
	if err := os.Chtimes(path, t2, t2); err != nil {
		t.Fatalf("chtimes (touch): %v", err)
	}

	second, err := SeedMarkdown(path)
	if err != nil {
		t.Fatalf("SeedMarkdown (second): %v", err)
	}

	if first.ID == second.ID {
		t.Errorf("ID unchanged across a content/mtime change: %q", first.ID)
	}
}
