// Package mcp implements a minimal Model Context Protocol (MCP) stdio server
// that exposes scry's symbol queries as tools Claude Code can call directly.
//
// Design:
//
//   - Speaks newline-delimited JSON-RPC 2.0 over stdin/stdout (the MCP stdio
//     transport).
//   - Implements only the three methods a host client actually calls to use
//     tools: initialize, tools/list, tools/call. Notifications that don't
//     require a response (initialized, notifications/cancelled, etc.) are
//     accepted and silently ignored.
//   - Translates every tool/call into a JSON-RPC call against the existing
//     scry daemon (via the Dialer interface). The daemon is auto-spawned by
//     the supplied dial function; the MCP server itself is stateless.
//   - Emits tool output as a single "text" content block containing either a
//     compact human-readable summary or the raw JSON. Claude can re-parse the
//     text if it wants structure back.
//
// What we deliberately don't implement:
//
//   - resources/list, resources/read, prompts/list, prompts/get. scry has
//     nothing to expose as a resource or prompt — it's a query tool.
//   - logging/setLevel. Logging stays on stderr; stdout is MCP protocol only.
//   - Cancellation plumbing. Queries return in <10ms; the host has no reason
//     to cancel mid-call.
//
// Every write to stdout happens in one contiguous operation so stdio-based
// clients see well-formed messages even under concurrent tool calls (which
// Claude Code doesn't send anyway, but it's cheap to be correct).
package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// protocolVersion is the MCP protocol version this server reports. We echo
// the client's requested version if it sends one we recognize; otherwise we
// fall back to this default.
const protocolVersion = "2025-06-18"

// Dialer is the minimal interface over the scry daemon client. rpc.Client
// satisfies it; tests can stub with any implementation.
type Dialer interface {
	Call(ctx context.Context, method string, params, out any) error
	Close() error
}

// DialFunc returns a fresh Dialer (typically an rpc.Client). The server
// opens one connection per tool call and closes it after; scry daemon calls
// are cheap enough that pooling isn't worth the complexity.
type DialFunc func() (Dialer, error)

// Server is a long-running MCP stdio server. Serve blocks until the input
// reader is closed.
type Server struct {
	dial DialFunc

	// mu serializes writes to the output stream so partial writes from
	// concurrent goroutines can't interleave. Reads are already serialized
	// by the Serve loop.
	mu  sync.Mutex
	out *bufio.Writer
}

// New constructs an unconnected MCP server. Call Serve to run it.
func New(dial DialFunc) *Server {
	return &Server{dial: dial}
}

// Serve runs the read-dispatch-write loop until ctx is cancelled or the
// input reader returns EOF. Every request is handled synchronously — the MCP
// stdio transport is single-threaded by design.
func (s *Server) Serve(ctx context.Context, in io.Reader, out io.Writer) error {
	s.out = bufio.NewWriter(out)

	scanner := bufio.NewScanner(in)
	// Claude Code has been observed sending messages up to ~64 KB. Bump the
	// buffer well past the default 64 KB so we don't truncate on large
	// tools/list responses round-tripping through a host.
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		s.handleLine(ctx, line)
	}
	if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}

// handleLine dispatches one incoming JSON-RPC message. Requests get a
// response written to stdout; notifications (no id field) do not.
func (s *Server) handleLine(ctx context.Context, line []byte) {
	var req request
	if err := json.Unmarshal(line, &req); err != nil {
		// Protocol violation — we can't map back to an id, so write a
		// parse error with id=null.
		s.writeError(nil, -32700, "parse error: "+err.Error(), nil)
		return
	}

	// Notifications never get a response.
	isNotification := req.ID == nil

	switch req.Method {
	case "initialize":
		s.handleInitialize(req)
	case "initialized", "notifications/initialized":
		// Client handshake completion. No response expected; no action needed.
	case "tools/list":
		s.handleToolsList(req)
	case "tools/call":
		s.handleToolsCall(ctx, req)
	case "ping":
		if !isNotification {
			s.writeResult(req.ID, map[string]any{})
		}
	default:
		if !isNotification {
			s.writeError(req.ID, -32601, "method not found: "+req.Method, nil)
		}
	}
}

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"` // keep raw so we can echo it back unchanged
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *responseError  `json:"error,omitempty"`
}

type responseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func (s *Server) writeResult(id json.RawMessage, result any) {
	s.writeMessage(&response{JSONRPC: "2.0", ID: id, Result: result})
}

func (s *Server) writeError(id json.RawMessage, code int, msg string, data any) {
	s.writeMessage(&response{JSONRPC: "2.0", ID: id, Error: &responseError{Code: code, Message: msg, Data: data}})
}

func (s *Server) writeMessage(resp *response) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if resp.ID == nil {
		resp.ID = json.RawMessage(`null`)
	}
	b, err := json.Marshal(resp)
	if err != nil {
		// This should be impossible — every shape we pass is JSON-encodable.
		fmt.Fprintf(os.Stderr, "scry mcp: marshal response: %v\n", err)
		return
	}
	_, _ = s.out.Write(b)
	_ = s.out.WriteByte('\n')
	_ = s.out.Flush()
}

// ---------------- initialize ----------------

type initializeParams struct {
	ProtocolVersion string          `json:"protocolVersion"`
	Capabilities    json.RawMessage `json:"capabilities"`
	ClientInfo      struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"clientInfo"`
}

func (s *Server) handleInitialize(req request) {
	var p initializeParams
	_ = json.Unmarshal(req.Params, &p) // missing fields are fine

	// Echo the client's version if it looks like something we recognize;
	// otherwise fall back to our default. MCP is liberal about this.
	version := p.ProtocolVersion
	if version == "" {
		version = protocolVersion
	}

	result := map[string]any{
		"protocolVersion": version,
		"capabilities": map[string]any{
			"tools": map[string]any{
				// We do not emit tools/list_changed notifications — the tool
				// set is static for the life of the server.
				"listChanged": false,
			},
		},
		"serverInfo": map[string]any{
			"name":    "scry",
			"version": "0.1.0",
		},
	}
	s.writeResult(req.ID, result)
}

// ---------------- tools/list ----------------

// tool is one entry in the tools/list response. The JSON shape matches the
// MCP spec exactly.
type tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

// toolDefinitions is the static list returned by tools/list. Descriptions
// are written for Claude Code's routing model — each one should make it
// obvious when to pick scry over Grep.
var toolDefinitions = []tool{
	{
		Name:        "scry_refs",
		Description: "PREFERRED over Grep for any identifier lookup. Finds every call site, usage, and reference of a symbol across a scry-indexed repo in under 10ms. Returns file:line:col with a context snippet for each occurrence. Accepts plain names (`processOrder`, `UserController`) AND method-call notation (`DB::table`, `auth->user`, `service.run`) — the leftmost token narrows to a specific class. Use this whenever the user asks \"where is X called\", \"where is X used\", \"find every reference to X\", \"who uses X\", or any question about where a named function / class / method / interface / property appears. Only fall back to Grep when the repo isn't indexed, when scry returns empty, or when searching for text inside strings/comments/docstrings.",
		InputSchema: mustMarshal(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"symbol": map[string]any{"type": "string", "description": "The symbol to look up. Plain name (e.g. `processOrder`, `UserController`) or method notation (e.g. `DB::table`, `User->save`, `client.Connect`). Case-insensitive exact match on the display name; compound forms are parsed and filtered so `DB::table` returns only the 92 `table()` calls on the DB class, not every `table()` on every class."},
				"repo":   map[string]any{"type": "string", "description": "Absolute path to the repo root. Defaults to the MCP server's working directory (which is Claude Code's cwd)."},
			},
			"required": []string{"symbol"},
		}),
	},
	{
		Name:        "scry_defs",
		Description: "Find the definition site(s) of a symbol. Returns file:line:col for each place the symbol is declared. Use for 'where is X defined' or 'jump to X'. Accepts the same compound notation as scry_refs (`DB::table` → defs of `table()` on the DB class). Note: vendor/framework/stdlib symbols may return empty because scry doesn't walk third-party sources as definitions — fall back to Grep or Read for those.",
		InputSchema: mustMarshal(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"symbol": map[string]any{"type": "string", "description": "The symbol to look up. Plain name or compound (`DB::table`, `Foo->bar`, `x.y`)."},
				"repo":   map[string]any{"type": "string", "description": "Absolute path to the repo root. Defaults to cwd."},
			},
			"required": []string{"symbol"},
		}),
	},
	{
		Name:        "scry_callers",
		Description: "Find every caller of a function or method — the set of references enriched with the containing function at each call site. Use this for 'who calls X' or 'what depends on X'. Coverage is full on TypeScript; partial on Go (scip-go doesn't populate enclosing_range for every file).",
		InputSchema: mustMarshal(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"symbol": map[string]any{"type": "string", "description": "The symbol name of the callee."},
				"repo":   map[string]any{"type": "string", "description": "Absolute path to the repo root. Defaults to cwd."},
			},
			"required": []string{"symbol"},
		}),
	},
	{
		Name:        "scry_callees",
		Description: "Find everything a function or method calls — the outgoing call graph edges. Use this for 'what does X call' or 'what does X depend on'. Requires the repo to be indexed.",
		InputSchema: mustMarshal(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"symbol": map[string]any{"type": "string", "description": "The symbol name of the caller."},
				"repo":   map[string]any{"type": "string", "description": "Absolute path to the repo root. Defaults to cwd."},
			},
			"required": []string{"symbol"},
		}),
	},
	{
		Name:        "scry_impls",
		Description: "Find every implementor of an interface or base class. Returns the concrete types that implement the named contract. Use this for 'what implements Repository' or 'what extends BaseHandler'.",
		InputSchema: mustMarshal(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"symbol": map[string]any{"type": "string", "description": "The interface or base class name."},
				"repo":   map[string]any{"type": "string", "description": "Absolute path to the repo root. Defaults to cwd."},
			},
			"required": []string{"symbol"},
		}),
	},
	{
		Name:        "scry_tests",
		Description: "Check which functions are covered by tests. Returns coverage status for a symbol — whether any test exercises it, and with what hit count. Use before deciding whether to write new tests or to identify which existing tests to run after a code change. Requires a coverage file (cover.out, coverage-final.json, clover.xml, or coverage.json) to be present in the repo root — generate it with your test runner (e.g. `go test -coverprofile=cover.out ./...`) then re-run `scry init`.",
		InputSchema: mustMarshal(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"symbol": map[string]any{"type": "string", "description": "The function or method name to check coverage for."},
				"repo":   map[string]any{"type": "string", "description": "Absolute path to the repo root. Defaults to cwd."},
			},
			"required": []string{"symbol"},
		}),
	},
	{
		Name:        "scry_status",
		Description: "List every repo scry has indexed, with document counts, ref counts, and last-indexed timestamps. Use this to check whether a repo is indexed before running a symbol query, or to see which repos are searchable.",
		InputSchema: mustMarshal(map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		}),
	},
}

func (s *Server) handleToolsList(req request) {
	all := make([]tool, 0, len(toolDefinitions)+len(gitToolDefinitions)+len(schemaToolDefinitions)+len(httpToolDefinitions)+len(graphToolDefinitions))
	all = append(all, toolDefinitions...)
	all = append(all, gitToolDefinitions...)
	all = append(all, schemaToolDefinitions...)
	all = append(all, httpToolDefinitions...)
	all = append(all, graphToolDefinitions...)
	s.writeResult(req.ID, map[string]any{"tools": all})
}

// ---------------- tools/call ----------------

type toolsCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type symbolArgs struct {
	Symbol string `json:"symbol"`
	Repo   string `json:"repo"`
}

func (s *Server) handleToolsCall(ctx context.Context, req request) {
	var p toolsCallParams
	if err := json.Unmarshal(req.Params, &p); err != nil {
		s.writeError(req.ID, -32602, "invalid params: "+err.Error(), nil)
		return
	}

	// Dispatch by tool name. Each branch dials the daemon, issues one
	// JSON-RPC call, and formats the response as a text content block.
	switch p.Name {
	case "scry_refs":
		s.callSymbolQuery(ctx, req.ID, "refs", p.Arguments)
	case "scry_defs":
		s.callSymbolQuery(ctx, req.ID, "defs", p.Arguments)
	case "scry_callers":
		s.callSymbolQuery(ctx, req.ID, "callers", p.Arguments)
	case "scry_callees":
		s.callSymbolQuery(ctx, req.ID, "callees", p.Arguments)
	case "scry_impls":
		s.callSymbolQuery(ctx, req.ID, "impls", p.Arguments)
	case "scry_tests":
		s.callSymbolQuery(ctx, req.ID, "tests", p.Arguments)
	case "scry_status":
		s.callStatus(ctx, req.ID)
	case "scry_blame":
		s.callGitQuery(ctx, req.ID, "scry_blame", "git.blame", p.Arguments)
	case "scry_history":
		s.callGitQuery(ctx, req.ID, "scry_history", "git.history", p.Arguments)
	case "scry_cochange":
		s.callGitQuery(ctx, req.ID, "scry_cochange", "git.cochange", p.Arguments)
	case "scry_hotspots":
		s.callGitQuery(ctx, req.ID, "scry_hotspots", "git.hotspots", p.Arguments)
	case "scry_contributors":
		s.callGitQuery(ctx, req.ID, "scry_contributors", "git.contributors", p.Arguments)
	case "scry_intent":
		s.callGitQuery(ctx, req.ID, "scry_intent", "git.intent", p.Arguments)
	case "scry_describe":
		s.callSchemaQuery(ctx, req.ID, "scry_describe", "schema.describe", p.Arguments)
	case "scry_relations":
		s.callSchemaQuery(ctx, req.ID, "scry_relations", "schema.relations", p.Arguments)
	case "scry_schema_search":
		s.callSchemaQuery(ctx, req.ID, "scry_schema_search", "schema.search", p.Arguments)
	case "scry_enums":
		s.callSchemaQuery(ctx, req.ID, "scry_enums", "schema.enums", p.Arguments)
	case "scry_requests":
		s.callHTTPQuery(ctx, req.ID, "scry_requests", "http.requests", p.Arguments)
	case "scry_request":
		s.callHTTPQuery(ctx, req.ID, "scry_request", "http.request", p.Arguments)
	case "scry_http_status":
		s.callHTTPQuery(ctx, req.ID, "scry_http_status", "http.status", p.Arguments)
	case "scry_graph_query":
		s.callGraphQuery(ctx, req.ID, "scry_graph_query", "graph.query", p.Arguments)
	case "scry_graph_path":
		s.callGraphQuery(ctx, req.ID, "scry_graph_path", "graph.path", p.Arguments)
	case "scry_graph_report":
		s.callGraphQuery(ctx, req.ID, "scry_graph_report", "graph.report", p.Arguments)
	default:
		s.writeToolError(req.ID, fmt.Sprintf("unknown tool %q", p.Name))
	}
}

// callSymbolQuery handles the five symbol-lookup tools. They all share the
// same argument shape (symbol + optional repo) and response shape.
//
// Compound symbol handling: scry's native index matches by display name
// (case-insensitive exact, e.g. "table" or "DB"), not by qualified
// expressions. Agents naturally ask in method-call notation ("DB::table",
// "auth->user", "user.id"), so we parse those forms here:
//
//  1. Split the input on `::`, `->`, or `.`
//  2. Try the rightmost token first (the method/property name)
//  3. If a left token exists, filter results whose symbol_id contains the
//     left token surrounded by descriptor separators (`/` or `#`). That way
//     "DB::table" narrows the 276 `table()` methods down to the ones defined
//     on a class whose descriptor ends with "/DB" or contains "/DB/".
//  4. If that filter is empty, fall back to the leftmost token alone.
//  5. If everything is empty, return the raw result for the original input
//     so the host sees an honest "no matches" rather than a false error.
func (s *Server) callSymbolQuery(ctx context.Context, id json.RawMessage, method string, rawArgs json.RawMessage) {
	start := nowUTC()

	var args symbolArgs
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		s.writeToolError(id, "invalid arguments: "+err.Error())
		return
	}
	if args.Symbol == "" {
		s.writeToolError(id, "`symbol` argument is required")
		return
	}
	repo, err := resolveRepo(args.Repo)
	if err != nil {
		s.writeToolError(id, err.Error())
		return
	}

	client, err := s.dial()
	if err != nil {
		logCall(callLogEntry{Timestamp: start.Format(time.RFC3339), Tool: "scry_" + method, Symbol: args.Symbol, Repo: repo, Results: 0, LatencyMs: time.Since(start).Milliseconds(), Error: "dial failed"})
		s.writeToolError(id, "dial scry daemon: "+err.Error())
		return
	}
	defer client.Close()

	container, tail := splitCompoundSymbol(args.Symbol)

	// Try the tail (method/property) first when we have a compound. Otherwise
	// use the input verbatim.
	first := args.Symbol
	if tail != "" {
		first = tail
	}
	raw, err := rawSymbolCall(ctx, client, method, repo, first)
	if err != nil {
		logCall(callLogEntry{Timestamp: start.Format(time.RFC3339), Tool: "scry_" + method, Symbol: args.Symbol, Repo: repo, Results: 0, LatencyMs: time.Since(start).Milliseconds(), Error: err.Error()})
		s.writeToolError(id, "scry "+method+": "+err.Error())
		return
	}

	// If the user gave us a container hint, filter to matches whose
	// symbol_id mentions the container. Keeps the signal high for
	// "DB::table" (92 DB::table sites, not 276 random table() sites).
	// If the filter wipes everything we return the empty set rather than
	// falling back to the container-only query — a "0 results" answer is
	// more useful to the caller than a pile of unrelated class references.
	if container != "" {
		if filtered, ok := filterResultByContainer(raw, container); ok {
			raw = filtered
		} else {
			raw = emptyResultJSON(args.Symbol)
		}
	}

	logCall(callLogEntry{Timestamp: start.Format(time.RFC3339), Tool: "scry_" + method, Symbol: args.Symbol, Repo: repo, Results: extractResultCount(raw), LatencyMs: time.Since(start).Milliseconds()})
	s.writeToolResult(id, prettyJSON(raw), false)
}

// emptyResultJSON returns a query.Result-shaped JSON payload with no
// matches, echoing the requested symbol so the caller sees an honest
// "I looked this up and found nothing" rather than a misleading substitute.
func emptyResultJSON(symbol string) json.RawMessage {
	out, _ := json.Marshal(map[string]any{
		"symbol":     symbol,
		"matches":    []any{},
		"total":      0,
		"elapsed_ms": 0,
	})
	return out
}

// splitCompoundSymbol parses a method-call expression into (container, tail).
// Returns empty container and empty tail if the symbol has no operator.
//
// Supported operators, in priority order: `::` (PHP static / Rust / C++),
// `->` (PHP instance), `.` (Go/JS/TS/Python/Ruby method access).
//
// Uses the LAST operator occurrence so chains like "Auth::user()->name"
// get split on the final access — the intent in that phrasing is "find
// refs to `name`, on whatever comes before".
func splitCompoundSymbol(s string) (container, tail string) {
	// Look for each separator and track the rightmost one.
	bestIdx := -1
	bestLen := 0
	for _, op := range []string{"::", "->", "."} {
		// rfindString
		if i := lastIndex(s, op); i >= 0 && i > bestIdx {
			bestIdx = i
			bestLen = len(op)
		}
	}
	if bestIdx < 0 {
		return "", ""
	}
	container = s[:bestIdx]
	tail = s[bestIdx+bestLen:]
	// Trim parentheses like "table()" → "table" so scry's display-name
	// match works.
	if i := indexByte(tail, '('); i >= 0 {
		tail = tail[:i]
	}
	if container == "" || tail == "" {
		return "", ""
	}
	return container, tail
}

// filterResultByContainer returns (filteredJSON, true) if any match's
// symbol_id contains the container as a descriptor segment, or (nil, false)
// if nothing matches. Matches are split into per-match filtering: one match
// record may have some occurrences and survive, so we keep it.
//
// Container matching is case-sensitive against the symbol_id's descriptor.
// We wrap in `/.../` to enforce segment boundaries — "DB" should match
// `Illuminate/Support/Facades/DB#` but not `Illuminate/Database/...`.
func filterResultByContainer(raw json.RawMessage, container string) (json.RawMessage, bool) {
	var result struct {
		Symbol    string            `json:"symbol"`
		Matches   []json.RawMessage `json:"matches"`
		Total     int               `json:"total"`
		ElapsedMs int               `json:"elapsed_ms"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, false
	}

	keep := make([]json.RawMessage, 0, len(result.Matches))
	for _, mRaw := range result.Matches {
		var m struct {
			SymbolID string `json:"symbol_id"`
		}
		if err := json.Unmarshal(mRaw, &m); err != nil {
			continue
		}
		if symbolIDMatchesContainer(m.SymbolID, container) {
			keep = append(keep, mRaw)
		}
	}
	if len(keep) == 0 {
		return nil, false
	}
	result.Matches = keep
	result.Total = len(keep)
	out, err := json.Marshal(result)
	if err != nil {
		return nil, false
	}
	return out, true
}

// symbolIDMatchesContainer is a cheap substring check that requires the
// container to appear between descriptor separators — so "DB" matches
// `Illuminate/Support/Facades/DB#method()` and `.../Facades/DB/Builder#...`
// but not `Illuminate/Database/Eloquent/Model#method()`.
func symbolIDMatchesContainer(symbolID, container string) bool {
	// Normalize: look for /container# or /container/ anywhere in the id.
	needle1 := "/" + container + "#"
	needle2 := "/" + container + "/"
	// Also allow a leading match for un-namespaced classes: "DB#..." at
	// the start of a descriptor token.
	needle3 := " " + container + "#"
	needle4 := " " + container + "/"
	return containsAny(symbolID, needle1, needle2, needle3, needle4)
}

// rawSymbolCall issues one daemon RPC and returns the raw JSON result.
func rawSymbolCall(ctx context.Context, client Dialer, method, repo, name string) (json.RawMessage, error) {
	var raw json.RawMessage
	params := map[string]string{"repo": repo, "name": name}
	if err := client.Call(ctx, method, params, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

// prettyJSON re-indents raw JSON; returns raw text if anything goes wrong.
func prettyJSON(raw json.RawMessage) string {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return string(raw)
	}
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return string(raw)
	}
	return string(out)
}

// --- tiny local string helpers to keep the mcp package dependency-light ---

func lastIndex(s, substr string) int {
	if len(substr) == 0 || len(substr) > len(s) {
		return -1
	}
	for i := len(s) - len(substr); i >= 0; i-- {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func indexByte(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

func containsAny(s string, needles ...string) bool {
	for _, n := range needles {
		if len(n) == 0 {
			continue
		}
		for i := 0; i <= len(s)-len(n); i++ {
			if s[i:i+len(n)] == n {
				return true
			}
		}
	}
	return false
}

// callStatus returns the daemon's status RPC verbatim.
func (s *Server) callStatus(ctx context.Context, id json.RawMessage) {
	start := nowUTC()

	client, err := s.dial()
	if err != nil {
		logCall(callLogEntry{Timestamp: start.Format(time.RFC3339), Tool: "scry_status", LatencyMs: time.Since(start).Milliseconds(), Error: "dial failed"})
		s.writeToolError(id, "dial scry daemon: "+err.Error())
		return
	}
	defer client.Close()

	var raw json.RawMessage
	if err := client.Call(ctx, "status", struct{}{}, &raw); err != nil {
		logCall(callLogEntry{Timestamp: start.Format(time.RFC3339), Tool: "scry_status", LatencyMs: time.Since(start).Milliseconds(), Error: err.Error()})
		s.writeToolError(id, "scry status: "+err.Error())
		return
	}
	var any any
	_ = json.Unmarshal(raw, &any)
	pretty, err := json.MarshalIndent(any, "", "  ")
	if err != nil {
		pretty = []byte(raw)
	}
	logCall(callLogEntry{Timestamp: start.Format(time.RFC3339), Tool: "scry_status", LatencyMs: time.Since(start).Milliseconds()})
	s.writeToolResult(id, string(pretty), false)
}

// writeToolResult wraps a text payload in the MCP content-block format.
func (s *Server) writeToolResult(id json.RawMessage, text string, isError bool) {
	s.writeResult(id, map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": text},
		},
		"isError": isError,
	})
}

// writeToolError returns a tool-level error (distinct from a JSON-RPC error:
// MCP tools report their own failures via an isError=true content block).
func (s *Server) writeToolError(id json.RawMessage, msg string) {
	s.writeToolResult(id, msg, true)
}

// resolveRepo picks the right repo path for a symbol query. If the tool call
// passed `repo`, we use it (after making it absolute). Otherwise we fall back
// to the MCP server's current working directory — that matches the cwd
// Claude Code passes when launching the server.
func resolveRepo(requested string) (string, error) {
	if requested != "" {
		if !filepath.IsAbs(requested) {
			abs, err := filepath.Abs(requested)
			if err != nil {
				return "", fmt.Errorf("resolve repo: %w", err)
			}
			return abs, nil
		}
		return requested, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve cwd: %w", err)
	}
	return cwd, nil
}

func mustMarshal(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
