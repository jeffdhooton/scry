// Package recall provides read-side queries over scry's memory graph:
// ranked fact search (query.go), as-of time travel over facts,
// breadth-first fact-path discovery between two entities, and a markdown
// orientation blurb meant to be injected into agent sessions.
package recall

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jeffdhooton/scry/internal/memory/store"
)

// filterAsOf filters facts to those valid at asOf, or to current facts
// (InvalidAt == nil) when asOf is nil.
func filterAsOf(facts []store.Fact, asOf *time.Time) []store.Fact {
	out := make([]store.Fact, 0, len(facts))
	for _, f := range facts {
		if asOf == nil {
			if f.InvalidAt == nil {
				out = append(out, f)
			}
			continue
		}
		if !f.ValidFrom.After(*asOf) && (f.InvalidAt == nil || f.InvalidAt.After(*asOf)) {
			out = append(out, f)
		}
	}
	return out
}

// collectEpisodes gathers the provenance episode IDs referenced by facts,
// deduped, fetches each via GetEpisode (skipping ones that no longer
// exist), and returns up to limit of the most recent (by OccurredAt).
func collectEpisodes(st *store.Store, facts []store.Fact, limit int) ([]store.Episode, error) {
	seen := make(map[string]bool)
	var ids []string
	for _, f := range facts {
		for _, id := range f.Episodes {
			if seen[id] {
				continue
			}
			seen[id] = true
			ids = append(ids, id)
		}
	}

	var eps []store.Episode
	for _, id := range ids {
		e, err := st.GetEpisode(id)
		if errors.Is(err, store.ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		eps = append(eps, e)
	}

	sort.Slice(eps, func(i, j int) bool { return eps[i].OccurredAt.After(eps[j].OccurredAt) })
	if limit > 0 && len(eps) > limit {
		eps = eps[:limit]
	}
	return eps, nil
}

// Episodes returns up to limit of the most-recent episodes referenced from
// slug's current facts' provenance lists.
func Episodes(st *store.Store, slug string, limit int) ([]store.Episode, error) {
	facts, err := st.FactsAbout(slug, false)
	if err != nil {
		return nil, err
	}
	return collectEpisodes(st, facts, limit)
}

// resolveEndpoint resolves a user-supplied name to a slug: first via the
// alias index, falling back to Slugify when there is no alias match.
func resolveEndpoint(st *store.Store, name string) (string, error) {
	slug, ok, err := st.ResolveAlias(name)
	if err != nil {
		return "", err
	}
	if ok {
		return slug, nil
	}
	return store.Slugify(name), nil
}

// Path finds the shortest chain of current facts connecting from to to,
// treating fact edges as undirected (traversal follows FactsAbout, which
// covers both the Src and Dst side of every fact). Returns store.ErrNotFound
// if either endpoint is unknown or no path exists.
func Path(st *store.Store, from, to string) ([]store.Fact, error) {
	fromSlug, err := resolveEndpoint(st, from)
	if err != nil {
		return nil, err
	}
	toSlug, err := resolveEndpoint(st, to)
	if err != nil {
		return nil, err
	}

	if _, err := st.GetEntity(fromSlug); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, store.ErrNotFound
		}
		return nil, err
	}
	if _, err := st.GetEntity(toSlug); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, store.ErrNotFound
		}
		return nil, err
	}

	visited := map[string]bool{fromSlug: true}
	parentFact := map[string]store.Fact{}
	parentPrev := map[string]string{}
	queue := []string{fromSlug}
	found := fromSlug == toSlug

	for len(queue) > 0 && !found {
		cur := queue[0]
		queue = queue[1:]

		facts, err := st.FactsAbout(cur, false)
		if err != nil {
			return nil, err
		}
		for _, f := range facts {
			if f.IsAttribute() {
				// A value is not a node; an attribute fact leads nowhere.
				continue
			}
			neighbor := f.Dst
			if neighbor == cur {
				neighbor = f.Src
			}
			if neighbor == cur || visited[neighbor] {
				continue
			}
			visited[neighbor] = true
			parentFact[neighbor] = f
			parentPrev[neighbor] = cur
			if neighbor == toSlug {
				found = true
				break
			}
			queue = append(queue, neighbor)
		}
	}

	if !found {
		return nil, store.ErrNotFound
	}

	var chain []store.Fact
	for cur := toSlug; cur != fromSlug; cur = parentPrev[cur] {
		chain = append(chain, parentFact[cur])
	}
	for i, j := 0, len(chain)-1; i < j; i, j = i+1, j-1 {
		chain[i], chain[j] = chain[j], chain[i]
	}
	return chain, nil
}

const (
	defaultOrientBudget = 2000
	orientHeader        = "## Memory orientation"
	orientFooter        = "_Query scry_recall for anything referenced here._"
	activeHeading       = "### Active projects (last 14d)"
	activeWindow        = 14 * 24 * time.Hour
	maxRepoFacts        = 3
	maxActiveEntities   = 10
)

// orientSection is a heading paired with the bullet lines under it; either
// can be dropped wholesale (or trimmed from the end) to fit a budget.
type orientSection struct {
	heading string
	bullets []string
}

// Orient renders a short markdown orientation blurb for injection into
// agent sessions: entities tied to cwd's repo, then recently active
// entities elsewhere, capped at budgetChars (0 uses a 2000-char default).
func Orient(st *store.Store, cwd string, budgetChars int, now time.Time) (string, error) {
	if budgetChars <= 0 {
		budgetChars = defaultOrientBudget
	}

	repoEntities, err := repoEntitiesForCwd(st, cwd)
	if err != nil {
		return "", err
	}
	repoSlugs := make(map[string]bool, len(repoEntities))
	repoBullets := make([]string, 0, len(repoEntities))
	for _, e := range repoEntities {
		repoSlugs[e.Slug] = true
		texts, err := recentFactTexts(st, e.Slug, maxRepoFacts)
		if err != nil {
			return "", err
		}
		repoBullets = append(repoBullets, fmt.Sprintf("- **%s** (%s): %s", e.Name, e.Type, strings.Join(texts, "; ")))
	}

	allEntities, err := st.Entities()
	if err != nil {
		return "", err
	}
	cutoff := now.Add(-activeWindow)
	var active []store.Entity
	for _, e := range allEntities {
		if repoSlugs[e.Slug] {
			continue
		}
		if e.LastSeen.Before(cutoff) {
			continue
		}
		active = append(active, e)
	}
	sort.Slice(active, func(i, j int) bool { return active[i].LastSeen.After(active[j].LastSeen) })
	if len(active) > maxActiveEntities {
		active = active[:maxActiveEntities]
	}

	activeBullets := make([]string, 0, len(active))
	for _, e := range active {
		texts, err := recentFactTexts(st, e.Slug, 1)
		if err != nil {
			return "", err
		}
		text := ""
		if len(texts) > 0 {
			text = texts[0]
		}
		activeBullets = append(activeBullets, fmt.Sprintf("- %s: %s", e.Name, text))
	}

	repoSec := orientSection{
		heading: fmt.Sprintf("### This repo (%s)", filepath.Base(filepath.Clean(cwd))),
		bullets: repoBullets,
	}
	activeSec := orientSection{heading: activeHeading, bullets: activeBullets}

	return renderOrient(repoSec, activeSec, budgetChars), nil
}

// recentFactTexts returns up to limit current facts' Fact text about slug,
// most-recent (by ValidFrom) first.
func recentFactTexts(st *store.Store, slug string, limit int) ([]string, error) {
	facts, err := st.FactsAbout(slug, false)
	if err != nil {
		return nil, err
	}
	sort.Slice(facts, func(i, j int) bool { return facts[i].ValidFrom.After(facts[j].ValidFrom) })
	if len(facts) > limit {
		facts = facts[:limit]
	}
	texts := make([]string, 0, len(facts))
	for _, f := range facts {
		texts = append(texts, f.Fact)
	}
	return texts, nil
}

// repoEntitiesForCwd tries EntitiesByRepoRef at cwd, then walks up one path
// component at a time (stopping at the /Users/<name> level, i.e. once the
// path is down to 2 components) until a match is found or candidates run
// out.
func repoEntitiesForCwd(st *store.Store, cwd string) ([]store.Entity, error) {
	for _, cand := range walkUpCandidates(cwd) {
		entities, err := st.EntitiesByRepoRef(cand)
		if err != nil {
			return nil, err
		}
		if len(entities) > 0 {
			return entities, nil
		}
	}
	return nil, nil
}

func walkUpCandidates(cwd string) []string {
	p := filepath.Clean(cwd)
	var candidates []string
	for {
		candidates = append(candidates, p)
		trimmed := strings.Trim(p, string(filepath.Separator))
		parts := strings.Split(trimmed, string(filepath.Separator))
		if len(parts) <= 2 {
			break
		}
		parent := filepath.Dir(p)
		if parent == p {
			break
		}
		p = parent
	}
	return candidates
}

// renderOrient assembles header, sections, and footer, trimming bullets
// from the end of the active section, then the repo section (dropping
// empty headings along the way), until the result fits budgetChars. If
// even the bare header+footer exceeds budget, that minimal form is
// returned regardless.
func renderOrient(repo, active orientSection, budget int) string {
	for {
		lines := []string{orientHeader}
		if len(repo.bullets) > 0 {
			lines = append(lines, repo.heading)
			lines = append(lines, repo.bullets...)
		}
		if len(active.bullets) > 0 {
			lines = append(lines, active.heading)
			lines = append(lines, active.bullets...)
		}
		lines = append(lines, orientFooter)
		out := strings.Join(lines, "\n")
		if len(out) <= budget {
			return out
		}

		switch {
		case len(active.bullets) > 0:
			active.bullets = active.bullets[:len(active.bullets)-1]
		case len(repo.bullets) > 0:
			repo.bullets = repo.bullets[:len(repo.bullets)-1]
		default:
			return orientHeader + "\n" + orientFooter
		}
	}
}
