package assess

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"syscall"
	"testing"
	"time"
)

const diagnosticSecret = "provider-secret-private-source-key"

// Losing the response subtype must fail without allowing provider text into errors.
func TestDispatchSafeResponseFailureCodes(t *testing.T) {
	for _, tt := range []struct{ name, body, want string }{
		{"json", diagnosticSecret, "response_json"},
		{"trailing json", validResponse + `{}`, "response_json"},
		{"size", strings.Repeat(" ", 2<<20) + validResponse, "response_size"},
		{"usage absent", strings.Replace(validResponse, `,"usage":{"input_tokens":1000,"output_tokens":30}`, "", 1), "usage"},
		{"usage null", strings.Replace(validResponse, `"input_tokens":1000`, `"input_tokens":null`, 1), "usage"},
		{"usage negative", strings.Replace(validResponse, `"output_tokens":30`, `"output_tokens":-1`, 1), "usage"},
		{"model", strings.Replace(validResponse, Model, diagnosticSecret, 1), "model_mismatch"},
		{"answers absent", `{"model":"jev-1.13.0","usage":{"input_tokens":1,"output_tokens":1}}`, "answer_count"},
		{"noul absent", strings.Replace(validResponse, `,"noul":0.91`, "", 1), "noul_fields"},
		{"noul null", strings.Replace(validResponse, `"noul":0.91`, `"noul":null`, 1), "noul_fields"},
		{"noul range", strings.Replace(validResponse, `"noul":0.8`, `"noul":1.1`, 1), "noul_fields"},
		{"noul type", strings.Replace(validResponse, `"type":"noul"`, `"type":"`+diagnosticSecret+`"`, 1), "noul_fields"},
		{"assertion confidence", strings.Replace(validResponse, `,"confidence":0.76`, "", 1), "assertion_fields"},
		{"assertion option absent", strings.Replace(validResponse, `,"unclear":0.03`, "", 1), "assertion_fields"},
		{"assertion option null", strings.Replace(validResponse, `"unclear":0.03`, `"unclear":null`, 1), "assertion_distribution"},
		{"assertion option range", strings.Replace(validResponse, `"unclear":0.03`, `"unclear":-0.03`, 1), "assertion_distribution"},
		{"assertion unknown choice", strings.Replace(validResponse, `"choice":"established"`, `"choice":"`+diagnosticSecret+`"`, 1), "assertion_consistency"},
		{"assertion sum", strings.Replace(validResponse, `"established":0.9`, `"established":0.8`, 1), "assertion_consistency"},
		{"assertion winner", strings.Replace(validResponse, `"choice":"established"`, `"choice":"planned"`, 1), "assertion_consistency"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++; fmt.Fprint(w, tt.body) }))
			defer srv.Close()
			c, _ := NewClient(diagnosticSecret)
			c.endpoint = srv.URL
			_, err := c.Assess(context.Background(), diagnosticSecret, "fact")
			assertSafeFailure(t, err, tt.want)
			if calls != 1 {
				t.Fatalf("dispatch retried: calls=%d", calls)
			}
		})
	}
}

type diagnosticRoundTripper func(*http.Request) (*http.Response, error)

func (f diagnosticRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type diagnosticReader struct{}

func (diagnosticReader) Read([]byte) (int, error) { return 0, errors.New(diagnosticSecret) }
func (diagnosticReader) Close() error             { return nil }

// Inspect structured transport causes, never their provider-controlled messages.
func TestDispatchSafeTransportFailureCodes(t *testing.T) {
	for _, tt := range []struct {
		name     string
		err      error
		want     string
		sentinel error
	}{
		{"generic", errors.New(diagnosticSecret), "transport", nil},
		{"dns", &net.DNSError{Err: diagnosticSecret, Name: diagnosticSecret}, "dns", nil},
		{"connect", &net.OpError{Op: "dial", Net: "tcp", Err: syscall.ECONNREFUSED}, "connect", nil},
		{"tls certificate", &tls.CertificateVerificationError{Err: x509.UnknownAuthorityError{}}, "tls", nil},
		{"tls protocol", tls.RecordHeaderError{Msg: diagnosticSecret}, "tls", nil},
		{"timeout", &net.DNSError{Err: diagnosticSecret, IsTimeout: true}, "timeout", context.DeadlineExceeded},
		{"deadline", fmt.Errorf("%s: %w", diagnosticSecret, context.DeadlineExceeded), "timeout", context.DeadlineExceeded},
		{"cancel", fmt.Errorf("%s: %w", diagnosticSecret, context.Canceled), "canceled", context.Canceled},
	} {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := NewClient(diagnosticSecret)
			calls := 0
			c.http.Transport = diagnosticRoundTripper(func(*http.Request) (*http.Response, error) { calls++; return nil, tt.err })
			_, err := c.Assess(context.Background(), "episode", "fact")
			assertSafeFailure(t, err, tt.want)
			if calls != 1 {
				t.Fatalf("dispatch retried: calls=%d", calls)
			}
			if tt.sentinel != nil && !errors.Is(err, tt.sentinel) {
				t.Fatalf("lost context sentinel: %v", err)
			}
			var op *net.OpError
			var dns *net.DNSError
			if errors.As(err, &op) || errors.As(err, &dns) {
				t.Fatal("retained unsafe transport cause")
			}
		})
	}
	c, _ := NewClient(diagnosticSecret)
	c.http.Transport = diagnosticRoundTripper(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: diagnosticReader{}}, nil
	})
	_, err := c.Assess(context.Background(), "episode", "fact")
	assertSafeFailure(t, err, "response_read")
}

// Classify deadlines that happen after headers as timeout, preserving errors.Is.
func TestDispatchResponseBodyDeadlineAndCancellation(t *testing.T) {
	for _, cancelRequest := range []bool{false, true} {
		t.Run(fmt.Sprint(cancelRequest), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(200)
				w.(http.Flusher).Flush()
				if cancelRequest {
					cancel()
				}
				<-r.Context().Done()
			}))
			defer srv.Close()
			c, _ := NewClient(diagnosticSecret)
			c.endpoint = srv.URL
			if err := c.SetTimeout(30 * time.Millisecond); err != nil {
				t.Fatal(err)
			}
			_, err := c.Assess(ctx, "episode", "fact")
			want, sentinel := "timeout", context.DeadlineExceeded
			if cancelRequest {
				want, sentinel = "canceled", context.Canceled
			}
			assertSafeFailure(t, err, want)
			if !errors.Is(err, sentinel) {
				t.Fatalf("lost sentinel %v: %v", sentinel, err)
			}
		})
	}
}

func TestFailureCodeRejectsUnknownErrorText(t *testing.T) {
	if got := FailureCode(nil); got != "" {
		t.Fatalf("nil code = %q", got)
	}
	for _, err := range []error{errors.New(diagnosticSecret), errors.New("assess: invalid TypeSafe JSON response")} {
		if got := FailureCode(err); got != "unknown" {
			t.Fatalf("untyped code = %q", got)
		}
	}
	if got := FailureCode(&HTTPError{StatusCode: 429}); got != "http" {
		t.Fatalf("HTTP code = %q", got)
	}
}

// The timeout setter must not allow a caller to silently remove the 15s ceiling.
func TestSetTimeoutPreservesUpperBound(t *testing.T) {
	c, _ := NewClient("key")
	for _, d := range []time.Duration{-1, 0, 15*time.Second + 1, time.Hour} {
		if err := c.SetTimeout(d); err == nil {
			t.Errorf("accepted timeout %s", d)
		}
		if c.http.Timeout != 15*time.Second {
			t.Fatalf("invalid setter changed timeout: %s", c.http.Timeout)
		}
	}
	if err := c.SetTimeout(time.Millisecond); err != nil {
		t.Fatal(err)
	}
}

// Huge provider integers must not wrap or shorten the requested cooldown.
func TestRetryAfterBoundsProviderValues(t *testing.T) {
	now := time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)
	for _, tt := range []struct {
		value string
		want  time.Duration
	}{
		{"17", 17 * time.Second}, {"-1", 0}, {"nonsense", 0},
		{"86400", 24 * time.Hour}, {"86401", 24*time.Hour + time.Second},
		{"9223372036854775807", time.Duration(1<<63 - 1)},
		{"999999999999999999999999999", time.Duration(1<<63 - 1)},
		{now.Add(48 * time.Hour).Format(http.TimeFormat), 48 * time.Hour},
		{now.Add(-time.Hour).Format(http.TimeFormat), 0},
	} {
		if got := parseRetryAfter(tt.value, now); got != tt.want {
			t.Errorf("Retry-After %q: %s want %s", tt.value, got, tt.want)
		}
	}
}

func assertSafeFailure(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatal("accepted invalid provider result")
	}
	if got := string(FailureCode(err)); got != want {
		t.Errorf("failure code=%q want=%q err=%v", got, want, err)
	}
	if got := string(FailureCode(fmt.Errorf("wrapper: %w", err))); got != want {
		t.Errorf("wrapped code=%q want=%q", got, want)
	}
	for cause := err; cause != nil; cause = errors.Unwrap(cause) {
		if strings.Contains(fmt.Sprintf("%+v %#v", cause, cause), diagnosticSecret) {
			t.Fatalf("unsafe error retained secret")
		}
	}
}

var _ io.ReadCloser = diagnosticReader{}

// Even an invalid internal code must not become persisted provider-controlled text.
func TestFailureAllowlistAndZeroValue(t *testing.T) {
	for _, err := range []*Failure{{code: ErrorCode(diagnosticSecret)}, {}, nil} {
		if got := FailureCode(err); got != CodeUnknown {
			t.Fatalf("unknown category = %q", got)
		}
		if strings.Contains(err.Error(), diagnosticSecret) {
			t.Fatal("unsafe code escaped Error")
		}
		if err.Unwrap() != nil {
			t.Fatal("unknown category retained a cause")
		}
	}
	if err := failure(ErrorCode(diagnosticSecret)); strings.Contains(fmt.Sprintf("%#v", err), diagnosticSecret) {
		t.Fatal("failure constructor retained unsafe code")
	}
}

// Invalid packets and pins must fail before any transport is invoked.
func TestDispatchRequestFailureCodesWithoutNetwork(t *testing.T) {
	c, _ := NewClient("key")
	c.http.Transport = diagnosticRoundTripper(func(*http.Request) (*http.Response, error) {
		t.Error("invalid packet reached network")
		return nil, errors.New("network")
	})
	for _, tt := range []struct{ body, model, want string }{
		{`{}`, "jev-latest", "model_mismatch"},
		{diagnosticSecret, Model, "request_invalid"},
		{`{"model":"` + diagnosticSecret + `"}`, Model, "model_mismatch"},
	} {
		_, err := c.Dispatch(context.Background(), []byte(tt.body), tt.model)
		assertSafeFailure(t, err, tt.want)
	}
	c.endpoint = "://" + diagnosticSecret
	_, err := c.Dispatch(context.Background(), []byte(`{"model":"jev-1.13.0"}`), Model)
	assertSafeFailure(t, err, "request_invalid")
}

// A real interrupted response must retain its read-failure category.
func TestDispatchTruncatedHTTPResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "10000")
		fmt.Fprint(w, diagnosticSecret)
	}))
	defer srv.Close()
	c, _ := NewClient("key")
	c.endpoint = srv.URL
	_, err := c.Assess(context.Background(), "episode", "fact")
	assertSafeFailure(t, err, "response_read")
}
