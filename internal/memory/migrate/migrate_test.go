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
	put(store.Entity{Slug: "hermes-ops", Name: "hermes-ops", Type: "project", Aliases: []string{"Hermes", "mini", "Mac Mini", "the machine", "box", "Hermes ops project", "you", "Hermes agent", "ai.hermes.gateway", "halo1"}})
	put(store.Entity{Slug: "halo-1", Name: "halo-1", Type: "machine"})
	put(store.Entity{Slug: "hermes", Name: "Hermes", Type: "service", Aliases: []string{"Hermes agent"}})
	put(store.Entity{Slug: "mac-mini", Name: "Mac mini", Type: "machine"})
	put(store.Entity{Slug: "qwen", Name: "Qwen38-27B", Type: "tool", Aliases: []string{"gpt-oss-120b", "qwen"}})
	put(store.Entity{Slug: "gpt-oss-120b", Name: "gpt-oss-120b", Type: "tool"})
	put(store.Entity{Slug: "in-progress", Name: "in-progress", Type: "concept", Aliases: []string{"partial"}})
	put(store.Entity{Slug: "main", Name: "main", Type: "concept"})
	put(store.Entity{Slug: "scry", Name: "scry", Type: "project"})
	put(store.Entity{Slug: "childscribe-laravel", Name: "childscribe-laravel", Type: "project"})
	put(store.Entity{Slug: "wren-home-cleaning", Name: "wren-home-cleaning", Type: "project"})
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
		{Src: "scry", Relation: "robots_method_now_welcomes", Dst: "wren-home-cleaning", Fact: "long tail verb"},
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
	if _, ok := facts["scry related_to wren-home-cleaning"]; !ok {
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
	// A status edge between two real entities is not a status: it keeps its
	// edge under related_to instead of making a project into scry's status.
	if f, ok := facts["scry related_to childscribe-laravel"]; !ok || f.Dst != "childscribe-laravel" {
		t.Errorf("status edge to a real entity must stay an edge: %v", keys(facts))
	}
	if _, ok := facts["scry status =childscribe-laravel"]; ok {
		t.Errorf("a real entity must not become another's status value: %v", keys(facts))
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
	for _, bad := range []string{"Hermes", "mini", "Mac Mini", "the machine", "box", "you", "Hermes agent", "ai.hermes.gateway", "halo1"} {
		if contains(ops.Aliases, bad) {
			t.Errorf("hermes-ops still carries %q: %v", bad, ops.Aliases)
		}
	}
	if !contains(ops.Aliases, "Hermes ops project") {
		t.Errorf("hermes-ops lost its own alias: %v", ops.Aliases)
	}
	if _, ok := facts["scry deployed_on mac-mini"]; !ok {
		t.Errorf("fact mentioning the Mac mini not reattached to the machine: %v\ndropped: %v\nreattached: %v", keys(facts), rep.Hygiene.DroppedAliasList, rep.Hygiene.Reattachments)
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
	if slug, _, _ := st.ResolveAlias("Hermes agent"); slug != "hermes" {
		t.Errorf("a split alias must be granted to the entity it names: Hermes agent → %q", slug)
	}
	if slug, _, _ := st.ResolveAlias("halo1"); slug != "halo-1" {
		t.Errorf("halo1 → %q", slug)
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

func TestRunRestoresWronglyDemotedAttributes(t *testing.T) {
	st := openTemp(t)
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	_ = st.PutEntity(store.Entity{Slug: "scry", Name: "scry", Type: "project", CreatedAt: now, LastSeen: now})
	facts := []store.Fact{
		{Src: "scry", Relation: "documents", Value: "docs/DECISIONS.md", Fact: "decisions live in docs/DECISIONS.md", ValidFrom: now},
		{Src: "scry", Relation: "status", Value: "in-progress", Fact: "in progress", ValidFrom: now.Add(time.Minute)},
		{Src: "scry", Relation: "uses", Value: "46 GiB", Fact: "uses 46 GiB", ValidFrom: now.Add(2 * time.Minute)},
	}
	for _, f := range facts {
		f.Confidence, f.Episodes = 0.9, []string{"e1"}
		if err := st.PutFact(f); err != nil {
			t.Fatal(err)
		}
	}
	rep, err := Run(st, Options{Backup: func() (string, error) { return "/tmp/b", nil }})
	if err != nil {
		t.Fatal(err)
	}
	if rep.AttributesRestored != 1 {
		t.Errorf("AttributesRestored = %d, want 1 (only the file path)", rep.AttributesRestored)
	}
	got := factsByKey(t, st)
	if f, ok := got["scry documents docsdecisionsmd"]; !ok || f.Value != "" {
		t.Errorf("file path not restored as an edge: %v", keys(got))
	}
	if _, err := st.GetEntity("docsdecisionsmd"); err != nil {
		t.Errorf("entity not recreated: %v", err)
	}
	if _, ok := got["scry status =in-progress"]; !ok {
		t.Errorf("status attribute must stay: %v", keys(got))
	}
	if _, ok := got["scry uses =46 GiB"]; !ok {
		t.Errorf("measurement attribute must stay: %v", keys(got))
	}
}

func TestRestoreDeployedOnBringsBackWhatExclusivityTook(t *testing.T) {
	st := openTemp(t)
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	later := now.Add(24 * time.Hour)
	for _, e := range []string{"cockpit", "cockpit-daemon", "mac-mini", "advocates", "forge", "retired-app", "old-host"} {
		if err := st.PutEntity(store.Entity{Slug: e, Name: e, Type: "project", CreatedAt: now, LastSeen: now}); err != nil {
			t.Fatal(err)
		}
	}
	put := func(f store.Fact) {
		if err := st.PutFact(f); err != nil {
			t.Fatal(err)
		}
	}
	// Retired by exclusivity: its invalid_at is the ValidFrom of the fact
	// that replaced it.
	put(store.Fact{Src: "cockpit", Relation: "deployed_on", Dst: "cockpit-daemon", Fact: "Cockpit ships its own MCP daemon at http://127.0.0.1:45679/mcp", ValidFrom: now, InvalidAt: &later, Confidence: 1})
	put(store.Fact{Src: "cockpit", Relation: "deployed_on", Dst: "mac-mini", Fact: "cockpit deployed to the mini", ValidFrom: later, Confidence: 1})
	// Retired for another reason: nothing current starts at its invalid_at.
	other := now.Add(72 * time.Hour)
	put(store.Fact{Src: "retired-app", Relation: "deployed_on", Dst: "old-host", Fact: "retired-app ran on old-host until it was decommissioned", ValidFrom: now, InvalidAt: &other, Confidence: 1})
	// Untouched current fact.
	put(store.Fact{Src: "advocates", Relation: "deployed_on", Dst: "forge", Fact: "Advocates runs on forge", ValidFrom: now, Confidence: 1})

	var rep Report
	if err := restoreDeployedOn(st, false, &rep); err != nil {
		t.Fatal(err)
	}
	if rep.DeployedOnRestored != 1 {
		t.Fatalf("restored = %d, want 1", rep.DeployedOnRestored)
	}
	facts, err := st.FactsFrom("cockpit", true)
	if err != nil {
		t.Fatal(err)
	}
	var current int
	for _, f := range facts {
		if f.InvalidAt == nil {
			current++
		}
	}
	if current != 2 {
		t.Errorf("cockpit current deployed_on facts = %d, want both", current)
	}
	retired, err := st.FactsFrom("retired-app", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(retired) != 1 || retired[0].InvalidAt == nil {
		t.Errorf("a fact retired for another reason must stay retired: %+v", retired)
	}
}

func TestDeployedOnIsNotExclusive(t *testing.T) {
	if resolve.DefaultExclusive["deployed_on"] {
		t.Error("deployed_on must not be exclusive: one thing is deployed in more than one place")
	}
	if !resolve.DefaultExclusive["status"] {
		t.Error("status must stay exclusive")
	}
}

func TestRepairInversionsPutsTheNewerFactBack(t *testing.T) {
	st := openTemp(t)
	jun := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	jul := time.Date(2026, 7, 22, 22, 24, 35, 0, time.UTC)
	born := jul.Add(300 * time.Millisecond)
	for _, e := range []string{"aasa", "not-cached", "live", "loom", "one", "two", "app", "hostA", "hostB"} {
		if err := st.PutEntity(store.Entity{Slug: e, Name: e, Type: "project", CreatedAt: jun, LastSeen: jun}); err != nil {
			t.Fatal(err)
		}
	}
	put := func(f store.Fact) {
		if err := st.PutFact(f); err != nil {
			t.Fatal(err)
		}
	}
	// The inversion: the July fact was retired the moment it was written,
	// leaving the June fact current.
	put(store.Fact{Src: "aasa", Relation: "status", Value: "not-cached", Fact: "CDN has not propagated", ValidFrom: jun, Confidence: 1})
	put(store.Fact{Src: "aasa", Relation: "status", Value: "live", Fact: "AASA file is live", ValidFrom: jul, InvalidAt: &born, Confidence: 1})
	// A real supersede: retired long after it began, and left alone.
	end := jul
	put(store.Fact{Src: "loom", Relation: "status", Value: "one", Fact: "loom was one", ValidFrom: jun, InvalidAt: &end, Confidence: 1})
	put(store.Fact{Src: "loom", Relation: "status", Value: "two", Fact: "loom is two", ValidFrom: jul, Confidence: 1})
	// A non-exclusive relation: the newer fact comes back and the older
	// one stays current, because both hold at once.
	put(store.Fact{Src: "app", Relation: "deployed_on", Dst: "hostA", Fact: "app runs on hostA", ValidFrom: jun, Confidence: 1})
	put(store.Fact{Src: "app", Relation: "deployed_on", Dst: "hostB", Fact: "app runs on hostB", ValidFrom: jul, InvalidAt: &born, Confidence: 1})

	var rep Report
	if err := repairInversions(st, false, &rep); err != nil {
		t.Fatal(err)
	}
	if rep.InversionsRepaired != 2 {
		t.Fatalf("repaired = %d, want 2", rep.InversionsRepaired)
	}
	state := func(slug string) map[string]bool {
		facts, err := st.FactsFrom(slug, true)
		if err != nil {
			t.Fatal(err)
		}
		out := map[string]bool{}
		for _, f := range facts {
			out[f.Value+f.Dst] = f.InvalidAt == nil
		}
		return out
	}
	if got := state("aasa"); !got["live"] || got["not-cached"] {
		t.Errorf("aasa = %v, want live current and not-cached retired", got)
	}
	if got := state("loom"); !got["two"] || got["one"] {
		t.Errorf("loom = %v, a real supersede must be left alone", got)
	}
	if got := state("app"); !got["hostA"] || !got["hostB"] {
		t.Errorf("app = %v, want both deployments current", got)
	}

	// Running it again changes nothing.
	var again Report
	if err := repairInversions(st, false, &again); err != nil {
		t.Fatal(err)
	}
	if again.InversionsRepaired != 0 {
		t.Errorf("second pass repaired %d, want a no-op", again.InversionsRepaired)
	}
}

func TestMigrateValuesTakesFactsPointingAtNothing(t *testing.T) {
	st := openTemp(t)
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	if err := st.PutEntity(store.Entity{Slug: "docket", Name: "docket", Type: "project", CreatedAt: now, LastSeen: now}); err != nil {
		t.Fatal(err)
	}
	// ready-to-merge has no entity record: an earlier pass retired it and
	// left this fact pointing at the slug.
	if err := st.PutFact(store.Fact{Src: "ready-to-merge", Relation: "related_to", Dst: "docket", Fact: "the branch was ready to merge", ValidFrom: now, Confidence: 1}); err != nil {
		t.Fatal(err)
	}
	var rep Report
	if err := migrateValues(st, false, &rep); err != nil {
		t.Fatal(err)
	}
	if rep.DanglingEndpoints != 1 {
		t.Fatalf("dangling endpoints = %d, want 1", rep.DanglingEndpoints)
	}
	facts, err := st.FactsAbout("docket", true)
	if err != nil {
		t.Fatal(err)
	}
	var kept int
	for _, f := range facts {
		if f.InvalidAt != nil {
			continue
		}
		kept++
		if f.Src != "docket" || f.Dst != "" || f.Value == "" {
			t.Errorf("fact = %+v, want an attribute of docket carrying the value", f)
		}
		if f.Fact != "the branch was ready to merge" {
			t.Errorf("the sentence must survive: %q", f.Fact)
		}
	}
	if kept != 1 {
		t.Errorf("current facts on docket = %d, want 1", kept)
	}
}
