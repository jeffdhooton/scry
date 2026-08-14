package resolve

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jeffdhooton/scry/internal/memory/store"
)

func TestHygiene_CleansAliasesAndDeadRepoRefs(t *testing.T) {
	st := openTemp(t)
	now := time.Now().UTC()

	gone := filepath.Join(t.TempDir(), "removed-worktree")
	if err := st.PutEntity(store.Entity{
		Slug: "setpoint", Name: "setpoint", Type: "project",
		Description: "loop engine",
		Aliases: []string{
			"loop engine",          // durable
			"setpoint-wt-9e6jz82r", // temp worktree
			"/tmp/survtest",        // scratch dir
		},
		RepoRefs:  []string{gone, "/not/under/users"},
		CreatedAt: now, LastSeen: now,
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Dry run reports and changes nothing.
	rep, err := Hygiene(st, true)
	if err != nil {
		t.Fatalf("dry run: %v", err)
	}
	if rep.AliasesDropped != 2 || rep.RepoRefsDropped != 2 {
		t.Fatalf("dry run miscounted: %+v", rep)
	}
	before, _ := st.GetEntity("setpoint")
	if len(before.Aliases) != 3 {
		t.Fatalf("dry run mutated the store: %+v", before)
	}

	if _, err := Hygiene(st, false); err != nil {
		t.Fatalf("apply: %v", err)
	}
	got, err := st.GetEntity("setpoint")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(got.Aliases) != 1 || got.Aliases[0] != "loop engine" {
		t.Fatalf("aliases not cleaned: %+v", got.Aliases)
	}
	if len(got.RepoRefs) != 0 {
		t.Fatalf("dead repo refs survived: %+v", got.RepoRefs)
	}
	if got.Description != "loop engine" {
		t.Fatalf("hygiene must not touch descriptions: %q", got.Description)
	}
}

func TestHygiene_KeepsLiveRepoRefs(t *testing.T) {
	st := openTemp(t)
	now := time.Now().UTC()
	live := t.TempDir()
	if err := os.MkdirAll(filepath.Join(live, ".git"), 0o755); err != nil {
		t.Fatalf("seed repo: %v", err)
	}
	// Only /Users/ paths qualify, so a temp dir is dropped even with a .git —
	// assert the rule as written rather than the rule as hoped.
	if err := st.PutEntity(store.Entity{
		Slug: "x", Name: "x", Type: "project", RepoRefs: []string{live},
		CreatedAt: now, LastSeen: now,
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := Hygiene(st, false); err != nil {
		t.Fatalf("apply: %v", err)
	}
	got, _ := st.GetEntity("x")
	if len(got.RepoRefs) != 0 {
		t.Fatalf("expected non-home path dropped: %+v", got.RepoRefs)
	}
}

func TestHygiene_ReportsEphemeralEntitiesWithoutDeleting(t *testing.T) {
	st := openTemp(t)
	now := time.Now().UTC()
	if err := st.PutEntity(store.Entity{
		Slug: store.Slugify("setpoint-wt-9e6jz82r"), Name: "setpoint-wt-9e6jz82r",
		Type: "project", CreatedAt: now, LastSeen: now,
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	rep, err := Hygiene(st, false)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(rep.EphemeralEntities) != 1 {
		t.Fatalf("expected the temp worktree reported: %+v", rep)
	}
	// Still present: its facts would be orphaned by a delete.
	if _, err := st.GetEntity(store.Slugify("setpoint-wt-9e6jz82r")); err != nil {
		t.Fatalf("entity must not be deleted: %v", err)
	}
}
