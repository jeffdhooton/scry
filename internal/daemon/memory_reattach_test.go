package daemon

import (
	"context"
	"strings"
	"testing"
	"time"

	memstore "github.com/jeffdhooton/scry/internal/memory/store"
)

// Reattach exists because three attempts to refile facts by rule were
// thrown away, so the whole value of it is that it cannot guess. Every
// refusal path matters more than the happy one.
func TestMemoryReattach(t *testing.T) {
	d := newTestMemoryDaemon(t)
	ctx := context.Background()
	st, err := d.memoryStore()
	if err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	for _, e := range []memstore.Entity{
		{Slug: "hermes-ops", Name: "hermes-ops", Type: "project"},
		{Slug: "hermes", Name: "Hermes", Type: "service"},
		{Slug: "cron-mode", Name: "cron mode", Type: "concept"},
	} {
		if err := st.PutEntity(e); err != nil {
			t.Fatal(err)
		}
	}
	fact := memstore.Fact{
		Src: "hermes-ops", Relation: "contains", Dst: "cron-mode",
		Fact: "cron_mode is set to deny", ValidFrom: at, Episodes: []string{"e1"},
	}
	if err := st.PutFact(fact); err != nil {
		t.Fatal(err)
	}

	move := MemoryReattachMove{Src: "hermes-ops", Relation: "contains", Dst: "cron-mode", ValidFrom: at, To: "hermes"}

	t.Run("a dry run writes nothing and takes no backup", func(t *testing.T) {
		out, err := d.handleMemoryReattach(ctx, mustJSON(t, MemoryReattachParams{Moves: []MemoryReattachMove{move}, DryRun: true}))
		if err != nil {
			t.Fatal(err)
		}
		res := out.(*MemoryReattachResult)
		if res.Moved != 1 || res.Refused != 0 {
			t.Errorf("dry run = %+v", res)
		}
		if res.BackupPath != "" {
			t.Error("a dry run must not take a backup")
		}
		if fs, _ := st.FactsFrom("hermes", true); len(fs) != 0 {
			t.Error("a dry run moved a fact")
		}
	})

	t.Run("a fact the caller did not read is refused", func(t *testing.T) {
		wrong := move
		wrong.ValidFrom = at.Add(time.Hour)
		out, err := d.handleMemoryReattach(ctx, mustJSON(t, MemoryReattachParams{Moves: []MemoryReattachMove{wrong}, DryRun: true}))
		if err != nil {
			t.Fatal(err)
		}
		res := out.(*MemoryReattachResult)
		if res.Moved != 0 || res.Refused != 1 {
			t.Errorf("a stale move should be refused: %+v", res)
		}
		if !strings.Contains(strings.Join(res.Details, " "), "no current fact") {
			t.Errorf("refusal should say why: %v", res.Details)
		}
	})

	t.Run("a destination that does not exist is refused", func(t *testing.T) {
		bad := move
		bad.To = "nonexistent"
		out, _ := d.handleMemoryReattach(ctx, mustJSON(t, MemoryReattachParams{Moves: []MemoryReattachMove{bad}, DryRun: true}))
		res := out.(*MemoryReattachResult)
		if res.Refused != 1 || !strings.Contains(strings.Join(res.Details, " "), "does not exist") {
			t.Errorf("%+v", res)
		}
	})

	t.Run("a move that would be a self-loop is refused", func(t *testing.T) {
		loop := move
		loop.To = "cron-mode"
		out, _ := d.handleMemoryReattach(ctx, mustJSON(t, MemoryReattachParams{Moves: []MemoryReattachMove{loop}, DryRun: true}))
		res := out.(*MemoryReattachResult)
		if res.Refused != 1 || !strings.Contains(strings.Join(res.Details, " "), "self-loop") {
			t.Errorf("%+v", res)
		}
	})

	t.Run("applying moves the fact and keeps everything about it", func(t *testing.T) {
		out, err := d.handleMemoryReattach(ctx, mustJSON(t, MemoryReattachParams{Moves: []MemoryReattachMove{move}}))
		if err != nil {
			t.Fatal(err)
		}
		res := out.(*MemoryReattachResult)
		if res.Moved != 1 {
			t.Fatalf("%+v", res)
		}
		if res.BackupPath == "" {
			t.Error("an applied reattach must take a backup first")
		}
		gone, _ := st.FactsFrom("hermes-ops", true)
		if len(gone) != 0 {
			t.Errorf("fact still on the project: %+v", gone)
		}
		got, _ := st.FactsFrom("hermes", true)
		if len(got) != 1 {
			t.Fatalf("fact did not land on the service: %+v", got)
		}
		if got[0].Fact != fact.Fact || got[0].Dst != "cron-mode" || len(got[0].Episodes) != 1 {
			t.Errorf("the fact changed on the way: %+v", got[0])
		}
		if !got[0].ValidFrom.Equal(at) {
			t.Errorf("validity changed: %v", got[0].ValidFrom)
		}
	})
}

// The pre-apply reviewer found that reattach claimed two guarantees it did
// not implement: it never compared the fact's text, and it matched
// invalidated facts. It also found that a key collision on the destination
// was resolved by nudging valid_from a nanosecond, silently, twice on the
// first real list. These cover all three.
func TestMemoryReattachRefusesWhatItCannotVerify(t *testing.T) {
	d := newTestMemoryDaemon(t)
	ctx := context.Background()
	st, _ := d.memoryStore()
	at := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	for _, e := range []memstore.Entity{
		{Slug: "ops", Name: "ops", Type: "project"},
		{Slug: "agent", Name: "agent", Type: "service"},
		{Slug: "thing", Name: "thing", Type: "concept"},
	} {
		if err := st.PutEntity(e); err != nil {
			t.Fatal(err)
		}
	}
	put := func(src, text string, at time.Time, invalid bool) {
		f := memstore.Fact{Src: src, Relation: "uses", Dst: "thing", Fact: text, ValidFrom: at, Episodes: []string{"e"}}
		if invalid {
			inv := at.Add(time.Hour)
			f.InvalidAt = &inv
		}
		if err := st.PutFact(f); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("a fact whose text has changed is refused", func(t *testing.T) {
		put("ops", "the sentence as it is now", at, false)
		out, _ := d.handleMemoryReattach(ctx, mustJSON(t, MemoryReattachParams{DryRun: true, Moves: []MemoryReattachMove{
			{Src: "ops", Relation: "uses", Dst: "thing", ValidFrom: at, Fact: "the sentence the reviewer read", To: "agent"},
		}}))
		res := out.(*MemoryReattachResult)
		if res.Moved != 0 || res.Refused != 1 || !strings.Contains(strings.Join(res.Details, " "), "reads differently") {
			t.Errorf("%+v", res)
		}
	})

	t.Run("an invalidated fact is refused", func(t *testing.T) {
		iat := at.Add(48 * time.Hour)
		put("ops", "retired", iat, true)
		out, _ := d.handleMemoryReattach(ctx, mustJSON(t, MemoryReattachParams{DryRun: true, Moves: []MemoryReattachMove{
			{Src: "ops", Relation: "uses", Dst: "thing", ValidFrom: iat, To: "agent"},
		}}))
		res := out.(*MemoryReattachResult)
		if res.Moved != 0 || res.Refused != 1 {
			t.Errorf("an invalidated fact must not move: %+v", res)
		}
	})

	t.Run("a destination already holding the fact at the same time is refused", func(t *testing.T) {
		cat := at.Add(72 * time.Hour)
		put("ops", "the project's copy", cat, false)
		put("agent", "the service's copy", cat, false)
		out, _ := d.handleMemoryReattach(ctx, mustJSON(t, MemoryReattachParams{DryRun: true, Moves: []MemoryReattachMove{
			{Src: "ops", Relation: "uses", Dst: "thing", ValidFrom: cat, To: "agent"},
		}}))
		res := out.(*MemoryReattachResult)
		if res.Moved != 0 || res.Refused != 1 || !strings.Contains(strings.Join(res.Details, " "), "same time") {
			t.Errorf("a key collision must be reported, not nudged: %+v", res)
		}
	})

	t.Run("a destination already saying it differently warns but proceeds", func(t *testing.T) {
		wat := at.Add(96 * time.Hour)
		put("ops", "the project's wording", wat, false)
		put("agent", "the service's older wording", wat.Add(-time.Hour), false)
		out, _ := d.handleMemoryReattach(ctx, mustJSON(t, MemoryReattachParams{DryRun: true, Moves: []MemoryReattachMove{
			{Src: "ops", Relation: "uses", Dst: "thing", ValidFrom: wat, To: "agent"},
		}}))
		res := out.(*MemoryReattachResult)
		if res.Moved != 1 || res.Warned != 1 {
			t.Errorf("a duplicate edge should warn and proceed: %+v", res)
		}
	})
}

// A fact can be filed under the wrong entity from either end. "The feedback
// digest job runs on the hermes Mac mini" was stored with the project as its
// destination, and moving the source would have been the wrong repair.
func TestMemoryReattachMovesTheFarEnd(t *testing.T) {
	d := newTestMemoryDaemon(t)
	ctx := context.Background()
	st, _ := d.memoryStore()
	at := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	for _, e := range []memstore.Entity{
		{Slug: "ops", Name: "ops", Type: "project"},
		{Slug: "mini", Name: "mini", Type: "machine"},
		{Slug: "job", Name: "job", Type: "service"},
	} {
		if err := st.PutEntity(e); err != nil {
			t.Fatal(err)
		}
	}
	if err := st.PutFact(memstore.Fact{
		Src: "job", Relation: "deployed_on", Dst: "ops",
		Fact: "the digest job runs on the mini", ValidFrom: at, Episodes: []string{"e"},
	}); err != nil {
		t.Fatal(err)
	}
	move := MemoryReattachMove{
		Src: "job", Relation: "deployed_on", Dst: "ops", ValidFrom: at,
		Fact: "the digest job runs on the mini", To: "mini", Side: "dst",
	}
	out, err := d.handleMemoryReattach(ctx, mustJSON(t, MemoryReattachParams{Moves: []MemoryReattachMove{move}}))
	if err != nil {
		t.Fatal(err)
	}
	if res := out.(*MemoryReattachResult); res.Moved != 1 {
		t.Fatalf("%+v", res)
	}
	onOps, _ := st.FactsAbout("ops", false)
	if len(onOps) != 0 {
		t.Errorf("the project still holds it: %+v", onOps)
	}
	got, _ := st.FactsFrom("job", false)
	if len(got) != 1 || got[0].Dst != "mini" || got[0].Src != "job" {
		t.Errorf("the far end did not move: %+v", got)
	}
	if got[0].Fact != "the digest job runs on the mini" {
		t.Errorf("the fact changed: %+v", got[0])
	}
}

// Reattaching facts without pruning the aliases that attracted them is, in
// a reviewer's phrase, bailing a boat with the hole still in it.
func TestMemoryUnalias(t *testing.T) {
	d := newTestMemoryDaemon(t)
	ctx := context.Background()
	st, _ := d.memoryStore()
	ops := memstore.Entity{Slug: "ops", Name: "hermes-ops", Type: "project",
		Aliases: []string{"Hermes Slack gateway", "Jeff's own Hermes", "ops repo"}}
	agent := memstore.Entity{Slug: "agent", Name: "Hermes", Type: "service"}
	for _, e := range []memstore.Entity{ops, agent} {
		if err := st.PutEntity(e); err != nil {
			t.Fatal(err)
		}
	}
	// Manufacture legacy disagreement: agent lists the gateway spelling,
	// while the leaking project still owns its index key.
	if err := st.ClaimAlias("Hermes Slack gateway", "agent"); err != nil {
		t.Fatal(err)
	}
	agent.Aliases = []string{"Hermes Slack gateway"}
	if err := st.PutEntity(agent); err != nil {
		t.Fatal(err)
	}
	if err := st.ClaimAlias("Hermes Slack gateway", "ops"); err != nil {
		t.Fatal(err)
	}
	// A spelling another entity has since claimed must survive on that one.
	if err := st.ClaimAlias("Jeff's own Hermes", "agent"); err != nil {
		t.Fatal(err)
	}

	t.Run("refuses what the entity does not list", func(t *testing.T) {
		out, _ := d.handleMemoryUnalias(ctx, mustJSON(t, MemoryUnaliasParams{DryRun: true,
			Drops: []MemoryUnaliasDrop{{Entity: "ops", Alias: "not held"}}}))
		if res := out.(*MemoryUnaliasResult); res.Refused != 1 || res.Dropped != 0 {
			t.Errorf("%+v", res)
		}
	})

	t.Run("refuses the entity's own name", func(t *testing.T) {
		out, _ := d.handleMemoryUnalias(ctx, mustJSON(t, MemoryUnaliasParams{DryRun: true,
			Drops: []MemoryUnaliasDrop{{Entity: "ops", Alias: "hermes-ops"}}}))
		if res := out.(*MemoryUnaliasResult); res.Refused != 1 {
			t.Errorf("%+v", res)
		}
	})

	t.Run("requires rehome when another entity lists an owned alias", func(t *testing.T) {
		out, _ := d.handleMemoryUnalias(ctx, mustJSON(t, MemoryUnaliasParams{DryRun: true,
			Drops: []MemoryUnaliasDrop{{Entity: "ops", Alias: "Hermes Slack gateway"}}}))
		if res := out.(*MemoryUnaliasResult); res.Refused != 1 || res.Dropped != 0 {
			t.Errorf("%+v", res)
		}
	})

	t.Run("drops the leak and leaves the index another entity owns", func(t *testing.T) {
		out, err := d.handleMemoryUnalias(ctx, mustJSON(t, MemoryUnaliasParams{
			Drops: []MemoryUnaliasDrop{
				{Entity: "ops", Alias: "Hermes Slack gateway", RehomeTo: "agent", Why: "the agent lists and owns this service name"},
				{Entity: "ops", Alias: "Jeff's own Hermes"},
			}}))
		if err != nil {
			t.Fatal(err)
		}
		res := out.(*MemoryUnaliasResult)
		if res.Dropped != 2 || res.BackupPath == "" {
			t.Fatalf("%+v", res)
		}
		e, _ := st.GetEntity("ops")
		if len(e.Aliases) != 1 || e.Aliases[0] != "ops repo" {
			t.Errorf("aliases = %v", e.Aliases)
		}
		// The gateway spelling pointed at ops, but the agent also listed it;
		// reviewed rehome must not leave that spelling unindexed.
		if slug, found, _ := st.ResolveAlias("Hermes Slack gateway"); !found || slug != "agent" {
			t.Errorf("the rehomed gateway alias resolves to %q, %v", slug, found)
		}
		// This one pointed at the agent, so the agent keeps it.
		slug, found, _ := st.ResolveAlias("Jeff's own Hermes")
		if !found || slug != "agent" {
			t.Errorf("a spelling another entity owns was stripped: %q %v", slug, found)
		}
	})
}
