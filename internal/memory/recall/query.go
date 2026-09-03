package recall

import (
	"encoding/json"
	"os"
	"sort"
	"strings"
	"time"
	"unicode"

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
	candidateFacts = 4000
	maxEntities    = 5
	maxEpisodes    = 3
	summaryChars   = 300
	// Per-field bounds so twenty facts can never exceed the cap on their
	// own: a sentence, a value, and a provenance list are each clipped.
	factChars     = 1000
	valueChars    = 200
	maxProvenance = 5
	queryChars    = 200
	// recencyBoost lifts a fact valid in the last thirty days;
	// episodeBoost a fact produced by an episode whose summary the query
	// matches, scaled by that episode's share of the best score.
	//
	// reasonBoost lifts a fact that carries a reason when the question
	// asks for one.
	recencyBoost   = 0.15
	episodeBoost   = 2.0
	maxEpisodeHits = 8
)

// synonyms expands a query token with the words the facts tend to use
// instead. Small and domain-specific on purpose: a general thesaurus would
// flood the query. Lexical, local, and inspectable.
// A synonym maps one English word to another. It never maps a word to
// the name of an entity: "box" once expanded to mini and halo, and every
// question containing the word came back full of facts about the two
// loudest machines in the graph whatever it had asked. A grader measured
// the table as negatively correlated with success on questions it had
// not been fitted to, and that entry was the mechanism.
var synonyms = map[string][]string{
	"ssh": {"login", "access", "tailscale", "user"}, "login": {"ssh", "user", "access", "credential", "account", "key"},
	"tailnet": {"tailscale"}, "tailscale": {"tailnet"},
	"wired": {"cable", "link", "connected", "10gbe"}, "cable": {"wired", "link"}, "link": {"cable", "wired"},
	"charges": {"cost", "spend", "billing", "api key", "balance"}, "cost": {"price", "spend", "charges", "cheap"},
	"spend": {"cost", "charges", "budget", "credit"}, "billing": {"balance", "cost", "402"},
	"commit": {"push", "merge", "main"}, "lands": {"merge", "push", "deploy"}, "landed": {"merged", "pushed"},
	"provider": {"model", "api", "endpoint"}, "fallback": {"falls back", "fall back"}, "fall": {"fallback"},
	"recording": {"audio", "record", "retain", "persist"}, "recordings": {"audio", "files"}, "voice": {"audio", "speech"},
	"watchdog": {"monitor", "guard", "cron", "kill"}, "runaway": {"watchdog", "limit", "cap"},
	"ingestion": {"sweep", "ingest", "launchd"},
	"process":   {"launchd", "job", "daemon"}, "pipeline": {"chain", "stages", "extraction"},
	"thinking": {"reasoning", "pro"}, "stages": {"chain", "pipeline"},
	"credentials": {"key", "token", "env"}, "secret": {"key", "token", "env"}, "password": {"keyring", "secret"},
	"address": {"ip", "host", "url", "endpoint"}, "port": {"endpoint", "listen"},
	"restart": {"kickstart", "launchctl"}, "schedule": {"cron", "launchd", "interval"},

	// How people ask about the same thing. Memory writes what a session
	// said; a question is asked in the words of whoever is asking, and
	// these are the pairs that kept missing each other.
	"watching":    {"monitors", "monitor", "watch", "checks"},
	"watches":     {"monitors", "monitor", "watch", "checks"},
	"eye":         {"monitors", "watch", "checks", "monitoring"},
	"monitor":     {"watch", "checks", "monitoring", "monitors"},
	"outstanding": {"remaining", "missing", "left", "todo", "pending"},
	"remaining":   {"outstanding", "missing", "left"},
	"holding":     {"blocked", "blocker", "blocking", "long pole", "waiting"},
	"blocker":     {"blocked", "blocking", "holding"},
	"broke":       {"broken", "failed", "failure", "regression", "caused"},
	"broken":      {"broke", "failed", "failure"},
	"wrong":       {"failed", "failure", "error", "bug", "incorrect"},
	"online":      {"live", "deployed", "reachable", "up", "serving"},
	"live":        {"online", "deployed", "production", "serving"},
	"bought":      {"registered", "purchased", "domain", "registrar"},
	"buy":         {"registered", "purchase", "registrar"},
	"allowed":     {"cap", "capped", "limit", "quota", "budget"},
	"budget":      {"cap", "capped", "limit", "quota", "spend"},
	"holds":       {"loaded", "resident", "memory", "capacity"},
	"hold":        {"loaded", "resident", "capacity"},
	"agent":       {"assistant", "worker", "session"},
	"overnight":   {"nightly", "night", "scheduled", "cron"},
	"criteria":    {"bar", "requirements", "acceptance", "definition of done"},
	"acceptance":  {"criteria", "bar", "requirements"},
	"disappeared": {"lost", "dropped", "silently", "missing", "never arrived"},
	"lost":        {"dropped", "disappeared", "missing"},
	"joined":      {"connect", "connected", "network", "wifi", "ssid"},
	"join":        {"connect", "connected", "network", "ssid"},
	"plugged":     {"attached", "connected", "usb", "adapter"},
	"outage":      {"down", "unavailable", "unreachable", "offline"},
	"traffic":     {"requests", "calls", "network", "http"},
	"red":         {"failing", "failed", "broken"},
	"gate":        {"check", "ci", "suite", "pipeline"},
	"primary":     {"first", "default", "main model"},
	"hiring":      {"recruiting", "candidates", "onboarding", "staffing"},
	"trunk":       {"main", "master"},
	"cloud":       {"eas", "hosted", "remote", "ci"},
}

// entityBoost is added to a fact's score for each entity the query names
// that the fact touches. Naming an entity is the strongest signal a
// question gives: the answer is nearly always a fact about the thing
// asked about, and BM25 alone puts the store's loudest fact first
// instead.
// entityBoost is added to a fact's score for each entity the query names
// that the fact touches. It is deliberately small: a bigger prior was
// tried and made retrieval worse on both question sets, because lifting
// every fact about the named thing pushes the one that answers the
// question out of the window along with the rest.
const entityBoost = 0.5

// reasonBoost lifts a fact that carries a reason when the question asks
// for one.
// reasonBoost lifts a fact that carries a reason when the question asks
// for one. Eight measured best on both sets; sixteen starts costing
// questions that are not about reasons.
const reasonBoost = 8.0

// meaningWeight is how much the vector model counts against the word
// match. Lexical scores run to a few tens, so eight is a nudge rather
// than a takeover: it moves a fact that means the right thing up past a
// fact that merely repeats a word, and leaves the rest of the order
// alone. Swept from zero to thirty-two on three question sets; above
// sixteen it starts costing questions that words answer perfectly well.
const meaningWeight = 8.0

// reasonRelations carry why something is the way it is, rather than what
// it is. A question asking why, or what broke, is answered by one of
// these far more often than by the most-mentioned fact about the subject.
var reasonRelations = map[string]bool{
	"causes": true, "caused_by": true, "blocked_by": true, "blocks": true,
	"decided": true, "fixes": true, "prevents": true, "requires": true,
	"lacks": true, "depends_on": true, "replaced_by": true,
}

// whyWords open, or appear in, a question about a reason or a failure.
var whyWords = map[string]bool{
	"why": true, "broke": true, "broken": true, "caused": true, "cause": true,
	"failed": true, "failing": true, "fail": true, "wrong": true, "blocked": true,
	"blocking": true, "holding": true, "happened": true, "decided": true,
	"reason": true, "because": true, "prevented": true, "stopped": true,
}

// asksWhy reports whether a question is after a reason or a failure.
func asksWhy(q string) bool {
	if os.Getenv("SCRY_NO_WHY") != "" {
		return false
	}
	for _, w := range strings.Fields(strings.ToLower(q)) {
		if whyWords[strings.Trim(w, "?.,'\"")] {
			return true
		}
	}
	return false
}

// expand appends synonym words to q so the BM25 pass sees them. The
// original words keep their full weight; synonyms only add candidates.
func expand(q string) string {
	if os.Getenv("SCRY_NO_SYN") != "" {
		return q
	}
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
		// The question in the vector model, used to rank what the words
		// found.
		qv := ix.EmbedQuery(q)
		mw := meaningWeight
		wantsReason := asksWhy(q)
		for _, h := range hits {
			f := h.Doc.Fact
			s := h.Score
			if wantsReason && reasonRelations[f.Relation] {
				s += reasonBoost
			}
			if mw > 0 && qv != nil {
				s += mw * ix.Meaning(qv, search.FactKey(f))
			}
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
		expanded := expand(q)
		var qv []float32
		if ix != nil {
			qv = ix.EmbedQuery(q)
		}
		// Scored the same way as the global candidates, meaning included:
		// a fact on the entity the question names should not be ranked by
		// a different rule than one that arrived through search.
		scoreFacts := func(slug string, w float64) error {
			facts, err := st.FactsAbout(slug, asOf != nil)
			if err != nil {
				return err
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
					s += ix.ScoreDoc(expanded, search.FactKey(f)) + entityBoost*(1+w/4)
					if now.Sub(f.ValidFrom) < 30*24*time.Hour {
						s += recencyBoost
					}
					if qv != nil {
						s += meaningWeight * ix.Meaning(qv, search.FactKey(f))
					}
				}
				h.Score = round3(s)
				scored = append(scored, h)
			}
			return nil
		}
		for slug, w := range named {
			if err := scoreFacts(slug, w); err != nil {
				return res, err
			}
		}
	}
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].Score != scored[j].Score {
			return scored[i].Score > scored[j].Score
		}
		return scored[i].ValidFrom.After(scored[j].ValidFrom)
	})
	scored = diversify(scored)
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

// Diversity. One thing said twenty times by twenty sessions is still one
// thing, and twenty restatements of it fill the answer window that the
// asked-about fact needed. Restatements are demoted below the last
// distinct fact rather than dropped, so a caller who asks for two hundred
// still gets them.
const (
	// maxPerPair is how many facts about the same two entities may hold
	// top slots. A pair with more to say than this says it in different
	// pairs.
	maxPerPair = 2
	// dupOverlap is the share of the shorter fact's words that must also
	// appear in a kept fact for the two to count as the same sentence.
	dupOverlap = 0.8
	// minDupTokens is the shortest fact that similarity may collapse.
	// Below it, only an exact repeat counts.
	minDupTokens = 4
	// diversifyWindow bounds the work: only the top of the ranking
	// competes for the answer window, and the tail is left alone.
	diversifyWindow = 300
	// maxSigs bounds how many kept facts a candidate is compared against.
	maxSigs = 80
)

// diversify demotes near-duplicate facts and third-and-later facts about
// the same pair of entities, preserving relative order otherwise.
func diversify(in []FactHit) []FactHit {
	if len(in) < 3 {
		return in
	}
	n := min(len(in), diversifyWindow)
	kept := make([]FactHit, 0, len(in))
	var demoted []FactHit
	pair := map[string]int{}
	var sigs []sig
	for _, h := range in[:n] {
		key := h.Src + "\x00" + h.Dst
		sig := textSig(h.Fact)
		dup := false
		for _, s := range sigs {
			if sameSentence(sig, s) {
				dup = true
				break
			}
		}
		if dup || pair[key] >= maxPerPair {
			demoted = append(demoted, h)
			continue
		}
		pair[key]++
		if len(sigs) < maxSigs {
			sigs = append(sigs, sig)
		}
		kept = append(kept, h)
	}
	kept = append(kept, in[n:]...)
	return append(kept, demoted...)
}

// sig is a fact reduced for comparison: its content words, and its
// numbers kept apart from them. Two facts about this machine differ in
// an address octet, a port, or a version far more often than in their
// prose, so numbers decide the question before words are weighed.
type sig struct {
	words map[string]bool
	nums  map[string]bool
}

// textSig splits a fact into content words and numbers, lowercased, with
// punctuation and one- and two-letter words dropped.
func textSig(s string) sig {
	out := sig{words: map[string]bool{}, nums: map[string]bool{}}
	for _, w := range strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}) {
		if isNumber(w) {
			out.nums[w] = true
			continue
		}
		if len(w) > 2 {
			out.words[w] = true
		}
	}
	return out
}

// isNumber reports whether a token is all digits.
func isNumber(w string) bool {
	if w == "" {
		return false
	}
	for _, r := range w {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// sameNums reports whether two facts carry the same numbers.
func sameNums(a, b sig) bool {
	if len(a.nums) != len(b.nums) {
		return false
	}
	for n := range a.nums {
		if !b.nums[n] {
			return false
		}
	}
	return true
}

// sameSentence reports whether two facts say the same thing: the same
// numbers, and most of the shorter one's words appearing in the longer.
func sameSentence(x, y sig) bool {
	if !sameNums(x, y) {
		return false
	}
	a, b := x.words, y.words
	if len(a) == 0 || len(b) == 0 {
		return false
	}
	shorter, longer := a, b
	if len(b) < len(a) {
		shorter, longer = b, a
	}
	hit := 0
	for w := range shorter {
		if longer[w] {
			hit++
		}
	}
	if len(shorter) < minDupTokens {
		return hit == len(shorter) && len(longer) == len(shorter)
	}
	return float64(hit)/float64(len(shorter)) >= dupOverlap
}
