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

// The grader's disproof: hygiene reported zero collisions for a machine
// and a project sharing a name byte for byte, because a dry run skipped
// every alias it believed it would clean before counting.
func TestHygieneCountsCollisionsItPlansToClean(t *testing.T) {
	cases := []struct{ machine, project string }{
		{"widget rig alpha", "widget rig alpha"},
		{"widget-rig-alpha", "widget rig alpha"},
		{"widgetrigalpha", "widget rig alpha"},
		{"widget rig alphas", "widget rig alpha"},
		{"halo/flashnext", "halo-flashnext"},
	}
	for _, c := range cases {
		st := openTemp(t)
		now := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
		if err := st.PutEntity(store.Entity{Slug: "m1", Name: c.machine, Type: "machine", CreatedAt: now, LastSeen: now}); err != nil {
			t.Fatal(err)
		}
		if err := st.PutEntity(store.Entity{Slug: "p1", Name: c.project, Type: "project", CreatedAt: now, LastSeen: now}); err != nil {
			t.Fatal(err)
		}
		for _, dry := range []bool{true, false} {
			rep, err := Hygiene(st, dry)
			if err != nil {
				t.Fatal(err)
			}
			if rep.CrossTypeCollisions == 0 {
				t.Errorf("machine %q and project %q, dryRun=%v: collisions = 0, want at least one", c.machine, c.project, dry)
			}
		}
	}
}

// A concept stub sharing a machine's name is counted: it is the path a
// machine's name takes to reach a project.
func TestHygieneCountsAConceptStubAgainstARealType(t *testing.T) {
	st := openTemp(t)
	now := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	if err := st.PutEntity(store.Entity{Slug: "mac-mini", Name: "Mac mini", Type: "machine", Aliases: []string{"mini"}, CreatedAt: now, LastSeen: now}); err != nil {
		t.Fatal(err)
	}
	if err := st.PutEntity(store.Entity{Slug: "mothership-thing", Name: "mothership thing", Type: "concept", Aliases: []string{"mini"}, CreatedAt: now, LastSeen: now}); err != nil {
		t.Fatal(err)
	}
	rep, err := Hygiene(st, true)
	if err != nil {
		t.Fatal(err)
	}
	if rep.CrossTypeCollisions == 0 {
		t.Errorf("a concept holding a machine's name counts as a collision, got 0: %+v", rep.CollisionSample)
	}
}

func TestFoldNameFoldsSpellingAndPlurals(t *testing.T) {
	same := [][2]string{
		{"Mac mini", "mac-mini"}, {"Mac mini", "macmini"}, {"Mac minis", "mac mini"},
		{"halo/flashnext", "halo-flashnext"}, {"Qwen3.8-27B", "qwen38 27b"},
		{"scry_status", "scry status"},
	}
	for _, p := range same {
		if foldName(p[0]) != foldName(p[1]) {
			t.Errorf("foldName(%q) = %q, foldName(%q) = %q, want the same", p[0], foldName(p[0]), p[1], foldName(p[1]))
		}
	}
	differ := [][2]string{{"halo", "halo2"}, {"mac mini", "mac studio"}, {"hermes", "hermes-ops"}}
	for _, p := range differ {
		if foldName(p[0]) == foldName(p[1]) {
			t.Errorf("foldName folded %q and %q together", p[0], p[1])
		}
	}
}

func TestHygieneMergesADuplicateStubButNeverTwoTypedEntities(t *testing.T) {
	st := openTemp(t)
	now := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	put := func(e store.Entity) {
		e.CreatedAt, e.LastSeen = now, now
		if err := st.PutEntity(e); err != nil {
			t.Fatal(err)
		}
	}
	put(store.Entity{Slug: "android-assetlinks", Name: "Android App Links", Type: "service"})
	put(store.Entity{Slug: "android-assetlinksjson", Name: "Android App Links", Type: "concept", Aliases: []string{"assetlinks.json"}, RepoRefs: []string{"/Users/jeff/workspace/mobile"}})
	put(store.Entity{Slug: "mac-mini", Name: "widget rig", Type: "machine"})
	put(store.Entity{Slug: "widget-rig-proj", Name: "widget rig", Type: "project"})
	if err := st.PutFact(store.Fact{Src: "android-assetlinksjson", Relation: "uses", Dst: "mac-mini", Fact: "the file is served from the mini", ValidFrom: now, Confidence: 1}); err != nil {
		t.Fatal(err)
	}

	rep, err := Hygiene(st, false)
	if err != nil {
		t.Fatal(err)
	}
	if rep.StubsMerged != 1 {
		t.Fatalf("stubs merged = %d, want 1: %v", rep.StubsMerged, rep.StubMergeSample)
	}
	if _, err := st.GetEntity("android-assetlinksjson"); err == nil {
		t.Error("the stub must be gone after its facts move")
	}
	target, err := st.GetEntity("android-assetlinks")
	if err != nil {
		t.Fatal(err)
	}
	var hasAlias, hasRef bool
	for _, a := range target.Aliases {
		if a == "assetlinks.json" {
			hasAlias = true
		}
	}
	for _, r := range target.RepoRefs {
		if r == "/Users/jeff/workspace/mobile" {
			hasRef = true
		}
	}
	if !hasAlias || !hasRef {
		t.Errorf("the stub's aliases and repository refs must come with it: %+v", target)
	}
	facts, err := st.FactsFrom("android-assetlinks", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(facts) != 1 || facts[0].Dst != "mac-mini" {
		t.Errorf("the stub's fact must move to the typed entity: %+v", facts)
	}
	// Two typed entities sharing a name are never merged.
	if _, err := st.GetEntity("mac-mini"); err != nil {
		t.Error("a machine must survive a project of the same name")
	}
	if _, err := st.GetEntity("widget-rig-proj"); err != nil {
		t.Error("a project must survive a machine of the same name")
	}
	if rep.CrossTypeCollisions == 0 {
		t.Error("the machine and the project still collide and must still be counted")
	}
}

// The ten fusions a grader found after the first merge ran: names that
// differ by a plural or a separator are different things, and a person or
// a decision never absorbs anything.
func TestStubMergeRefusesTheFusionsAGraderFound(t *testing.T) {
	cases := []struct{ stub, stubType, target, targetType string }{
		{"reports.ts", "concept", "report.ts", "tool"},
		{"webhooks.ts", "concept", "webhook.ts", "tool"},
		{"db/seeds/", "concept", "db:seed", "runbook"},
		{"books", "concept", "book", "project"},
		{"broker", "concept", "brokers", "service"},
		{"CellData", "concept", "cell-datas", "project"},
		{"Stop", "concept", "stops/", "machine"},
		{"API integrations", "concept", "api-integration", "person"},
		{"claude-opus-4-8", "concept", "claude-opus-4.8", "person"},
		{"db/migrations", "concept", "db-migrations", "decision"},
	}
	for _, c := range cases {
		st := openTemp(t)
		now := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
		if err := st.PutEntity(store.Entity{Slug: store.Slugify(c.stub), Name: c.stub, Type: c.stubType, CreatedAt: now, LastSeen: now}); err != nil {
			t.Fatal(err)
		}
		if err := st.PutEntity(store.Entity{Slug: store.Slugify(c.target), Name: c.target, Type: c.targetType, CreatedAt: now, LastSeen: now}); err != nil {
			t.Fatal(err)
		}
		ents, err := st.Entities()
		if err != nil {
			t.Fatal(err)
		}
		n, sample, err := mergeDuplicateStubs(st, ents, true)
		if err != nil {
			t.Fatal(err)
		}
		if n != 0 {
			t.Errorf("%q (%s) must not merge into %q (%s): %v", c.stub, c.stubType, c.target, c.targetType, sample)
		}
	}
}

// The drop for hardware named on a non-machine and a role named on a
// person is the part of the alias cleanup that measured well, and a
// revert of the part that did not once removed it by accident.
func TestHygieneStillDropsRolesAndHardware(t *testing.T) {
	st := openTemp(t)
	now := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	put := func(e store.Entity) {
		e.CreatedAt, e.LastSeen = now, now
		if err := st.PutEntity(e); err != nil {
			t.Fatal(err)
		}
	}
	put(store.Entity{Slug: "jeff", Name: "Jeff", Type: "person",
		Aliases: []string{"jeffdhooton", "Claude agent", "review subagent", "/Users/jeff", "coding-agent"}})
	put(store.Entity{Slug: "some-project", Name: "some project", Type: "project",
		Aliases: []string{"some-proj", "the mac mini box"}})
	if err := st.PutFact(store.Fact{Src: "jeff", Relation: "owns", Value: "a laptop", Fact: "Jeff owns a laptop", ValidFrom: now, Confidence: 1}); err != nil {
		t.Fatal(err)
	}

	if _, err := Hygiene(st, false); err != nil {
		t.Fatal(err)
	}
	jeff, err := st.GetEntity("jeff")
	if err != nil {
		t.Fatal(err)
	}
	if len(jeff.Aliases) != 1 || jeff.Aliases[0] != "jeffdhooton" {
		t.Errorf("a person keeps only their names, got %v", jeff.Aliases)
	}
	proj, err := st.GetEntity("some-project")
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range proj.Aliases {
		if a == "the mac mini box" {
			t.Error("hardware named on a project must be dropped")
		}
	}
}

// Every identical-alias collision in the store had a concept on one
// side. A stub never keeps a name a typed entity answers to, and
// dropping it is safe where transferring was not: the name still
// resolves, to the entity that has a type.
func TestStubsGiveUpNamesATypedEntityAnswersTo(t *testing.T) {
	st := openTemp(t)
	now := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	put := func(e store.Entity) {
		e.CreatedAt, e.LastSeen = now, now
		if err := st.PutEntity(e); err != nil {
			t.Fatal(err)
		}
	}
	put(store.Entity{Slug: "photon-node-sidecar", Name: "photon node sidecar", Type: "service", Aliases: []string{"sidecar"}})
	put(store.Entity{Slug: "meta", Name: "-meta", Type: "concept", Aliases: []string{"sidecar", "its own thing"}})
	put(store.Entity{Slug: "db-migrations", Name: "db-migrations", Type: "decision", Aliases: []string{"migration"}})
	put(store.Entity{Slug: "add-is-beta", Name: "add is beta to users table", Type: "concept", Aliases: []string{"migration"}})

	if _, err := Hygiene(st, false); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ slug, gone, kept string }{
		{"meta", "sidecar", "its own thing"},
		{"add-is-beta", "migration", ""},
	} {
		e, err := st.GetEntity(c.slug)
		if err != nil {
			t.Fatal(err)
		}
		for _, a := range e.Aliases {
			if a == c.gone {
				t.Errorf("%s kept %q, which a typed entity answers to", c.slug, c.gone)
			}
		}
		if c.kept != "" {
			var found bool
			for _, a := range e.Aliases {
				if a == c.kept {
					found = true
				}
			}
			if !found {
				t.Errorf("%s lost %q, which nothing else claims", c.slug, c.kept)
			}
		}
	}
	// The typed entities keep theirs.
	svc, err := st.GetEntity("photon-node-sidecar")
	if err != nil {
		t.Fatal(err)
	}
	if len(svc.Aliases) != 1 || svc.Aliases[0] != "sidecar" {
		t.Errorf("the service must keep its own alias: %v", svc.Aliases)
	}
	// The name still resolves.
	if slug, ok, _ := st.ResolveAlias("sidecar"); !ok || slug != "photon-node-sidecar" {
		t.Errorf("sidecar resolves to %q, want the service", slug)
	}
}
