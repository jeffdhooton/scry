package assess

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDispatchExactPacketAndTypedRetryAfter(t *testing.T) {
	packet := []byte(`{ "model" : "jev-1.13.0", "state":"exact", "questions":{} }`)
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		b, _ := io.ReadAll(r.Body)
		if string(b) != string(packet) {
			t.Errorf("packet changed: %q", b)
		}
		w.Header().Set("Retry-After", "17")
		w.WriteHeader(429)
		fmt.Fprint(w, "private-source test-key")
	}))
	defer srv.Close()
	c, _ := NewClient("test-key")
	c.endpoint = srv.URL
	_, err := c.Dispatch(context.Background(), packet, Model)
	var he *HTTPError
	if !errors.As(err, &he) || he.StatusCode != 429 || he.RetryAfter != 17*time.Second || calls != 1 || strings.Contains(err.Error(), "private-source") || strings.Contains(err.Error(), "test-key") {
		t.Fatalf("error %v, calls %d", err, calls)
	}
}
func TestDispatchRejectsUnsupportedPinAndTimeout(t *testing.T) {
	c, _ := NewClient("test-key")
	if _, err := c.Dispatch(context.Background(), []byte(`{"model":"jev-latest"}`), "jev-latest"); err == nil {
		t.Fatal("accepted alias")
	}
	if err := c.SetTimeout(0); err == nil {
		t.Fatal("accepted zero timeout")
	}
	if err := c.SetTimeout(2 * time.Second); err != nil || c.http.Timeout != 2*time.Second {
		t.Fatal("timeout not set")
	}
}

func TestAssessmentRejectsIncompatibleAnswerFields(t *testing.T) {
	secret := "sk-abcdefghijklmnopqrstuvwxyz123456"
	for _, body := range []string{
		strings.Replace(validResponse, `"noul":0.91`, `"noul":0.91,"choice":"`+secret+`"`, 1),
		strings.Replace(validResponse, `"noul":0.91`, `"noul":0.91,"confidence":0.9`, 1),
		strings.Replace(validResponse, `"noul":0.91`, `"noul":0.91,"probabilities":{"secret":1}`, 1),
		strings.Replace(validResponse, `"choice":"established"`, `"noul":0.9,"choice":"established"`, 1),
	} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, body) }))
		c, _ := NewClient("test-key")
		c.endpoint = srv.URL
		if _, err := c.Assess(context.Background(), "episode", "fact"); err == nil {
			t.Fatalf("accepted incompatible answer: %s", body)
		}
		srv.Close()
	}
}
func TestAssessmentValidateForRejectsInvalidMetadata(t *testing.T) {
	var a Assessment
	if err := json.Unmarshal([]byte(validResponse), &a); err != nil {
		t.Fatal(err)
	}
	a.Usage.InputTokens = -1
	if err := a.ValidateFor(Model); err == nil {
		t.Fatal("negative usage accepted")
	}
	a.Usage.InputTokens = 1000
	a.LatencyMS = math.NaN()
	if err := a.ValidateFor(Model); err == nil {
		t.Fatal("nonfinite latency accepted")
	}
	a.LatencyMS = 1
	if err := a.ValidateFor("jev-latest"); err == nil {
		t.Fatal("unsupported pin accepted")
	}
}
