// Tiny PHP scanner for the non-PSR-4 walker.
//
// We need to extract two things from a PHP file:
//
//  1. `use` declarations (class aliases)
//  2. `Foo\Bar::class` references with their source positions
//
// A full PHP parser would be overkill. This scanner walks bytes,
// understands enough syntax to skip strings/comments/heredocs without
// false-matching `::class` inside them, and emits the two structures the
// walker needs.
//
// What we deliberately don't handle:
//   - PHP attributes (#[Attribute]) — we treat the # as a line comment which
//     swallows them; refs inside attributes are not extracted. Routes don't
//     use attributes for their controller bindings, so this is fine.
//   - Conditional class loading (`if ($cond) class Foo {}`) — irrelevant here.
//   - Variable variables, eval'd strings — irrelevant here.
package php

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

type classRef struct {
	name string // raw token, may be qualified or absolute
	line int    // 1-indexed
	col  int    // 1-indexed
}

// stringCallRef captures a function-style call where the first argument is
// a string literal — i.e. `view('users.show')`, `config('mail.from.address')`,
// `__('messages.welcome')`. The walker uses these to synthesize blade-file
// and config-file edges that scip-php's static analysis can't see.
type stringCallRef struct {
	funcName string // identifier the call was made on (e.g. "view", "config")
	value    string // the literal string contents (single- or double-quoted, escapes resolved best-effort)
	line     int    // 1-indexed line of the literal's opening quote
	col      int    // 1-indexed column of the literal's opening quote
}

type phpScanner struct {
	src   []byte
	pos   int
	line  int // 1-indexed
	col   int // 1-indexed
	inPhp bool
}

func newPhpScanner(src []byte) *phpScanner {
	return &phpScanner{
		src:   src,
		line:  1,
		col:   1,
		inPhp: false,
	}
}

// scanResult is what the scanner returns after one pass over a file.
type scanResult struct {
	uses       map[string]string
	classRefs  []classRef
	stringRefs []stringCallRef
}

// collect runs the scanner across the source and returns everything we
// extract: the use map, every `::class` reference, and every
// `funcname('literal')` call site. Errors are silent — partial output is the
// right default for a best-effort post-processor.
func (s *phpScanner) collect() scanResult {
	res := scanResult{uses: map[string]string{}}
	for s.pos < len(s.src) {
		if !s.inPhp {
			s.scanUntilPhpOpen()
			continue
		}
		c := s.src[s.pos]
		switch {
		case c == '?' && s.peekAt(1) == '>':
			s.inPhp = false
			s.advance(2)
		case c == '/' && s.peekAt(1) == '/':
			s.skipLineComment()
		case c == '#':
			// Could be a PHP attribute (#[...]) or a line comment. Either way
			// we skip to end of line — attributes can span lines but we don't
			// extract refs from them anyway.
			s.skipLineComment()
		case c == '/' && s.peekAt(1) == '*':
			s.skipBlockComment()
		case c == '\'':
			s.skipSingleQuotedString()
		case c == '"':
			s.skipDoubleQuotedString()
		case c == '<' && s.peekAt(1) == '<' && s.peekAt(2) == '<':
			s.skipHeredoc()
		case isIdentStartByte(c, s.src[s.pos:]):
			before := s.pos
			s.scanIdentifierOrKeyword(&res)
			// Defense against the rune-widening trap: if a multibyte char's
			// first byte happens to be in the Latin-1 letter range
			// (e.g. `\xE2` → rune 0xE2 → unicode.IsLetter true) but the
			// underlying utf8.DecodeRune returns RuneError because the
			// sequence is incomplete or malformed, scanIdentifierOrKeyword
			// returns without advancing. Force progress so the main loop
			// can never spin in place.
			if s.pos == before {
				s.advance(1)
			}
		case c == '\\' && s.pos+1 < len(s.src) && isIdentStartByte(s.src[s.pos+1], s.src[s.pos+1:]):
			// Absolute name like \Foo\Bar — treat as identifier.
			before := s.pos
			s.scanIdentifierOrKeyword(&res)
			if s.pos == before {
				s.advance(1)
			}
		default:
			s.advance(1)
		}
	}
	return res
}

func (s *phpScanner) advance(n int) {
	for i := 0; i < n && s.pos < len(s.src); i++ {
		if s.src[s.pos] == '\n' {
			s.line++
			s.col = 1
		} else {
			s.col++
		}
		s.pos++
	}
}

func (s *phpScanner) peekAt(off int) byte {
	if s.pos+off >= len(s.src) {
		return 0
	}
	return s.src[s.pos+off]
}

func (s *phpScanner) scanUntilPhpOpen() {
	idx := strings.Index(string(s.src[s.pos:]), "<?php")
	if idx < 0 {
		// Also accept short tag <?
		idx = strings.Index(string(s.src[s.pos:]), "<?")
		if idx < 0 {
			s.advance(len(s.src) - s.pos)
			return
		}
		s.advance(idx + 2)
		s.inPhp = true
		return
	}
	s.advance(idx + 5)
	s.inPhp = true
}

func (s *phpScanner) skipLineComment() {
	for s.pos < len(s.src) && s.src[s.pos] != '\n' {
		s.advance(1)
	}
}

func (s *phpScanner) skipBlockComment() {
	s.advance(2) // skip /*
	for s.pos+1 < len(s.src) {
		if s.src[s.pos] == '*' && s.src[s.pos+1] == '/' {
			s.advance(2)
			return
		}
		s.advance(1)
	}
	s.advance(len(s.src) - s.pos)
}

func (s *phpScanner) skipSingleQuotedString() {
	s.advance(1)
	for s.pos < len(s.src) {
		c := s.src[s.pos]
		if c == '\\' && s.pos+1 < len(s.src) {
			s.advance(2)
			continue
		}
		if c == '\'' {
			s.advance(1)
			return
		}
		s.advance(1)
	}
}

func (s *phpScanner) skipDoubleQuotedString() {
	s.advance(1)
	for s.pos < len(s.src) {
		c := s.src[s.pos]
		if c == '\\' && s.pos+1 < len(s.src) {
			s.advance(2)
			continue
		}
		if c == '"' {
			s.advance(1)
			return
		}
		s.advance(1)
	}
}

func (s *phpScanner) skipHeredoc() {
	// Heredoc: <<<LABEL\n...\nLABEL;  (or <<<'LABEL'\n...\nLABEL; for nowdoc)
	s.advance(3)
	// Skip optional quote.
	if s.pos < len(s.src) && (s.src[s.pos] == '\'' || s.src[s.pos] == '"') {
		s.advance(1)
	}
	// Read label until newline or quote.
	labelStart := s.pos
	for s.pos < len(s.src) {
		c := s.src[s.pos]
		if c == '\n' || c == '\'' || c == '"' {
			break
		}
		s.advance(1)
	}
	label := string(s.src[labelStart:s.pos])
	// Consume the rest of the opening line.
	for s.pos < len(s.src) && s.src[s.pos] != '\n' {
		s.advance(1)
	}
	if label == "" {
		return
	}
	// Look for the closing label at the start of a line. PHP 7.3+ allows
	// indentation; we accept any leading whitespace.
	for s.pos < len(s.src) {
		// Move to next line start.
		for s.pos < len(s.src) && s.src[s.pos] != '\n' {
			s.advance(1)
		}
		if s.pos >= len(s.src) {
			return
		}
		s.advance(1) // consume newline
		// Skip leading whitespace.
		lineStart := s.pos
		for lineStart < len(s.src) && (s.src[lineStart] == ' ' || s.src[lineStart] == '\t') {
			lineStart++
		}
		if lineStart+len(label) <= len(s.src) && string(s.src[lineStart:lineStart+len(label)]) == label {
			// Match if followed by `;`, `,`, `)`, end-of-line, etc.
			after := lineStart + len(label)
			if after >= len(s.src) || !isIdentPart(rune(s.src[after])) {
				s.advance(after - s.pos)
				return
			}
		}
	}
}

// scanIdentifierOrKeyword reads an identifier (which may be qualified by
// backslashes) and either records a `use` clause, a `::class` reference,
// a `funcname('literal')` call, or just consumes it.
func (s *phpScanner) scanIdentifierOrKeyword(res *scanResult) {
	startLine, startCol := s.line, s.col
	identStart := s.pos
	// Allow leading backslash for absolute names.
	if s.src[s.pos] == '\\' {
		s.advance(1)
	}
	for s.pos < len(s.src) {
		r, size := utf8.DecodeRune(s.src[s.pos:])
		if isIdentPart(r) || r == '\\' {
			s.advance(size)
			continue
		}
		break
	}
	ident := string(s.src[identStart:s.pos])
	if ident == "" {
		return
	}

	lower := strings.ToLower(ident)
	switch lower {
	case "use":
		s.handleUse(res.uses)
		return
	case "namespace":
		s.handleNamespace()
		return
	}

	// Look ahead for `::class`. Whitespace is allowed between `::` and `class`.
	save := s.pos
	s.skipInlineWhitespace()
	if s.pos+1 < len(s.src) && s.src[s.pos] == ':' && s.src[s.pos+1] == ':' {
		s.advance(2)
		s.skipInlineWhitespace()
		// Read next identifier word
		wordStart := s.pos
		for s.pos < len(s.src) && isIdentPart(rune(s.src[s.pos])) {
			s.advance(1)
		}
		word := string(s.src[wordStart:s.pos])
		if word == "class" {
			res.classRefs = append(res.classRefs, classRef{
				name: ident,
				line: startLine,
				col:  startCol,
			})
			return
		}
		// Not ::class — rewind and fall through to the call-site check below.
		s.pos = save
		s.skipInlineWhitespace()
	}

	// Look ahead for `(` followed by a string literal: `view('users.show', ...)`.
	// We only care about UNqualified identifiers here — namespace-prefixed
	// idents like `Foo\bar('x')` are valid PHP, but our consumers (view,
	// config, __, trans) are all root-namespace globals.
	if !strings.Contains(ident, "\\") && s.pos < len(s.src) && s.src[s.pos] == '(' {
		s.advance(1) // (
		s.skipInlineAndNewlineWhitespace()
		if s.pos < len(s.src) && (s.src[s.pos] == '\'' || s.src[s.pos] == '"') {
			litLine, litCol := s.line, s.col
			value, ok := s.consumeStringLiteral()
			if ok {
				res.stringRefs = append(res.stringRefs, stringCallRef{
					funcName: ident,
					value:    value,
					line:     litLine,
					col:      litCol,
				})
				return
			}
		}
		return
	}

	// Not a recognized pattern — rewind so the main loop sees the next char.
	s.pos = save
}

// consumeStringLiteral reads a single- or double-quoted string starting at
// s.pos and returns its decoded value. The opening quote is at s.pos. The
// scanner advances past the closing quote.
//
// We don't need PHP-perfect escape handling — only the literal contents,
// for matching against view/config/etc. keys. We do unescape `\\`, `\'`,
// and `\"` because those affect the string's identity.
func (s *phpScanner) consumeStringLiteral() (string, bool) {
	if s.pos >= len(s.src) {
		return "", false
	}
	q := s.src[s.pos]
	if q != '\'' && q != '"' {
		return "", false
	}
	s.advance(1)
	var b strings.Builder
	for s.pos < len(s.src) {
		c := s.src[s.pos]
		if c == '\\' && s.pos+1 < len(s.src) {
			next := s.src[s.pos+1]
			if next == q || next == '\\' {
				b.WriteByte(next)
				s.advance(2)
				continue
			}
			// Other escapes are passed through unchanged — for our purposes
			// (matching dot-notation keys) we never expect them.
			b.WriteByte(c)
			s.advance(1)
			continue
		}
		if c == q {
			s.advance(1)
			return b.String(), true
		}
		// Reject newlines and dollar-sign interpolation in double-quoted
		// strings — those mean the string isn't a static literal.
		if c == '\n' {
			return "", false
		}
		if q == '"' && c == '$' {
			return "", false
		}
		b.WriteByte(c)
		s.advance(1)
	}
	return "", false
}

// handleUse parses a use statement starting just after the `use` keyword.
// Supports:
//
//	use Foo\Bar;
//	use Foo\Bar as Baz;
//	use Foo\{A, B as C, D};
//	use function Foo\bar;       (skipped — function imports)
//	use const Foo\BAR;          (skipped — const imports)
func (s *phpScanner) handleUse(uses map[string]string) {
	s.skipInlineWhitespace()
	if s.pos >= len(s.src) {
		return
	}

	// Detect `use function` / `use const` and skip them — we don't track
	// non-class imports.
	if s.matchKeyword("function") || s.matchKeyword("const") {
		s.skipUntilSemicolon()
		return
	}

	// Read the leading namespace prefix until we hit `{`, `as`, `,`, or `;`.
	prefix := s.readNamespacePart()
	s.skipInlineWhitespace()

	if s.pos < len(s.src) && s.src[s.pos] == '{' {
		// Group use: use Foo\{A, B as C, D};
		s.advance(1) // {
		for s.pos < len(s.src) {
			s.skipInlineAndNewlineWhitespace()
			if s.pos >= len(s.src) || s.src[s.pos] == '}' {
				break
			}
			part := s.readNamespacePart()
			alias := lastSegment("\\" + part)
			s.skipInlineAndNewlineWhitespace()
			if s.matchKeyword("as") {
				s.skipInlineWhitespace()
				alias = s.readNamespacePart()
			}
			if part != "" {
				base := strings.TrimSuffix(strings.TrimPrefix(prefix, "\\"), "\\")
				tail := strings.TrimPrefix(part, "\\")
				uses[alias] = base + "\\" + tail
			}
			s.skipInlineAndNewlineWhitespace()
			if s.pos < len(s.src) && s.src[s.pos] == ',' {
				s.advance(1)
				continue
			}
			break
		}
		// Consume up to semicolon.
		s.skipUntilSemicolon()
		return
	}

	// Single use: maybe with `as`.
	alias := ""
	if s.matchKeyword("as") {
		s.skipInlineWhitespace()
		alias = s.readNamespacePart()
	}
	if alias == "" {
		alias = lastSegment("\\" + prefix)
	}
	if prefix != "" {
		uses[alias] = strings.TrimPrefix(prefix, "\\")
	}
	s.skipUntilSemicolon()
}

// handleNamespace consumes a namespace declaration. We don't currently use
// the namespace name (the four target dirs are root-namespace files), but
// we still need to consume the tokens so the main loop doesn't see them
// as identifiers and try to interpret `namespace` as a class.
func (s *phpScanner) handleNamespace() {
	s.skipInlineWhitespace()
	_ = s.readNamespacePart()
	s.skipInlineWhitespace()
	// Either `;` (file-scope) or `{` (block-scope). Consume the delimiter
	// and let the main loop continue.
	if s.pos < len(s.src) && (s.src[s.pos] == ';' || s.src[s.pos] == '{') {
		s.advance(1)
	}
}

// readNamespacePart reads a (possibly qualified) name like `Foo\Bar\Baz`.
func (s *phpScanner) readNamespacePart() string {
	start := s.pos
	for s.pos < len(s.src) {
		c := s.src[s.pos]
		if isIdentPart(rune(c)) || c == '\\' {
			s.advance(1)
			continue
		}
		break
	}
	return string(s.src[start:s.pos])
}

func (s *phpScanner) matchKeyword(kw string) bool {
	if s.pos+len(kw) > len(s.src) {
		return false
	}
	if !strings.EqualFold(string(s.src[s.pos:s.pos+len(kw)]), kw) {
		return false
	}
	if s.pos+len(kw) < len(s.src) && isIdentPart(rune(s.src[s.pos+len(kw)])) {
		return false
	}
	s.advance(len(kw))
	return true
}

func (s *phpScanner) skipInlineWhitespace() {
	for s.pos < len(s.src) {
		c := s.src[s.pos]
		if c == ' ' || c == '\t' {
			s.advance(1)
			continue
		}
		break
	}
}

func (s *phpScanner) skipInlineAndNewlineWhitespace() {
	for s.pos < len(s.src) {
		c := s.src[s.pos]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			s.advance(1)
			continue
		}
		break
	}
}

func (s *phpScanner) skipUntilSemicolon() {
	for s.pos < len(s.src) {
		c := s.src[s.pos]
		if c == ';' {
			s.advance(1)
			return
		}
		s.advance(1)
	}
}

func isIdentStart(r rune) bool {
	return r == '_' || unicode.IsLetter(r)
}

func isIdentPart(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

// isIdentStartByte answers "does the byte b start a PHP identifier?", reading
// just enough subsequent bytes (via tail) to handle multibyte UTF-8 properly.
// We can't naively zero-extend a byte to a rune because Latin-1 letters
// (0xC0..0xFF) include the leading bytes of UTF-8 sequences that aren't
// themselves identifier characters in the runtime — and an undecodable
// sequence would otherwise misroute into the identifier scanner.
func isIdentStartByte(b byte, tail []byte) bool {
	if b < 0x80 {
		return b == '_' || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
	}
	r, size := utf8.DecodeRune(tail)
	if size == 0 || r == utf8.RuneError {
		return false
	}
	return unicode.IsLetter(r)
}

func trimLeading(s string) string { return strings.TrimPrefix(s, "\\") }
