package recall

import "testing"

func hit(src, dst, text string, score float64) FactHit {
	return FactHit{Src: src, Dst: dst, Fact: text, Score: score}
}

func TestDiversifyDemotesRestatementsOfOneThing(t *testing.T) {
	// The shape that lost an answer: one load test restated by six
	// sessions filled the window before the asked-about fact.
	in := []FactHit{
		hit("laptop", "queue", "Twenty consecutive remembers were submitted from the laptop MCP host during the load test", 9),
		hit("laptop", "queue", "Issued twenty consecutive remembers to the scry memory queue during the load test", 8),
		hit("laptop", "queue", "The laptop MCP host sent twenty consecutive remember calls to the scry memory queue during the load test", 7),
		hit("laptop", "queue", "Twenty consecutive remembers were issued from the laptop MCP host to the scry memory queue during the load test", 6),
		hit("laptop", "mini", "The laptop reaches the store through a launchd-managed SSH StreamLocalForward to the mini", 5),
	}
	got := diversify(in)
	if len(got) != len(in) {
		t.Fatalf("diversify dropped facts: %d, want %d", len(got), len(in))
	}
	// Two restatements keep their slots; the pair cap promotes the fact
	// that answers a different question into the third.
	want := "The laptop reaches the store through a launchd-managed SSH StreamLocalForward to the mini"
	if got[2].Fact != want {
		t.Errorf("third slot = %q, want the distinct fact", got[2].Fact)
	}
	if got[len(got)-1].Score >= got[0].Score {
		t.Errorf("restatements must be demoted below the distinct facts")
	}
}

func TestDiversifyKeepsDistinctFactsAboutOnePair(t *testing.T) {
	in := []FactHit{
		hit("mini", "tailscale", "The mini is reached at jclaw@100.96.45.73 over the tailnet", 9),
		hit("mini", "tailscale", "Tailscale SSH is unavailable on the mini because no ACL grants are configured", 8),
		hit("mini", "tailscale", "MagicDNS name resolution is not working, so aliases use the raw address", 7),
		hit("mini", "scry", "The mini holds the authoritative memory store", 6),
	}
	got := diversify(in)
	if got[0].Fact != in[0].Fact || got[1].Fact != in[1].Fact {
		t.Errorf("the first two facts about a pair must keep their slots")
	}
	if got[2].Src != "mini" || got[2].Dst != "scry" {
		t.Errorf("slot 3 = %+v, want the other pair's fact promoted", got[2])
	}
	if got[3].Fact != in[2].Fact {
		t.Errorf("the third fact about a pair is demoted, not dropped: %+v", got[3])
	}
}

func TestSameSentenceIsNotFooledByShortFacts(t *testing.T) {
	cases := []struct {
		a, b string
		same bool
	}{
		{"Hermes gateway falls back to hosted DeepSeek", "The Hermes gateway falls back to hosted DeepSeek.", true},
		{"Scry uses BadgerDB", "Scry uses BadgerDB", true},
		{"Scry uses BadgerDB", "Scry uses SQLite", false},
		{"The mini runs the daemon", "The laptop runs the daemon", false},
		{"Extraction runs on GLM", "Extraction runs on GLM first, then DeepSeek, and never on a third party", false},
		// Two facts whose prose is identical and whose address is not.
		{"halo enp191s0 has static IP 192.168.100.1/30 with MTU 9000", "halo2 enp191s0 has carrier and static IP 192.168.100.2/30 with MTU 9000", false},
		{"The daemon listens on port 45679", "The daemon listens on port 7279", false},
		{"The daemon listens on port 45679", "The daemon listens on port 45679.", true},
	}
	for _, c := range cases {
		if got := sameSentence(textSig(c.a), textSig(c.b)); got != c.same {
			t.Errorf("sameSentence(%q, %q) = %v, want %v", c.a, c.b, got, c.same)
		}
	}
}
