package store

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/dgraph-io/badger/v4"
)

// Independently authored boundary probes. All data are disposable synthetic data.
func TestSerialCodeIndependentObserverReentersExclusiveMaintenance(t *testing.T) {
	for _, serialized := range []bool{false, true} {
		t.Run(fmt.Sprint(serialized), func(t *testing.T) {
			f := ownerValidMaintenance(t)
			var observed error
			calls := 0
			f.s.SetObserver(func(Event) {
				calls++
				f.s.SetObserver(nil)
				_, observed = f.s.MergeEntitiesChecked(f.merge, nil)
			})
			result := serialAsync(func() error {
				body := func(tx *Store) error { return tx.PutEpisode(Episode{ID: "observer-trigger"}) }
				if serialized {
					return runSerializedIdentityAdmission(f.s, body, func(*Store) error { return nil })
				}
				return f.s.AtomicWrite(body)
			})
			if err := result.wait(t); err != nil || observed != nil || calls != 1 {
				t.Fatalf("observer result %v/%v calls %d", err, observed, calls)
			}
			if _, err := f.s.GetEpisode("observer-trigger"); err != nil {
				t.Fatal(err)
			}
			if _, err := f.s.GetEntity(f.merge.Retire[0]); !errors.Is(err, ErrNotFound) {
				t.Fatalf("merge did not remove loser: %v", err)
			}
		})
	}
}

func TestSerialCodeIndependentQueueCannotEscapeLiteralPrefix(t *testing.T) {
	for _, phase := range []string{"body", "finalizer"} {
		t.Run(phase, func(t *testing.T) {
			st := openTemp(t)
			if err := st.ClaimAlias("sentinel", "owner"); err != nil {
				t.Fatal(err)
			}
			before := dpRows(t, st)
			release, owner := serialHold(t, st, phase, nil)
			ids := []string{"", "../al:sentinel", "\x00al:sentinel", "\xff:en:sentinel", "meta:identity_sentinel", "pq:../en:sentinel", "al:sentinel\n"}
			for i, id := range ids {
				p := PendingEpisode{ID: id, Text: fmt.Sprintf("queue-%d", i)}
				if err := serialAsync(func() error { return st.PutPending(p) }).wait(t); err != nil {
					t.Fatal(err)
				}
				if _, ok := dpRows(t, st)["pq:"+id]; !ok {
					t.Fatalf("literal key absent for %q", id)
				}
				if err := serialAsync(func() error { return st.DeletePending(id) }).wait(t); err != nil {
					t.Fatal(err)
				}
			}
			if !reflect.DeepEqual(before, dpRows(t, st)) {
				t.Fatal("queue operation changed another family")
			}
			release()
			if err := owner.wait(t); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestSerialCodeIndependentCompetingOwnersSeeCommittedPredecessors(t *testing.T) {
	st := openTemp(t)
	release, first := serialHold(t, st, "finalizer", nil)
	const n = 20
	results := make([]*serialResult, n)
	for i := range results {
		i := i
		results[i] = serialAsync(func() error {
			return runSerializedIdentityAdmission(st, func(tx *Store) error {
				claims, err := tx.AliasClaims()
				if err != nil {
					return err
				}
				if _, found := claims["last"]; found {
					if _, found := claims[claims["last"]]; !found {
						return errors.New("predecessor is partial")
					}
				}
				var count int
				for key := range claims {
					if strings.HasPrefix(key, "owner-") {
						count++
					}
				}
				id := fmt.Sprintf("owner-%02d", i)
				if err := tx.ClaimAlias(id, fmt.Sprint(count)); err != nil {
					return err
				}
				return tx.ClaimAlias("last", id)
			}, func(*Store) error { return nil })
		})
	}
	for _, r := range results {
		serialBlocked(t, r)
	}
	release()
	if err := first.wait(t); err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, r := range results {
		if err := r.wait(t); err != nil {
			t.Fatal(err)
		}
	}
	claims, err := st.AliasClaims()
	if err != nil {
		t.Fatal(err)
	}
	for key, value := range claims {
		if strings.HasPrefix(key, "owner-") {
			if seen[value] {
				t.Fatal("two scopes saw same predecessor count")
			}
			seen[value] = true
		}
	}
	if len(seen) != n {
		t.Fatalf("committed owners %d", len(seen))
	}
	for i := 0; i < n; i++ {
		if !seen[fmt.Sprint(i)] {
			t.Fatalf("snapshot sequence missing %d", i)
		}
	}
}

func TestSerialCodeIndependentOrdinaryScopesRemainShared(t *testing.T) {
	for _, oldOwner := range []bool{false, true} {
		t.Run(fmt.Sprint(oldOwner), func(t *testing.T) {
			st := openTemp(t)
			entered, gate := make(chan struct{}), make(chan struct{})
			body := func(tx *Store) error { close(entered); <-gate; return tx.PutEpisode(Episode{ID: "first"}) }
			first := serialAsync(func() error {
				if oldOwner {
					return runIdentityAdmission(st, body, func(*Store) error { return nil })
				}
				return st.AtomicWrite(body)
			})
			serialSignal(t, entered)
			second := serialAsync(func() error {
				return st.AtomicWrite(func(tx *Store) error { return tx.PutEpisode(Episode{ID: "second"}) })
			})
			err := second.wait(t)
			close(gate)
			if err != nil || first.wait(t) != nil {
				t.Fatal("ordinary shared transaction blocked or failed")
			}
		})
	}
}

func TestSerialCodeIndependentNestedCaughtPanicPoisonsQueue(t *testing.T) {
	st := openTemp(t)
	before := dpRows(t, st)
	finalized := false
	err := runSerializedIdentityAdmission(st, func(tx *Store) error {
		func() {
			defer func() { _ = recover() }()
			_ = tx.AtomicWrite(func(nested *Store) error {
				if err := nested.PutPending(PendingEpisode{ID: "must-rollback"}); err != nil {
					return err
				}
				panic("synthetic nested panic")
			})
		}()
		_ = tx.DeletePending("unrelated")
		return nil
	}, func(*Store) error { finalized = true; return nil })
	if !errors.Is(err, errIdentityAdmissionScope) || finalized || !reflect.DeepEqual(before, dpRows(t, st)) {
		t.Fatal("caught nested panic committed or finalized")
	}
	if err := serialAsync(func() error { return st.PutPending(PendingEpisode{ID: "after"}) }).wait(t); err != nil {
		t.Fatal(err)
	}
	if err := serialAsync(func() error { return st.PutMetaTime("after", time.Now()) }).wait(t); err != nil {
		t.Fatal(err)
	}
}

func TestSerialCodeIndependentExcludedRawAndPostVerifyWritesRemainPossible(t *testing.T) {
	for _, mode := range []string{"raw-phantom", "post-verify"} {
		t.Run(mode, func(t *testing.T) {
			st := openTemp(t)
			var ledger *identityMutationLedger
			var report identityMutationReport
			err := runSerializedIdentityAdmission(st, func(tx *Store) error {
				var err error
				ledger, err = beginIdentityMutationLedger(tx)
				if err != nil {
					return err
				}
				if mode == "raw-phantom" {
					if err := st.db.Update(func(other *badger.Txn) error { return other.Set([]byte("al:excluded"), []byte("missing-owner")) }); err != nil {
						return err
					}
				}
				return tx.PutEpisode(Episode{ID: "committed"})
			}, func(tx *Store) error {
				var err error
				report, err = ledger.verify()
				if err != nil {
					return err
				}
				if mode == "post-verify" {
					return tx.txn.Set([]byte("al:excluded"), []byte("missing-owner"))
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := report.Final.Rows["al:excluded"]; ok {
				t.Fatal("excluded future/raw write unexpectedly accounted")
			}
			if string(dpRows(t, st)["al:excluded"]) != "missing-owner" {
				t.Fatal("documented counterexample did not commit")
			}
		})
	}
}
