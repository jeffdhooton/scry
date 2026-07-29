package browse

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestRenderInjectsData covers the core contract: Render must produce a
// well-formed HTML document (doctype + charset meta present, from the
// embedded template) with the caller's data marshaled and spliced in where
// __SCRY_MEMORY_DATA__ was.
func TestRenderInjectsData(t *testing.T) {
	type tiny struct {
		Foo string `json:"foo"`
	}

	out, err := Render(tiny{Foo: "bar"})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	html := string(out)

	if !strings.Contains(html, "<!doctype html>") {
		t.Errorf("output missing doctype: %q", html[:min(80, len(html))])
	}
	if !strings.Contains(html, `<meta charset="utf-8">`) {
		t.Errorf("output missing charset meta")
	}
	if strings.Contains(html, dataPlaceholder) {
		t.Errorf("placeholder %q was not replaced", dataPlaceholder)
	}

	want, err := json.Marshal(tiny{Foo: "bar"})
	if err != nil {
		t.Fatalf("marshal want: %v", err)
	}
	if !strings.Contains(html, string(want)) {
		t.Errorf("output does not contain injected JSON %q", want)
	}
}

// TestRenderEscapesClosingScriptTag covers the "</script>" injection guard:
// a literal "</" in the marshaled data must not survive into the output,
// since the injected JSON lives inside an inline <script> block and a raw
// "</script>" there would prematurely close it. json.Marshal already
// HTML-escapes "<" by default (to "<"), so this is largely
// belt-and-suspenders on top of that — but it's the documented contract, so
// assert it holds regardless of which layer provides the escaping.
func TestRenderEscapesClosingScriptTag(t *testing.T) {
	type tiny struct {
		Foo string `json:"foo"`
	}

	out, err := Render(tiny{Foo: "</script><script>alert(1)</script>"})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	html := string(out)

	if strings.Contains(html, "</script><script>alert(1)") {
		t.Errorf("unescaped </script> sequence leaked into output: %s", html)
	}
}
