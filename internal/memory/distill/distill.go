// Package distill turns raw agent-session transcripts (JSONL from Claude
// Code, and eventually other sources) into small, redacted text episodes,
// entirely in pure Go and before any LLM sees the content. It is the first
// stage of scry's episodic memory pipeline: distill -> extract -> ingest.
//
// Distillation is deliberately dumb and cheap: tolerant JSONL parsing, turn
// extraction, secret redaction, and turn-boundary chunking. No network
// calls, no LLM calls, no dependency on the rest of scry.
package distill

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// RawEpisode is a distilled, redacted slice of a session transcript, ready
// to be handed to an LLM-based extractor. Text is plain conversational
// text with tool_use breadcrumbs; tool_result bodies are never included.
type RawEpisode struct {
	ID         string // sha256 hex of SourceRef
	Source     string // "claude-session" etc.
	SourceRef  string // "<path>#<startByte>-<endByte>"
	Text       string // distilled, redacted conversational text
	OccurredAt time.Time
	Cwd        string // session working directory, "" if unknown
}

// maxEpisodeChars bounds the size of a single episode's Text (~4k tokens).
// Longer conversations are split on turn boundaries with a 1-turn overlap
// between consecutive episodes so extraction never loses context at a seam.
const maxEpisodeChars = 16000

// turn is one user or assistant message, already reduced to plain text
// (tool_use blocks rendered as breadcrumbs, tool_result content dropped),
// along with the byte range it occupied in the source file.
type turn struct {
	role  string // "user" or "assistant"
	text  string // pre-redaction turn text; never empty for a substantive turn
	start int64  // byte offset of the start of the source line, inclusive
	end   int64  // byte offset of the end of the source line, exclusive
	ts    time.Time
	cwd   string
}

// renderTurn formats a turn for inclusion in episode text.
func renderTurn(t turn) string {
	label := "User"
	if t.role == "assistant" {
		label = "Assistant"
	} else if t.role != "user" {
		label = t.role
	}
	return label + ": " + t.text + "\n\n"
}

// makeID derives a stable episode ID from its SourceRef.
func makeID(sourceRef string) string {
	sum := sha256.Sum256([]byte(sourceRef))
	return hex.EncodeToString(sum[:])
}

// chunkTurns assembles turns into one or more RawEpisodes, each bounded by
// maxEpisodeChars, splitting only on turn boundaries. Consecutive episodes
// share exactly one overlapping turn so extraction has context across the
// seam. IDs are deterministic: same turns in, same IDs out.
func chunkTurns(source, path string, turns []turn) []RawEpisode {
	var episodes []RawEpisode
	n := len(turns)
	if n == 0 {
		return episodes
	}

	i := 0
	for i < n {
		var sb strings.Builder
		j := i
		for j < n {
			piece := renderTurn(turns[j])
			if sb.Len() > 0 && sb.Len()+len(piece) > maxEpisodeChars {
				break
			}
			sb.WriteString(piece)
			j++
		}
		if j == i {
			// A single turn alone exceeds maxEpisodeChars; include it
			// anyway so we always make forward progress.
			sb.WriteString(renderTurn(turns[i]))
			j = i + 1
		}
		last := j - 1

		sourceRef := fmt.Sprintf("%s#%d-%d", path, turns[i].start, turns[last].end)
		episodes = append(episodes, RawEpisode{
			ID:         makeID(sourceRef),
			Source:     source,
			SourceRef:  sourceRef,
			Text:       Redact(sb.String()),
			OccurredAt: turns[i].ts,
			Cwd:        turns[i].cwd,
		})

		if j >= n {
			break
		}
		if last > i {
			// 1-turn overlap: the next episode starts at the last turn of
			// this one.
			i = last
		} else {
			i = last + 1
		}
	}

	return episodes
}
