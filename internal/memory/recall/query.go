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
	candidateFacts = 1000
	maxEntities    = 5
	maxEpisodes    = 3
	summaryChars   = 300
	// Per-field bounds so twenty facts can never exceed the cap on their
	// own: a sentence, a value, and a provenance list are each clipped.
	factChars     = 1000
	valueChars    = 200
	maxProvenance = 5
	queryChars    = 200
	// entityBoost is added to a fact's score per query-named entity it
	// touches; recencyBoost to a fact valid in the last thirty days;
	// episodeBoost to a fact produced by an episode whose summary the
	// query matches, scaled by that episode's share of the best score.
	entityBoost    = 0.5
	recencyBoost   = 0.15
	episodeBoost   = 2.0
	maxEpisodeHits = 8
)

// synonyms expands a query token with the words the facts tend to use
// instead. Small and domain-specific on purpose: a general thesaurus would
// flood the query. Lexical, local, and inspectable.
var synonyms = map[string][]string{
	"ssh": {"login", "access", "tailscale", "user"}, "login": {"ssh", "user", "access"},
	"tailnet": {"tailscale"}, "tailscale": {"tailnet"},
	"box": {"machine", "host", "server", "mini", "halo"}, "boxes": {"machines", "hosts", "halo", "halo2"},
	"machine": {"box", "host", "mini"}, "host": {"machine", "box", "server"},
	"wired": {"cable", "link", "connected", "10gbe"}, "cable": {"wired", "link"}, "link": {"cable", "wired"},
	"charges": {"cost", "spend", "billing", "api key", "balance"}, "cost": {"price", "spend", "charges", "cheap"},
	"spend": {"cost", "charges", "budget", "credit"}, "billing": {"balance", "cost", "402"},
	"container": {"docker", "compose"}, "containers": {"docker", "compose"}, "docker": {"container", "compose"},
	"natively": {"launchd", "brew", "docker"},
	"commit":   {"push", "merge", "main"}, "lands": {"merge", "push", "deploy"}, "landed": {"merged", "pushed"},
	"provider": {"model", "api", "endpoint"}, "fallback": {"falls back", "fall back"}, "fall": {"fallback"},
	"database": {"postgres", "mysql", "sqlite", "db"}, "databases": {"postgres", "mysql", "db"},
	"object store": {"minio", "s3", "r2"}, "storage": {"store", "disk", "bucket"},
	"recording": {"audio", "record", "retain", "persist"}, "recordings": {"audio", "files"}, "voice": {"audio", "speech"},
	"watchdog": {"monitor", "guard", "cron", "kill"}, "runaway": {"watchdog", "limit", "cap"},
	"ingestion": {"sweep", "ingest", "launchd"}, "drives": {"runs", "sweep", "launchd", "cron"},
	"process": {"launchd", "job", "daemon"}, "pipeline": {"chain", "stages", "extraction"},
	"cheap": {"flash", "cost"}, "thinking": {"reasoning", "pro"}, "stages": {"chain", "pipeline"},
	"environments": {"sites", "forge", "deploys", "staging", "production"}, "web": {"site", "forge"},
	"credentials": {"key", "token", "env"}, "secret": {"key", "token", "env"}, "password": {"keyring", "secret"},
	"address": {"ip", "host", "url", "endpoint"}, "port": {"endpoint", "listen"},
	"restart": {"kickstart", "launchctl"}, "schedule": {"cron", "launchd", "interval"},
	"gpu": {"halo", "inference", "vram"}, "inference": {"model", "halo", "llm"},
	"phone": {"mobile", "ios", "android", "expo"}, "mobile": {"phone", "expo", "app"},
	"email": {"gmail", "gog", "mail"}, "calendar": {"gog", "gcal"},
}

// expand appends synonym words to q so the BM25 pass sees them. The
// original words keep their full weight; synonyms only add candidates.
func expand(q string) string {
	lower := strings.ToLower(q)
	var extra []string
	seen := map[string]bool{}
	for k, vs := range synonyms {
		if !containsWord(lower, k) {
			continue
		}
		for _, v := range vs {
			if !seen[v] {
				seen[v] = true
				extra = append(extra, v)
			}
		}
	}
	if len(extra) == 0 {
		return q
	}
	sort.Strings(extra)
	return q + " " + strings.Join(extra, " ")
}

func containsWord(text, w string) bool {
	i := strings.Index(text, w)
	for i >= 0 {
		before := i == 0 || !isWordChar(text[i-1])
		after := i+len(w) >= len(text) || !isWordChar(text[i+len(w)])
		if before && after {
			return true
		}
		next := strings.Index(text[i+1:], w)
		if next < 0 {
			return false
		}
		i += 1 + next
	}
	return false
}

func isWordChar(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_'
}

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
	// Episodes holds up to maxProvenance ids; EpisodeCount is the full
	// number, since a long-lived fact can be restated by hundreds.
	Episodes     []string `json:"episodes,omitempty"`
	EpisodeCount int      `json:"episode_count,omitempty"`
	Score        float64  `json:"score"`
}

// EpisodeHead is a provenance pointer with a short summary.
type EpisodeHead struct {
	ID         string    `json:"id"`
	Source     string    `json:"source"`
	OccurredAt time.Time `json:"occurred_at"`
	Summary    string    `json:"summary"`
}

// Recall runs the v2 query. ix may be nil (the index failed to build), in
// which case only facts on entities the query names directly are returned,
// with the same output shape and cap.
func Recall(st *store.Store, ix *search.Index, q string, asOf *time.Time, limit int) (Result, error) {
	if limit <= 0 {
		limit = DefaultFactLimit
	}
	res := Result{Query: clip(q, queryChars), Entities: []EntityHead{}, Facts: []FactHit{}}

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
		expanded := expand(q)
		// Episodes whose summary matches: their facts get a boost scaled by
		// the episode's share of the best episode score.
		epScore := map[string]float64{}
		best := 0.0
		for _, h := range ix.Search(expanded, []string{search.KindEpisode}, nil, maxEpisodeHits) {
			id := strings.TrimPrefix(h.Doc.Key, "ep:")
			epScore[id] = h.Score
			if h.Score > best {
				best = h.Score
			}
		}
		hits := ix.Search(expanded, []string{search.KindFact}, asOf, candidateFacts)
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
			if best > 0 {
				top := 0.0
				for _, id := range f.Episodes {
					if es := epScore[id]; es > top {
						top = es
					}
				}
				s += episodeBoost * top / best
			}
			scored = append(scored, toHit(f, s))
		}
	}
	// Facts on entities the query names, scored the same way as the global
	// candidates. An entity with two hundred facts pushes its own best
	// match out of the global window, so without this the answer to "how
	// do I ssh into the mini" loses to two hundred other facts about the
	// mini.
	if len(named) > 0 {
		seen := map[string]bool{}
		for _, h := range scored {
			seen[hitKey(h)] = true
		}
		now := time.Now()
		for slug, w := range named {
			facts, err := st.FactsAbout(slug, asOf != nil)
			if err != nil {
				return res, err
			}
			facts = filterAsOf(facts, asOf)
			for _, f := range facts {
				h := toHit(f, 0)
				if seen[hitKey(h)] {
					continue
				}
				seen[hitKey(h)] = true
				s := 0.1 * w
				if ix != nil {
					s += ix.ScoreDoc(expand(q), search.FactKey(f)) + entityBoost*(1+w/4)
					if now.Sub(f.ValidFrom) < 30*24*time.Hour {
						s += recencyBoost
					}
				}
				h.Score = round3(s)
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
			if len(res.Facts) == 1 && len(res.Facts[0].Fact) > 256 {
				res.Facts[0].Fact = clip(res.Facts[0].Fact, 256)
				res.Facts[0].Episodes = nil
				res.Truncated = true
				continue
			}
			return res.Facts
		}
	}
}

func toHit(f store.Fact, score float64) FactHit {
	eps := f.Episodes
	if len(eps) > maxProvenance {
		eps = eps[:maxProvenance]
	}
	return FactHit{Src: f.Src, Relation: f.Relation, Dst: f.Dst, Value: clip(f.Value, valueChars), Fact: clip(f.Fact, factChars),
		ValidFrom: f.ValidFrom, InvalidAt: f.InvalidAt, Confidence: f.Confidence, Episodes: eps, EpisodeCount: len(f.Episodes), Score: round3(score)}
}

func hitKey(h FactHit) string {
	return h.Src + "|" + h.Relation + "|" + h.Dst + "|" + h.Value + "|" + h.ValidFrom.UTC().Format(time.RFC3339Nano)
}

// clip bounds s to n runes, never splitting a multi-byte character.
func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

func round3(f float64) float64 { return float64(int(f*1000+0.5)) / 1000 }
