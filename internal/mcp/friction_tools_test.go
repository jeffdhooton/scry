package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestFrictionProfilesAndExactForwarding(t *testing.T) {
	for _, profile := range []ToolProfile{ToolProfileAll, ToolProfileLocal, ToolProfileMemory} {
		t.Run(string(profile), func(t *testing.T) {
			fake := &fakeMemoryDialer{response: json.RawMessage(`{"event_id":"e","created":false,"sha256":"receipt"}`)}
			s := NewWithProfile(func() (Dialer, error) { return fake, nil }, profile)
			var out bytes.Buffer
			if err := s.Serve(context.Background(), strings.NewReader("{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tools/list\"}\n"), &out); err != nil {
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
			count := 0
			for _, td := range listed.Result.Tools {
				if strings.HasPrefix(td.Name, "scry_friction_") {
					count++
				}
			}
			want := 4
			if profile == ToolProfileMemory {
				want = 0
			}
			if count != want {
				t.Fatalf("friction tool count: %d want %d", count, want)
			}
			for _, action := range []string{"record", "get", "list", "review"} {
				// Even unknown fields reach daemon validation rather than disappearing
				// in a lossy MCP superset argument struct.
				args := json.RawMessage(`{"event_id":"e","observed":"literal <>&","measured_user_time_cost_seconds":null,"evidence":["a"],"unknown_field":"reject me"}`)
				request := map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": "scry_friction_" + action, "arguments": args}}
				b, _ := json.Marshal(request)
				out.Reset()
				if err := s.Serve(context.Background(), bytes.NewReader(append(b, '\n')), &out); err != nil {
					t.Fatal(err)
				}
				if profile == ToolProfileMemory {
					if len(fake.calls) != 0 || !strings.Contains(out.String(), `"isError":true`) {
						t.Fatal("memory profile routed local journal")
					}
					continue
				}
				last := fake.calls[len(fake.calls)-1]
				if last.method != "friction."+action {
					t.Fatalf("wrong method: %s", last.method)
				}
				got, _ := json.Marshal(last.params)
				var a, z any
				_ = json.Unmarshal(args, &a)
				_ = json.Unmarshal(got, &z)
				if fmt.Sprint(a) != fmt.Sprint(z) {
					t.Fatalf("lossy params: %s", got)
				}
				if !strings.Contains(out.String(), "receipt") {
					t.Fatalf("lost receipt: %s", out.String())
				}
			}
		})
	}
}

func TestFrictionToolDialFailureIsError(t *testing.T) {
	s := New(func() (Dialer, error) { return nil, fmt.Errorf("unavailable") })
	var out bytes.Buffer
	in := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"scry_friction_get","arguments":{"event_id":"e"}}}` + "\n"
	if err := s.Serve(context.Background(), strings.NewReader(in), &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"isError":true`) || !strings.Contains(out.String(), "unavailable") {
		t.Fatalf("hidden failure: %s", out.String())
	}
}

func TestRecordSchemaOffersDestinationKindAsOptional(t *testing.T) {
	var record tool
	for _, td := range frictionToolDefinitions {
		if td.Name == "scry_friction_record" {
			record = td
		}
	}
	if record.Name == "" {
		t.Fatal("record tool missing")
	}
	var schema struct {
		Properties map[string]json.RawMessage `json:"properties"`
		Required   []string                   `json:"required"`
	}
	if err := json.Unmarshal(record.InputSchema, &schema); err != nil {
		t.Fatal(err)
	}
	desc, ok := schema.Properties["destination_kind"]
	if !ok {
		t.Fatal("destination_kind missing from the record schema")
	}
	// The description is the only place a calling agent learns the vocabulary.
	for _, kind := range []string{"fact", "decision", "policy", "skill", "worker", "gate"} {
		if !strings.Contains(string(desc), kind) {
			t.Fatalf("description omits %q: %s", kind, desc)
		}
	}
	for _, r := range schema.Required {
		if r == "destination_kind" {
			t.Fatal("destination_kind must stay optional")
		}
	}
}
