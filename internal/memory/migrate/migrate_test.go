package migrate

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jeffdhooton/scry/internal/memory/resolve"
	"github.com/jeffdhooton/scry/internal/memory/store"
)

func openTemp(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "badger"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

// seedAuditStore builds a small store shaped like the 2026-09-02 audit:
// fused identities, an unbounded relation vocabulary, value entities, and
// a self-loop.
func seedAuditStore(t *testing.T, st *store.Store) {
	t.Helper()
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	put := func(e store.Entity) {
		e.CreatedAt, e.LastSeen = now, now
		if err := st.PutEntity(e); err != nil {
			t.Fatal(err)
		}
	}
	put(store.Entity{Slug: "hermes-ops", Name: "hermes-ops", Type: "project", Aliases: []string{"Hermes", "mini", "Mac Mini", "the machine", "box", "Hermes ops project", "you"}})
	put(store.Entity{Slug: "hermes", Name: "Hermes", Type: "service", Aliases: []string{"Hermes agent"}})
	put(store.Entity{Slug: "mac-mini", Name: "Mac mini", Type: "machine"})
	put(store.Entity{Slug: "qwen", Name: "Qwen38-27B", Type: "tool", Aliases: []string{"gpt-oss-120b", "qwen"}})
	put(store.Entity{Slug: "gpt-oss-120b", Name: "gpt-oss-120b", Type: "tool"})
	put(store.Entity{Slug: "in-progress", Name: "in-progress", Type: "concept", Aliases: []string{"partial"}})
	put(store.Entity{Slug: "main", Name: "main", Type: "concept"})
	put(store.Entity{Slug: "scry", Name: "scry", Type: "project"})
	put(store.Entity{Slug: "childscribe-laravel", Name: "childscribe-laravel", Type: "project"})
	put(store.Entity{Slug: "51b-active-parameters", Name: "51B active parameters", Type: "concept"})

	facts := []store.Fact{
		{Src: "scry", Relation: "used_by", Dst: "childscribe-laravel", Fact: "childscribe-laravel uses scry"},
		{Src: "scry", Relation: "installed_on", Dst: "hermes-ops", Fact: "scry is installed on the Mac mini"},
		{Src: "scry", Relation: "deployed_on", Dst: "hermes-ops", Fact: "scry deploys to hermes-ops via ssh"},
		{Src: "hermes-ops", Relation: "monitors", Dst: "childscribe-laravel", Fact: "Hermes watches childscribe for alerts"},
		{Src: "scry", Relation: "status", Dst: "in-progress", Fact: "scry work is in progress"},
		{Src: "main", Relation: "contains", Dst: "scry", Fact: "main has the scry code"},
		{Src: "in-progress", Relation: "related_to", Dst: "main", Fact: "nonsense between two values"},
		{Src: "gpt-oss-120b", Relation: "has_active_parameters", Dst: "51b-active-parameters", Fact: "gpt-oss-120b has 51B active parameters"},
		{Src: "qwen", Relation: "runs_on", Dst: "hermes-ops", Fact: "gpt-oss-120b runs on halo"},
		{Src: "scry", Relation: "status", Dst: "childscribe-laravel", Fact: "odd status edge"},
		{Src: "scry", Relation: "robots_method_now_welcomes", Dst: "childscribe-laravel", Fact: "long tail verb"},
		{Src: "scry", Relation: "uses", Dst: "scry", Fact: "self loop"},
	}
	for i, f := range facts {
		f.ValidFrom = now.Add(time.Duration(i) * time.Minute)
		f.Confidence = 0.9
		f.Episodes = []string{"e1"}
		if err := st.PutFact(f); err != nil {
			t.Fatal(err)
		}
	}
}

func factsByKey(t *testing.T, st *store.Store) map[string]store.Fact {
	t.Helper()
	all, err := st.AllFacts()
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]store.Fact{}
	for _, f := range all {
		key := f.Src + " " + f.Relation + " " + f.Dst
		if f.Dst == "" {
			key = f.Src + " " + f.Relation + " =" + f.Value
		}
		out[key] = f
	}
	return out
}

func TestRunDryRunWritesNothing(t *testing.T) {
	st := openTemp(t)
	seedAuditStore(t, st)
	before := factsByKey(t, st)
	rep, err := Run(st, Options{DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if rep.RelationsRewritten == 0 || rep.ValueEntities != 3 || rep.Hygiene.AliasesDropped == 0 {
		t.Errorf("dry run found nothing: %+v", rep)
	}
	after := factsByKey(t, st)
	if len(before) != len(after) {
		t.Errorf("dry run changed facts: %d → %d", len(before), len(after))
	}
	for k := range before {
		if _, ok := after[k]; !ok {
			t.Errorf("dry run removed %q", k)
		}
	}
	if e, err := st.GetEntity("main"); err != nil || len(e.Aliases) != 0 {
		t.Errorf("dry run touched entities: %+v %v", e, err)
	}
	if rep.BackupPath != "" {
		t.Error("dry run must not back up")
	}
}

func TestRunAppliesEveryRule(t *testing.T) {
	st := openTemp(t)
	seedAuditStore(t, st)
	backups := 0
	rep, err := Run(st, Options{Backup: func() (string, error) { backups++; return "/tmp/b.badger", nil }})
	if err != nil {
		t.Fatal(err)
	}
	if backups != 1 || rep.BackupPath != "/tmp/b.badger" {
		t.Errorf("backup not taken first: %+v", rep)
	}
	if rep.NonCanonicalAfter != 0 || rep.ValueEntitiesAfter != 0 || rep.Hygiene.CrossTypeCollisions != 0 {
		t.Errorf("audit after apply: %+v", rep)
	}
	facts := factsByKey(t, st)

	// Relations: inverse flipped, synonym mapped, long tail on related_to.
	if f, ok := facts["childscribe-laravel uses scry"]; !ok || f.RawRelation != "used_by" {
		t.Errorf("used_by not flipped into uses: %v", keys(facts))
	}
	if _, ok := facts["scry related_to childscribe-laravel"]; !ok {
		t.Errorf("long tail verb not on related_to: %v", keys(facts))
	}

	// Values: attributes, no entities.
	for _, slug := range []string{"in-progress", "main", "51b-active-parameters"} {
		if _, err := st.GetEntity(slug); err == nil {
			t.Errorf("value entity %q survived", slug)
		}
	}
	if f, ok := facts["scry status =in-progress"]; !ok || f.Dst != "" {
		t.Errorf("status edge not converted: %v", keys(facts))
	}
	if _, ok := facts["scry contains =main"]; !ok {
		t.Errorf("value-src fact not flipped into an attribute: %v", keys(facts))
	}
	if _, ok := facts["gpt-oss-120b status =51B active parameters"]; !ok {
		t.Errorf("measurement not converted: %v", keys(facts))
	}
	if f, ok := facts["scry status =childscribe-laravel"]; !ok || f.Dst != "" {
		t.Errorf("status edge to a real entity must become an attribute: %v", keys(facts))
	}
	if f, ok := facts["in-progress related_to main"]; !ok || f.InvalidAt == nil {
		t.Errorf("value-to-value fact must be invalidated, not deleted: %+v", f)
	}

	// Self loop invalidated.
	if f, ok := facts["scry uses scry"]; !ok || f.InvalidAt == nil {
		t.Errorf("self loop: %+v", f)
	}

	// Hygiene: hermes-ops loses the machine's and the service's names and
	// the reference words; the mention facts moved.
	ops, err := st.GetEntity("hermes-ops")
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"Hermes", "mini", "Mac Mini", "the machine", "box", "you"} {
		if contains(ops.Aliases, bad) {
			t.Errorf("hermes-ops still carries %q: %v", bad, ops.Aliases)
		}
	}
	if !contains(ops.Aliases, "Hermes ops project") {
		t.Errorf("hermes-ops lost its own alias: %v", ops.Aliases)
	}
	if _, ok := facts["scry deployed_on mac-mini"]; !ok {
		t.Errorf("fact mentioning the Mac mini not reattached to the machine: %v", keys(facts))
	}
	if _, ok := facts["scry deployed_on hermes-ops"]; !ok {
		t.Errorf("fact about hermes-ops itself must stay: %v", keys(facts))
	}
	if _, ok := facts["hermes monitors childscribe-laravel"]; !ok {
		t.Errorf("fact naming Hermes not reattached to the service: %v", keys(facts))
	}
	if slug, _, _ := st.ResolveAlias("Mac mini"); slug != "mac-mini" {
		t.Errorf("Mac mini resolves to %q", slug)
	}
	if slug, _, _ := st.ResolveAlias("Hermes"); slug != "hermes" {
		t.Errorf("Hermes resolves to %q", slug)
	}
	qwen, _ := st.GetEntity("qwen")
	if contains(qwen.Aliases, "gpt-oss-120b") {
		t.Errorf("qwen still carries gpt-oss-120b: %v", qwen.Aliases)
	}
	if _, ok := facts["gpt-oss-120b runs_on hermes-ops"]; !ok {
		t.Errorf("fact mentioning gpt-oss-120b not reattached: %v", keys(facts))
	}
	if slug, _, _ := st.ResolveAlias("gpt-oss-120b"); slug != "gpt-oss-120b" {
		t.Errorf("gpt-oss-120b resolves to %q", slug)
	}
	if rep.Hygiene.FactsReattached != 3 {
		t.Errorf("FactsReattached = %d, want 3", rep.Hygiene.FactsReattached)
	}

	// Idempotent.
	second, err := Run(st, Options{Backup: func() (string, error) { return "/tmp/c.badger", nil }})
	if err != nil {
		t.Fatal(err)
	}
	if second.RelationsRewritten != 0 || second.ValueEntities != 0 || second.ValueFactsConverted != 0 || second.Hygiene.AliasesDropped != 0 || second.Hygiene.AliasesSplit != 0 || second.Hygiene.FactsReattached != 0 || second.Hygiene.SelfLoopsInvalidated != 0 {
		t.Errorf("second run was not a no-op: %+v", second)
	}
	if second.Hygiene.CrossTypeCollisions != 0 {
		t.Errorf("collisions after second run: %d", second.Hygiene.CrossTypeCollisions)
	}
}

func TestRunRequiresBackupToApply(t *testing.T) {
	st := openTemp(t)
	if _, err := Run(st, Options{}); err == nil || !strings.Contains(err.Error(), "backup") {
		t.Fatalf("err = %v, want a backup requirement", err)
	}
}

func keys(m map[string]store.Fact) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

var _ = resolve.RelStatus
