package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/dgraph-io/badger/v4"
)

func TestEpisodeSelectionRealStorageFailureAndCapturedCommitConflict(t *testing.T) {
	for _, mode := range []string{"oversized", "capacity", "conflict", "panic"} {
		t.Run(mode, func(t *testing.T) {
			db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true).WithLogger(nil).WithMemTableSize(2 << 20).WithValueThreshold(4096))
			if err != nil {
				t.Fatal("synthetic database open")
			}
			st := &Store{db: db}
			defer st.Close()
			_, p := selectionFixture(t, st, "storage", true)
			if mode == "oversized" {
				p.Outcomes[1].Materialization = dpJSON(t, map[string]string{"description": strings.Repeat("synthetic-private-value ", 1000)})
			}
			before := dpRows(t, st)
			var staged episodeSelection
			var caught any
			events := 0
			st.SetObserver(func(Event) { events++ })
			func() {
				defer func() { caught = recover() }()
				err = runSerializedIdentityAdmission(st, func(tx *Store) error {
					if mode != "capacity" {
						return nil
					}
					for i := 0; i < 20000; i++ {
						if err := tx.txn.Set([]byte(fmt.Sprintf("synthetic-fill:%05d", i)), bytes.Repeat([]byte{'x'}, 1000)); err != nil {
							if !errors.Is(err, badger.ErrTxnTooBig) {
								t.Fatal("unexpected fill failure")
							}
							return nil // finalizer must itself encounter real full txn
						}
					}
					t.Fatal("capacity fixture never filled")
					return nil
				}, func(tx *Store) error {
					var local error
					staged, local = putEpisodeSelection(tx, "storage", episodeHeadExpectation{}, p)
					if mode == "oversized" || mode == "capacity" {
						if local == nil || strings.Contains(local.Error(), "synthetic-private-value") {
							t.Fatal("storage failure missing or unsafe")
						}
						if mode == "capacity" && !errors.Is(local, badger.ErrTxnTooBig) {
							t.Fatal("not actual capacity refusal")
						}
						return nil // swallowed writer failure must poison
					}
					if local != nil {
						return local
					}
					if mode == "panic" {
						panic("synthetic finalizer panic")
					}
					return st.db.Update(func(other *badger.Txn) error {
						return other.Set([]byte(episodeHeadKey("storage")), []byte("synthetic competing bytes"))
					})
				})
			}()
			if mode == "panic" {
				if caught != "synthetic finalizer panic" {
					t.Fatal("panic did not propagate")
				}
			} else if caught != nil || err == nil {
				t.Fatal("storage failure did not refuse")
			}
			if mode == "conflict" {
				if !errors.Is(err, badger.ErrConflict) || !staged.Selected {
					t.Fatal("expected actual conflict after captured staging")
				}
				after := dpRows(t, st)
				if !bytes.Equal(after[episodeHeadKey("storage")], []byte("synthetic competing bytes")) {
					t.Fatal("competitor not retained")
				}
				delete(after, episodeHeadKey("storage"))
				if !reflect.DeepEqual(before, after) || after[staged.Head.ResultKey] != nil {
					t.Fatal("captured staged selection persisted")
				}
			} else if !reflect.DeepEqual(before, dpRows(t, st)) {
				t.Fatal("failed scope changed raw bytes")
			}
			if (mode == "capacity" || mode == "oversized") && !reflect.DeepEqual(staged, episodeSelection{}) {
				t.Fatal("writer error returned staged data")
			}
			if events != 0 {
				t.Fatal("selector emitted events")
			}
		})
	}
}

func TestEpisodeSelectionPinnedChunksLongIDAndReopen(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { st.Close() }()
	id := strings.Repeat("long-界-", 3000)
	_, p := selectionFixture(t, st, id, false)
	first, err := selectionPut(st, p, episodeHeadExpectation{})
	if err != nil {
		t.Fatal(err)
	}
	token, err := readEpisodeSelectionToken(st, id, 512)
	if err != nil {
		t.Fatal(err)
	}
	tokenRaw, _ := json.Marshal(token)
	if len(tokenRaw) > 512 || strings.Contains(string(tokenRaw), "long-") || token.ResultKey != first.Head.ResultKey {
		t.Fatal("long ID escaped bounded token")
	}
	var assembled []byte
	offset := 0
	pages := 0
	for {
		chunk, err := readEpisodeResultChunk(st, id, token.ResultKey, offset, 4096)
		if err != nil {
			t.Fatal(err)
		}
		raw, _ := json.Marshal(chunk)
		if len(raw) > 4096 {
			t.Fatal("chunk exceeds complete envelope budget")
		}
		assembled = append(assembled, chunk.Data...)
		pages++
		if pages == 1 {
			p.Result.Assertions[0].Reason = "assertion-dependency"
			if _, err := selectionPut(st, p, episodeHeadExpectation{Exists: true, Raw: first.HeadRaw}); err != nil {
				t.Fatal(err)
			}
		}
		if chunk.NextOffset == -1 {
			break
		}
		if chunk.NextOffset <= offset || len(chunk.Data) == 0 {
			t.Fatal("chunk did not progress")
		}
		offset = chunk.NextOffset
	}
	if pages < 2 || !bytes.Equal(assembled, first.ResultRaw) {
		t.Fatal("changing head spliced result bytes")
	}
	eof, err := readEpisodeResultChunk(st, id, token.ResultKey, len(assembled), 512)
	if err != nil || len(eof.Data) != 0 || eof.NextOffset != -1 {
		t.Fatal("EOF invalid")
	}
	if _, err := readEpisodeResultChunk(st, id, token.ResultKey, len(assembled)+1, 512); err == nil {
		t.Fatal("past EOF accepted")
	}
	before := dpRows(t, st)
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	st, err = Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, dpRows(t, st)) {
		t.Fatal("reopen altered selection bytes")
	}
	current, err := readEpisodeSelection(st, id)
	if err != nil || current.Head.Revision != 2 {
		t.Fatal("reopened current result wrong", err)
	}
}

func TestEpisodeSelectionHeadLocalCounterAndOverflow(t *testing.T) {
	st := openTemp(t)
	_, p := selectionFixture(t, st, "counter", false)
	a, err := selectionPut(st, p, episodeHeadExpectation{})
	if err != nil {
		t.Fatal(err)
	}
	p.Result.Assertions[0].Reason = "assertion-dependency"
	b, err := selectionPut(st, p, episodeHeadExpectation{Exists: true, Raw: a.HeadRaw})
	if err != nil {
		t.Fatal(err)
	}
	// Deliberate raw fixture demonstrates LOCAL validity is not history count.
	b.Head.Revision = 999
	raw, err := encodeEpisodeHead(b.Head)
	if err != nil {
		t.Fatal(err)
	}
	dpSet(t, st, episodeHeadKey("counter"), raw)
	read, err := readEpisodeSelection(st, "counter")
	if err != nil || read.Head.Revision != 999 {
		t.Fatal("unexpected full-history counter claim")
	}
	b.Head.Revision = ^uint64(0)
	raw, err = encodeEpisodeHead(b.Head)
	if err != nil {
		t.Fatal(err)
	}
	dpSet(t, st, episodeHeadKey("counter"), raw)
	before := dpRows(t, st)
	if _, err := selectionPut(st, p, episodeHeadExpectation{Exists: true, Raw: raw}); err != nil {
		t.Fatal("max-counter unchanged retry should remain no-op")
	}
	p.Result.Assertions[0].Reason = "identity-dependency"
	if _, err := selectionPut(st, p, episodeHeadExpectation{Exists: true, Raw: raw}); err == nil {
		t.Fatal("counter overflow accepted")
	}
	if !reflect.DeepEqual(before, dpRows(t, st)) {
		t.Fatal("overflow changed rows")
	}
	b.Head.Revision = 1
	raw, _ = encodeEpisodeHead(b.Head)
	dpSet(t, st, episodeHeadKey("counter"), raw)
	if _, err := readEpisodeSelection(st, "counter"); err == nil {
		t.Fatal("revision1 with predecessor accepted")
	}
}

func TestEpisodeSelectionAdoptionReservedFamilies(t *testing.T) {
	for _, prefix := range []string{identityEpisodePrefix, identityEpisodeHeadPrefix} {
		for _, suffix := range []string{"", "malformed"} {
			t.Run(prefix+suffix, func(t *testing.T) {
				st := openTemp(t)
				_, manifest := adoptionFixture(t, st)
				dpSet(t, st, prefix+suffix, []byte{})
				assertAdoptionRefusal(t, st, manifest, false)
				assertAdoptionRefusal(t, st, manifest, true)
			})
		}
	}
}
