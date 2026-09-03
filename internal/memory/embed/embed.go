// Package embed gives every fact a vector built from the store's own
// words, so a question can reach a fact that shares no word with it.
//
// The house rules allow local embeddings and forbid a hosted embedding
// API, and the spec's deferral of embeddings is what this run exists to
// undo. That rules out a downloaded model as well: nothing here needs a
// model, a network call, or a dependency. It is random indexing, the
// one-pass approximation of latent semantic analysis. Each word gets a
// sparse random vector; a word's meaning is the sum of the vectors of
// the words it appears with; a fact is the sum of its words' meanings.
// Words that keep the same company end up pointing the same way, which
// is what lets "no domain knowledge" answer "cannot fake his way
// through".
package embed

import (
	"encoding/binary"
	"hash/fnv"
	"math"
	"sort"
	"strings"
)

const (
	// Dims is the width of a vector. Random indexing needs enough room
	// for near-orthogonality between tens of thousands of words; 128 is
	// the usual floor and keeps the whole model near 30 MB.
	Dims = 128
	// seeds is how many positions a word's own random vector occupies,
	// half of them positive. Sparse index vectors are what make the
	// method cheap and near-orthogonal.
	seeds = 8
	// maxDocShare drops words that appear in too much of the corpus to
	// carry meaning, the same cut the search index makes.
	maxDocShare = 0.08
	// minTermLen is the shortest word worth a vector.
	minTermLen = 3
	// minCommon is how many documents a word must appear in before the
	// share cut can drop it.
	minCommon = 20
)

// Model holds a context vector per word and a vector per fact.
type Model struct {
	terms  map[string][]float32
	facts  map[string][]float32
	weight map[string]float64
}

// Doc is one document as the model sees it. Every document teaches the
// model what words keep company; only a document marked Index gets a
// vector of its own. Episode summaries are good company and are not
// things recall returns.
type Doc struct {
	Key   string
	Terms []string
	Index bool
}

// Build learns a model from docs. Terms are expected to be already
// tokenised by the caller, so the model and the lexical index agree on
// what a word is.
func Build(docs []Doc) *Model {
	df := map[string]int{}
	for _, d := range docs {
		for _, t := range uniq(d.Terms) {
			df[t]++
		}
	}
	n := float64(len(docs))
	keep := map[string]float64{}
	for t, c := range df {
		// The share cut only means something once there is a corpus: on a
		// handful of documents every word is in a large share of them.
		if len(t) < minTermLen || (c > minCommon && float64(c) > n*maxDocShare) {
			continue
		}
		keep[t] = math.Log(1 + (n-float64(c)+0.5)/(float64(c)+0.5))
	}

	m := &Model{terms: make(map[string][]float32, len(keep)), facts: make(map[string][]float32, len(docs)), weight: keep}
	// A word's meaning accumulates from the company it keeps.
	for _, d := range docs {
		ts := make([]string, 0, len(d.Terms))
		for _, t := range uniq(d.Terms) {
			if _, ok := keep[t]; ok {
				ts = append(ts, t)
			}
		}
		if len(ts) < 2 {
			continue
		}
		sum := make([]float32, Dims)
		for _, t := range ts {
			addIndexVector(sum, t, float32(keep[t]))
		}
		for _, t := range ts {
			v, ok := m.terms[t]
			if !ok {
				v = make([]float32, Dims)
				m.terms[t] = v
			}
			// The company, without the word itself.
			for i := range v {
				v[i] += sum[i]
			}
			addIndexVector(v, t, -float32(keep[t]))
		}
	}
	for t := range m.terms {
		normalize(m.terms[t])
	}
	for _, d := range docs {
		if !d.Index {
			continue
		}
		if v := m.vector(d.Terms, keep); v != nil {
			m.facts[d.Key] = v
		}
	}
	return m
}

// vector embeds a bag of words, weighting each by how rare it is.
func (m *Model) vector(terms []string, weight map[string]float64) []float32 {
	out := make([]float32, Dims)
	any := false
	for _, t := range uniq(terms) {
		v, ok := m.terms[t]
		if !ok {
			continue
		}
		w := float32(1)
		if weight != nil {
			w = float32(weight[t])
		}
		for i := range out {
			out[i] += v[i] * w
		}
		any = true
	}
	if !any {
		return nil
	}
	normalize(out)
	return out
}

// Query embeds a question's words, weighting each by how rare it is —
// the same weighting the facts were embedded with, so a common word in
// the question does not drag the vector toward everything.
func (m *Model) Query(terms []string) []float32 { return m.vector(terms, m.weight) }

// Similarity returns the cosine between a query vector and a fact, in
// [-1, 1], or zero when either is unknown.
func (m *Model) Similarity(q []float32, key string) float64 {
	if q == nil {
		return 0
	}
	v, ok := m.facts[key]
	if !ok {
		return 0
	}
	var dot float32
	for i := range q {
		dot += q[i] * v[i]
	}
	return float64(dot)
}

// Facts reports how many facts the model covers.
func (m *Model) Facts() int { return len(m.facts) }

// Terms reports how many words the model knows.
func (m *Model) Terms() int { return len(m.terms) }

// addIndexVector adds a word's own sparse random vector, scaled. The
// positions come from the word's hash, so nothing has to be stored.
func addIndexVector(dst []float32, term string, scale float32) {
	h := fnv.New64a()
	_, _ = h.Write([]byte(term))
	seed := h.Sum64()
	var b [8]byte
	for i := 0; i < seeds; i++ {
		binary.LittleEndian.PutUint64(b[:], seed+uint64(i)*0x9e3779b97f4a7c15)
		x := fnv.New64a()
		_, _ = x.Write(b[:])
		v := x.Sum64()
		pos := int(v % Dims)
		if v&(1<<40) != 0 {
			dst[pos] += scale
		} else {
			dst[pos] -= scale
		}
	}
}

func normalize(v []float32) {
	var sum float32
	for _, x := range v {
		sum += x * x
	}
	if sum == 0 {
		return
	}
	inv := float32(1 / math.Sqrt(float64(sum)))
	for i := range v {
		v[i] *= inv
	}
}

// uniq returns the distinct words of a bag, lowercased and sorted so a
// build is deterministic.
func uniq(terms []string) []string {
	seen := make(map[string]bool, len(terms))
	out := make([]string, 0, len(terms))
	for _, t := range terms {
		t = strings.TrimPrefix(strings.ToLower(t), "^")
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}
