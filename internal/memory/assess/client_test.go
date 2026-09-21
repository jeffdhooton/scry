package assess

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const validResponse = `{"model":"jev-1.13.0","answers":{"supported":{"type":"noul","noul":0.91},"durable":{"type":"noul","noul":0.8},"assertion":{"type":"choice","choice":"established","probabilities":{"established":0.9,"planned":0.04,"hypothetical":0.02,"denied":0.01,"unclear":0.03},"confidence":0.76}},"usage":{"input_tokens":1000,"output_tokens":30}}`

// Removing redaction, leaking labels, or sending an incompatible wire shape must fail.
func TestClientSendsOnlyRedactedEvidenceAndQuestions(t *testing.T) {
	secret := "sk-abcdefghijklmnopqrstuvwxyz"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/v1/systemone" || r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("bad HTTP contract: %s %s", r.Method, r.URL.Path)
		}
		var body map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		raw, _ := json.Marshal(body)
		if strings.Contains(string(raw), secret) || strings.Contains(string(raw), "expected") || len(body) != 3 {
			t.Errorf("unexpected data sent: %s", raw)
		}
		var state map[string]string
		json.Unmarshal(body["state"], &state)
		if state["episode"] != "User: use [REDACTED]" || state["fact"] != "The credential is [REDACTED]" {
			t.Errorf("state = %v", state)
		}
		var questions map[string]Question
		json.Unmarshal(body["questions"], &questions)
		if len(questions) != 3 || questions["supported"].Type != "noul" || questions["assertion"].Type != "choice" {
			t.Errorf("questions = %v", questions)
		}
		fmt.Fprint(w, validResponse)
	}))
	defer srv.Close()
	c, err := NewClient("test-key")
	if err != nil {
		t.Fatal(err)
	}
	c.endpoint = srv.URL + "/v1/systemone"
	got, err := c.Assess(context.Background(), "User: use "+secret, "The credential is "+secret)
	if err != nil {
		t.Fatal(err)
	}
	if got.Model != "jev-1.13.0" || *got.Answers["supported"].Noul != .91 || got.Usage.InputTokens != 1000 {
		t.Fatalf("result = %+v", got)
	}
}

// Missing and invalid probabilities must never become confident zero values.
func TestClientRejectsMalformedAnswers(t *testing.T) {
	for name, body := range map[string]string{
		"missing probability":     strings.Replace(validResponse, `,"noul":0.91`, "", 1),
		"null probability":        strings.Replace(validResponse, `"noul":0.91`, `"noul":null`, 1),
		"range":                   strings.Replace(validResponse, `"noul":0.91`, `"noul":1.1`, 1),
		"wrong answer type":       strings.Replace(validResponse, `"type":"noul"`, `"type":"score"`, 1),
		"unknown choice":          strings.Replace(validResponse, `"choice":"established"`, `"choice":"complete"`, 1),
		"missing choice option":   strings.Replace(validResponse, `,"unclear":0.03`, "", 1),
		"null choice probability": strings.Replace(strings.Replace(validResponse, `"unclear":0.03`, `"unclear":null`, 1), `"established":0.9`, `"established":0.93`, 1),
		"bad distribution":        strings.Replace(validResponse, `"established":0.9`, `"established":0.1`, 1),
		"wrong winner":            strings.Replace(validResponse, `"choice":"established"`, `"choice":"planned"`, 1),
		"missing confidence":      strings.Replace(validResponse, `,"confidence":0.76`, "", 1),
		"missing answer":          strings.Replace(validResponse, `"durable"`, `"other"`, 1),
		"wrong model":             strings.Replace(validResponse, "jev-1.13.0", "jev-1.14.0", 1),
		"missing usage":           strings.Replace(validResponse, `,"usage":{"input_tokens":1000,"output_tokens":30}`, "", 1),
		"negative usage":          strings.Replace(validResponse, `"input_tokens":1000`, `"input_tokens":-1`, 1),
		"trailing JSON":           validResponse + `{}`,
		"oversize":                strings.Repeat(" ", 2<<20) + validResponse,
	} {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, body) }))
			defer srv.Close()
			c, _ := NewClient("test-key")
			c.endpoint = srv.URL
			if _, err := c.Assess(context.Background(), "User: deploy completed", "Deploy completed"); err == nil {
				t.Fatal("accepted invalid response")
			}
		})
	}
}

func TestClientStopsOnHTTPErrorWithoutEchoingProviderBody(t *testing.T) {
	for _, status := range []int{401, 403, 429, 529} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			calls := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				w.WriteHeader(status)
				fmt.Fprint(w, "private-episode test-key")
			}))
			defer srv.Close()
			c, _ := NewClient("test-key")
			c.endpoint = srv.URL
			_, err := c.Assess(context.Background(), "private-episode", "fact")
			if err == nil || !strings.Contains(err.Error(), fmt.Sprint(status)) || strings.Contains(err.Error(), "private-episode") || strings.Contains(err.Error(), "test-key") || calls != 1 {
				t.Fatalf("calls=%d err=%v", calls, err)
			}
		})
	}
}

func TestClientDoesNotFollowRedirectOrIgnoreCancellation(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		http.Redirect(w, r, "/elsewhere", http.StatusTemporaryRedirect)
	}))
	defer srv.Close()
	c, _ := NewClient("test-key")
	c.endpoint = srv.URL
	if _, err := c.Assess(context.Background(), "episode", "fact"); err == nil || calls != 1 {
		t.Fatalf("redirect: calls=%d err=%v", calls, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := c.Assess(ctx, "episode", "fact"); err == nil || calls != 1 {
		t.Fatalf("canceled request: calls=%d err=%v", calls, err)
	}
	if c.http.Timeout <= 0 || c.http.Timeout > 30*time.Second {
		t.Fatal("unbounded request timeout")
	}
}

func TestRejectsEmptyInputsBeforeNetwork(t *testing.T) {
	if _, err := NewClient(" "); err == nil {
		t.Fatal("accepted missing key")
	}
	for _, pair := range [][2]string{{"", "fact"}, {"episode", " "}} {
		if _, err := BuildRequest(pair[0], pair[1]); err == nil {
			t.Fatal("accepted empty evidence")
		}
	}
}
