package store

import (
	"bytes"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/dgraph-io/badger/v4"
)

type serialResult struct {
	done chan struct{}
	err  error
}

func serialAsync(fn func() error) *serialResult {
	r := &serialResult{done: make(chan struct{})}
	go func() { defer close(r.done); r.err = fn() }()
	return r
}

func (r *serialResult) wait(t *testing.T) error {
	t.Helper()
	select {
	case <-r.done:
		return r.err
	case <-time.After(5 * time.Second):
		t.Fatal("coordinated operation did not complete")
		return nil
	}
}

func serialSignal(t *testing.T, c <-chan struct{}) {
	t.Helper()
	select {
	case <-c:
	case <-time.After(5 * time.Second):
		t.Fatal("coordination signal missing")
	}
}

func serialBlocked(t *testing.T, r *serialResult) {
	t.Helper()
	select {
	case <-r.done:
		t.Fatal("writer crossed held coordination boundary")
	case <-time.After(20 * time.Millisecond):
	}
}

func serialHold(t *testing.T, st *Store, phase string, finish error) (func(), *serialResult) {
	t.Helper()
	entered, gate := make(chan struct{}), make(chan struct{})
	var once sync.Once
	release := func() { once.Do(func() { close(gate) }) }
	block := func(*Store) error { close(entered); <-gate; return finish }
	body, finalizer := func(*Store) error { return nil }, func(*Store) error { return nil }
	if phase == "body" {
		body = block
	} else {
		finalizer = block
	}
	r := serialAsync(func() error { return runSerializedIdentityAdmission(st, body, finalizer) })
	t.Cleanup(func() { release(); r.wait(t) })
	serialSignal(t, entered)
	return release, r
}

func serialFixture(t *testing.T) (*Store, Entity, Fact) {
	t.Helper()
	st := openTemp(t)
	at := time.Unix(123, 456).UTC()
	e := Entity{Slug: "alpha", Name: "Alpha", Type: "project", Aliases: []string{"transfer"}, CreatedAt: at, LastSeen: at}
	for _, v := range []Entity{e, {Slug: "beta", Name: "Beta", Type: "project", CreatedAt: at}, {Slug: "gamma", Name: "Gamma", Type: "project", CreatedAt: at}} {
		if err := st.PutEntity(v); err != nil {
			t.Fatal("entity fixture failed")
		}
	}
	b, _ := st.GetEntity("beta")
	b.Aliases = []string{"transfer"}
	// Explicit synthetic legacy listing; index remains owned by alpha.
	dpSet(t, st, "en:beta", dpJSON(t, b))
	f := Fact{Src: "alpha", Relation: "related_to", Dst: "beta", Fact: "synthetic assertion", ValidFrom: at, Confidence: 0.5}
	if err := st.PutFact(f); err != nil {
		t.Fatal("fact fixture failed")
	}
	return st, e, f
}

func TestSerializedAdmissionAllGraphProducers(t *testing.T) {
	for _, phase := range []string{"body", "finalizer"} {
		for _, refuse := range []bool{false, true} {
			for _, name := range []string{"entity", "delete-entity", "episode", "fact", "invalidate", "delete-fact", "relocate", "cursor", "claim", "drop", "drop-rehome", "meta-time", "meta-json", "attest", "value-evidence", "atomic"} {
				t.Run(phase+"/"+name+map[bool]string{false: "/commit", true: "/refuse"}[refuse], func(t *testing.T) {
					st, e, f := serialFixture(t)
					before := dpRows(t, st)
					var finish error
					if refuse {
						finish = errors.New("synthetic refusal")
					}
					release, admission := serialHold(t, st, phase, finish)
					started := make(chan struct{})
					writer := serialAsync(func() error {
						close(started)
						switch name {
						case "entity":
							e.Description = "changed"
							return st.PutEntity(e)
						case "delete-entity":
							return st.DeleteEntity("gamma")
						case "episode":
							return st.PutEpisode(Episode{ID: "new"})
						case "fact":
							f.Confidence = 0.9
							return st.PutFact(f)
						case "invalidate":
							return st.InvalidateFact(f.Src, f.Relation, f.Dst, f.ValidFrom, f.ValidFrom.Add(time.Second))
						case "delete-fact":
							return st.DeleteFact(f.Src, f.Relation, f.Dst, f.ValidFrom)
						case "relocate":
							updated := f
							updated.Dst = "gamma"
							return st.RelocateFact(f, updated)
						case "cursor":
							return st.PutCursor(Cursor{Path: "/synthetic", Size: 1})
						case "claim":
							return st.ClaimAlias("new", e.Slug)
						case "drop":
							_, err := st.DropAlias(e.Slug, "transfer")
							return err
						case "drop-rehome":
							_, err := st.DropAliasRehome(e.Slug, "transfer", "beta")
							return err
						case "meta-time":
							return st.PutMetaTime("identity_synthetic", e.CreatedAt)
						case "meta-json":
							return st.PutMetaJSON("identity_synthetic", true)
						case "attest":
							_, err := st.AttestAlias(e.Slug, "new", "episode")
							return err
						case "value-evidence":
							return st.RecordValueEvidence("synthetic", "episode")
						case "atomic":
							return st.AtomicWrite(func(tx *Store) error { return tx.PutEpisode(Episode{ID: "atomic"}) })
						}
						return errors.New("missing test operation")
					})
					serialSignal(t, started)
					serialBlocked(t, writer)
					if !reflect.DeepEqual(before, dpRows(t, st)) {
						t.Fatal("blocked producer mutated raw store")
					}
					release()
					if admission.wait(t) != finish || writer.wait(t) != nil || reflect.DeepEqual(before, dpRows(t, st)) {
						t.Fatal("real producer did not execute after release")
					}
				})
			}
		}
	}
}

func TestSerializedAdmissionWaitsForPublicAtomicAndPreventsPhantom(t *testing.T) {
	st := openTemp(t)
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	t.Cleanup(unblock)
	writer := serialAsync(func() error {
		return st.AtomicWrite(func(tx *Store) error {
			if err := tx.ClaimAlias("before", "owner"); err != nil {
				return err
			}
			close(entered)
			<-release
			return nil
		})
	})
	serialSignal(t, entered)
	var ledger *identityMutationLedger
	var report identityMutationReport
	admissionEntered, admissionRelease := make(chan struct{}), make(chan struct{})
	var admissionOnce sync.Once
	endAdmission := func() { admissionOnce.Do(func() { close(admissionRelease) }) }
	t.Cleanup(endAdmission)
	admission := serialAsync(func() error {
		return runSerializedIdentityAdmission(st, func(tx *Store) error {
			var err error
			ledger, err = beginIdentityMutationLedger(tx)
			if err != nil {
				return err
			}
			close(admissionEntered)
			<-admissionRelease
			return nil
		}, func(*Store) error { var err error; report, err = ledger.verify(); return err })
	})
	serialBlocked(t, admission)
	select {
	case <-admissionEntered:
		t.Fatal("snapshot opened before prior writer committed")
	default:
	}
	unblock()
	if writer.wait(t) != nil {
		t.Fatal("prior writer failed")
	}
	serialSignal(t, admissionEntered)
	phantom := serialAsync(func() error { return st.ClaimAlias("phantom", "missing-owner") })
	serialBlocked(t, phantom)
	endAdmission()
	if admission.wait(t) != nil || phantom.wait(t) != nil {
		t.Fatal("coordinated writes failed")
	}
	if string(report.Baseline.Rows["al:before"]) != "owner" || !reflect.DeepEqual(report.Baseline.Rows, report.Final.Rows) {
		t.Fatal("admission did not observe prior complete commit")
	}
	if _, exists := report.Final.Rows["al:phantom"]; exists {
		t.Fatal("future claim leaked into serialized report")
	}
	if string(dpRows(t, st)["al:phantom"]) != "missing-owner" {
		t.Fatal("post-scope public claim did not execute; lifecycle policy is separate")
	}
}

func TestSerializedAdmissionQueueProgressDurabilityAndFacadeGuards(t *testing.T) {
	for _, phase := range []string{"body", "finalizer"} {
		t.Run(phase, func(t *testing.T) {
			dir := t.TempDir()
			st, err := Open(dir)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { st.Close() })
			if err := st.PutPending(PendingEpisode{ID: "delete", Text: "synthetic"}); err != nil {
				t.Fatal(err)
			}
			release, admission := serialHold(t, st, phase, errors.New("rollback"))
			p := PendingEpisode{ID: "kept", Text: "synthetic durable input", Hints: []string{"Synthetic"}, OccurredAt: time.Unix(100, 2).UTC()}
			queue := serialAsync(func() error {
				if err := st.PutPending(p); err != nil {
					return err
				}
				if err := st.DeletePending("delete"); err != nil {
					return err
				}
				if has, err := st.HasPending(p.ID); err != nil || !has {
					return errors.New("pending missing")
				}
				ready, backoff, parked, err := st.PendingCounts(time.Now())
				if err != nil || ready != 1 || backoff != 0 || parked != 0 {
					return errors.New("queue count mismatch")
				}
				return nil
			})
			if queue.wait(t) != nil {
				t.Fatal("queue could not progress under graph lock")
			}
			want := bytes.Clone(dpRows(t, st)["pq:kept"])
			release()
			if admission.wait(t) == nil {
				t.Fatal("expected owner refusal")
			}
			if err := st.Close(); err != nil {
				t.Fatal(err)
			}
			reopened, err := Open(dir)
			if err != nil {
				t.Fatal(err)
			}
			defer reopened.Close()
			got, err := reopened.GetPending(p.ID)
			if err != nil || !reflect.DeepEqual(got, p) || !bytes.Equal(want, dpRows(t, reopened)["pq:kept"]) {
				t.Fatal("queue bytes not durable")
			}
			if _, err := reopened.GetPending("delete"); !errors.Is(err, ErrNotFound) {
				t.Fatal("queue deletion not durable")
			}
		})
	}
	for _, op := range []string{"put", "delete"} {
		for _, phase := range []string{"body-rollback", "finalizer", "closed", "poisoned"} {
			t.Run(op+"/"+phase, func(t *testing.T) {
				st := openTemp(t)
				if err := st.PutPending(PendingEpisode{ID: "old"}); err != nil {
					t.Fatal(err)
				}
				before := dpRows(t, st)
				invoke := func(tx *Store) error {
					if op == "put" {
						return tx.PutPending(PendingEpisode{ID: "new"})
					}
					return tx.DeletePending("old")
				}
				prior := errors.New("earlier")
				var saved *Store
				err := runSerializedIdentityAdmission(st, func(tx *Store) error {
					saved = tx
					if phase == "body-rollback" {
						if err := invoke(tx); err != nil {
							return err
						}
						return prior
					}
					if phase == "poisoned" {
						tx.poisonAdmission(prior)
						_ = invoke(tx)
					}
					return nil
				}, func(tx *Store) error {
					if phase == "finalizer" {
						_ = invoke(tx)
					}
					return nil
				})
				if phase == "closed" {
					if err != nil || invoke(saved) == nil {
						t.Fatal("closed facade queue bypass")
					}
				} else if err == nil {
					t.Fatal("ignored queue failure committed")
				}
				if (phase == "poisoned" || phase == "body-rollback") && err != prior {
					t.Fatal("prior error lost")
				}
				if !reflect.DeepEqual(before, dpRows(t, st)) {
					t.Fatal("facade queue mutation escaped rollback")
				}
			})
		}
	}
}

func TestSerializedAdmissionFailuresReleaseAndSuppressEvents(t *testing.T) {
	for _, mode := range []string{"body-error", "final-error", "body-panic", "final-panic", "nested", "late-write", "conflict"} {
		t.Run(mode, func(t *testing.T) {
			st := openTemp(t)
			dpSet(t, st, "al:observed", []byte("old"))
			before := dpRows(t, st)
			events := 0
			st.SetObserver(func(Event) { events++ })
			var saved *Store
			var err error
			panicked := false
			prior := errors.New("synthetic refusal")
			func() {
				defer func() {
					if recover() != nil {
						panicked = true
					}
				}()
				err = runSerializedIdentityAdmission(st, func(tx *Store) error {
					saved = tx
					if err := tx.AtomicWrite(func(nested *Store) error { return nested.PutEpisode(Episode{ID: "staged"}) }); err != nil {
						return err
					}
					if _, err := tx.txn.Get([]byte("al:observed")); err != nil {
						return err
					}
					if err := tx.txn.Set([]byte("al:observed"), []byte("staged")); err != nil {
						return err
					}
					switch mode {
					case "body-error":
						return prior
					case "body-panic":
						panic("synthetic")
					case "nested":
						_ = runSerializedIdentityAdmission(tx, func(*Store) error { return nil }, func(*Store) error { return nil })
					case "conflict":
						return st.db.Update(func(other *badger.Txn) error { return other.Set([]byte("al:observed"), []byte("competitor")) })
					}
					return nil
				}, func(tx *Store) error {
					switch mode {
					case "final-error":
						return prior
					case "final-panic":
						panic("synthetic")
					case "late-write":
						_ = tx.PutEntity(Entity{Slug: "late", Name: "Late"})
					}
					return nil
				})
			}()
			if mode == "conflict" {
				before["al:observed"] = []byte("competitor")
				if !errors.Is(err, badger.ErrConflict) {
					t.Fatal("not real commit conflict")
				}
			}
			if err == nil && !panicked {
				t.Fatal("expected failure")
			}
			if events != 0 || saved.admissionOwner.phase != admissionClosed || !reflect.DeepEqual(before, dpRows(t, st)) {
				t.Fatal("failure leaked bytes/events or owner")
			}
			if serialAsync(func() error { return st.ClaimAlias("after", "owner") }).wait(t) != nil {
				t.Fatal("graph lock leaked")
			}
			if serialAsync(func() error { return st.PutPending(PendingEpisode{ID: "after"}) }).wait(t) != nil {
				t.Fatal("maintenance lock leaked")
			}
		})
	}
	st := openTemp(t)
	if runSerializedIdentityAdmission(nil, nil, nil) == nil || runSerializedIdentityAdmission(st, nil, func(*Store) error { return nil }) == nil || runSerializedIdentityAdmission(st, func(*Store) error { return nil }, nil) == nil {
		t.Fatal("malformed entry accepted")
	}
	if err := st.AtomicWrite(func(tx *Store) error {
		_ = runSerializedIdentityAdmission(tx, nil, nil)
		return tx.txn.Set([]byte("opaque:staged"), []byte("rollback"))
	}); err == nil {
		t.Fatal("ordinary nested entry did not poison")
	}
	if _, ok := dpRows(t, st)["opaque:staged"]; ok {
		t.Fatal("nested entry leaked raw write")
	}
}

func TestSerializedAdmissionObserverCanWriteAfterCommit(t *testing.T) {
	st := openTemp(t)
	var observed error
	st.SetObserver(func(Event) { observed = st.ClaimAlias("observer", "owner") })
	r := serialAsync(func() error {
		return runSerializedIdentityAdmission(st, func(tx *Store) error { return tx.PutEpisode(Episode{ID: "committed"}) }, func(*Store) error { return nil })
	})
	if r.wait(t) != nil || observed != nil {
		t.Fatal("postcommit observer could not write")
	}
	if got := dpRows(t, st); string(got["al:observer"]) != "owner" || got["ep:committed"] == nil {
		t.Fatal("postcommit write missing")
	}
}

func TestSerializedAdmissionQueueAtomicAndOtherAdmissionWait(t *testing.T) {
	for _, phase := range []string{"body", "finalizer"} {
		for _, mode := range []string{"queue-atomic", "other-admission"} {
			t.Run(phase+"/"+mode, func(t *testing.T) {
				st := openTemp(t)
				release, owner := serialHold(t, st, phase, nil)
				body := func(tx *Store) error { return tx.PutPending(PendingEpisode{ID: "coordinated"}) }
				writer := serialAsync(func() error {
					if mode == "queue-atomic" {
						return st.AtomicWrite(body)
					}
					return runSerializedIdentityAdmission(st, body, func(*Store) error { return nil })
				})
				serialBlocked(t, writer)
				release()
				if owner.wait(t) != nil || writer.wait(t) != nil {
					t.Fatal("coordinated queue transaction failed")
				}
				if has, err := st.HasPending("coordinated"); err != nil || !has {
					t.Fatal("coordinated queue transaction did not commit")
				}
			})
		}
	}
}

func TestSerializedAdmissionQueueConflictIsNotWholeDBIsolation(t *testing.T) {
	st := openTemp(t)
	if err := st.PutPending(PendingEpisode{ID: "shared", Text: "before"}); err != nil {
		t.Fatal(err)
	}
	events := 0
	st.SetObserver(func(Event) { events++ })
	err := runSerializedIdentityAdmission(st, func(tx *Store) error {
		if _, err := tx.GetPending("shared"); err != nil {
			return err
		}
		if err := tx.PutEpisode(Episode{ID: "rollback"}); err != nil {
			return err
		}
		return st.PutPending(PendingEpisode{ID: "shared", Text: "competitor"})
	}, func(*Store) error { return nil })
	if !errors.Is(err, badger.ErrConflict) || events != 0 {
		t.Fatal("queue conflict did not roll back owner")
	}
	if has, err := st.HasEpisode("rollback"); err != nil || has {
		t.Fatal("owner leaked staged episode")
	}
	if p, err := st.GetPending("shared"); err != nil || p.Text != "competitor" {
		t.Fatal("queue competitor lost durable input")
	}
	if serialAsync(func() error { return st.ClaimAlias("after", "owner") }).wait(t) != nil {
		t.Fatal("conflict leaked graph lock")
	}
}
