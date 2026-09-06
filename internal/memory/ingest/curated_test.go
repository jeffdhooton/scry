package ingest

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCuratedRequiresExplicitBoundedMapping(t *testing.T) {
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0700); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(repo, "constraint.txt")
	if err := os.WriteFile(file, []byte("Scry requires no CGO.\n"), 0600); err != nil {
		t.Fatal(err)
	}
	foreign := filepath.Join(t.TempDir(), "foreign.txt")
	if err := os.WriteFile(foreign, []byte("Unrelated rule."), 0600); err != nil {
		t.Fatal(err)
	}
	symlink := filepath.Join(repo, "outside.txt")
	if err := os.Symlink(foreign, symlink); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, repo, path string
		force            bool
	}{
		{"no repo", "", file, false}, {"no path", repo, "", false},
		{"not a repo", t.TempDir(), file, false}, {"outside", repo, foreign, false},
		{"symlink outside", repo, symlink, false}, {"directory", repo, repo, false},
		{"force", repo, file, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := newFakeDaemon()
			_, err := File(context.Background(), Options{Source: "curated", Repo: tc.repo, Path: tc.path, Force: tc.force, Daemon: d})
			if err == nil || len(d.enqueued) != 0 || len(d.cursors) != 0 {
				t.Fatalf("unsafe mapping accepted: %v %+v", err, d)
			}
		})
	}
	for _, text := range []string{"", "\n", "first rule\nsecond rule", strings.Repeat("x", 601), "bad\xff", "control\x00text"} {
		d := newFakeDaemon()
		if err := os.WriteFile(file, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := File(context.Background(), Options{Source: "curated", Repo: repo, Path: file, Daemon: d}); err == nil || len(d.cursors) != 0 {
			t.Fatalf("invalid source accepted: %q", text)
		}
	}
}
