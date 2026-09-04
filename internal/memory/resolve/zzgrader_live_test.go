package resolve

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/jeffdhooton/scry/internal/memory/store"
)

func openLive(t *testing.T) *store.Store {
	dir := os.Getenv("GRADER_STORE")
	if dir == "" {
		t.Skip("no GRADER_STORE")
	}
	st, err := store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

// TaskA: entity + fact dump for the identities under test.
func TestGraderTaskA(t *testing.T) {
	st := openLive(t)
	for _, slug := range strings.Split(os.Getenv("GRADER_SLUGS"), ",") {
		slug = strings.TrimSpace(slug)
		if slug == "" {
			continue
		}
		e, err := st.GetEntity(slug)
		if err != nil {
			fmt.Printf("### %s: ERR %v\n", slug, err)
			continue
		}
		facts, _ := st.FactsAbout(slug, false)
		fmt.Printf("### %s (%s) aliases=%d facts=%d\n", e.Slug, e.Type, len(e.Aliases), len(facts))
		for i, f := range facts {
			fmt.Printf("F%03d [%s] src=%s rel=%s dst=%s val=%q :: %s\n", i, strings.Join(f.Episodes, "|"), f.Src, f.Relation, f.Dst, f.Value, f.Fact)
		}
	}
}

type refusal struct {
	Slug, Type, Alias, Reason string
}

// TaskB: which live aliases would the current admission rules refuse?
func TestGraderTaskB(t *testing.T) {
	st := openLive(t)
	ents, err := st.Entities()
	if err != nil {
		t.Fatal(err)
	}
	if err := RefreshCompactIndex(st); err != nil {
		t.Fatal(err)
	}
	bySlug := map[string]store.Entity{}
	for _, e := range ents {
		bySlug[e.Slug] = e
	}
	var never, leak, named, xtype, revalDrop []refusal
	total := 0
	for _, e := range ents {
		// RevalidateAliases on a copy: read-only w.r.t. the store.
		cp := e
		cp.Aliases = append([]string(nil), e.Aliases...)
		before := map[string]bool{}
		for _, a := range cp.Aliases {
			before[a] = true
		}
		if err := RevalidateAliases(st, &cp); err != nil {
			t.Fatal(err)
		}
		after := map[string]bool{}
		for _, a := range cp.Aliases {
			after[a] = true
		}
		for a := range before {
			if !after[a] {
				revalDrop = append(revalDrop, refusal{e.Slug, e.Type, a, "revalidate drops"})
			}
		}
		for _, a := range e.Aliases {
			total++
			if neverAlias(a) {
				never = append(never, refusal{e.Slug, e.Type, a, "neverAlias"})
				continue
			}
			if why := leakReason(a, e); why != "" {
				leak = append(leak, refusal{e.Slug, e.Type, a, why})
				continue
			}
			if n, err := namedByKindWords(st, a, e); err == nil && n != "" && n != e.Slug {
				named = append(named, refusal{e.Slug, e.Type, a, "names " + n})
				continue
			}
			owner, ok, err := st.ResolveAlias(a)
			if err != nil {
				t.Fatal(err)
			}
			if ok && owner != e.Slug {
				if o, gerr := st.GetEntity(owner); gerr == nil && !TypesCompatible(o.Type, e.Type) {
					xtype = append(xtype, refusal{e.Slug, e.Type, a, "owned by " + owner + " (" + o.Type + ")"})
				}
			}
		}
	}
	fmt.Printf("TASKB total_aliases=%d never=%d leak=%d named_other=%d cross_type_owned=%d revalidate_drops=%d\n",
		total, len(never), len(leak), len(named), len(xtype), len(revalDrop))
	dump := func(label string, rs []refusal) {
		sort.Slice(rs, func(i, j int) bool { return rs[i].Slug < rs[j].Slug })
		fmt.Printf("--- %s (%d)\n", label, len(rs))
		for i, r := range rs {
			if i >= 60 {
				break
			}
			fmt.Printf("  %s [%s] %q :: %s\n", r.Slug, r.Type, r.Alias, r.Reason)
		}
	}
	dump("neverAlias", never)
	dump("leakReason", leak)
	dump("namedByKindWords", named)
	dump("crossTypeOwned", xtype)
	dump("revalidateDrops", revalDrop)
	b, _ := json.Marshal(map[string]any{"never": never, "leak": leak, "named": named, "xtype": xtype, "reval": revalDrop})
	os.WriteFile(os.Getenv("GRADER_OUT")+"/taskb.json", b, 0o644)
}

// TaskC: hygiene dry-run on the copy, and the collision audit.
func TestGraderTaskCHygieneDry(t *testing.T) {
	st := openLive(t)
	rep, err := Hygiene(st, true)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.MarshalIndent(rep, "", " ")
	os.WriteFile(os.Getenv("GRADER_OUT")+"/hygiene-dry.json", b, 0o644)
	fmt.Printf("DRY scanned=%d changed=%d dropped=%d split=%d reattached=%d selfloops=%d crossType=%d unref=%d stubClaims=%d stubsMerged=%d\n",
		rep.EntitiesScanned, rep.EntitiesChanged, rep.AliasesDropped, rep.AliasesSplit, rep.FactsReattached,
		rep.SelfLoopsInvalidated, rep.CrossTypeCollisions, rep.EntitiesUnreferenced, rep.StubClaimsDropped, rep.StubsMerged)
	for i, s := range rep.CollisionSample {
		if i >= 40 {
			break
		}
		fmt.Println("  C:", s)
	}
}
