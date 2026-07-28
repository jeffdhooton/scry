package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// fakeMemoryDialer records the RPC method + params of each Call so tests can
// assert on what callMemoryQuery forwarded, without needing a real daemon.
type fakeMemoryDialer struct {
	calls []fakeMemoryCall
}

type fakeMemoryCall struct {
	method string
	params any
}

func (f *fakeMemoryDialer) Call(_ context.Context, method string, params, out any) error {
	f.calls = append(f.calls, fakeMemoryCall{method: method, params: params})
	b, _ := json.Marshal(map[string]any{"ok": true})
	return json.Unmarshal(b, out)
}

func (f *fakeMemoryDialer) Close() error { return nil }

func TestMemoryToolDefinitions(t *testing.T) {
	if len(memoryToolDefinitions) != 4 {
		t.Fatalf("expected 4 memory tool definitions, got %d", len(memoryToolDefinitions))
	}

	wantRequired := map[string][]string{
		"scry_recall":      {"query"},
		"scry_memory_path": {"from", "to"},
		"scry_episodes":    {"entity"},
		"scry_remember":    {"fact"},
	}

	seen := map[string]bool{}
	for _, td := range memoryToolDefinitions {
		seen[td.Name] = true
		if td.Description == "" {
			t.Errorf("%s: empty description", td.Name)
		}
		want, ok := wantRequired[td.Name]
		if !ok {
			t.Errorf("unexpected tool %q in memoryToolDefinitions", td.Name)
			continue
		}
		var schema struct {
			Required []string `json:"required"`
		}
		if err := json.Unmarshal(td.InputSchema, &schema); err != nil {
			t.Fatalf("unmarshal InputSchema for %s: %v", td.Name, err)
		}
		if !reflect.DeepEqual(schema.Required, want) {
			t.Errorf("%s required = %v, want %v", td.Name, schema.Required, want)
		}
	}
	for name := range wantRequired {
		if !seen[name] {
			t.Errorf("missing tool %q in memoryToolDefinitions", name)
		}
	}
}

// TestToolsListIncludesMemoryTools guards against the append in
// handleToolsList silently dropping the memory domain.
func TestToolsListIncludesMemoryTools(t *testing.T) {
	s := New(func() (Dialer, error) { return &fakeMemoryDialer{}, nil })
	var out bytes.Buffer
	in := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}` + "\n")
	if err := s.Serve(context.Background(), in, &out); err != nil {
		t.Fatalf("Serve: %v", err)
	}

	var resp struct {
		Result struct {
			Tools []tool `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v\nraw: %s", err, out.String())
	}

	names := map[string]bool{}
	for _, td := range resp.Result.Tools {
		names[td.Name] = true
	}
	for _, want := range []string{"scry_recall", "scry_memory_path", "scry_episodes", "scry_remember"} {
		if !names[want] {
			t.Errorf("tools/list missing %q", want)
		}
	}
}

// TestCallMemoryQueryForwardsRecall mirrors the git-tools dispatch shape:
// arg parse -> RPC name + params. scry_recall stands in for all four since
// they share callMemoryQuery.
func TestCallMemoryQueryForwardsRecall(t *testing.T) {
	fd := &fakeMemoryDialer{}
	s := New(func() (Dialer, error) { return fd, nil })

	var out bytes.Buffer
	in := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"scry_recall","arguments":{"query":"hermes","as_of":"2026-01-01T00:00:00Z","limit":3}}}` + "\n")
	if err := s.Serve(context.Background(), in, &out); err != nil {
		t.Fatalf("Serve: %v", err)
	}

	if len(fd.calls) != 1 {
		t.Fatalf("expected 1 daemon call, got %d", len(fd.calls))
	}
	call := fd.calls[0]
	if call.method != "memory.recall" {
		t.Errorf("method = %q, want memory.recall", call.method)
	}
	params, ok := call.params.(map[string]any)
	if !ok {
		t.Fatalf("params type = %T, want map[string]any", call.params)
	}
	if params["query"] != "hermes" {
		t.Errorf("query = %v, want hermes", params["query"])
	}
	if params["as_of"] != "2026-01-01T00:00:00Z" {
		t.Errorf("as_of = %v, want 2026-01-01T00:00:00Z", params["as_of"])
	}
	if params["limit"] != 3 {
		t.Errorf("limit = %v, want 3", params["limit"])
	}
	if _, present := params["from"]; present {
		t.Errorf("unexpected from param leaked into recall call: %v", params)
	}

	if strings.Contains(out.String(), `"isError":true`) {
		t.Errorf("unexpected tool error in response: %s", out.String())
	}
}

// TestCallMemoryQueryForwardsRemember checks the array-valued entities field
// round-trips and scry_remember dispatches to memory.remember.
func TestCallMemoryQueryForwardsRemember(t *testing.T) {
	fd := &fakeMemoryDialer{}
	s := New(func() (Dialer, error) { return fd, nil })

	var out bytes.Buffer
	in := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"scry_remember","arguments":{"fact":"book-system deploys to hermes-mini","entities":["book-system","hermes-mini"]}}}` + "\n")
	if err := s.Serve(context.Background(), in, &out); err != nil {
		t.Fatalf("Serve: %v", err)
	}

	if len(fd.calls) != 1 {
		t.Fatalf("expected 1 daemon call, got %d", len(fd.calls))
	}
	call := fd.calls[0]
	if call.method != "memory.remember" {
		t.Errorf("method = %q, want memory.remember", call.method)
	}
	params, ok := call.params.(map[string]any)
	if !ok {
		t.Fatalf("params type = %T, want map[string]any", call.params)
	}
	if params["fact"] != "book-system deploys to hermes-mini" {
		t.Errorf("fact = %v", params["fact"])
	}
	entities, ok := params["entities"].([]string)
	if !ok || !reflect.DeepEqual(entities, []string{"book-system", "hermes-mini"}) {
		t.Errorf("entities = %v", params["entities"])
	}
}
