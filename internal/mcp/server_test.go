package mcp

import (
	"encoding/json"
	"testing"
)

func TestSplitCompoundSymbol(t *testing.T) {
	cases := []struct {
		in        string
		container string
		tail      string
	}{
		{"DB::table", "DB", "table"},
		{"Illuminate\\DB::table", "Illuminate\\DB", "table"},
		{"Auth::user()", "Auth", "user"},
		{"$user->name", "$user", "name"},
		{"client.Connect", "client", "Connect"},
		{"a::b::c", "a::b", "c"},        // rightmost split
		{"Auth::user->name", "Auth::user", "name"}, // chain → last op
		{"plainIdent", "", ""},
		{"", "", ""},
		{"::method", "", ""}, // no container → don't split
		{"method::", "", ""}, // no tail → don't split
	}
	for _, c := range cases {
		container, tail := splitCompoundSymbol(c.in)
		if container != c.container || tail != c.tail {
			t.Errorf("splitCompoundSymbol(%q) = (%q,%q), want (%q,%q)", c.in, container, tail, c.container, c.tail)
		}
	}
}

func TestSymbolIDMatchesContainer(t *testing.T) {
	cases := []struct {
		id        string
		container string
		want      bool
	}{
		{"scip-php composer laravel/framework abc Illuminate/Support/Facades/DB#table().", "DB", true},
		{"scip-php composer laravel/framework abc Illuminate/Database/DatabaseManager#table().", "DB", false},
		{"scip-php composer laravel/framework abc Illuminate/Database/Connection#table().", "DB", false},
		{"scip-php composer laravel/framework abc Illuminate/Database/Eloquent/Builder#table().", "DB", false},
		{"scip-typescript npm . . src/foo.ts/Foo#bar().", "Foo", true},
		{"scip-go gomod . . github.com/foo/Bar#baz().", "Bar", true},
		{"scip-go gomod . . github.com/foobar/Baz#qux().", "Bar", false}, // segment boundary
	}
	for _, c := range cases {
		got := symbolIDMatchesContainer(c.id, c.container)
		if got != c.want {
			t.Errorf("symbolIDMatchesContainer(%q, %q) = %v, want %v", c.id, c.container, got, c.want)
		}
	}
}

func TestFilterResultByContainer(t *testing.T) {
	// Simulate `scry refs table` returning three matches across DB, Connection,
	// and an unrelated Eloquent Builder. Asking for DB::table should keep only
	// the DB one.
	raw := []byte(`{
		"symbol": "table",
		"matches": [
			{"symbol_id": "scip-php composer laravel/framework v Illuminate/Support/Facades/DB#table().", "display_name": "table", "kind": "Method", "occurrences": [{"file":"a.php","line":1}]},
			{"symbol_id": "scip-php composer laravel/framework v Illuminate/Database/Connection#table().", "display_name": "table", "kind": "Method", "occurrences": [{"file":"b.php","line":2}]},
			{"symbol_id": "scip-php composer laravel/framework v Illuminate/Database/Eloquent/Builder#table().", "display_name": "table", "kind": "Method", "occurrences": [{"file":"c.php","line":3}]}
		],
		"total": 3,
		"elapsed_ms": 5
	}`)
	filtered, ok := filterResultByContainer(raw, "DB")
	if !ok {
		t.Fatal("filter returned ok=false; expected at least one match for DB")
	}
	var result struct {
		Matches []json.RawMessage `json:"matches"`
		Total   int               `json:"total"`
	}
	if err := json.Unmarshal(filtered, &result); err != nil {
		t.Fatalf("unmarshal filtered: %v", err)
	}
	if result.Total != 1 || len(result.Matches) != 1 {
		t.Errorf("filtered total/len = %d/%d, want 1/1", result.Total, len(result.Matches))
	}
	var m struct{ SymbolID string `json:"symbol_id"` }
	_ = json.Unmarshal(result.Matches[0], &m)
	if !containsAny(m.SymbolID, "/Facades/DB#") {
		t.Errorf("kept wrong match: %s", m.SymbolID)
	}

	// Asking for a container that matches nothing returns ok=false.
	if _, ok := filterResultByContainer(raw, "Nonexistent"); ok {
		t.Errorf("expected ok=false for Nonexistent container")
	}
}
