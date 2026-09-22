package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func reviewToolCall(t *testing.T, s *Server, action string, args json.RawMessage) string {
	t.Helper()
	request := map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": "scry_review_" + action, "arguments": args}}
	b, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := s.Serve(context.Background(), bytes.NewReader(append(b, '\n')), &out); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func TestReviewToolsProfilesAndDispatch(t *testing.T) {
	for _, profile := range []ToolProfile{ToolProfileAll, ToolProfileLocal, ToolProfileMemory} {
		t.Run(string(profile), func(t *testing.T) {
			fake := &fakeMemoryDialer{response: json.RawMessage(`{"snapshot_id":"snapshot-1","freshness":"stale","provisional":true}`)}
			s := NewWithProfile(func() (Dialer, error) { return fake, nil }, profile)
			var out bytes.Buffer
			if err := s.Serve(context.Background(), strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`+"\n"), &out); err != nil {
				t.Fatal(err)
			}
			var listed struct {
				Result struct {
					Tools []tool `json:"tools"`
				} `json:"result"`
			}
			if err := json.Unmarshal(out.Bytes(), &listed); err != nil {
				t.Fatal(err)
			}
			names := map[string]bool{}
			for _, td := range listed.Result.Tools {
				names[td.Name] = true
			}
			for _, tc := range []struct {
				action string
				args   string
			}{
				{"status", `{}`}, {"preview", `{"repo":"/repo"}`}, {"run", `{"repo":"/repo"}`}, {"list", `{"repo":"/repo"}`}, {"get", `{"id":"review-1"}`},
			} {
				if got, want := names["scry_review_"+tc.action], profile != ToolProfileMemory; got != want {
					t.Errorf("%s advertised=%v want %v", tc.action, got, want)
				}
				result := reviewToolCall(t, s, tc.action, json.RawMessage(tc.args))
				if profile == ToolProfileMemory {
					if len(fake.calls) != 0 || !strings.Contains(result, `"isError":true`) {
						t.Fatalf("memory profile routed review: %s", result)
					}
					continue
				}
				if strings.Contains(result, `"isError":true`) || !strings.Contains(result, "snapshot-1") || !strings.Contains(result, "stale") || !strings.Contains(result, "provisional") {
					t.Fatalf("lost snapshot-bound output: %s", result)
				}
				last := fake.calls[len(fake.calls)-1]
				if last.method != "review."+tc.action {
					t.Fatalf("method=%s", last.method)
				}
				gotJSON, _ := json.Marshal(last.params)
				var got, want any
				_ = json.Unmarshal(gotJSON, &got)
				_ = json.Unmarshal([]byte(tc.args), &want)
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("params=%s want %s", gotJSON, tc.args)
				}
			}
		})
	}
}

func TestReviewToolsResolveRepository(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for _, action := range []string{"preview", "run", "list"} {
		for _, tc := range []struct{ args, want string }{{`{}`, cwd}, {`{"repo":"relative"}`, filepath.Join(cwd, "relative")}} {
			fake := &fakeMemoryDialer{}
			s := New(func() (Dialer, error) { return fake, nil })
			result := reviewToolCall(t, s, action, json.RawMessage(tc.args))
			if strings.Contains(result, `"isError":true`) || len(fake.calls) != 1 {
				t.Fatalf("%s: %s", action, result)
			}
			raw, _ := json.Marshal(fake.calls[0].params)
			var got struct {
				Repo string `json:"repo"`
			}
			_ = json.Unmarshal(raw, &got)
			if got.Repo != tc.want {
				t.Fatalf("repo=%q want %q", got.Repo, tc.want)
			}
		}
	}
}

func TestReviewToolsRejectInvalidArgumentsBeforeDial(t *testing.T) {
	for _, tc := range []struct{ action, args string }{
		{"get", `{}`}, {"get", `{"id":""}`}, {"get", `{"id":"../secret"}`}, {"get", `{"id":"review with spaces"}`},
		{"get", `{"id":true}`}, {"get", `{"id":"` + strings.Repeat("a", 201) + `"}`}, {"get", `{"id":"ok","repo":"/other"}`},
		{"run", `{"repo":5}`}, {"run", `{"repo":"/repo","max_requests":999}`}, {"status", `{"repo":"/repo"}`}, {"status", `{"":"unexpected"}`}, {"preview", `[]`}, {"preview", `null`}, {"run", `{"repo":null}`},
	} {
		t.Run(tc.action+tc.args, func(t *testing.T) {
			dialed := false
			s := New(func() (Dialer, error) { dialed = true; return &fakeMemoryDialer{}, nil })
			result := reviewToolCall(t, s, tc.action, json.RawMessage(tc.args))
			if dialed || !strings.Contains(result, `"isError":true`) {
				t.Fatalf("dialed=%v result=%s", dialed, result)
			}
		})
	}
}

func TestReviewToolDaemonFailure(t *testing.T) {
	s := New(func() (Dialer, error) { return nil, fmt.Errorf("unavailable") })
	result := reviewToolCall(t, s, "status", json.RawMessage(`{}`))
	if !strings.Contains(result, `"isError":true`) || !strings.Contains(result, "unavailable") {
		t.Fatalf("hidden error: %s", result)
	}
}
