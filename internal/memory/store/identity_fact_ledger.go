package store

// PRIVATE, UNCALLED write-set accounting. A recorded ordinal is descriptive;
// allowed assertion semantics/support and public mutator integration are absent.

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"sort"
	"strings"

	"github.com/dgraph-io/badger/v4"
)

var errIdentityFactLedger = errors.New("memory: invalid identity fact mutation ledger")

type identityFactState struct {
	Exists bool
	Raw    []byte
}

type identityFactMutation struct {
	Ordinal int
	Key     []byte
	Before  identityFactState
	After   identityFactState
}

type identityFactLedgerEntry struct {
	before identityFactState
	after  identityFactState
}

type identityFactLedger struct {
	st        *Store
	owner     *identityAdmissionOwner
	baseline  identityReferenceInventory
	entries   map[string]identityFactLedgerEntry
	mutations []identityFactMutation
	verified  bool
}

type identityFactLedgerReport struct {
	Final     identityReferenceInventory
	Mutations []identityFactMutation
}

func factLedgerFailure(st *Store, err error) error {
	safe := error(errIdentityFactLedger)
	for _, kind := range []error{badger.ErrTxnTooBig, badger.ErrConflict} {
		if errors.Is(err, kind) {
			safe = errors.Join(errIdentityFactLedger, kind)
			break
		}
	}
	if st != nil && st.admissionOwner != nil {
		st.poisonAdmission(safe)
	}
	return safe
}

func beginIdentityFactLedger(st *Store) (*identityFactLedger, error) {
	if st == nil || st.admissionOwner == nil || st.admissionOwner.phase != admissionBody || st.admissionFailure != nil {
		return nil, factLedgerFailure(st, errIdentityFactLedger)
	}
	if err := generationTransaction(st); err != nil {
		return nil, factLedgerFailure(st, err)
	}
	baseline, err := scanIdentityReferenceInventory(st)
	if err != nil {
		return nil, factLedgerFailure(st, err)
	}
	return &identityFactLedger{st: st, owner: st.admissionOwner, baseline: baseline, entries: make(map[string]identityFactLedgerEntry), mutations: []identityFactMutation{}}, nil
}

func (l *identityFactLedger) checkPhase(phase int) error {
	if l == nil {
		return errIdentityFactLedger
	}
	if l.st == nil || l.owner == nil || l.st.admissionOwner != l.owner || l.owner.phase != phase || l.verified || l.entries == nil || l.st.admissionFailure != nil {
		return factLedgerFailure(l.st, errIdentityFactLedger)
	}
	if err := generationTransaction(l.st); err != nil {
		return factLedgerFailure(l.st, err)
	}
	return nil
}

func cloneIdentityFactState(s identityFactState) identityFactState {
	return identityFactState{Exists: s.Exists, Raw: bytes.Clone(s.Raw)}
}

func equalIdentityFactState(a, b identityFactState) bool {
	return a.Exists == b.Exists && bytes.Equal(a.Raw, b.Raw)
}

func (l *identityFactLedger) put(ordinal int, key, raw []byte) error {
	return l.mutate(ordinal, key, identityFactState{Exists: true, Raw: raw})
}

func (l *identityFactLedger) delete(ordinal int, key []byte) error {
	return l.mutate(ordinal, key, identityFactState{})
}

func (l *identityFactLedger) mutate(ordinal int, key []byte, after identityFactState) error {
	if err := l.checkPhase(admissionBody); err != nil {
		return err
	}
	fail := func(err error) error { return factLedgerFailure(l.st, err) }
	if ordinal < 0 || !bytes.HasPrefix(key, []byte(prefixFact)) {
		return fail(errIdentityFactLedger)
	}
	key = bytes.Clone(key)
	after = cloneIdentityFactState(after)
	if after.Exists {
		if _, err := decodeIdentityReference(key, after.Raw); err != nil {
			return fail(err)
		}
	}
	before := identityFactState{}
	item, err := l.st.txn.Get(key)
	if err == nil {
		before.Exists = true
		before.Raw, err = item.ValueCopy(nil)
		if err != nil {
			return fail(err)
		}
		if _, err := decodeIdentityReference(key, before.Raw); err != nil {
			return fail(err)
		}
	} else if !errors.Is(err, badger.ErrKeyNotFound) {
		return fail(err)
	}
	if !after.Exists && !before.Exists {
		return fail(errIdentityFactLedger)
	}
	entry, tracked := l.entries[string(key)]
	if tracked && !equalIdentityFactState(entry.after, before) {
		return fail(errIdentityFactLedger)
	}
	if !tracked {
		entry.before = cloneIdentityFactState(before)
	}
	if after.Exists {
		err = l.st.txn.Set(key, after.Raw)
	} else {
		err = l.st.txn.Delete(key)
	}
	if err != nil {
		return fail(err)
	}
	entry.after = cloneIdentityFactState(after)
	l.entries[string(key)] = entry
	l.mutations = append(l.mutations, identityFactMutation{Ordinal: ordinal, Key: bytes.Clone(key), Before: cloneIdentityFactState(before), After: cloneIdentityFactState(after)})
	return nil
}

func (l *identityFactLedger) verify() (identityFactLedgerReport, error) {
	if err := l.checkPhase(admissionFinalizing); err != nil {
		return identityFactLedgerReport{}, err
	}
	fail := func(err error) (identityFactLedgerReport, error) {
		return identityFactLedgerReport{}, factLedgerFailure(l.st, err)
	}
	final, err := scanIdentityReferenceInventory(l.st)
	if err != nil {
		return fail(err)
	}
	keys := make([]string, 0, len(l.entries))
	for key := range l.entries {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	digest := sha256.New()
	_, _ = digest.Write([]byte("identity-reference-inventory-v1\x00"))
	var length [8]byte
	var count uint64
	add := func(key []byte, state identityFactState) {
		if !state.Exists {
			return
		}
		for _, part := range [][]byte{key, state.Raw} {
			binary.BigEndian.PutUint64(length[:], uint64(len(part)))
			_, _ = digest.Write(length[:])
			_, _ = digest.Write(part)
		}
		count++
	}
	missing := func(key string) error {
		entry := l.entries[key]
		if entry.after.Exists {
			return errIdentityFactLedger
		}
		add([]byte(key), entry.before)
		return nil
	}
	it := l.st.txn.NewIterator(badger.DefaultIteratorOptions)
	defer it.Close()
	i := 0
	prefix := []byte(prefixFact)
	for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
		key := it.Item().KeyCopy(nil)
		for i < len(keys) && strings.Compare(keys[i], string(key)) < 0 {
			if err := missing(keys[i]); err != nil {
				return fail(err)
			}
			i++
		}
		raw, err := it.Item().ValueCopy(nil)
		if err != nil {
			return fail(err)
		}
		if i < len(keys) && keys[i] == string(key) {
			entry := l.entries[keys[i]]
			if !entry.after.Exists || !bytes.Equal(entry.after.Raw, raw) {
				return fail(errIdentityFactLedger)
			}
			add(key, entry.before)
			i++
		} else {
			add(key, identityFactState{Exists: true, Raw: raw})
		}
	}
	for i < len(keys) {
		if err := missing(keys[i]); err != nil {
			return fail(err)
		}
		i++
	}
	if count != l.baseline.Scanned || hex.EncodeToString(digest.Sum(nil)) != l.baseline.Digest {
		return fail(errIdentityFactLedger)
	}
	mutations := make([]identityFactMutation, len(l.mutations))
	for i, m := range l.mutations {
		mutations[i] = identityFactMutation{Ordinal: m.Ordinal, Key: bytes.Clone(m.Key), Before: cloneIdentityFactState(m.Before), After: cloneIdentityFactState(m.After)}
	}
	l.verified = true
	return identityFactLedgerReport{Final: final, Mutations: mutations}, nil
}
