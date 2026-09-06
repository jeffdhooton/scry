package store

// PRIVATE, UNCALLED complete identity/control write accounting. No support,
// authorization, undo, post-materialization closure or producer coordination.

import (
	"bytes"
	"errors"

	"github.com/dgraph-io/badger/v4"
)

var errIdentityMutationLedger = errors.New("memory: invalid identity mutation ledger")

type identityMutationLedger struct {
	st       *Store
	owner    *identityAdmissionOwner
	baseline identityRelationshipInventory
	writer   *identityWriterHistory
	verified bool
}

type identityMutationReport struct {
	Baseline identityRelationshipInventory
	Final    identityRelationshipInventory
	History  identityWriterSnapshot
}

func identityMutationFailure(st *Store, err error) error {
	safe := error(errIdentityMutationLedger)
	for _, kind := range []error{badger.ErrTxnTooBig, badger.ErrConflict} {
		if errors.Is(err, kind) {
			safe = errors.Join(errIdentityMutationLedger, kind)
			break
		}
	}
	if st != nil && st.admissionOwner != nil {
		st.poisonAdmission(safe)
	}
	return safe
}

func beginIdentityMutationLedger(st *Store) (*identityMutationLedger, error) {
	if st == nil || st.admissionOwner == nil || st.admissionOwner.phase != admissionBody || st.admissionFailure != nil {
		return nil, identityMutationFailure(st, errIdentityMutationLedger)
	}
	if err := generationTransaction(st); err != nil {
		return nil, identityMutationFailure(st, err)
	}
	baseline, err := scanIdentityRelationships(st)
	if err != nil {
		return nil, identityMutationFailure(st, err)
	}
	writer, err := newIdentityWriterHistory(st)
	if err != nil {
		return nil, identityMutationFailure(st, err)
	}
	return &identityMutationLedger{st: st, owner: st.admissionOwner, baseline: baseline, writer: writer}, nil
}

func (l *identityMutationLedger) phase(want int) error {
	if l == nil {
		return errIdentityMutationLedger
	}
	if l.st == nil || l.owner == nil || l.st.admissionOwner != l.owner || l.owner.phase != want || l.writer == nil || l.writer.st != l.st || l.writer.owner != l.owner || l.baseline.Rows == nil || l.verified || l.st.admissionFailure != nil {
		return identityMutationFailure(l.st, errIdentityMutationLedger)
	}
	if err := generationTransaction(l.st); err != nil {
		return identityMutationFailure(l.st, err)
	}
	return nil
}

func (l *identityMutationLedger) put(actor string, key, raw []byte) error {
	if err := l.phase(admissionBody); err != nil {
		return err
	}
	if err := l.writer.put(actor, key, raw); err != nil {
		return identityMutationFailure(l.st, err)
	}
	return nil
}

func (l *identityMutationLedger) delete(actor string, key []byte) error {
	if err := l.phase(admissionBody); err != nil {
		return err
	}
	if err := l.writer.delete(actor, key); err != nil {
		return identityMutationFailure(l.st, err)
	}
	return nil
}

func cloneIdentityRelationshipInventory(v identityRelationshipInventory) identityRelationshipInventory {
	out := identityRelationshipInventory{Rows: map[string][]byte{}, Entities: map[string]Entity{}, Listings: map[string][]identityListing{}, NaturalListings: map[string][]identityListing{}, IndexTargets: map[string][]string{}, FamilyCounts: map[string]uint64{}, Scanned: v.Scanned, Digest: v.Digest}
	for k, raw := range v.Rows {
		out.Rows[k] = bytes.Clone(raw)
	}
	for k, e := range v.Entities {
		e.Aliases = appendPreservingNil(e.Aliases)
		e.RepoRefs = appendPreservingNil(e.RepoRefs)
		out.Entities[k] = e
	}
	for k, entries := range v.Listings {
		out.Listings[k] = append([]identityListing{}, entries...)
	}
	for k, entries := range v.NaturalListings {
		out.NaturalListings[k] = append([]identityListing{}, entries...)
	}
	for k, keys := range v.IndexTargets {
		out.IndexTargets[k] = appendPreservingNil(keys)
	}
	for k, count := range v.FamilyCounts {
		out.FamilyCounts[k] = count
	}
	return out
}

func appendPreservingNil(v []string) []string {
	if v == nil {
		return nil
	}
	return append([]string{}, v...)
}

func (l *identityMutationLedger) verify() (identityMutationReport, error) {
	if err := l.phase(admissionFinalizing); err != nil {
		return identityMutationReport{}, err
	}
	fail := func(err error) (identityMutationReport, error) {
		return identityMutationReport{}, identityMutationFailure(l.st, err)
	}
	history, err := l.writer.freeze()
	if err != nil {
		return fail(err)
	}
	final, err := scanIdentityRelationships(l.st)
	if err != nil {
		return fail(err)
	}
	expected := make(map[string][]byte, len(l.baseline.Rows))
	for key, raw := range l.baseline.Rows {
		expected[key] = raw // Replay never mutates these owned baseline bytes.
	}
	for i, entry := range history.Entries {
		raw, exists := expected[string(entry.Key)]
		if entry.Sequence != uint64(i) || exists != entry.Before.Exists || !bytes.Equal(raw, entry.Before.Value) {
			return fail(errIdentityMutationLedger)
		}
		if entry.After.Exists {
			expected[string(entry.Key)] = entry.After.Value
		} else {
			delete(expected, string(entry.Key))
		}
	}
	if len(expected) != len(final.Rows) {
		return fail(errIdentityMutationLedger)
	}
	for key, raw := range expected {
		actual, exists := final.Rows[key]
		if !exists || !bytes.Equal(raw, actual) {
			return fail(errIdentityMutationLedger)
		}
	}
	l.verified = true
	return identityMutationReport{Baseline: cloneIdentityRelationshipInventory(l.baseline), Final: final, History: history}, nil
}
