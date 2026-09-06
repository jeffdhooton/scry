package store

// PRIVATE PROTOTYPE: not used by any production caller. This is an exact-key
// before-image primitive, not an admission policy or entity-retirement API.

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/dgraph-io/badger/v4"
)

var errIdentityJournal = errors.New("memory: identity journal precondition failed")

type identityImage struct {
	Exists bool
	Value  []byte
}

type identityUndo struct {
	Key      []byte
	Expected identityImage
}

// identityJournal lives only inside one AtomicWrite callback. Capture must run
// BEFORE the corresponding mutation. It never reconstructs JSON, scans for an
// owner, touches facts, or deletes a family of keys. Errors must abort the outer
// transaction, just like all other Store writes.
type identityJournal struct {
	st     *Store
	before map[string]identityImage
}

func newIdentityJournal(st *Store) (*identityJournal, error) {
	if st == nil || st.txn == nil || st.pendingEvents == nil {
		return nil, errIdentityJournal
	}
	j := &identityJournal{st: st, before: make(map[string]identityImage)}
	if err := j.live(); err != nil {
		return nil, err
	}
	return j, nil
}

func (j *identityJournal) live() error {
	_, err := j.st.txn.Get([]byte(keySchemaVersion))
	if errors.Is(err, badger.ErrKeyNotFound) {
		return nil
	}
	return err
}

func identityJournalKey(key []byte) bool {
	for _, prefix := range []string{prefixEntity, prefixAlias, prefixAttest} {
		if len(key) > len(prefix) && bytes.HasPrefix(key, []byte(prefix)) {
			return true
		}
	}
	return false
}

func (j *identityJournal) image(key []byte) (identityImage, error) {
	item, err := j.st.txn.Get(key)
	if errors.Is(err, badger.ErrKeyNotFound) {
		return identityImage{}, nil
	}
	if err != nil {
		return identityImage{}, err
	}
	value, err := item.ValueCopy(nil)
	return identityImage{Exists: true, Value: value}, err
}

func (j *identityJournal) capture(key []byte) error {
	if err := j.live(); err != nil {
		return err
	}
	if !identityJournalKey(key) {
		return errIdentityJournal
	}
	if _, ok := j.before[string(key)]; ok {
		return nil
	}
	image, err := j.image(key)
	if err != nil {
		return err
	}
	j.before[string(key)] = image
	return nil
}

// restore preflights the COMPLETE explicit undo set before staging any change.
// Expected is the exact current staged value, not a typed projection or hash.
func (j *identityJournal) restore(undo []identityUndo) error {
	if err := j.live(); err != nil {
		return err
	}
	// Badger retains key/value buffers until transaction completion. Freeze an
	// owned plan so a caller reusing its input buffers after this method returns
	// cannot redirect a staged identity undo into any other record family.
	type plannedUndo struct {
		key      []byte
		expected identityImage
		old      identityImage
	}
	plan := make([]plannedUndo, 0, len(undo))
	seen := make(map[string]bool, len(undo))
	for _, u := range undo {
		ownedKey := bytes.Clone(u.Key)
		key := string(ownedKey)
		if seen[key] {
			return errIdentityJournal
		}
		seen[key] = true
		old, ok := j.before[key]
		if !ok {
			return errIdentityJournal
		}
		plan = append(plan, plannedUndo{
			key:      ownedKey,
			expected: identityImage{Exists: u.Expected.Exists, Value: bytes.Clone(u.Expected.Value)},
			old:      identityImage{Exists: old.Exists, Value: bytes.Clone(old.Value)},
		})
	}
	for _, u := range plan {
		current, err := j.image(u.key)
		if err != nil {
			return err
		}
		if current.Exists != u.expected.Exists || !bytes.Equal(current.Value, u.expected.Value) {
			return errIdentityJournal
		}
	}
	for _, u := range plan {
		var err error
		if u.old.Exists {
			err = j.st.txn.Set(u.key, u.old.Value)
		} else {
			err = j.st.txn.Delete(u.key)
		}
		if err != nil {
			return fmt.Errorf("identity journal restore: %w", err)
		}
	}
	return nil
}

// suppressAbsentEntityEvents is separate from restore: it requires that every
// explicitly named entity was absent when captured AND is absent now. It only
// filters entity events; fact/episode events and existing entities are untouched.
// The caller must separately prove factual support, preserve all metadata and
// restore exact claims/evidence before this low-level primitive is useful.
func (j *identityJournal) suppressAbsentEntityEvents(slugs []string) error {
	if err := j.live(); err != nil {
		return err
	}
	remove := make(map[string]bool, len(slugs))
	for _, slug := range slugs {
		key := prefixEntity + slug
		old, captured := j.before[key]
		if slug == "" || !captured || old.Exists {
			return errIdentityJournal
		}
		current, err := j.image([]byte(key))
		if err != nil {
			return err
		}
		if current.Exists {
			return errIdentityJournal
		}
		remove[slug] = true
	}
	events := *j.st.pendingEvents
	filtered := make([]Event, 0, len(events))
	for _, event := range events {
		slug := event.Entity.Slug
		if event.Op == "delete" {
			slug = event.Slug
		}
		if event.Kind == "entity" && remove[slug] {
			continue
		}
		filtered = append(filtered, event)
	}
	*j.st.pendingEvents = filtered
	return nil
}
