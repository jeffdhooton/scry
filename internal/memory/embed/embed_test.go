package embed

import "testing"

// Words that keep the same company must end up pointing the same way.
// That is the whole claim of the method, and the reason a question can
// reach a fact that shares no word with it.
func TestWordsInTheSameCompanyPointTheSameWay(t *testing.T) {
	docs := []Doc{
		{Key: "1", Index: true, Terms: []string{"jeff", "monitors", "cellsaviors", "hermeswatch", "ssh"}},
		{Key: "2", Index: true, Terms: []string{"jeff", "watches", "cellsaviors", "hermeswatch", "ssh"}},
		{Key: "3", Index: true, Terms: []string{"hermeswatch", "monitors", "uptime", "cellsaviors"}},
		{Key: "4", Index: true, Terms: []string{"hermeswatch", "watches", "uptime", "cellsaviors"}},
		{Key: "5", Index: true, Terms: []string{"forge", "deploys", "laravel", "staging", "production"}},
		{Key: "6", Index: true, Terms: []string{"forge", "ships", "laravel", "staging", "production"}},
		{Key: "7", Index: true, Terms: []string{"deploys", "laravel", "production", "release"}},
		{Key: "8", Index: true, Terms: []string{"ships", "laravel", "production", "release"}},
	}
	m := Build(docs)
	if m.Terms() == 0 || m.Facts() == 0 {
		t.Fatalf("model is empty: %d terms, %d facts", m.Terms(), m.Facts())
	}

	// "watches" never appears with "monitors", but both appear with
	// hermeswatch, cellsaviors and uptime.
	q := m.Query([]string{"watches", "cellsaviors"})
	near := m.Similarity(q, "3") // the monitors fact
	far := m.Similarity(q, "5")  // an unrelated deploy fact
	if near <= far {
		t.Errorf("a question about watching must reach the monitoring fact: %.3f vs %.3f", near, far)
	}

	// A question in the other vocabulary reaches the other cluster.
	q2 := m.Query([]string{"ships", "laravel"})
	if m.Similarity(q2, "7") <= m.Similarity(q2, "1") {
		t.Errorf("a deploy question must reach a deploy fact: %.3f vs %.3f",
			m.Similarity(q2, "7"), m.Similarity(q2, "1"))
	}
}

func TestUnknownQueryAndFactScoreZero(t *testing.T) {
	m := Build([]Doc{{Key: "1", Index: true, Terms: []string{"alpha", "beta", "gamma"}}})
	if got := m.Similarity(m.Query([]string{"nothing", "here"}), "1"); got != 0 {
		t.Errorf("an unknown question scores %v, want 0", got)
	}
	if got := m.Similarity(m.Query([]string{"alpha"}), "missing"); got != 0 {
		t.Errorf("an unknown fact scores %v, want 0", got)
	}
}

func TestBuildIsDeterministic(t *testing.T) {
	docs := []Doc{
		{Key: "1", Index: true, Terms: []string{"one", "two", "three"}},
		{Key: "2", Index: true, Terms: []string{"two", "three", "four"}},
	}
	a, b := Build(docs), Build(docs)
	q := []string{"two"}
	if a.Similarity(a.Query(q), "1") != b.Similarity(b.Query(q), "1") {
		t.Error("two builds of the same corpus must agree")
	}
}
