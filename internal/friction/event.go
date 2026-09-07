// Package friction retains explicit workflow observations independently of
// memory extraction. Events are immutable; recommendations are never instructions.
package friction

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const MaxEventBytes = 16 << 10
const MaxResults = 100

var ErrInvalid = errors.New("invalid friction input")
var ErrConflict = errors.New("friction event ID already has different content")
var ErrNotFound = errors.New("friction event not found")
var ErrReviewTooLarge = errors.New("review exceeds 100 events; narrow repository, run_id, signature, since or until (use list to paginate)")

// Event preserves the pilot's authored observations and their attribution.
// RecordedAt is supplied by the caller so a retry never gets a new timestamp.
// ResolutionState is descriptive, not a policy or instruction activation flag.
type Event struct {
	EventID                     string            `json:"event_id"`
	RunID                       string            `json:"run_id"`
	Repository                  string            `json:"repository"`
	RecordedAt                  string            `json:"recorded_at"`
	Signature                   string            `json:"signature"`
	Observed                    string            `json:"observed"`
	Resolution                  string            `json:"resolution"`
	ResolutionState             string            `json:"resolution_state"`
	Evidence                    []string          `json:"evidence"`
	EvidenceSHA256              map[string]string `json:"evidence_sha256,omitempty"`
	CauseStatus                 string            `json:"cause_status,omitempty"`
	Cause                       string            `json:"cause,omitempty"`
	ProposedChange              string            `json:"proposed_change,omitempty"`
	ProposedOwner               string            `json:"proposed_owner,omitempty"`
	ProposedFile                string            `json:"proposed_file,omitempty"`
	ProposedVerification        string            `json:"proposed_verification,omitempty"`
	Priority                    string            `json:"priority,omitempty"`
	OccurrencesObservedThisRun  int               `json:"occurrences_observed_this_run"`
	DistinctPriorRunsVerified   int               `json:"distinct_prior_runs_verified"`
	MeasuredUserTimeCostSeconds *float64          `json:"measured_user_time_cost_seconds"`
	ChangeApproved              bool              `json:"change_approved"`
}

var identifier = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,199}$`)

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalid, fmt.Sprintf(format, args...))
}

// Decode rejects misspelled fields rather than silently losing evidence.
func Decode(raw []byte, out any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return invalid("%v", err)
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return invalid("expected one JSON object")
	}
	return nil
}

func ValidateID(id string) error {
	if !identifier.MatchString(id) {
		return invalid("event_id must be 1–200 ASCII letters, digits, dots, underscores, colons or hyphens, starting with a letter or digit")
	}
	return nil
}

func validateRepo(repo string) error {
	if !filepath.IsAbs(repo) || filepath.Clean(repo) != repo || strings.ContainsRune(repo, 0) {
		return invalid("repository must be an absolute clean path supplied by the caller")
	}
	return nil
}

func (e Event) Validate() error {
	if err := ValidateID(e.EventID); err != nil {
		return err
	}
	if !identifier.MatchString(e.RunID) || !identifier.MatchString(e.Signature) {
		return invalid("run_id and signature must use the event_id character rules")
	}
	if err := validateRepo(e.Repository); err != nil {
		return err
	}
	if _, err := time.Parse(time.RFC3339Nano, e.RecordedAt); err != nil {
		return invalid("recorded_at must be RFC3339: %v", err)
	}
	if strings.TrimSpace(e.Observed) == "" || strings.TrimSpace(e.Resolution) == "" || strings.TrimSpace(e.ResolutionState) == "" {
		return invalid("observed, resolution and resolution_state are required (describe unresolved outcomes explicitly)")
	}
	if len(e.Evidence) == 0 {
		return invalid("at least one evidence reference is required")
	}
	refs := map[string]bool{}
	for _, ref := range e.Evidence {
		if strings.TrimSpace(ref) == "" {
			return invalid("evidence references cannot be empty")
		}
		refs[ref] = true
	}
	for ref, hash := range e.EvidenceSHA256 {
		b, err := hex.DecodeString(hash)
		if !refs[ref] || err != nil || len(b) != 32 {
			return invalid("evidence_sha256 must map an evidence reference to a SHA-256 hex digest")
		}
	}
	if e.MeasuredUserTimeCostSeconds != nil {
		n := *e.MeasuredUserTimeCostSeconds
		if n < 0 || math.IsNaN(n) || math.IsInf(n, 0) {
			return invalid("measured_user_time_cost_seconds must be finite and nonnegative, or null for unknown")
		}
	}
	if e.OccurrencesObservedThisRun < 0 || e.DistinctPriorRunsVerified < 0 {
		return invalid("observation counts cannot be negative")
	}
	if e.ChangeApproved {
		return invalid("change_approved must be false; this journal records proposals, not authorization")
	}
	b, err := json.Marshal(e)
	if err != nil {
		return invalid("%v", err)
	}
	if len(b) > MaxEventBytes {
		return invalid("event exceeds %d bytes", MaxEventBytes)
	}
	return nil
}

// Filter scopes a bounded query. Time bounds apply to the caller's recorded_at:
// since is inclusive, until exclusive. Results/cursors use event ID order.
type Filter struct {
	Repository string `json:"repository"`
	RunID      string `json:"run_id,omitempty"`
	Signature  string `json:"signature,omitempty"`
	Since      string `json:"since,omitempty"`
	Until      string `json:"until,omitempty"`
	After      string `json:"after,omitempty"`
	Limit      int    `json:"limit,omitempty"`
}

func (f Filter) Validate() error {
	if err := validateRepo(f.Repository); err != nil {
		return err
	}
	if f.RunID != "" && !identifier.MatchString(f.RunID) {
		return invalid("invalid run_id")
	}
	if f.Signature != "" && !identifier.MatchString(f.Signature) {
		return invalid("invalid signature")
	}
	if f.After != "" {
		if err := ValidateID(f.After); err != nil {
			return err
		}
	}
	var since, until time.Time
	var err error
	if f.Since != "" {
		since, err = time.Parse(time.RFC3339Nano, f.Since)
		if err != nil {
			return invalid("since must be RFC3339")
		}
	}
	if f.Until != "" {
		until, err = time.Parse(time.RFC3339Nano, f.Until)
		if err != nil {
			return invalid("until must be RFC3339")
		}
	}
	if !since.IsZero() && !until.IsZero() && !since.Before(until) {
		return invalid("since must precede until")
	}
	if f.Limit < 0 || f.Limit > MaxResults {
		return invalid("limit must be between 1 and %d, or omitted", MaxResults)
	}
	return nil
}

func (f Filter) matches(e Event) bool {
	if e.Repository != f.Repository || (f.RunID != "" && e.RunID != f.RunID) || (f.Signature != "" && e.Signature != f.Signature) {
		return false
	}
	at, _ := time.Parse(time.RFC3339Nano, e.RecordedAt)
	if f.Since != "" {
		since, _ := time.Parse(time.RFC3339Nano, f.Since)
		if at.Before(since) {
			return false
		}
	}
	if f.Until != "" {
		until, _ := time.Parse(time.RFC3339Nano, f.Until)
		if !at.Before(until) {
			return false
		}
	}
	return true
}
