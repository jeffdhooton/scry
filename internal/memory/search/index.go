// Package search is the lexical entry point into the memory graph: an
// in-memory BM25 index over fact sentences and entity names. Recall used
// to find entities by substring and return every fact on them; this ranks
// the facts themselves, so "why did we switch off deepseek" finds the
// sentence that says why rather than every fact on a Qwen model whose
// alias contains "switch".
//
// The index is rebuilt from the store at daemon start (a second or so for
// 30k facts) and kept current through store.Observer. Nothing is persisted:
// there is no key layout to migrate and no way for it to drift from the
// store.
package search

import (
	"math"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jeffdhooton/scry/internal/memory/store"
)

// Kind says what a document is.
const (
	KindFact    = "fact"
	KindEntity  = "entity"
	KindEpisode = "episode"
)

// prefixWeight is the share of a full-token match a 4-character prefix
// token earns: enough to bridge "tailnet" and "Tailscale" or "deploy" and
// "deployment", not enough to outrank a real match.
const prefixWeight = 0.4

// Doc is one indexed item. For a fact, Key is the store key string and
// Slugs holds the src (and dst) entity slugs; for an entity, Key is
// "en:<slug>" and Slugs is just the slug.
type Doc struct {
	Kind      string
	Key       string
	Text      string
	Slugs     []string
	ValidFrom time.Time
	InvalidAt *time.Time
	Fact      store.Fact // populated for facts
}

// Hit is one ranked result.
type Hit struct {
	Doc   Doc
	Score float64
}

const (
	k1 = 1.2
	b  = 0.75
)

type posting struct {
	doc int
	tf  int
}

// Index is a BM25 index. Safe for concurrent use.
type Index struct {
	mu       sync.RWMutex
	docs     []Doc
	byKey    map[string]int
	postings map[string][]posting
	lengths  []int
	totalLen int
	live     int // docs not removed
	dead     int // tombstoned slots since the last compaction
	// names maps entity slugs to display names so a fact indexed later
	// still carries "Mac mini" for mac-mini.
	names map[string]string
}

// New returns an empty index.
func New() *Index {
	return &Index{byKey: map[string]int{}, postings: map[string][]posting{}, names: map[string]string{}}
}

// Build indexes every fact and entity in st into a new index. Callers that
// need writes made during the build to land should subscribe the store's
// observer to the index BEFORE calling Load; see Load.
func Build(st *store.Store) (*Index, error) {
	ix := New()
	if err := ix.Load(st); err != nil {
		return nil, err
	}
	return ix, nil
}

// Load fills ix from st. It holds the write lock for the duration, so an
// observer that upserts concurrently blocks and then applies after the
// snapshot: a write made mid-build is never lost, only deferred a second.
func (ix *Index) Load(st *store.Store) error {
	facts, err := st.AllFacts()
	if err != nil {
		return err
	}
	entities, err := st.Entities()
	if err != nil {
		return err
	}
	episodes, err := st.AllEpisodes()
	if err != nil {
		return err
	}
	ix.mu.Lock()
	defer ix.mu.Unlock()
	for _, e := range entities {
		ix.names[e.Slug] = e.Name
	}
	for _, f := range facts {
		ix.upsertLocked(FactDoc(f, ix.names))
	}
	for _, e := range entities {
		ix.upsertLocked(EntityDoc(e))
	}
	for _, ep := range episodes {
		ix.upsertLocked(EpisodeDoc(ep))
	}
	return nil
}

// EpisodeDoc renders an episode summary as a document. Summaries are the
// model's paraphrase of a session, so they carry words a fact sentence
// leaves out; a query that hits one boosts the facts it produced.
func EpisodeDoc(ep store.Episode) Doc {
	return Doc{Kind: KindEpisode, Key: "ep:" + ep.ID, Text: ep.Summary + " " + ep.Source, ValidFrom: ep.OccurredAt}
}

// UpsertEpisode indexes ep.
func (ix *Index) UpsertEpisode(ep store.Episode) {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	ix.upsertLocked(EpisodeDoc(ep))
}

// UpsertFact indexes f using the index's own name table.
func (ix *Index) UpsertFact(f store.Fact) {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	ix.upsertLocked(FactDoc(f, ix.names))
}

// UpsertEntity indexes e and records its display name.
func (ix *Index) UpsertEntity(e store.Entity) {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	ix.names[e.Slug] = e.Name
	ix.upsertLocked(EntityDoc(e))
}

// RemoveEntity drops an entity document and its name.
func (ix *Index) RemoveEntity(slug string) {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	delete(ix.names, slug)
	if i, ok := ix.byKey["en:"+slug]; ok {
		ix.removeLocked(i)
	}
}

// FactDoc renders a fact as a document. names maps slugs to display names
// so "scry deployed_on mac-mini" also indexes "Mac mini".
func FactDoc(f store.Fact, names map[string]string) Doc {
	var sb strings.Builder
	sb.WriteString(f.Fact)
	sb.WriteString(" ")
	sb.WriteString(f.Src)
	if n := names[f.Src]; n != "" {
		sb.WriteString(" " + n)
	}
	sb.WriteString(" " + strings.ReplaceAll(f.Relation, "_", " "))
	if f.RawRelation != "" {
		sb.WriteString(" " + strings.ReplaceAll(f.RawRelation, "_", " "))
	}
	if f.Dst != "" {
		sb.WriteString(" " + f.Dst)
		if n := names[f.Dst]; n != "" {
			sb.WriteString(" " + n)
		}
	} else {
		sb.WriteString(" " + f.Value)
	}
	slugs := []string{f.Src}
	if f.Dst != "" {
		slugs = append(slugs, f.Dst)
	}
	return Doc{Kind: KindFact, Key: FactKey(f), Text: sb.String(), Slugs: slugs, ValidFrom: f.ValidFrom, InvalidAt: f.InvalidAt, Fact: f}
}

// EntityDoc renders an entity as a document: name, aliases, description.
func EntityDoc(e store.Entity) Doc {
	text := e.Name + " " + e.Slug + " " + strings.Join(e.Aliases, " ") + " " + e.Description + " " + e.Type
	return Doc{Kind: KindEntity, Key: "en:" + e.Slug, Text: text, Slugs: []string{e.Slug}, ValidFrom: e.CreatedAt}
}

// FactKey is the index key for a fact: the store's own key layout.
func FactKey(f store.Fact) string {
	return "fa:" + f.Src + ":" + f.Relation + ":" + f.KeyDst() + ":" + f.ValidFrom.UTC().Format(time.RFC3339Nano)
}

// Upsert adds or replaces a document.
func (ix *Index) Upsert(d Doc) {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	ix.upsertLocked(d)
}

func (ix *Index) upsertLocked(d Doc) {
	if i, ok := ix.byKey[d.Key]; ok {
		ix.removeLocked(i)
	}
	toks := Tokenize(d.Text)
	i := len(ix.docs)
	ix.docs = append(ix.docs, d)
	ix.byKey[d.Key] = i
	ix.lengths = append(ix.lengths, len(toks))
	ix.totalLen += len(toks)
	ix.live++
	counts := map[string]int{}
	for _, t := range toks {
		counts[t]++
	}
	for t, c := range counts {
		ix.postings[t] = append(ix.postings[t], posting{doc: i, tf: c})
	}
}

// Remove drops a document by key.
func (ix *Index) Remove(key string) {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	if i, ok := ix.byKey[key]; ok {
		ix.removeLocked(i)
	}
}

// removeLocked tombstones a document. Postings keep pointing at it and are
// skipped at query time. Once tombstones outnumber half the live documents
// (and there are enough to matter) the index compacts itself, so a daemon
// that runs for months does not scan a graveyard on every query.
func (ix *Index) removeLocked(i int) {
	d := ix.docs[i]
	if d.Key == "" {
		return
	}
	delete(ix.byKey, d.Key)
	ix.totalLen -= ix.lengths[i]
	ix.live--
	ix.dead++
	ix.docs[i] = Doc{}
	if ix.dead > 1000 && ix.dead > ix.live/2 {
		ix.compactLocked()
	}
}

// compactLocked rebuilds the arrays from live documents only.
func (ix *Index) compactLocked() {
	liveDocs := make([]Doc, 0, ix.live)
	for _, d := range ix.docs {
		if d.Key != "" {
			liveDocs = append(liveDocs, d)
		}
	}
	ix.docs = nil
	ix.byKey = make(map[string]int, len(liveDocs))
	ix.postings = map[string][]posting{}
	ix.lengths = nil
	ix.totalLen = 0
	ix.live = 0
	ix.dead = 0
	for _, d := range liveDocs {
		ix.upsertLocked(d)
	}
}

// Dead reports the number of tombstoned slots, for tests.
func (ix *Index) Dead() int {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	return ix.dead
}

// Len is the number of live documents.
func (ix *Index) Len() int {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	return ix.live
}

// Search ranks documents for q. kinds filters by Kind (nil = both). asOf
// nil returns current facts only; otherwise facts valid at asOf. Entities
// are always eligible. Returns up to k hits, best first.
func (ix *Index) Search(q string, kinds []string, asOf *time.Time, k int) []Hit {
	toks := Tokenize(q)
	if len(toks) == 0 || k <= 0 {
		return nil
	}
	want := map[string]bool{}
	for _, kd := range kinds {
		want[kd] = true
	}
	phrase := strings.ToLower(strings.TrimSpace(q))

	ix.mu.RLock()
	defer ix.mu.RUnlock()
	if ix.live == 0 {
		return nil
	}
	avg := float64(ix.totalLen) / float64(ix.live)
	scores := map[int]float64{}
	seenTok := map[string]bool{}
	for _, t := range toks {
		if seenTok[t] {
			continue
		}
		seenTok[t] = true
		pl := ix.postings[t]
		df := 0
		for _, p := range pl {
			if ix.docs[p.doc].Key != "" {
				df++
			}
		}
		if df == 0 {
			continue
		}
		idf := math.Log(1 + (float64(ix.live)-float64(df)+0.5)/(float64(df)+0.5))
		weight := 1.0
		if strings.HasPrefix(t, "^") {
			weight = prefixWeight
		}
		for _, p := range pl {
			d := ix.docs[p.doc]
			if d.Key == "" {
				continue
			}
			tf := float64(p.tf)
			norm := tf * (k1 + 1) / (tf + k1*(1-b+b*float64(ix.lengths[p.doc])/avg))
			scores[p.doc] += weight * idf * norm
		}
	}
	hits := make([]Hit, 0, len(scores))
	for i, s := range scores {
		d := ix.docs[i]
		if len(want) > 0 && !want[d.Kind] {
			continue
		}
		if d.Kind == KindFact && !validAt(d, asOf) {
			continue
		}
		if len(phrase) >= 6 && strings.Contains(strings.ToLower(d.Text), phrase) {
			s *= 1.5
		}
		hits = append(hits, Hit{Doc: d, Score: s})
	}
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].Score != hits[j].Score {
			return hits[i].Score > hits[j].Score
		}
		return hits[i].Doc.ValidFrom.After(hits[j].Doc.ValidFrom)
	})
	if len(hits) > k {
		hits = hits[:k]
	}
	return hits
}

func validAt(d Doc, asOf *time.Time) bool {
	if asOf == nil {
		return d.InvalidAt == nil
	}
	return !d.ValidFrom.After(*asOf) && (d.InvalidAt == nil || d.InvalidAt.After(*asOf))
}

var (
	splitRE     = regexp.MustCompile(`[^a-z0-9]+`)
	camelRE     = regexp.MustCompile(`([a-z0-9])([A-Z])`)
	stopwords   = map[string]bool{"the": true, "and": true, "for": true, "with": true, "that": true, "this": true, "from": true, "into": true, "was": true, "are": true, "were": true, "has": true, "have": true, "had": true, "did": true, "does": true, "why": true, "what": true, "when": true, "where": true, "how": true, "who": true, "which": true, "our": true, "you": true, "your": true, "its": true, "not": true, "but": true, "can": true, "will": true, "about": true, "off": true, "we": true, "us": true, "it": true, "is": true, "be": true, "to": true, "of": true, "in": true, "on": true, "at": true, "by": true, "as": true, "or": true, "an": true, "if": true, "so": true, "do": true}
	suffixOrder = []string{"ing", "ies", "ed", "es", "s"}
)

// Tokenize lowercases, drops stopwords and tokens under two characters,
// and applies light stemming so "switched", "switches", and "switch" meet.
// A word with internal structure yields both its whole form and its parts:
// "deepseek-v4-flash" gives deepseek-v4-flash, deepseek, v4, flash, and
// "DeepSeek" gives deepseek, deep, seek — so exact names score highest and
// partial mentions still match.
func Tokenize(s string) []string {
	var out []string
	for _, raw := range strings.Fields(s) {
		raw = strings.Trim(raw, "\"'`()[]{}<>,.;:!?")
		if raw == "" {
			continue
		}
		lower := strings.ToLower(raw)
		parts := splitRE.Split(strings.ToLower(camelRE.ReplaceAllString(raw, "$1 $2")), -1)
		var kept []string
		for _, p := range parts {
			if p != "" {
				kept = append(kept, p)
			}
		}
		if len(kept) > 1 && len(lower) >= 4 {
			if whole := strings.Trim(splitRE.ReplaceAllString(lower, "-"), "-"); whole != "" {
				out = append(out, whole)
			}
		}
		for _, p := range kept {
			if len(p) < 2 || stopwords[p] {
				continue
			}
			st := stem(p)
			out = append(out, st)
			// A 4-character prefix token bridges word forms and compounds
			// that stemming misses ("tailnet"/"Tailscale", "record"/
			// "recording"). Scored at prefixWeight.
			if len(st) >= 6 {
				out = append(out, "^"+st[:4])
			}
		}
	}
	return out
}

// stem strips one common suffix when the stem stays 3+ characters.
func stem(t string) string {
	if len(t) < 4 {
		return t
	}
	for _, suf := range suffixOrder {
		if strings.HasSuffix(t, suf) && len(t)-len(suf) >= 3 {
			s := t[:len(t)-len(suf)]
			if suf == "ies" {
				s += "y"
			}
			return s
		}
	}
	return t
}
