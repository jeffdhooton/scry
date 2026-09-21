// Package assessstore owns the independent, redacted assessment sidecar.
// Logical limits exclude Badger overhead; value-log collection may lag deletion.
package assessstore

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dgraph-io/badger/v4"
	"github.com/jeffdhooton/scry/internal/memory/assess"
	"github.com/jeffdhooton/scry/internal/memory/distill"
	"github.com/jeffdhooton/scry/internal/memory/extract"
	"math"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	ErrNotFound            = errors.New("assessstore: not found")
	ErrCapacity            = errors.New("assessstore: capacity exhausted")
	ErrOwnership           = errors.New("assessstore: claim ownership mismatch")
	ErrEvidenceUnavailable = errors.New("assessstore: evidence unavailable")
	ErrBlocked             = errors.New("assessstore: dispatch blocked")
	ErrCooldown            = errors.New("assessstore: cooldown active")
	ErrReadOnly            = errors.New("assessstore: read only")
	ErrConflict            = errors.New("assessstore: immutable content conflict")
)

type State string

const (
	Pending   State = "pending"
	Running   State = "running"
	Completed State = "completed"
	Failed    State = "failed"
	Blocked   State = "blocked"
	Oversize  State = "oversize"
)

type SourceMetadata struct {
	Namespace       string `json:"namespace,omitempty"`
	SessionID       string `json:"session_id,omitempty"`
	Order           int64  `json:"order,omitempty"`
	OrderKnown      bool   `json:"order_known,omitempty"`
	SpanKnown       bool   `json:"span_known,omitempty"`
	Start           int64  `json:"start,omitempty"`
	End             int64  `json:"end,omitempty"`
	RepositoryScope string `json:"repository_scope,omitempty"`
	Cwd             string `json:"cwd,omitempty"`
	CwdIsRepo       bool   `json:"cwd_is_repo,omitempty"`
}
type SourceTurn struct {
	Speaker string `json:"speaker"`
	Text    string `json:"text"`
	Start   int64  `json:"start"`
	End     int64  `json:"end"`
}
type Source struct {
	Kind              string       `json:"kind"`
	SourceUnavailable bool         `json:"source_unavailable"`
	ID                string       `json:"id"`
	EpisodeID         string       `json:"episode_id"`
	Source            string       `json:"source"`
	SourceRef         string       `json:"source_ref"`
	Text              string       `json:"text,omitempty"`
	Turns             []SourceTurn `json:"turns,omitempty"`
	OccurredAt        time.Time    `json:"occurred_at"`
	CapturedAt        time.Time    `json:"captured_at"`
	Revision          uint64       `json:"revision"`
	Digest            string       `json:"digest"`
	Unavailable       bool         `json:"payload_unavailable"`
	SourceMetadata
}
type Versions struct {
	Model         string `json:"model"`
	Rubric        string `json:"rubric"`
	ContextPolicy string `json:"context_policy"`
}
type Job struct {
	ID                  string             `json:"id"`
	CandidateID         string             `json:"candidate_id"`
	EpisodeID           string             `json:"episode_id"`
	SourceID            string             `json:"source_id"`
	ExtractionDigest    string             `json:"extraction_digest"`
	Extraction          json.RawMessage    `json:"extraction,omitempty"`
	Ordinal             int                `json:"ordinal"`
	Candidate           extract.Fct        `json:"candidate"`
	SourceCutoff        uint64             `json:"source_cutoff"`
	Versions            Versions           `json:"versions"`
	Status              State              `json:"status"`
	Owner               string             `json:"owner,omitempty"`
	Error               string             `json:"error,omitempty"`
	CreatedAt           time.Time          `json:"created_at"`
	UpdatedAt           time.Time          `json:"updated_at"`
	DispatchStartedAt   time.Time          `json:"dispatch_started_at"`
	ParentID            string             `json:"parent_id,omitempty"`
	Attempt             uint64             `json:"attempt"`
	Packet              []byte             `json:"packet,omitempty"`
	PacketHash          string             `json:"packet_hash,omitempty"`
	Manifest            assess.Manifest    `json:"manifest"`
	Assessment          *assess.Assessment `json:"assessment,omitempty"`
	EvidenceUnavailable bool               `json:"evidence_unavailable"`
	// TargetSourceUnavailable is missing raw source at capture time. It is
	// separate from EvidenceUnavailable, the retention payload-cleared flag.
	TargetSourceUnavailable bool `json:"target_source_unavailable"`
}
type Limits struct {
	PendingJobs   int
	PendingBytes  int64
	PayloadBytes  int64
	MetadataBytes int64
	PayloadAge    time.Duration
	MetadataAge   time.Duration
}
type Options struct {
	ReadOnly bool
	Now      func() time.Time
	Limits   Limits
}
type Store struct {
	db   *badger.DB
	mu   sync.Mutex
	opts Options
}
type Status struct {
	Revision       uint64            `json:"revision"`
	Counts         map[State]int     `json:"counts"`
	PendingBytes   int64             `json:"pending_bytes"`
	PayloadBytes   int64             `json:"payload_bytes"`
	MetadataBytes  int64             `json:"metadata_bytes"`
	CoverageGaps   map[string]uint64 `json:"coverage_gaps"`
	BlockedReason  string            `json:"blocked_reason,omitempty"`
	RetryAfter     time.Time         `json:"retry_after"`
	ReplaySequence uint64            `json:"replay_sequence"`
	Dispatches     uint64            `json:"dispatches"`
}

func Open(path string, o Options) (*Store, error) {
	if o.Now == nil {
		o.Now = time.Now
	}
	l := &o.Limits
	if l.PendingJobs == 0 {
		l.PendingJobs = 10000
	}
	if l.PendingBytes == 0 {
		l.PendingBytes = 256 << 20
	}
	if l.PayloadBytes == 0 {
		l.PayloadBytes = 512 << 20
	}
	if l.MetadataBytes == 0 {
		l.MetadataBytes = 128 << 20
	}
	if l.PayloadAge == 0 {
		l.PayloadAge = 30 * 24 * time.Hour
	}
	if l.MetadataAge == 0 {
		l.MetadataAge = 90 * 24 * time.Hour
	}
	if o.ReadOnly {
		if _, e := os.Stat(path); e != nil {
			return nil, e
		}
	} else if e := os.MkdirAll(path, 0700); e != nil {
		return nil, e
	}
	// A committed reservation must survive a crash before its external request.
	db, e := badger.Open(badger.DefaultOptions(path).WithLogger(nil).WithReadOnly(o.ReadOnly).WithSyncWrites(true))
	if e != nil {
		return nil, e
	}
	s := &Store{db: db, opts: o}
	e = db.View(func(tx *badger.Txn) error {
		var v int
		e := get(tx, "schema", &v)
		if errors.Is(e, ErrNotFound) && !o.ReadOnly {
			it := tx.NewIterator(badger.DefaultIteratorOptions)
			defer it.Close()
			it.Rewind()
			if it.Valid() {
				return errors.New("assessstore: missing schema marker")
			}
			return nil
		}
		if e != nil {
			return e
		}
		if v != 1 {
			return errors.New("assessstore: incompatible schema")
		}
		return nil
	})
	if e != nil {
		db.Close()
		return nil, e
	}
	if !o.ReadOnly {
		e = s.write(func(tx *badger.Txn, st *Status) error {
			if e := put(tx, "schema", 1); e != nil {
				return e
			}
			// Recompute reservations when opening older schema-1 sidecars that
			// predate reserved completion accounting. Existing pending work is
			// honored; additional intake remains capacity-limited.
			st.MetadataBytes = 0
			if e := scan(tx, "s:", func(k string, b []byte) error {
				var v Source
				if e := json.Unmarshal(b, &v); e != nil {
					return e
				}
				st.MetadataBytes += sourceMeta(v)
				return nil
			}); e != nil {
				return e
			}
			if e := scan(tx, "j:", func(k string, b []byte) error {
				var j Job
				if e := json.Unmarshal(b, &j); e != nil {
					return e
				}
				st.MetadataBytes += jobMeta(j)
				return nil
			}); e != nil {
				return e
			}
			return scan(tx, "j:", func(k string, b []byte) error {
				var j Job
				if e := json.Unmarshal(b, &j); e != nil {
					return e
				}
				if j.Status == Running {
					old := j
					j.Status = Failed
					j.Error = "interrupted_delivery_unknown"
					j.Owner = ""
					j.UpdatedAt = o.Now()
					return s.saveJob(tx, st, &old, j)
				}
				return nil
			})
		})
		if e != nil {
			db.Close()
			return nil, e
		}
	}
	return s, nil
}
func (s *Store) Close() error { return s.db.Close() }
func hash(b []byte) string    { v := sha256.Sum256(b); return hex.EncodeToString(v[:]) }
func encoded(v any) []byte    { b, _ := json.Marshal(v); return b }
func get(tx *badger.Txn, k string, v any) error {
	i, e := tx.Get([]byte(k))
	if errors.Is(e, badger.ErrKeyNotFound) {
		return ErrNotFound
	}
	if e != nil {
		return e
	}
	return i.Value(func(b []byte) error { return json.Unmarshal(b, v) })
}
func put(tx *badger.Txn, k string, v any) error {
	b, e := json.Marshal(v)
	if e != nil {
		return e
	}
	return tx.Set([]byte(k), b)
}
func scan(tx *badger.Txn, p string, f func(string, []byte) error) error {
	it := tx.NewIterator(badger.DefaultIteratorOptions)
	defer it.Close()
	for it.Seek([]byte(p)); it.ValidForPrefix([]byte(p)); it.Next() {
		b, e := it.Item().ValueCopy(nil)
		if e != nil {
			return e
		}
		if e = f(string(it.Item().KeyCopy(nil)), b); e != nil {
			return e
		}
	}
	return nil
}
func state(tx *badger.Txn) (Status, error) {
	st := Status{}
	e := get(tx, "state", &st)
	if errors.Is(e, ErrNotFound) {
		e = nil
	}
	if st.Counts == nil {
		st.Counts = map[State]int{}
	}
	if st.CoverageGaps == nil {
		st.CoverageGaps = map[string]uint64{}
	}
	return st, e
}
func (s *Store) write(f func(*badger.Txn, *Status) error) error {
	if s.opts.ReadOnly {
		return ErrReadOnly
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.db.Update(func(tx *badger.Txn) error {
		st, e := state(tx)
		if e != nil {
			return e
		}
		if e = f(tx, &st); e != nil {
			return e
		}
		return put(tx, "state", st)
	})
}
func (s *Store) Status() (Status, error) {
	var st Status
	e := s.db.View(func(tx *badger.Txn) error { var e error; st, e = state(tx); return e })
	return st, e
}
func (s *Store) RecordGap(reason string) error {
	return s.write(func(tx *badger.Txn, st *Status) error {
		r := safeReason(reason)
		if len(st.CoverageGaps) >= 64 && st.CoverageGaps[r] == 0 {
			r = "other"
		}
		st.CoverageGaps[r]++
		return nil
	})
}
func safeReason(r string) string {
	if len(r) > 80 || r == "" {
		return "assessment_error"
	}
	for _, c := range r {
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_') {
			return "assessment_error"
		}
	}
	return r
}
func active(j Job) bool { return j.Status == Pending || j.Status == Running }
func jobPayload(j Job) int64 {
	return int64(len(j.Packet) + len(j.Extraction) + len(encoded(j.Candidate)))
}

// The reservation is charged before a job can be claimed. The validated
// three-answer result fits within 4 KiB; the remainder covers owner, terminal
// reason/status, and timestamp changes. Packet/manifest growth is charged
// separately by SavePacket before dispatch.
const completionMetadataReserve int64 = 8 << 10
const maxAssessmentMetadataBytes = 4 << 10
const maxOwnerBytes = 128

func jobMeta(j Job) int64 {
	j.Packet = nil
	j.Extraction = nil
	j.Candidate = extract.Fct{}
	if active(j) {
		j.Status = Pending
		j.Owner = ""
		j.Error = ""
		j.Assessment = nil
		j.UpdatedAt = j.CreatedAt
		j.DispatchStartedAt = time.Time{}
		return int64(len(encoded(j))) + completionMetadataReserve
	}
	return int64(len(encoded(j)))
}
func sourcePayload(v Source) int64 { return int64(len(v.Text) + len(encoded(v.Turns))) }
func sourceMeta(v Source) int64    { v.Text = ""; v.Turns = nil; return int64(len(encoded(v))) }
func (s *Store) saveJob(tx *badger.Txn, st *Status, old *Job, j Job) error {
	if old != nil {
		st.Counts[old.Status]--
		st.PayloadBytes -= jobPayload(*old)
		st.MetadataBytes -= jobMeta(*old)
		if active(*old) {
			st.PendingBytes -= jobPayload(*old)
		}
	}
	st.Counts[j.Status]++
	st.PayloadBytes += jobPayload(j)
	st.MetadataBytes += jobMeta(j)
	if active(j) {
		st.PendingBytes += jobPayload(j)
	}
	if e := put(tx, "j:"+j.ID, j); e != nil {
		return e
	}
	if j.Status == Pending {
		return put(tx, "p:"+j.ID, j.ID)
	}
	return tx.Delete([]byte("p:" + j.ID))
}
func (s *Store) capacity(st *Status) bool {
	l := s.opts.Limits
	return st.PendingBytes > l.PendingBytes || st.Counts[Pending]+st.Counts[Running] > l.PendingJobs || st.PayloadBytes > l.PayloadBytes || st.MetadataBytes > l.MetadataBytes
}
func (s *Store) Capture(v Source) (Source, error) {
	out, e := s.capture(v)
	if errors.Is(e, ErrCapacity) {
		if ce := s.Cleanup(); ce == nil {
			out, e = s.capture(v)
		}
	}
	if e != nil {
		reason := "source_write_failed"
		if errors.Is(e, ErrCapacity) {
			reason = "payload_capacity"
		}
		_ = s.RecordGap(reason)
	}
	return out, e
}
func (s *Store) capture(v Source) (Source, error) {
	v.ID = ""
	v.Revision = 0
	v.CapturedAt = time.Time{}
	v.Digest = ""
	v.Unavailable = false
	if v.Kind == "" {
		v.Kind = "raw_source"
	}
	if v.Kind == "derived_summary" {
		v.SourceUnavailable = true
	}
	// Redact all string fields, including local provenance, before persistence.
	b := redactedJSON(v)
	if e := json.Unmarshal(b, &v); e != nil {
		return Source{}, e
	}
	canonical := encoded(v)
	v.ID = hash(canonical)
	v.Digest = hash([]byte(v.Text))
	var out Source
	e := s.write(func(tx *badger.Txn, st *Status) error {
		var old Source
		e := get(tx, "s:"+v.ID, &old)
		if e == nil {
			check := old
			check.ID = ""
			check.Revision = 0
			check.CapturedAt = time.Time{}
			check.Digest = ""
			if old.Unavailable {
				return ErrEvidenceUnavailable
			}
			if !bytes.Equal(encoded(check), canonical) {
				return ErrConflict
			}
			out = old
			return nil
		}
		if !errors.Is(e, ErrNotFound) {
			return e
		}
		st.Revision++
		v.Revision = st.Revision
		v.CapturedAt = s.opts.Now()
		st.PayloadBytes += sourcePayload(v)
		st.MetadataBytes += sourceMeta(v)
		if s.capacity(st) {
			return ErrCapacity
		}
		out = v
		if e = put(tx, fmt.Sprintf("r:%020d", v.Revision), v.ID); e != nil {
			return e
		}
		return put(tx, "s:"+v.ID, v)
	})

	return out, e
}
func redactedJSON(v any) []byte {
	var x any
	decoder := json.NewDecoder(bytes.NewReader(encoded(v)))
	decoder.UseNumber()
	_ = decoder.Decode(&x)
	var walk func(any) any
	walk = func(v any) any {
		switch x := v.(type) {
		case string:
			return distill.Redact(x)
		case []any:
			for i := range x {
				x[i] = walk(x[i])
			}
		case map[string]any:
			for k, v := range x {
				x[k] = walk(v)
			}
		}
		return v
	}
	return encoded(walk(x))
}
func (s *Store) GetSource(id string) (Source, error) {
	var v Source
	e := s.db.View(func(tx *badger.Txn) error { return get(tx, "s:"+id, &v) })
	return v, e
}
func (s *Store) GetJob(id string) (Job, error) {
	var v Job
	e := s.db.View(func(tx *badger.Txn) error { return get(tx, "j:"+id, &v) })
	return v, e
}
func (s *Store) Enqueue(sourceID string, result extract.Result, versions Versions) ([]Job, error) {
	out, e := s.enqueue(sourceID, result, versions)
	if errors.Is(e, ErrCapacity) {
		if ce := s.Cleanup(); ce == nil {
			out, e = s.enqueue(sourceID, result, versions)
		}
	}
	if e != nil {
		reason := "job_write_failed"
		if errors.Is(e, ErrCapacity) {
			reason = "pending_capacity"
		}
		if errors.Is(e, ErrEvidenceUnavailable) {
			reason = "source_unavailable"
		}
		_ = s.RecordGap(reason)
	}
	return out, e
}
func (s *Store) enqueue(sourceID string, result extract.Result, versions Versions) ([]Job, error) {
	canonical := redactedJSON(result)
	var clean extract.Result
	if e := json.Unmarshal(canonical, &clean); e != nil {
		return nil, e
	}
	digest := hash(canonical)
	var out []Job
	e := s.write(func(tx *badger.Txn, st *Status) error {
		var src Source
		if e := get(tx, "s:"+sourceID, &src); e != nil {
			return e
		}
		// A pre-extracted commit can retain only a derived summary. Keep its
		// candidate occurrence so the builder records evidence_unavailable;
		// it must never dispatch a summary as the required raw target.
		if src.Unavailable || (src.SourceUnavailable && src.Kind != "derived_summary") || src.Text == "" {
			return ErrEvidenceUnavailable
		}
		for i, f := range clean.Facts {
			cid := hash(encoded([]any{src.EpisodeID, digest, i}))
			id := hash(encoded([]any{cid, versions}))
			var old Job
			e := get(tx, "j:"+id, &old)
			if e == nil {
				if old.Versions != versions || old.SourceID != src.ID || old.Ordinal != i || (!bytes.Equal(old.Extraction, canonical) && !old.EvidenceUnavailable) {
					return ErrConflict
				}
				out = append(out, old)
				continue
			}
			if !errors.Is(e, ErrNotFound) {
				return e
			}
			j := Job{ID: id, CandidateID: cid, EpisodeID: src.EpisodeID, SourceID: src.ID, ExtractionDigest: digest, Extraction: canonical, Ordinal: i, Candidate: f, SourceCutoff: st.Revision, Versions: versions, Status: Pending, CreatedAt: s.opts.Now(), UpdatedAt: s.opts.Now()}
			j.TargetSourceUnavailable = src.SourceUnavailable
			if e = s.saveJob(tx, st, nil, j); e != nil {
				return e
			}
			out = append(out, j)
		}
		if s.capacity(st) {
			return ErrCapacity
		}
		return nil
	})

	if e != nil {
		return nil, e
	}
	return out, nil
}
func (s *Store) Claim(owner string) (Job, error) {
	var j Job
	if owner == "" || len(owner) > maxOwnerBytes {
		return j, ErrOwnership
	}
	e := s.write(func(tx *badger.Txn, st *Status) error {
		if st.BlockedReason != "" {
			return ErrBlocked
		}
		it := tx.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		it.Seek([]byte("p:"))
		if !it.ValidForPrefix([]byte("p:")) {
			return ErrNotFound
		}
		var id string
		if e := it.Item().Value(func(b []byte) error { return json.Unmarshal(b, &id) }); e != nil {
			return e
		}
		if e := get(tx, "j:"+id, &j); e != nil {
			return e
		}
		old := j
		j.Status = Running
		j.Owner = owner
		j.UpdatedAt = s.opts.Now()
		return s.saveJob(tx, st, &old, j)
	})
	return j, e
}
func (s *Store) SavePacket(id, owner string, packet []byte, manifest assess.Manifest) error {
	var safeManifest assess.Manifest
	if e := json.Unmarshal(redactedJSON(manifest), &safeManifest); e != nil {
		return e
	}
	manifest = safeManifest
	if !json.Valid(packet) || !bytes.Equal(redactedJSON(json.RawMessage(packet)), encodedJSON(packet)) {
		return errors.New("assessstore: packet must be redacted JSON")
	}
	return s.write(func(tx *badger.Txn, st *Status) error {
		var j Job
		if e := get(tx, "j:"+id, &j); e != nil {
			return e
		}
		if j.Status != Running || j.Owner != owner {
			return ErrOwnership
		}
		if len(j.Packet) > 0 {
			if !bytes.Equal(j.Packet, packet) || !bytes.Equal(encoded(j.Manifest), encoded(manifest)) {
				return ErrConflict
			}
			return nil
		}
		old := j
		j.Packet = append([]byte(nil), packet...)
		j.PacketHash = hash(packet)
		j.Manifest = manifest
		j.UpdatedAt = s.opts.Now()
		if e := s.saveJob(tx, st, &old, j); e != nil {
			return e
		}
		if s.capacity(st) {
			return ErrCapacity
		}
		return nil
	})
}
func encodedJSON(b []byte) []byte {
	var v any
	d := json.NewDecoder(bytes.NewReader(b))
	d.UseNumber()
	_ = d.Decode(&v)
	return encoded(v)
}
func (s *Store) Finish(id, owner string, status State, reason string, a *assess.Assessment) error {
	if status != Completed && a != nil {
		return errors.New("assessstore: judgments require completed status")
	}
	if status != Completed && status != Failed && status != Blocked && status != Oversize {
		return errors.New("assessstore: invalid terminal state")
	}
	return s.write(func(tx *badger.Txn, st *Status) error {
		var j Job
		if e := get(tx, "j:"+id, &j); e != nil {
			return e
		}
		if j.Status != Running || j.Owner != owner {
			return ErrOwnership
		}
		if status == Completed && (a == nil || len(j.Packet) == 0 || a.Model != j.Versions.Model) {
			return errors.New("assessstore: completed assessment required")
		}
		if status == Completed {
			if len(encoded(a)) > maxAssessmentMetadataBytes {
				return errors.New("assessstore: assessment metadata exceeds reserved bound")
			}
			if e := a.ValidateFor(j.Versions.Model); e != nil {
				return errors.New("assessstore: invalid assessment")
			}
			if a.Usage.InputTokens < 0 || a.Usage.OutputTokens < 0 || a.LatencyMS < 0 || math.IsNaN(a.LatencyMS) || math.IsInf(a.LatencyMS, 0) {
				return errors.New("assessstore: invalid assessment usage or latency")
			}
		}
		old := j
		j.Status = status
		j.Owner = ""
		j.Error = safeReason(reason)
		if status == Completed {
			j.Error = ""
		}
		j.Assessment = a
		j.UpdatedAt = s.opts.Now()
		if e := s.saveJob(tx, st, &old, j); e != nil {
			return e
		}
		// Completion consumes already-reserved space and releases queue
		// capacity. Do not reject this terminal transition merely because an
		// older sidecar or a newly lowered test limit is already over capacity.
		return nil
	})
}
func (s *Store) Block(reason string, retryAfter time.Time) error {
	return s.write(func(tx *badger.Txn, st *Status) error {
		st.BlockedReason = safeReason(reason)
		if retryAfter.After(st.RetryAfter) {
			st.RetryAfter = retryAfter
		}
		return nil
	})
}
func (s *Store) Resume() error {
	return s.write(func(tx *badger.Txn, st *Status) error {
		if s.opts.Now().Before(st.RetryAfter) {
			return ErrCooldown
		}
		st.BlockedReason = ""
		st.RetryAfter = time.Time{}
		return nil
	})
}
func (s *Store) Replay(id string) (Job, error) {
	if e := s.Cleanup(); e != nil {
		return Job{}, e
	}
	var out Job
	e := s.write(func(tx *badger.Txn, st *Status) error {
		var j Job
		if e := get(tx, "j:"+id, &j); e != nil {
			return e
		}
		if len(j.Packet) == 0 || j.EvidenceUnavailable {
			return ErrEvidenceUnavailable
		}
		st.ReplaySequence++
		j.ParentID = id
		j.Attempt = st.ReplaySequence
		j.ID = hash(encoded([]any{id, j.Attempt}))
		j.Status = Pending
		j.Owner = ""
		j.Error = ""
		j.Assessment = nil
		j.CreatedAt = s.opts.Now()
		j.DispatchStartedAt = time.Time{}
		j.UpdatedAt = j.CreatedAt
		if e := s.saveJob(tx, st, nil, j); e != nil {
			return e
		}
		if s.capacity(st) {
			return ErrCapacity
		}
		out = j
		return nil
	})
	return out, e
}

// Retain all source revisions visible to active jobs: conservative pinning ensures
// delayed context assembly cannot lose history before its immutable cutoff.
func (s *Store) Cleanup() error {
	if s.opts.ReadOnly {
		return ErrReadOnly
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.opts.Now()
	var pin uint64
	if e := s.db.View(func(tx *badger.Txn) error {
		return scan(tx, "j:", func(k string, b []byte) error {
			var j Job
			if e := json.Unmarshal(b, &j); e != nil {
				return e
			}
			if active(j) && j.SourceCutoff > pin {
				pin = j.SourceCutoff
			}
			return nil
		})
	}); e != nil {
		return e
	}
	for _, prefix := range []string{"j:", "s:"} {
		after := ""
		done := false
		for !done {
			e := s.db.Update(func(tx *badger.Txn) error {
				st, e := state(tx)
				if e != nil {
					return e
				}
				it := tx.NewIterator(badger.DefaultIteratorOptions)
				defer it.Close()
				seek := prefix + after
				it.Seek([]byte(seek))
				count := 0
				done = true
				for ; it.ValidForPrefix([]byte(prefix)); it.Next() {
					key := string(it.Item().KeyCopy(nil))
					if after != "" && key == seek {
						continue
					}
					b, e := it.Item().ValueCopy(nil)
					if e != nil {
						return e
					}
					if prefix == "j:" {
						var j Job
						if e = json.Unmarshal(b, &j); e != nil {
							return e
						}
						if !active(j) {
							if now.Sub(j.UpdatedAt) >= s.opts.Limits.MetadataAge {
								st.Counts[j.Status]--
								st.PayloadBytes -= jobPayload(j)
								st.MetadataBytes -= jobMeta(j)
								if e = tx.Delete([]byte(key)); e != nil {
									return e
								}
							} else if !j.EvidenceUnavailable && now.Sub(j.UpdatedAt) >= s.opts.Limits.PayloadAge {
								old := j
								j.Packet = nil
								j.Extraction = nil
								j.Candidate = extract.Fct{}
								j.EvidenceUnavailable = true
								if e = s.saveJob(tx, &st, &old, j); e != nil {
									return e
								}
							}
						}
					} else {
						var v Source
						if e = json.Unmarshal(b, &v); e != nil {
							return e
						}
						if v.Revision > pin {
							if now.Sub(v.CapturedAt) >= s.opts.Limits.MetadataAge {
								st.PayloadBytes -= sourcePayload(v)
								st.MetadataBytes -= sourceMeta(v)
								if e = tx.Delete([]byte(fmt.Sprintf("r:%020d", v.Revision))); e != nil {
									return e
								}
								if e = tx.Delete([]byte(key)); e != nil {
									return e
								}
							} else if !v.Unavailable && now.Sub(v.CapturedAt) >= s.opts.Limits.PayloadAge {
								st.PayloadBytes -= sourcePayload(v)
								st.MetadataBytes -= sourceMeta(v)
								v.Text = ""
								v.Turns = nil
								v.Unavailable = true
								v.SourceUnavailable = true
								st.PayloadBytes += sourcePayload(v)
								st.MetadataBytes += sourceMeta(v)
								if e = put(tx, key, v); e != nil {
									return e
								}
							}
						}
					}
					count++
					after = strings.TrimPrefix(key, prefix)
					if count >= 64 {
						done = false
						break
					}
				}
				return put(tx, "state", st)
			})
			if e != nil {
				return e
			}
		}
	}
	return nil
}

type ListQuery struct {
	IncludeContext bool
	EpisodeID      string
	After          string
	Limit          int
}
type JobPage struct {
	Jobs []Job  `json:"jobs"`
	Next string `json:"next,omitempty"`
}

func pageLimit(n int) int {
	if n <= 0 {
		return 100
	}
	if n > 1000 {
		return 1000
	}
	return n
}
func (s *Store) List(q ListQuery) (JobPage, error) {
	var out JobPage
	e := s.db.View(func(tx *badger.Txn) error {
		it := tx.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		start := "j:" + q.After
		it.Seek([]byte(start))
		scanned := 0
		for ; it.ValidForPrefix([]byte("j:")); it.Next() {
			key := string(it.Item().KeyCopy(nil))
			if key == start && q.After != "" {
				continue
			}
			var j Job
			if e := it.Item().Value(func(b []byte) error { return json.Unmarshal(b, &j) }); e != nil {
				return e
			}
			scanned++
			out.Next = j.ID
			if q.EpisodeID == "" || j.EpisodeID == q.EpisodeID {
				if !q.IncludeContext {
					j.Packet = nil
					j.Extraction = nil
				}
				out.Jobs = append(out.Jobs, j)
			}
			if len(out.Jobs) >= pageLimit(q.Limit) || scanned >= 1000 {
				return nil
			}
		}
		out.Next = ""
		return nil
	})
	return out, e
}

type SourceQuery struct {
	Descending bool
	Cutoff     uint64
	After      uint64
	Limit      int
}
type SourcePage struct {
	Sources []Source `json:"sources"`
	Next    uint64   `json:"next,omitempty"`
}

func (s *Store) Sources(q SourceQuery) (SourcePage, error) {
	var out SourcePage
	e := s.db.View(func(tx *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Reverse = q.Descending
		it := tx.NewIterator(opts)
		defer it.Close()
		start := q.After + 1
		if q.Descending {
			start = q.Cutoff
			if start == 0 {
				st, e := state(tx)
				if e != nil {
					return e
				}
				start = st.Revision
			}
			if q.After > 0 && q.After <= start {
				start = q.After - 1
			}
		}
		it.Seek([]byte(fmt.Sprintf("r:%020d", start)))
		for ; it.ValidForPrefix([]byte("r:")); it.Next() {
			var id string
			if e := it.Item().Value(func(b []byte) error { return json.Unmarshal(b, &id) }); e != nil {
				return e
			}
			var src Source
			if e := get(tx, "s:"+id, &src); e != nil {
				return e
			}
			if q.Cutoff != 0 && src.Revision > q.Cutoff {
				break
			}
			out.Sources = append(out.Sources, src)
			if len(out.Sources) >= pageLimit(q.Limit) {
				out.Next = src.Revision
				return nil
			}
		}
		return nil
	})
	return out, e
}

// SessionReference removes a known span suffix without opening a client path.
func SessionReference(ref string) string {
	if i := strings.LastIndex(ref, "#"); i >= 0 {
		return ref[:i]
	}
	return ref
}
