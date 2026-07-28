package distill

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
