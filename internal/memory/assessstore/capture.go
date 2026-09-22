package assessstore

import (
	"github.com/jeffdhooton/scry/internal/memory/distill"
	"github.com/jeffdhooton/scry/internal/memory/store"
)

// CapturePending handles legacy queue rows too; absent namespace/order remains
// absent so a remote client's path never becomes an inferred local session.
func (s *Store) CapturePending(p store.PendingEpisode, m SourceMetadata) (Source, error) {
	return s.CaptureRaw(distill.RawEpisode{ID: p.ID, Source: p.Source, SourceRef: p.SourceRef, Text: p.Text, OccurredAt: p.OccurredAt, Cwd: p.Cwd, CwdIsRepo: p.CwdIsRepo, SourceNamespace: p.SourceNamespace, SourceSpanKnown: p.SourceSpanKnown, SourceStart: p.SourceStart, SourceEnd: p.SourceEnd, SourceTurns: p.SourceTurns}, m)
}
func (s *Store) CaptureRaw(p distill.RawEpisode, m SourceMetadata) (Source, error) {
	if m.Namespace == "" {
		m.Namespace = p.SourceNamespace
	}
	if m.SessionID == "" && m.Namespace != "" {
		m.SessionID = SessionReference(p.SourceRef)
	}
	if !m.SpanKnown && p.SourceSpanKnown {
		m.SpanKnown = true
		m.Start = p.SourceStart
		m.End = p.SourceEnd
	}
	if m.Cwd == "" {
		m.Cwd = p.Cwd
		m.CwdIsRepo = p.CwdIsRepo
	}
	src := Source{EpisodeID: p.ID, Source: p.Source, SourceRef: p.SourceRef, Text: p.Text, OccurredAt: p.OccurredAt, SourceMetadata: m}
	for _, t := range p.SourceTurns {
		src.Turns = append(src.Turns, SourceTurn{Speaker: t.Speaker, Text: t.Text, Start: t.Start, End: t.End})
	}
	return s.Capture(src)
}
