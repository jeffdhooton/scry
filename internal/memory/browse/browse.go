// Package browse renders the scry memory graph visualization: a single
// self-contained HTML file (vanilla JS + CSS, no CDN/network calls) with the
// memory export data injected inline as a JSON literal. It backs two
// callers: `scry memory browse` (cmd/scry/memory.go, writes the result to a
// file and opens it) and the daemon's live memory UI HTTP server
// (internal/daemon/memory_ui.go, serves it fresh on every request).
package browse

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
)

// template is the embedded UI shell. It contains a single placeholder,
// __SCRY_MEMORY_DATA__, inside `const DATA = __SCRY_MEMORY_DATA__;`.
//
//go:embed template.html
var template string

// dataPlaceholder is the token in template.html that Render replaces with
// the JSON-marshaled export.
const dataPlaceholder = "__SCRY_MEMORY_DATA__"

// Render marshals data to JSON and injects it into the embedded template in
// place of __SCRY_MEMORY_DATA__. "</" sequences in the JSON are escaped to
// "<\/" so a fact or episode summary containing a literal "</script>" can't
// prematurely close the inline <script> tag the data lives in.
func Render(data any) ([]byte, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshal memory export: %w", err)
	}
	safe := strings.ReplaceAll(string(b), "</", "<\\/")
	html := strings.Replace(template, dataPlaceholder, safe, 1)
	return []byte(html), nil
}
