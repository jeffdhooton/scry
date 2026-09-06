package store

// PRIVATE, UNCALLED mechanical actor history. This is not an ownership or undo
// policy and does not certify untracked keys or the final relationship graph.

import (
	"bytes"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/dgraph-io/badger/v4"
)

var errIdentityWriterHistory = errors.New("memory: invalid identity writer history")

type identityWriterEntry struct {
	Sequence uint64
	Actor    string
	Key      []byte
	Before   identityImage
	After    identityImage
}

type identityWriterSnapshot struct {
	Entries []identityWriterEntry
	Before  map[string]identityImage
	Final   map[string]identityImage
}

type identityWriterHistory struct {
	st      *Store
	owner   *identityAdmissionOwner
	journal *identityJournal
	final   map[string]identityImage
	entries []identityWriterEntry
	frozen  bool
}

func identityWriterFailure(st *Store, err error) error {
	safe := error(errIdentityWriterHistory)
	for _, kind := range []error{badger.ErrTxnTooBig, badger.ErrConflict} {
		if errors.Is(err, kind) {
			safe = errors.Join(errIdentityWriterHistory, kind)
			break
		}
	}
	if st != nil && st.admissionOwner != nil {
		st.poisonAdmission(safe)
	}
	return safe
}

func newIdentityWriterHistory(st *Store) (*identityWriterHistory, error) {
	if st == nil || st.admissionOwner == nil || st.admissionOwner.phase != admissionBody || st.admissionFailure != nil {
		return nil, identityWriterFailure(st, errIdentityWriterHistory)
	}
	if err := generationTransaction(st); err != nil {
		return nil, identityWriterFailure(st, err)
	}
	j, err := newIdentityJournal(st)
	if err != nil {
		return nil, identityWriterFailure(st, err)
	}
	return &identityWriterHistory{st: st, owner: st.admissionOwner, journal: j, final: make(map[string]identityImage), entries: []identityWriterEntry{}}, nil
}

func (h *identityWriterHistory) phase(want int) error {
	if h == nil {
		return errIdentityWriterHistory
	}
	if h.st == nil || h.owner == nil || h.st.admissionOwner != h.owner || h.owner.phase != want || h.journal == nil || h.journal.st != h.st || h.final == nil || h.frozen || h.st.admissionFailure != nil {
		return identityWriterFailure(h.st, errIdentityWriterHistory)
	}
	if err := generationTransaction(h.st); err != nil {
		return identityWriterFailure(h.st, err)
	}
	return nil
}

func cloneWriterImage(v identityImage) identityImage {
	return identityImage{Exists: v.Exists, Value: bytes.Clone(v.Value)}
}

func writerImageEqual(a, b identityImage) bool {
	return a.Exists == b.Exists && bytes.Equal(a.Value, b.Value)
}

func identityWriterKey(actor string, key []byte) (entity bool, err error) {
	if !validEntitySlug(actor) || !utf8.Valid(key) {
		return false, errIdentityWriterHistory
	}
	s := string(key)
	if strings.HasPrefix(s, prefixEntity) {
		if strings.TrimPrefix(s, prefixEntity) != actor {
			return false, errIdentityWriterHistory
		}
		return true, nil
	}
	if strings.HasPrefix(s, prefixAlias) {
		norm := strings.TrimPrefix(s, prefixAlias)
		if norm != "" && Normalize(norm) == norm {
			return false, nil
		}
	}
	return false, errIdentityWriterHistory
}

func (h *identityWriterHistory) put(actor string, key, raw []byte) error {
	return h.mutate(actor, key, identityImage{Exists: true, Value: raw})
}

func (h *identityWriterHistory) delete(actor string, key []byte) error {
	return h.mutate(actor, key, identityImage{})
}

func (h *identityWriterHistory) mutate(actor string, key []byte, after identityImage) error {
	if err := h.phase(admissionBody); err != nil {
		return err
	}
	fail := func(err error) error { return identityWriterFailure(h.st, err) }
	isEntity, err := identityWriterKey(actor, key)
	if err != nil {
		return fail(err)
	}
	key = bytes.Clone(key)
	after = cloneWriterImage(after)
	if after.Exists {
		if isEntity {
			if _, err := decodeLegacyEntity(key, after.Value); err != nil {
				return fail(err)
			}
		} else if !bytes.Equal(after.Value, []byte(actor)) {
			return fail(errIdentityWriterHistory)
		}
	}
	before, err := h.journal.image(key)
	if err != nil {
		return fail(err)
	}
	if !after.Exists {
		if !before.Exists || (!isEntity && !bytes.Equal(before.Value, []byte(actor))) {
			return fail(errIdentityWriterHistory)
		}
	}
	if previous, ok := h.final[string(key)]; ok && !writerImageEqual(previous, before) {
		return fail(errIdentityWriterHistory)
	}
	if err := h.journal.capture(key); err != nil {
		return fail(err)
	}
	if after.Exists {
		err = h.st.txn.Set(key, after.Value)
	} else {
		err = h.st.txn.Delete(key)
	}
	if err != nil {
		return fail(err)
	}
	h.final[string(key)] = cloneWriterImage(after)
	h.entries = append(h.entries, identityWriterEntry{Sequence: uint64(len(h.entries)), Actor: actor, Key: bytes.Clone(key), Before: cloneWriterImage(before), After: cloneWriterImage(after)})
	return nil
}

func (h *identityWriterHistory) freeze() (identityWriterSnapshot, error) {
	if err := h.phase(admissionFinalizing); err != nil {
		return identityWriterSnapshot{}, err
	}
	fail := func(err error) (identityWriterSnapshot, error) {
		return identityWriterSnapshot{}, identityWriterFailure(h.st, err)
	}
	if len(h.journal.before) != len(h.final) {
		return fail(errIdentityWriterHistory)
	}
	for key, expected := range h.final {
		actual, err := h.journal.image([]byte(key))
		if err != nil {
			return fail(err)
		}
		if !writerImageEqual(actual, expected) {
			return fail(errIdentityWriterHistory)
		}
		if _, ok := h.journal.before[key]; !ok {
			return fail(errIdentityWriterHistory)
		}
	}
	out := identityWriterSnapshot{Entries: make([]identityWriterEntry, len(h.entries)), Before: make(map[string]identityImage, len(h.final)), Final: make(map[string]identityImage, len(h.final))}
	for key, value := range h.final {
		out.Before[key] = cloneWriterImage(h.journal.before[key])
		out.Final[key] = cloneWriterImage(value)
	}
	for i, entry := range h.entries {
		out.Entries[i] = identityWriterEntry{Sequence: entry.Sequence, Actor: entry.Actor, Key: bytes.Clone(entry.Key), Before: cloneWriterImage(entry.Before), After: cloneWriterImage(entry.After)}
	}
	h.frozen = true
	return out, nil
}
