package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/dgraph-io/badger/v4"
)

func gradeGenerationBirth() identityBirth {
	return identityBirth{EpisodeID: "initial-exact-episode", Occurrence: 7, Slug: "atlas-guide", Name: "Atlas Guide", Origin: "endpoint", CreatedAt: time.Date(2026, 9, 6, 2, 3, 4, 456, time.UTC)}
}

func gradeGenerationSnapshot(t *testing.T, st *Store) map[string][]byte {
	t.Helper()
	out := make(map[string][]byte)
	err := st.db.View(func(tx *badger.Txn) error {
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
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func gradeGenerationSet(t *testing.T, st *Store, key string, value []byte) {
	t.Helper()
	if err := st.db.Update(func(tx *badger.Txn) error {
		if value == nil {
			return tx.Delete([]byte(key))
		}
		return tx.Set([]byte(key), bytes.Clone(value))
	}); err != nil {
		t.Fatal(err)
	}
}

func gradeGenerationBirthCommit(t *testing.T, st *Store, b identityBirth) {
	t.Helper()
	if err := st.AtomicWrite(func(tx *Store) error {
		s, err := beginIdentityGeneration(tx, b)
		if err != nil {
			return err
		}
		if err = tx.PutEntity(Entity{Slug: b.Slug, Name: b.Name, CreatedAt: b.CreatedAt, Type: "concept"}); err != nil {
			return err
		}
		if err = tx.PutFact(Fact{Src: b.Slug, Relation: "status", Value: "ready", Fact: "Synthetic Atlas guide is ready.", ValidFrom: b.CreatedAt, Confidence: .875, Episodes: []string{b.EpisodeID}}); err != nil {
			return err
		}
		if err = tx.PutEpisode(Episode{ID: b.EpisodeID, OccurredAt: b.CreatedAt}); err != nil {
			return err
		}
		return s.finishSupported()
	}); err != nil {
		t.Fatal(err)
	}
}

func gradeGenerationVote(t *testing.T, st *Store, slug, id, alias string, expected int) {
	t.Helper()
	if err := st.AtomicWrite(func(tx *Store) error {
		s, err := loadIdentityGeneration(tx, slug, id)
		if err != nil {
			return err
		}
		for repeat := 0; repeat < 3; repeat++ {
			count, err := s.attest(alias)
			if err != nil {
				return err
			}
			if count != expected {
				t.Fatalf("count=%d expected=%d", count, expected)
			}
		}
		if err := tx.PutEpisode(Episode{ID: id, OccurredAt: time.Date(2026, 9, 6, 3, 0, 0, 0, time.UTC)}); err != nil {
			return err
		}
		return s.finishSupported()
	}); err != nil {
		t.Fatal(err)
	}
}

func TestGradeGenerationDelayedEvidenceCapacityAndExactRestore(t *testing.T) {
	for _, oldCount := range []int{1, 8} {
		t.Run(fmt.Sprint(oldCount), func(t *testing.T) {
			st := openTemp(t)
			b := gradeGenerationBirth()
			alias := Normalize("Northern Guide")
			for i := 0; i < oldCount; i++ {
				if _, err := st.AttestAlias(b.Slug, alias, fmt.Sprintf("orphan-%d", i)); err != nil {
					t.Fatal(err)
				}
			}
			// Retain opaque legacy JSON that ordinary canonical rewrites would lose.
			legacyKey := prefixAttest + b.Slug + ":" + alias
			legacy := gradeGenerationSnapshot(t, st)[legacyKey]
			legacy = append(bytes.Clone(legacy[:len(legacy)-1]), []byte(", \"future_field\" : [1,2,3] }\n")...)
			gradeGenerationSet(t, st, legacyKey, legacy)
			gradeGenerationBirthCommit(t, st, b) // birth has no alias proposals
			factsBefore, err := st.AllFacts()
			if err != nil {
				t.Fatal(err)
			}
			for i := 1; i <= 14; i++ {
				id := fmt.Sprintf("new-%02d-", i) + strings.Repeat("full-id-", 40)
				gradeGenerationVote(t, st, b.Slug, id, alias, i)
				if !bytes.Equal(gradeGenerationSnapshot(t, st)[legacyKey], legacy) {
					t.Fatal("legacy bytes changed")
				}
			}
			gradeGenerationVote(t, st, b.Slug, "different-alias-episode", "southern-guide", 1)
			b2 := b
			b2.Slug = "zephyr-guide"
			b2.Name = "Zephyr Guide"
			b2.EpisodeID = "other-birth"
			gradeGenerationBirthCommit(t, st, b2)
			gradeGenerationVote(t, st, b2.Slug, "new-01-"+strings.Repeat("full-id-", 40), alias, 1)
			factsAfter, err := st.FactsFrom(b.Slug, true)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(factsBefore, factsAfter) {
				t.Fatal("existing real fact content or provenance changed")
			}
			if _, found, err := st.ResolveAlias(alias); err != nil || found {
				t.Fatal("primitive wrote routing claim")
			}
			before := gradeGenerationSnapshot(t, st)
			rec, _, _ := identityBirthRecord(b)
			for i := 1; i <= 14; i++ {
				id := fmt.Sprintf("new-%02d-", i) + strings.Repeat("full-id-", 40)
				a := generationAttestation{Generation: rec.ID, Alias: alias, EpisodeID: id}
				var got generationAttestation
				if err := json.Unmarshal(before[string(generationAttestationKey(a))], &got); err != nil || got != a {
					t.Fatal("complete fresh ID not retained")
				}
			}
			var backup bytes.Buffer
			if n, err := st.Backup(&backup); err != nil || n == 0 {
				t.Fatalf("backup: %v", err)
			}
			dir := filepath.Join(t.TempDir(), "restore")
			db, err := badger.Open(badger.DefaultOptions(dir).WithLogger(nil).WithCompression(0))
			if err != nil {
				t.Fatal(err)
			}
			if err = db.Load(bytes.NewReader(backup.Bytes()), 16); err != nil {
				t.Fatal(err)
			}
			if err = db.Close(); err != nil {
				t.Fatal(err)
			}
			r, err := Open(dir)
			if err != nil {
				t.Fatal(err)
			}
			defer r.Close()
			if !reflect.DeepEqual(before, gradeGenerationSnapshot(t, r)) {
				t.Fatal("direct restore/reopen changed raw keys or values")
			}
			gradeGenerationVote(t, r, b.Slug, "new-14-"+strings.Repeat("full-id-", 40), alias, 14)
			gradeGenerationVote(t, r, b.Slug, "fifteenth-fresh", alias, 15)
		})
	}
}

func TestGradeGenerationRejectsSelectorAndEntityCorruption(t *testing.T) {
	cases := []string{"missing", "invalid-json", "unknown-field", "version", "id", "tuple", "key-body-slug", "noncanonical", "name-drift", "time-drift", "missing-entity", "body-slug-drift"}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			st := openTemp(t)
			b := gradeGenerationBirth()
			gradeGenerationBirthCommit(t, st, b)
			key := identityGenerationPrefix + b.Slug
			raw := gradeGenerationSnapshot(t, st)[key]
			var record identityGenerationRecord
			if err := json.Unmarshal(raw, &record); err != nil {
				t.Fatal(err)
			}
			switch name {
			case "missing":
				raw = nil
			case "invalid-json":
				raw = []byte("{")
			case "unknown-field":
				raw = append(bytes.Clone(raw[:len(raw)-1]), []byte(",\"unknown\":true}")...)
			case "version":
				record.Version = 2
				raw, _ = json.Marshal(record)
			case "id":
				record.ID = strings.Repeat("a", 64)
				raw, _ = json.Marshal(record)
			case "tuple":
				record.Birth.Occurrence++
				raw, _ = json.Marshal(record)
			case "key-body-slug":
				record.Birth.Slug = "other-guide"
				record.Birth.Name = "Other Guide"
				_, raw, _ = identityBirthRecord(record.Birth)
			case "noncanonical":
				raw = append(raw, ' ')
			case "missing-entity":
				gradeGenerationSet(t, st, prefixEntity+b.Slug, nil)
			case "name-drift", "time-drift", "body-slug-drift":
				e, err := st.GetEntity(b.Slug)
				if err != nil {
					t.Fatal(err)
				}
				if name == "name-drift" {
					e.Name = "Atlas guide"
				}
				if name == "time-drift" {
					e.CreatedAt = e.CreatedAt.Add(time.Nanosecond)
				}
				if name == "body-slug-drift" {
					e.Slug = "other-guide"
				}
				eraw, _ := json.Marshal(e)
				gradeGenerationSet(t, st, prefixEntity+b.Slug, eraw)
			}
			gradeGenerationSet(t, st, key, raw)
			before := gradeGenerationSnapshot(t, st)
			if err := st.AtomicWrite(func(tx *Store) error { _, err := loadIdentityGeneration(tx, b.Slug, "next"); return err }); err == nil {
				t.Fatal("corrupt selector/entity accepted")
			}
			if !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
				t.Fatal("refusal rewrote preserved state")
			}
		})
	}
}

func TestGradeGenerationRejectsLedgerCorruptionAndMissingEpisodes(t *testing.T) {
	for _, name := range []string{"unknown-field", "invalid-json", "noncanonical", "generation", "alias", "empty-episode", "episode-body-key", "extra-key-suffix", "alias-key", "missing-old-provenance", "missing-current-provenance", "missing-birth-provenance"} {
		t.Run(name, func(t *testing.T) {
			st := openTemp(t)
			b := gradeGenerationBirth()
			gradeGenerationBirthCommit(t, st, b)
			gradeGenerationVote(t, st, b.Slug, "vote-1", "north-guide", 1)
			rec, _, _ := identityBirthRecord(b)
			a := generationAttestation{Generation: rec.ID, Alias: "north-guide", EpisodeID: "vote-1"}
			key := string(generationAttestationKey(a))
			raw := gradeGenerationSnapshot(t, st)[key]
			switch name {
			case "unknown-field":
				raw = append(bytes.Clone(raw[:len(raw)-1]), []byte(",\"extra\":1}")...)
			case "invalid-json":
				raw = []byte("[")
			case "noncanonical":
				raw = append(raw, '\n')
			case "generation":
				a.Generation = strings.Repeat("a", 64)
				raw, _ = json.Marshal(a)
			case "alias":
				a.Alias = "south-guide"
				raw, _ = json.Marshal(a)
			case "empty-episode":
				a.EpisodeID = ""
				raw, _ = json.Marshal(a)
			case "episode-body-key":
				a.EpisodeID = "different-full-id"
				raw, _ = json.Marshal(a)
			case "extra-key-suffix":
				gradeGenerationSet(t, st, key+":extra", raw)
			case "alias-key":
				gradeGenerationSet(t, st, identityGenerationAttestPrefix+rec.ID+":"+strings.Repeat("b", 64)+":"+generationDigest([]byte(a.EpisodeID)), raw)
			case "missing-old-provenance":
				gradeGenerationSet(t, st, prefixEpisode+"vote-1", nil)
			case "missing-birth-provenance":
				gradeGenerationSet(t, st, prefixEpisode+b.EpisodeID, nil)
			}
			gradeGenerationSet(t, st, key, raw)
			before := gradeGenerationSnapshot(t, st)
			events := 0
			st.SetObserver(func(Event) { events++ })
			err := st.AtomicWrite(func(tx *Store) error {
				s, err := loadIdentityGeneration(tx, b.Slug, "vote-2")
				if err != nil {
					return err
				}
				if _, err = s.attest("north-guide"); err != nil {
					return err
				}
				if name != "missing-current-provenance" {
					if err = tx.PutEpisode(Episode{ID: "vote-2"}); err != nil {
						return err
					}
				}
				return s.finishSupported()
			})
			if err == nil {
				t.Fatal("malformed ledger or missing provenance accepted")
			}
			if events != 0 || !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
				t.Fatal("failed ledger validation changed bytes/events")
			}
		})
	}
}

func TestGradeGenerationLifetimeRollbackBuffersAndStaleHandle(t *testing.T) {
	for _, mode := range []string{"error", "panic", "staging-failure", "commit"} {
		t.Run(mode, func(t *testing.T) {
			st := openTemp(t)
			b := gradeGenerationBirth()
			gradeGenerationBirthCommit(t, st, b)
			if _, err := st.AttestAlias(b.Slug, "old-alias", "old-episode"); err != nil {
				t.Fatal(err)
			}
			before := gradeGenerationSnapshot(t, st)
			events := 0
			st.SetObserver(func(Event) { events++ })
			var saved *identityGenerationSession
			forced := errors.New("forced private rollback")
			var outcome error
			func() {
				defer func() {
					if p := recover(); p != nil {
						if mode != "panic" || p != forced {
							panic(p)
						}
					}
				}()
				outcome = st.AtomicWrite(func(tx *Store) error {
					s, err := loadIdentityGeneration(tx, b.Slug, "next-episode")
					if err != nil {
						return err
					}
					saved = s
					buffer := []byte("fresh-alias")
					alias := string(buffer)
					if _, err = s.attest(alias); err != nil {
						return err
					}
					for i := range buffer {
						buffer[i] = 'x'
					} // caller reuse cannot alter staged keys/values
					if err = tx.PutEpisode(Episode{ID: "next-episode"}); err != nil {
						return err
					}
					switch mode {
					case "error":
						return forced
					case "panic":
						panic(forced)
					case "staging-failure":
						return tx.txn.Set(make([]byte, 70000), []byte("too large key"))
					}
					return s.finishSupported()
				})
			}()
			if saved == nil {
				t.Fatal("did not stage fixture")
			}
			if mode == "commit" {
				if outcome != nil {
					t.Fatal(outcome)
				}
			} else {
				if mode != "panic" && outcome == nil {
					t.Fatal("expected failure")
				}
				if events != 0 || !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
					t.Fatal("rollback changed bytes or emitted events")
				}
			}
			if _, err := saved.attest("should-refuse"); err == nil {
				t.Fatal("expired attest accepted")
			}
			if err := saved.finishSupported(); err == nil {
				t.Fatal("expired finish accepted")
			}
			if err := st.AtomicWrite(func(tx *Store) error {
				if _, err := saved.attest("later"); err == nil {
					t.Fatal("expired handle reused by later transaction")
				}
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			if mode == "commit" {
				gradeGenerationVote(t, st, b.Slug, "next-episode", "fresh-alias", 1)
			}
		})
	}
	st := openTemp(t)
	b := gradeGenerationBirth()
	gradeGenerationBirthCommit(t, st, b)
	before := gradeGenerationSnapshot(t, st)
	err := st.AtomicWrite(func(tx *Store) error {
		s, err := loadIdentityGeneration(tx, b.Slug, "new")
		if err != nil {
			return err
		}
		if err = tx.txn.Set([]byte(identityGenerationPrefix+b.Slug), []byte("changed")); err != nil {
			return err
		}
		_, err = s.attest("north-guide")
		return err
	})
	if err == nil || !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
		t.Fatal("stale handle redirected or failed rollback")
	}
}

func TestGradeGenerationBirthFailuresAndDerivedLedgerOccupancy(t *testing.T) {
	for _, mode := range []string{"existing-selector", "existing-entity", "derived-ledger", "birth-rollback", "missing-birth-episode", "retired-selector"} {
		t.Run(mode, func(t *testing.T) {
			st := openTemp(t)
			b := gradeGenerationBirth()
			rec, raw, _ := identityBirthRecord(b)
			switch mode {
			case "existing-selector":
				gradeGenerationSet(t, st, identityGenerationPrefix+b.Slug, raw)
			case "existing-entity":
				gradeGenerationSet(t, st, prefixEntity+b.Slug, []byte("{}"))
			case "derived-ledger":
				gradeGenerationSet(t, st, identityGenerationAttestPrefix+rec.ID+":opaque", []byte("{\"unknown\":true}"))
			case "retired-selector":
				gradeGenerationBirthCommit(t, st, b)
				gradeGenerationSet(t, st, prefixEntity+b.Slug, nil)
			}
			before := gradeGenerationSnapshot(t, st)
			events := 0
			st.SetObserver(func(Event) { events++ })
			err := st.AtomicWrite(func(tx *Store) error {
				s, err := beginIdentityGeneration(tx, b)
				if err != nil {
					return err
				}
				if _, err = s.attest("north-guide"); err != nil {
					return err
				}
				if mode == "birth-rollback" {
					return errors.New("forced after selector and ledger")
				}
				if err = tx.PutEntity(Entity{Slug: b.Slug, Name: b.Name, CreatedAt: b.CreatedAt}); err != nil {
					return err
				}
				return s.finishSupported()
			})
			if err == nil || events != 0 || !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
				t.Fatal("birth precondition/finalization failure escaped rollback")
			}
		})
	}
}

func TestGradeGenerationPropagatesOwnStagingFailures(t *testing.T) {
	for _, mode := range []string{"selector", "ledger"} {
		t.Run(mode, func(t *testing.T) {
			db, err := badger.Open(badger.DefaultOptions(filepath.Join(t.TempDir(), "small-transaction")).WithLogger(nil).WithCompression(0).WithMemTableSize(2 << 20).WithValueThreshold(200000))
			if err != nil {
				t.Fatal(err)
			}
			st := &Store{db: db}
			defer st.Close()
			if err := st.ensureSchema(); err != nil {
				t.Fatal(err)
			}
			b := gradeGenerationBirth()
			gradeGenerationBirthCommit(t, st, b)
			if _, err := st.AttestAlias(b.Slug, "legacy-alias", "legacy-episode"); err != nil {
				t.Fatal(err)
			}
			before := gradeGenerationSnapshot(t, st)
			events := 0
			st.SetObserver(func(Event) { events++ })
			err = st.AtomicWrite(func(tx *Store) error {
				if err := tx.txn.Set([]byte("private-filler"), bytes.Repeat([]byte("f"), 190000)); err != nil {
					t.Fatalf("fixture staging failed early: %v", err)
				}
				if mode == "selector" {
					b.Slug = "second-guide"
					b.Name = "Second Guide"
					b.EpisodeID = strings.Repeat("e", 190000)
					_, err := beginIdentityGeneration(tx, b)
					return err
				}
				s, err := loadIdentityGeneration(tx, b.Slug, "later")
				if err != nil {
					return err
				}
				_, err = s.attest(strings.Repeat("a", 190000))
				return err
			})
			if !errors.Is(err, badger.ErrTxnTooBig) {
				t.Fatalf("expected primitive staging error, got %v", err)
			}
			if events != 0 || !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
				t.Fatal("primitive staging failure escaped rollback")
			}
		})
	}
}
