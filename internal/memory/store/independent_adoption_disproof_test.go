package store

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/dgraph-io/badger/v4"
)

func dpRows(t *testing.T, s *Store) map[string][]byte {
	t.Helper()
	out := map[string][]byte{}
	if err := s.db.View(func(tx *badger.Txn) error {
		it := tx.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			v, err := it.Item().ValueCopy(nil)
			if err != nil {
				return err
			}
			out[string(it.Item().KeyCopy(nil))] = v
		}
		return nil
	}); err != nil {
		t.Fatal("snapshot failed")
	}
	return out
}

func dpSet(t *testing.T, s *Store, k string, v []byte) {
	t.Helper()
	if err := s.db.Update(func(tx *badger.Txn) error {
		if v == nil {
			return tx.Delete([]byte(k))
		}
		return tx.Set([]byte(k), v)
	}); err != nil {
		t.Fatal("fixture mutation failed")
	}
}

func dpJSON(t *testing.T, v any) []byte {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal("fixture JSON failed")
	}
	return raw
}

func dpFixture(t *testing.T) (*Store, []byte, Entity) {
	t.Helper()
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal("fixture open failed")
	}
	t.Cleanup(func() { s.Close() })
	e := Entity{Slug: "center", Name: "Synthetic Center", Type: "project", Description: "original", Aliases: []string{"first", "second"}, RepoRefs: []string{"/synthetic/one", "/synthetic/two"}, CreatedAt: time.Date(2025, 3, 2, 1, 2, 3, 456, time.UTC), LastSeen: time.Date(2026, 3, 2, 1, 2, 3, 456, time.UTC)}
	for _, slug := range []string{"center", "aaa", "zzz"} {
		other := e
		other.Slug = slug
		dpSet(t, s, "en:"+slug, dpJSON(t, other))
	}
	for _, k := range []string{"fa:opaque", "adj:opaque", "ep:opaque", "pq:opaque", "al:opaque", "future:opaque"} {
		dpSet(t, s, k, []byte{255, 0, 254})
	}
	raw, err := captureLegacyInventory(s)
	if err != nil {
		t.Fatal("fixture capture failed")
	}
	return s, raw, e
}

func dpRefuse(t *testing.T, s *Store, raw []byte) {
	t.Helper()
	before := dpRows(t, s)
	for _, apply := range []bool{false, true} {
		r, err := adoptLegacyInventory(s, raw, apply)
		if err != errLegacyAdoption || r != (legacyAdoptionReport{}) {
			t.Fatal("refusal missing, unsafe error, or partial report")
		}
		if !reflect.DeepEqual(before, dpRows(t, s)) {
			t.Fatal("failed transaction changed original rows")
		}
	}
}

func dpRoundTrip(t *testing.T, s *Store, raw []byte) {
	t.Helper()
	before := dpRows(t, s)
	var m legacyInventoryManifest
	if json.Unmarshal(raw, &m) != nil {
		t.Fatal("manifest invalid")
	}
	digest := sha256.Sum256(raw)
	events := 0
	s.SetObserver(func(Event) { events++ })
	p, err := adoptLegacyInventory(s, raw, false)
	if err != nil || p.Applied || p.Inventory != hex.EncodeToString(digest[:]) || p.Entities != len(m.Entries) {
		t.Fatal("preview report wrong")
	}
	if !reflect.DeepEqual(before, dpRows(t, s)) || events != 0 {
		t.Fatal("preview wrote")
	}
	r, err := adoptLegacyInventory(s, raw, true)
	p.Applied = true
	if err != nil || r != p {
		t.Fatal("apply diverged")
	}
	after := dpRows(t, s)
	if len(after) != len(before)+len(m.Entries)+1 {
		t.Fatal("unexpected write set size")
	}
	for k, v := range before {
		if !bytes.Equal(v, after[k]) {
			t.Fatal("original row changed")
		}
	}
	var addedBytes uint64
	for k, v := range after {
		if _, ok := before[k]; !ok {
			addedBytes += uint64(len(k) + len(v))
			if !strings.HasPrefix(k, "il:") && k != legacyAdoptionMarkerKey {
				t.Fatal("unexpected added family")
			}
		}
	}
	if addedBytes != r.Bytes || events != 0 {
		t.Fatal("bytes/events wrong")
	}
	for _, entry := range m.Entries {
		var expected Entity
		if json.Unmarshal(entry.Value, &expected) != nil {
			t.Fatal("source decode failed")
		}
		a, e, err := readActiveLegacyIdentity(s, expected.Slug)
		if err != nil || a.Inventory != r.Inventory || !reflect.DeepEqual(e, expected) {
			t.Fatal("recognition failed")
		}
	}
	dpRefuse(t, s, raw)
	t.Logf("rows=%d entities=%d manifest_bytes=%d manifest_sha256=%s added_bytes=%d originals_equal=true events=%d", len(before), len(m.Entries), len(raw), r.Inventory, r.Bytes, events)
}

func TestDisproofAdoptionExactWritesAndOwnedBuffers(t *testing.T) {
	s, raw, _ := dpFixture(t)
	owned := bytes.Clone(raw)
	m, _, err := decodeLegacyInventory(raw)
	if err != nil {
		t.Fatal("decode failed")
	}
	m.Entries[0].Key[0] = 'x'
	m.Entries[0].Value[0] = 'x'
	if !bytes.Equal(raw, owned) {
		t.Fatal("decoded buffers alias input")
	}
	second, err := captureLegacyInventory(s)
	if err != nil {
		t.Fatal("capture failed")
	}
	second[0] = 'x'
	if !bytes.Equal(raw, owned) {
		t.Fatal("captured buffers alias")
	}
	dpRoundTrip(t, s, raw)
}

func TestDisproofAdoptionEveryMetadataAndInventoryDrift(t *testing.T) {
	for _, kind := range []string{"name", "type", "description", "aliases-order", "repo-order", "created", "last-seen", "unknown-json", "byte-space", "missing-first", "missing-middle", "missing-last", "extra-first", "extra-middle", "extra-last"} {
		t.Run(kind, func(t *testing.T) {
			s, raw, e := dpFixture(t)
			switch kind {
			case "name":
				e.Name += "!"
			case "type":
				e.Type = "tool"
			case "description":
				e.Description += "!"
			case "aliases-order":
				e.Aliases[0], e.Aliases[1] = e.Aliases[1], e.Aliases[0]
			case "repo-order":
				e.RepoRefs[0], e.RepoRefs[1] = e.RepoRefs[1], e.RepoRefs[0]
			case "created":
				e.CreatedAt = e.CreatedAt.Add(time.Nanosecond)
			case "last-seen":
				e.LastSeen = e.LastSeen.Add(time.Nanosecond)
			case "unknown-json":
				v := dpJSON(t, e)
				dpSet(t, s, "en:center", append(v[:len(v)-1], []byte(`,"new":1}`)...))
				dpRefuse(t, s, raw)
				return
			case "byte-space":
				dpSet(t, s, "en:center", append(dpJSON(t, e), ' '))
				dpRefuse(t, s, raw)
				return
			case "missing-first", "missing-middle", "missing-last":
				slug := map[string]string{"missing-first": "aaa", "missing-middle": "center", "missing-last": "zzz"}[kind]
				dpSet(t, s, "en:"+slug, nil)
				dpRefuse(t, s, raw)
				return
			case "extra-first", "extra-middle", "extra-last":
				e.Slug = map[string]string{"extra-first": "a", "extra-middle": "middle", "extra-last": "zzzz"}[kind]
			}
			dpSet(t, s, "en:"+e.Slug, dpJSON(t, e))
			dpRefuse(t, s, raw)
		})
	}
}

func TestDisproofAdoptionReservedFamiliesEmptyAndMalformed(t *testing.T) {
	for _, p := range []string{"il:", "il-consumed:", "ig:", "iga:", "io:", "io-result:", "meta:identity_"} {
		for _, suffix := range []string{"", "a", string([]byte{255})} {
			for _, value := range [][]byte{{}, {255}, []byte("null")} {
				s, raw, _ := dpFixture(t)
				dpSet(t, s, p+suffix, value)
				dpRefuse(t, s, raw)
			}
		}
	}
}

func TestDisproofAdoptionCanonicalInputs(t *testing.T) {
	s, raw, _ := dpFixture(t)
	for _, bad := range [][]byte{
		nil, []byte("null"), []byte(`{"version":1,"entries":[] }`),
		[]byte(`{"version":1,"entries":[],"version":1}`), []byte(`{"version":1,"entries":null}`),
		bytes.Replace(raw, []byte(`"version":1`), []byte(`"version":1.0`), 1),
		bytes.Replace(raw, []byte(`"version":1`), []byte(`"Version":1`), 1),
		bytes.Replace(raw, []byte(`"key":`), []byte(`"unknown":0,"key":`), 1),
		append(bytes.Clone(raw), '\n'), append(bytes.Clone(raw), raw...),
	} {
		dpRefuse(t, s, bad)
	}
	for _, kind := range []string{"reversed", "duplicate", "wrong-prefix", "wrong-value", "missing-member"} {
		var m legacyInventoryManifest
		json.Unmarshal(raw, &m)
		switch kind {
		case "reversed":
			m.Entries[0], m.Entries[2] = m.Entries[2], m.Entries[0]
		case "duplicate":
			m.Entries[1] = m.Entries[0]
		case "wrong-prefix":
			m.Entries[0].Key = []byte("il:aaa")
		case "wrong-value":
			m.Entries[0].Value = m.Entries[1].Value
		case "missing-member":
			m.Entries = m.Entries[1:]
		}
		dpRefuse(t, s, dpJSON(t, m))
	}
}

func TestDisproofAdoptionCanonicalControlsAndReadYourWrites(t *testing.T) {
	for _, kind := range []string{"marker-duplicate", "marker-unknown", "marker-spaced", "marker-negative", "marker-upperhash", "anchor-duplicate", "anchor-unknown", "anchor-nonutc", "entity-noncanonical", "consumed-empty", "consumed-corrupt", "generation-empty", "recreated-consumed"} {
		t.Run(kind, func(t *testing.T) {
			s, raw, e := dpFixture(t)
			if _, err := adoptLegacyInventory(s, raw, true); err != nil {
				t.Fatal("apply failed")
			}
			before := dpRows(t, s)
			err := s.AtomicWrite(func(tx *Store) error {
				k := legacyAdoptionMarkerKey
				v := bytes.Clone(before[k])
				switch kind {
				case "marker-duplicate":
					v = append(v[:len(v)-1], []byte(`,"version":1}`)...)
				case "marker-unknown":
					v = append(v[:len(v)-1], []byte(`,"unknown":0}`)...)
				case "marker-spaced":
					v = append(v, ' ')
				case "marker-negative":
					v = bytes.Replace(v, []byte(`"entities":3`), []byte(`"entities":-1`), 1)
				case "marker-upperhash":
					var m legacyAdoptionMarker
					json.Unmarshal(v, &m)
					m.Inventory = strings.ToUpper(m.Inventory)
					v = dpJSON(t, m)
				case "anchor-duplicate", "anchor-unknown", "anchor-nonutc":
					k = "il:" + e.Slug
					v = bytes.Clone(before[k])
					if kind == "anchor-duplicate" {
						v = append(v[:len(v)-1], []byte(`,"version":1}`)...)
					} else if kind == "anchor-unknown" {
						v = append(v[:len(v)-1], []byte(`,"unknown":0}`)...)
					} else {
						v = bytes.Replace(v, []byte("Z\""), []byte("+00:00\""), 1)
					}
				case "entity-noncanonical":
					k = "en:" + e.Slug
					v = append(bytes.Clone(before[k]), ' ')
				case "consumed-empty", "consumed-corrupt", "recreated-consumed":
					k = "il-consumed:" + e.Slug
					v = []byte{}
					if kind == "consumed-corrupt" {
						v = []byte{255}
					}
					if kind == "recreated-consumed" {
						if err := tx.DeleteEntity(e.Slug); err != nil {
							return err
						}
						if err := tx.PutEntity(e); err != nil {
							return err
						}
					}
				case "generation-empty":
					k = "ig:" + e.Slug
					v = []byte{}
				}
				if err := tx.txn.Set([]byte(k), v); err != nil {
					return err
				}
				a, got, err := readActiveLegacyIdentity(tx, e.Slug)
				if err != errLegacyAdoption || a != (legacyIdentityAnchor{}) || !reflect.DeepEqual(got, Entity{}) {
					t.Fatal("reader accepted staged corrupt/consumed controls")
				}
				return errors.New("independent rollback")
			})
			if err == nil || !reflect.DeepEqual(before, dpRows(t, s)) {
				t.Fatal("reader rollback changed store")
			}
			if _, _, err := readActiveLegacyIdentity(s, e.Slug); err != nil {
				t.Fatal("staged corruption escaped rollback")
			}
		})
	}
}

func TestDisproofAdoptionAllOrdinaryWriterLocks(t *testing.T) {
	for _, name := range []string{"entity", "delete-entity", "episode", "fact", "invalidate", "delete-fact", "relocate", "cursor", "claim", "drop", "drop-rehome", "pending", "delete-pending", "meta-time", "meta-json", "attest", "value-evidence", "atomic"} {
		t.Run(name, func(t *testing.T) {
			s, _, e := dpFixture(t)
			f := Fact{Src: e.Slug, Relation: "related_to", Dst: "aaa", Fact: "synthetic", ValidFrom: e.CreatedAt, Confidence: 1}
			if err := s.PutFact(f); err != nil {
				t.Fatal("fact fixture failed")
			}
			call := func() error {
				switch name {
				case "entity":
					return s.PutEntity(e)
				case "delete-entity":
					return s.DeleteEntity(e.Slug)
				case "episode":
					return s.PutEpisode(Episode{ID: "probe"})
				case "fact":
					return s.PutFact(f)
				case "invalidate":
					return s.InvalidateFact(f.Src, f.Relation, f.Dst, f.ValidFrom, e.LastSeen)
				case "delete-fact":
					return s.DeleteFact(f.Src, f.Relation, f.Dst, f.ValidFrom)
				case "relocate":
					updated := f
					updated.Dst = "zzz"
					return s.RelocateFact(f, updated)
				case "cursor":
					return s.PutCursor(Cursor{Path: "/synthetic"})
				case "claim":
					return s.ClaimAlias("probe", e.Slug)
				case "drop":
					_, err := s.DropAlias(e.Slug, "first")
					return err
				case "drop-rehome":
					_, err := s.DropAliasRehome(e.Slug, "first", "aaa")
					return err
				case "pending":
					return s.PutPending(PendingEpisode{ID: "probe"})
				case "delete-pending":
					return s.DeletePending("probe")
				case "meta-time":
					return s.PutMetaTime("probe", e.LastSeen)
				case "meta-json":
					return s.PutMetaJSON("probe", true)
				case "attest":
					_, err := s.AttestAlias(e.Slug, "first", "probe")
					return err
				case "value-evidence":
					return s.RecordValueEvidence("probe", "episode")
				case "atomic":
					return s.AtomicWrite(func(tx *Store) error { return tx.PutCursor(Cursor{Path: "/synthetic"}) })
				}
				return nil
			}
			s.maintenanceMu.Lock()
			started, done := make(chan struct{}), make(chan error, 1)
			go func() { close(started); done <- call() }()
			<-started
			select {
			case <-done:
				s.maintenanceMu.Unlock()
				t.Fatal("ordinary writer bypassed root maintenance lock")
			case <-time.After(15 * time.Millisecond):
			}
			s.maintenanceMu.Unlock()
			select {
			case <-done:
			case <-time.After(3 * time.Second):
				t.Fatal("ordinary writer deadlocked")
			}
		})
	}
}

func TestDisproofAdoptionFacadeRejectsBeforeDecodeAndPoisons(t *testing.T) {
	for _, apply := range []bool{false, true} {
		for _, phase := range []string{"ordinary-nested", "body", "finalizer"} {
			s, _, _ := dpFixture(t)
			before := dpRows(t, s)
			refuse := func(tx *Store) error {
				if err := tx.txn.Set([]byte("independent:staged"), []byte("rollback")); err != nil {
					return err
				}
				r, err := adoptLegacyInventory(tx, []byte("invalid-manifest"), apply)
				if err != errLegacyAdoption || r != (legacyAdoptionReport{}) {
					t.Fatal("facade refusal missing")
				}
				return nil
			}
			var err error
			if phase == "ordinary-nested" {
				err = s.AtomicWrite(func(tx *Store) error { _ = tx.AtomicWrite(refuse); return nil })
			} else if phase == "body" {
				err = runIdentityAdmission(s, refuse, func(*Store) error { return nil })
			} else {
				err = runIdentityAdmission(s, func(*Store) error { return nil }, refuse)
			}
			if err != errLegacyAdoption || !reflect.DeepEqual(before, dpRows(t, s)) {
				t.Fatal("ignored facade refusal committed")
			}
		}
	}
}

func TestDisproofAdoptionStorageErrorsStatic(t *testing.T) {
	unsafe := fmt.Errorf("secret source data: %w", badger.ErrTxnTooBig)
	for _, err := range []error{unsafe, fmt.Errorf("secret source data: %w", badger.ErrConflict), errors.New("secret source data")} {
		got := legacyAdoptionFailure(err)
		if strings.Contains(got.Error(), "secret") || !errors.Is(got, errLegacyAdoption) {
			t.Fatal("unsafe error disclosure")
		}
	}
	s, raw, _ := dpFixture(t)
	if err := s.Close(); err != nil {
		t.Fatal("close failed")
	}
	for _, apply := range []bool{false, true} {
		r, err := adoptLegacyInventory(s, raw, apply)
		if err != errLegacyAdoption || r != (legacyAdoptionReport{}) {
			t.Fatal("closed DB returned unsafe error/report")
		}
	}
}

func TestDisproofAdoptionMeasuredLateFailureRollback(t *testing.T) {
	for _, kind := range []string{"transaction", "value"} {
		t.Run(kind, func(t *testing.T) {
			db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true).WithLogger(nil).WithMemTableSize(2 << 20).WithValueThreshold(4096))
			if err != nil {
				t.Fatal("storage fixture open failed")
			}
			s := &Store{db: db}
			defer s.Close()
			count := 3000
			if kind == "value" {
				count = 3
			}
			for i := 0; i < count; i++ {
				e := Entity{Slug: fmt.Sprintf("independent-%04d", i), Name: "Synthetic Identity", CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
				raw := dpJSON(t, e)
				if kind == "value" && i == count-1 {
					found := false
					for n := 3800; n < 4100; n++ {
						e.Name = strings.Repeat("q", n)
						raw = dpJSON(t, e)
						_, anchor, err := makeLegacyIdentityAnchor(strings.Repeat("b", 64), []byte("en:"+e.Slug), raw)
						if err != nil {
							t.Fatal("anchor probe failed")
						}
						if len(raw) <= 4096 && len(anchor) > 4096 {
							found = true
							break
						}
					}
					if !found {
						t.Fatal("no late value boundary fixture")
					}
				}
				dpSet(t, s, "en:"+e.Slug, raw)
			}
			raw, err := captureLegacyInventory(s)
			if err != nil {
				t.Fatal("late fixture capture failed")
			}
			m, id, err := decodeLegacyInventory(raw)
			if err != nil {
				t.Fatal("late fixture decode failed")
			}
			before := dpRows(t, s)
			probe := db.NewTransaction(true)
			staged := 0
			for _, entry := range m.Entries {
				k, v, err := makeLegacyIdentityAnchor(id, entry.Key, entry.Value)
				if err != nil {
					probe.Discard()
					t.Fatal("probe anchor failed")
				}
				if err := probe.Set(k, v); err != nil {
					break
				}
				staged++
			}
			probe.Discard()
			if staged == 0 || staged >= len(m.Entries) {
				t.Fatal("fixture did not prove late staging failure")
			}
			for _, apply := range []bool{false, true} {
				r, err := adoptLegacyInventory(s, raw, apply)
				if !errors.Is(err, errLegacyAdoption) || r != (legacyAdoptionReport{}) {
					t.Fatal("late failure produced report")
				}
				if kind == "transaction" && !errors.Is(err, badger.ErrTxnTooBig) {
					t.Fatal("transaction class lost")
				}
				if kind == "value" && err != errLegacyAdoption {
					t.Fatal("unsafe value error")
				}
				if !reflect.DeepEqual(before, dpRows(t, s)) {
					t.Fatal("late failure left partial writes")
				}
			}
			t.Logf("independently staged=%d before failure; originals_equal=true preview_and_apply_rollback=true", staged)
		})
	}
}

func TestDisproofAdoptionIndependentActualBackup(t *testing.T) {
	if os.Getenv("SCRY_DISPROOF_RESTORE") != "1" {
		t.Skip("explicit private restore only")
	}
	const source = "/tmp/scry-foundation-closure-sep06.8IEPu5/memory-20260906T051519Z.badger"
	f, err := os.Open(source)
	if err != nil {
		t.Fatal("backup open failed")
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		t.Fatal("backup hash failed")
	}
	if hex.EncodeToString(h.Sum(nil)) != "193f19b3491c74782d7556e3059714cbb7ba65005603ce38f84d70efcf52c256" {
		t.Fatal("backup hash mismatch")
	}
	if _, err := f.Seek(0, 0); err != nil {
		t.Fatal("backup rewind failed")
	}
	path, err := os.MkdirTemp("/tmp/scry-adoption-disproof-jRBRuU", "replica-")
	if err != nil {
		t.Fatal("private directory failed")
	}
	s, err := Open(path)
	if err != nil {
		t.Fatal("private replica open failed")
	}
	defer s.Close()
	if err := s.Restore(f); err != nil {
		t.Fatal("private replica restore failed")
	}
	raw, err := captureLegacyInventory(s)
	if err != nil {
		t.Fatal("private replica capture failed")
	}
	dpRoundTrip(t, s, raw)
	t.Log("private replica path=" + path + "; bounded measurement only, no live or lifecycle approval")
}
