package distill

import (
	"os"
	"strings"
)

// seedSource is the Source value stamped on episodes produced by
// SeedMarkdown.
const seedSource = "seed"

// SeedMarkdown distills a single hand-authored "seed" markdown file (e.g. a
// Claude skill's SKILL.md) into one RawEpisode. A seed file is expected to
// carry YAML frontmatter bounded by "---" lines with at least "name" and
// "description" keys, followed by a markdown body. The episode text is the
// description followed by the body; the frontmatter block itself (name,
// and any other keys) is not conversational content and is dropped once
// its description has been pulled out.
//
// Parsing is tolerant, matching the rest of this package: a file with no
// frontmatter block at all is not an error, its entire content is simply
// treated as the body with no description.
func SeedMarkdown(path string) (RawEpisode, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return RawEpisode{}, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return RawEpisode{}, err
	}

	description, body := parseSeedFrontmatter(string(data))

	var b strings.Builder
	if description != "" {
		b.WriteString(description)
		b.WriteString("\n\n")
	}
	b.WriteString(body)

	return RawEpisode{
		ID:         makeID(path),
		Source:     seedSource,
		SourceRef:  path,
		Text:       Redact(strings.TrimSpace(b.String())),
		OccurredAt: info.ModTime(),
	}, nil
}

// parseSeedFrontmatter splits content into its YAML frontmatter's
// "description" field and the markdown body that follows. Frontmatter is
// recognized only when the very first line is exactly "---" and a matching
// "---" line closes it further down; otherwise the whole content is
// returned as the body with an empty description. Frontmatter values are
// parsed with a simple "key: value" line scan (sufficient for the
// single-line name/description fields these files use), not a full YAML
// parser.
func parseSeedFrontmatter(content string) (description, body string) {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return "", content
	}

	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			end = i
			break
		}
	}
	if end < 0 {
		return "", content
	}

	for _, line := range lines[1:end] {
		trimmed := strings.TrimSpace(line)
		if v, ok := strings.CutPrefix(trimmed, "description:"); ok {
			description = seedUnquote(strings.TrimSpace(v))
			break
		}
	}

	body = strings.TrimLeft(strings.Join(lines[end+1:], "\n"), "\n")
	return description, body
}

// seedUnquote strips a single layer of matching double or single quotes
// from a YAML scalar value, if present.
func seedUnquote(s string) string {
	if len(s) >= 2 {
		first, last := s[0], s[len(s)-1]
		if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}
