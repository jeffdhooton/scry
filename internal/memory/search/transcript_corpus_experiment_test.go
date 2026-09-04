package search

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/jeffdhooton/scry/internal/memory/embed"
	"github.com/jeffdhooton/scry/internal/memory/store"
)

// TestTranscriptCorpusExperiment answers the question the retention decision
// turns on, without changing retention: if the vector model had the machine's
// transcripts to learn from, would it retrieve the facts the lexical index
// misses?
//
// Offline, throwaway, nothing written. Runs only when SCRY_REPLICA and
// SCRY_QUESTIONS are set.
func TestTranscriptCorpusExperiment(t *testing.T) {
	replica := os.Getenv("SCRY_REPLICA")
	qfile := os.Getenv("SCRY_QUESTIONS")
	if replica == "" || qfile == "" {
		t.Skip("set SCRY_REPLICA and SCRY_QUESTIONS")
	}
	st, err := store.Open(replica)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	facts, err := st.AllFacts()
	if err != nil {
		t.Fatal(err)
	}
	ents, err := st.Entities()
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]string{}
	for _, e := range ents {
		names[e.Slug] = e.Name
	}

	// Only current facts, matching what recall may return.
	var cur []store.Fact
	for _, f := range facts {
		if f.InvalidAt == nil {
			cur = append(cur, f)
		}
	}
	t.Logf("current facts: %d", len(cur))

	factDocs := make([]embed.Doc, 0, len(cur))
	text := make([]string, len(cur))
	for i, f := range cur {
		d := FactDoc(f, names)
		text[i] = d.Text
		factDocs = append(factDocs, embed.Doc{Key: key(i), Terms: Tokenize(d.Text), Index: true})
	}

	base := embed.Build(factDocs)
	t.Logf("baseline model: %d terms, %d vectors", base.Terms(), base.Facts())

	// The transcripts already on disk, as company only — never indexed, so
	// nothing new becomes retrievable, only better understood.
	tdocs := transcriptDocs(t, envInt("SCRY_MAX_FILES", 600), envInt("SCRY_MAX_MB", 400))
	t.Logf("transcript docs: %d", len(tdocs))
	aug := embed.Build(append(append([]embed.Doc{}, factDocs...), tdocs...))
	t.Logf("augmented model: %d terms, %d vectors", aug.Terms(), aug.Facts())

	qs := loadQuestions(t, qfile)
	var baseRanks, augRanks []int
	scored := 0
	for _, q := range qs {
		want := -1
		for i, tx := range text {
			if matchesAll(tx, q.terms) {
				want = i
				break
			}
		}
		if want < 0 {
			continue
		}
		scored++
		baseRanks = append(baseRanks, rankOf(base, Tokenize(q.question), want, len(cur)))
		augRanks = append(augRanks, rankOf(aug, Tokenize(q.question), want, len(cur)))
	}
	t.Logf("questions with a findable answer: %d", scored)
	for _, n := range []int{20, 50, 100, 500, 2000} {
		t.Logf("  answer in vector top-%-5d  baseline %3d   with transcripts %3d",
			n, within(baseRanks, n), within(augRanks, n))
	}
}

func key(i int) string { return "f" + itoa(i) }

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b []byte
	for i > 0 {
		b = append([]byte{byte('0' + i%10)}, b...)
		i /= 10
	}
	return string(b)
}

func envInt(name string, def int) int {
	v := os.Getenv(name)
	if v == "" {
		return def
	}
	n := 0
	for _, c := range v {
		if c < '0' || c > '9' {
			return def
		}
		n = n*10 + int(c-'0')
	}
	return n
}

func rankOf(m *embed.Model, qterms []string, want, n int) int {
	q := m.Query(qterms)
	if len(q) == 0 {
		return n
	}
	target := m.Similarity(q, key(want))
	better := 0
	for i := 0; i < n; i++ {
		if i == want {
			continue
		}
		if m.Similarity(q, key(i)) > target {
			better++
		}
	}
	return better + 1
}

func within(ranks []int, n int) int {
	c := 0
	for _, r := range ranks {
		if r <= n {
			c++
		}
	}
	return c
}

type bq struct {
	question string
	terms    []string
}

func loadQuestions(t *testing.T, path string) []bq {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var raw []struct {
		Question string `json:"question"`
		Expect   struct {
			AllOf []string `json:"all_of"`
		} `json:"expect"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatal(err)
	}
	out := make([]bq, 0, len(raw))
	for _, r := range raw {
		if len(r.Expect.AllOf) == 0 {
			continue
		}
		out = append(out, bq{question: r.Question, terms: r.Expect.AllOf})
	}
	return out
}

func matchesAll(text string, terms []string) bool {
	low := strings.ToLower(text)
	for _, t := range terms {
		if !strings.Contains(low, strings.ToLower(t)) {
			return false
		}
	}
	return true
}

// chunk splits s into pieces of about n characters, on word boundaries.
func chunk(s string, n int) []string {
	words := strings.Fields(s)
	var out []string
	var b strings.Builder
	for _, w := range words {
		b.WriteString(w)
		b.WriteByte(' ')
		if b.Len() >= n {
			out = append(out, b.String())
			b.Reset()
		}
	}
	if b.Len() > 40 {
		out = append(out, b.String())
	}
	return out
}

var textRE = regexp.MustCompile(`"text"\s*:\s*"((?:[^"\\]|\\.){20,})"`)

// transcriptDocs reads the machine's own transcripts as plain word company.
func transcriptDocs(t *testing.T, maxFiles, maxMB int) []embed.Doc {
	t.Helper()
	home, _ := os.UserHomeDir()
	var files []string
	for _, root := range []string{
		filepath.Join(home, ".claude", "projects"),
		filepath.Join(home, ".codex", "sessions"),
	} {
		_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(p, ".jsonl") {
				return nil
			}
			files = append(files, p)
			return nil
		})
	}
	sort.Strings(files)
	if len(files) > maxFiles {
		// Spread across the whole set rather than taking the oldest.
		step := len(files) / maxFiles
		var pick []string
		for i := 0; i < len(files); i += step {
			pick = append(pick, files[i])
		}
		files = pick
	}
	budget := int64(maxMB) * 1024 * 1024
	var docs []embed.Doc
	var used int64
	for _, p := range files {
		if used > budget {
			break
		}
		f, err := os.Open(p)
		if err != nil {
			continue
		}
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 1<<20), 8<<20)
		var sb strings.Builder
		for sc.Scan() {
			for _, m := range textRE.FindAllStringSubmatch(sc.Text(), -1) {
				sb.WriteString(m[1])
				sb.WriteByte(' ')
			}
			if sb.Len() > 3_000_000 {
				break
			}
		}
		f.Close()
		if sb.Len() < 500 {
			continue
		}
		used += int64(sb.Len())
		// Chunked to roughly the size of a fact so document frequency
		// stays comparable. One 3 MB document per file would give nearly
		// every term a document hit and flatten the weights, which is a
		// property of the packaging rather than of the text.
		for _, c := range chunk(sb.String(), envInt("SCRY_CHUNK", 400)) {
			docs = append(docs, embed.Doc{Terms: Tokenize(c)})
		}
	}
	t.Logf("transcript bytes read: %.1f MB from %d files", float64(used)/1e6, len(docs))
	return docs
}
