package recall

import (
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/jeffdhooton/scry/internal/memory/search"
	"github.com/jeffdhooton/scry/internal/memory/store"
)

// Recall v2 answers a question with the handful of facts that answer it.
// The entry point is lexical (BM25 over fact text and entity names, see
// internal/memory/search); facts are ranked, not entities; a fact that
// touches an entity the query names is boosted; and the payload is capped
// so an MCP host never truncates an arbitrary slice of it.

const (
	// DefaultFactLimit is how many facts a recall returns unless asked
	// otherwise. Twenty is what an agent can read; the old behaviour
	// returned thousands.
	DefaultFactLimit = 20
	// MaxPayloadBytes bounds the serialized Result. Codex and Claude both
	// truncate tool output somewhere above this; below it every byte is
	// seen.
	MaxPayloadBytes = 24 * 1024
	// candidateFacts is how many BM25 hits are re-ranked.
	candidateFacts = 200
	maxEntities    = 5
	maxEpisodes    = 3
	summaryChars   = 300
	// entityBoost is added to a fact's score per query-named entity it
	// touches; recencyBoost to a fact valid in the last thirty days.
	entityBoost  = 0.5
	recencyBoost = 0.15
)

// Result is a recall answer.
type Result struct {
	Query     string        `json:"query"`
	Entities  []EntityHead  `json:"entities"`
	Facts     []FactHit     `json:"facts"`
	Episodes  []EpisodeHead `json:"episodes,omitempty"`
	Truncated bool          `json:"truncated,omitempty"`
	// Total is how many facts matched before the limit and the cap.
	Total int `json:"total_matches"`
}

// EntityHead is an entity mentioned by the returned facts or named by the
// query, without its fact list: the facts below are the facts.
type EntityHead struct {
	Slug        string  `json:"slug"`
	Name        string  `json:"name"`
	Type        string  `json:"type"`
	Description string  `json:"description,omitempty"`
	FactCount   int     `json:"fact_count"`
	Score       float64 `json:"score"`
}

// FactHit is one ranked fact.
type FactHit struct {
	Src        string     `json:"src"`
	Relation   string     `json:"relation"`
	Dst        string     `json:"dst,omitempty"`
	Value      string     `json:"value,omitempty"`
	Fact       string     `json:"fact"`
	ValidFrom  time.Time  `json:"valid_from"`
	InvalidAt  *time.Time `json:"invalid_at,omitempty"`
	Confidence float64    `json:"confidence"`
	Episodes   []string   `json:"episodes,omitempty"`
	Score      float64    `json:"score"`
}

// EpisodeHead is a provenance pointer with a short summary.
type EpisodeHead struct {
	ID         string    `json:"id"`
	Source     string    `json:"source"`
	OccurredAt time.Time `json:"occurred_at"`
	Summary    string    `json:"summary"`
}

// Recall runs the v2 query. ix may be nil, in which case it falls back to
// the entity-substring path with the same output shape and cap.
func Recall(st *store.Store, ix *search.Index, q string, asOf *time.Time, limit int) (Result, error) {
	if limit <= 0 {
		limit = DefaultFactLimit
	}
	res := Result{Query: q, Entities: []EntityHead{}, Facts: []FactHit{}}

	// Entities the query names directly: exact or alias match, or a BM25
	// entity hit. Their facts get the boost.
	named := map[string]float64{}
	if slug, ok, _ := st.ResolveAlias(q); ok {
		named[slug] = 2
	}
	if ix != nil {
		for _, h := range ix.Search(q, []string{search.KindEntity}, nil, maxEntities) {
			if len(h.Doc.Slugs) > 0 {
				if named[h.Doc.Slugs[0]] < h.Score {
					named[h.Doc.Slugs[0]] = h.Score
				}
			}
		}
	}
	for _, tok := range strings.Fields(q) {
		if slug, ok, _ := st.ResolveAlias(tok); ok {
			if named[slug] < 1 {
				named[slug] = 1
			}
		}
	}

	var scored []FactHit
	if ix != nil {
		hits := ix.Search(q, []string{search.KindFact}, asOf, candidateFacts)
		now := time.Now()
		for _, h := range hits {
			f := h.Doc.Fact
			s := h.Score
			for _, slug := range h.Doc.Slugs {
				if w, ok := named[slug]; ok {
					s += entityBoost * (1 + w/4)
				}
			}
			if now.Sub(f.ValidFrom) < 30*24*time.Hour {
				s += recencyBoost
			}
			scored = append(scored, toHit(f, s))
		}
	}
	// Facts on named entities that BM25 missed (the question used none of
	// the fact's words) still belong: add them at a low score so a query
	// that is only an entity name behaves like the old recall, ranked.
	if len(named) > 0 {
		seen := map[string]bool{}
		for _, h := range scored {
			seen[hitKey(h)] = true
		}
		for slug, w := range named {
			facts, err := st.FactsAbout(slug, asOf != nil)
			if err != nil {
				return res, err
			}
			facts = filterAsOf(facts, asOf)
			for _, f := range facts {
				h := toHit(f, 0.1*w)
				if seen[hitKey(h)] {
					continue
				}
				seen[hitKey(h)] = true
				scored = append(scored, h)
			}
		}
	}
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].Score != scored[j].Score {
			return scored[i].Score > scored[j].Score
		}
		return scored[i].ValidFrom.After(scored[j].ValidFrom)
	})
	res.Total = len(scored)
	if len(scored) > limit {
		scored = scored[:limit]
		res.Truncated = true
	}
	res.Facts = scored

	// Entity headers: query-named entities plus the ones the returned
	// facts touch most, up to maxEntities.
	entScore := map[string]float64{}
	for slug, w := range named {
		entScore[slug] += w
	}
	for _, h := range res.Facts {
		entScore[h.Src] += h.Score
		if h.Dst != "" {
			entScore[h.Dst] += h.Score * 0.5
		}
	}
	type es struct {
		slug  string
		score float64
	}
	var ranked []es
	for slug, s := range entScore {
		ranked = append(ranked, es{slug, s})
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].score != ranked[j].score {
			return ranked[i].score > ranked[j].score
		}
		return ranked[i].slug < ranked[j].slug
	})
	for _, r := range ranked {
		if len(res.Entities) >= maxEntities {
			break
		}
		e, err := st.GetEntity(r.slug)
		if err != nil {
			continue
		}
		facts, _ := st.FactsAbout(r.slug, false)
		res.Entities = append(res.Entities, EntityHead{
			Slug: e.Slug, Name: e.Name, Type: e.Type, Description: clip(e.Description, summaryChars),
			FactCount: len(facts), Score: r.score,
		})
	}

	// Episodes: the most recent provenance of the top facts.
	var provenance []store.Fact
	for i, h := range res.Facts {
		if i >= 5 {
			break
		}
		provenance = append(provenance, store.Fact{Episodes: h.Episodes})
	}
	eps, err := collectEpisodes(st, provenance, maxEpisodes)
	if err != nil {
		return res, err
	}
	for _, e := range eps {
		res.Episodes = append(res.Episodes, EpisodeHead{ID: e.ID, Source: e.Source, OccurredAt: e.OccurredAt, Summary: clip(e.Summary, summaryChars)})
	}

	res.Facts = capPayload(&res)
	return res, nil
}

// capPayload trims the result until its JSON fits MaxPayloadBytes: first
// episodes, then facts from the tail.
func capPayload(res *Result) []FactHit {
	for {
		b, err := json.Marshal(res)
		if err != nil || len(b) <= MaxPayloadBytes {
			return res.Facts
		}
		switch {
		case len(res.Episodes) > 0:
			res.Episodes = nil
		case len(res.Facts) > 1:
			res.Facts = res.Facts[:len(res.Facts)-1]
			res.Truncated = true
		case len(res.Entities) > 1:
			res.Entities = res.Entities[:len(res.Entities)-1]
		default:
			if len(res.Facts) == 1 && len(res.Facts[0].Fact) > 512 {
				res.Facts[0].Fact = clip(res.Facts[0].Fact, 512)
				res.Truncated = true
				continue
			}
			return res.Facts
		}
	}
}

func toHit(f store.Fact, score float64) FactHit {
	return FactHit{Src: f.Src, Relation: f.Relation, Dst: f.Dst, Value: f.Value, Fact: f.Fact,
		ValidFrom: f.ValidFrom, InvalidAt: f.InvalidAt, Confidence: f.Confidence, Episodes: f.Episodes, Score: round3(score)}
}

func hitKey(h FactHit) string {
	return h.Src + "|" + h.Relation + "|" + h.Dst + "|" + h.Value + "|" + h.ValidFrom.UTC().Format(time.RFC3339Nano)
}

func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

func round3(f float64) float64 { return float64(int(f*1000+0.5)) / 1000 }
