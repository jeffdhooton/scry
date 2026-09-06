package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/dgraph-io/badger/v4"
)

func TestIndependentOutcomeCorrectionStaticErrorsAndLargeSuccess(t *testing.T) {
	for _, mode := range []string{"memory-value-refusal", "disk-value-refusal", "transaction-size-refusal", "large-disk-success"} {
		t.Run(mode, func(t *testing.T) {
			opts := badger.DefaultOptions(filepath.Join(t.TempDir(), "db")).WithLogger(nil)
			size := 1170000
			switch mode {
			case "memory-value-refusal":
				opts = badger.DefaultOptions("").WithLogger(nil).WithInMemory(true).WithValueThreshold(4096)
				size = 26000
			case "disk-value-refusal":
				opts = opts.WithValueLogFileSize(1 << 20)
			case "transaction-size-refusal":
				opts = opts.WithCompression(0).WithMemTableSize(2 << 20).WithValueThreshold(200000)
				size = 120000
			case "large-disk-success":
				opts = opts.WithValueLogFileSize(4 << 20)
			}
			db, err := badger.Open(opts)
			if err != nil {
				t.Fatal(err)
			}
			st := &Store{db: db}
			defer st.Close()
			if err := st.ensureSchema(); err != nil {
				t.Fatal(err)
			}
			input, ik := independentOutcomeInput(t, st, "src", 1)
			o := independentOutcomeAssertion(input, ik)
			o.Birth = &identityBirth{EpisodeID: input.EpisodeID, Occurrence: input.Ordinal, Slug: "cedar-atlas", Name: input.Fact.Src, Origin: "endpoint", CreatedAt: input.OccurredAt}
			o.AssertionOrdinal = nil
			o.Disposition = "no-assertion"
			o.Materialization = []byte(`{"original":"` + strings.Repeat("x", size) + `","unknown":true}`)
			want, wantKey, err := encodeIdentityOutcome(o)
			if err != nil {
				t.Fatal(err)
			}
			before := gradeGenerationSnapshot(t, st)
			events := 0
			st.SetObserver(func(Event) { events++ })
			var key string
			err = st.AtomicWrite(func(tx *Store) error {
				if mode == "transaction-size-refusal" {
					if err := tx.txn.Set([]byte("independent-transaction-filler"), bytes.Repeat([]byte("f"), 190000)); err != nil {
						t.Fatalf("fixture did not reach outcome Set: %v", err)
					}
				}
				var err error
				key, err = putIdentityOutcome(tx, o)
				return err
			})
			if events != 0 {
				t.Fatal("outcome emitted graph event")
			}
			if mode == "large-disk-success" {
				if err != nil || key != wantKey {
					t.Fatalf("valid oversized write refused: %v", err)
				}
				if len(want) <= 1<<20 {
					t.Fatal("success fixture below real disk failure size")
				}
				var assembled []byte
				for offset := 0; ; {
					chunk, err := readIdentityOutcomeChunk(st, input.EpisodeID, key, offset, 24576)
					if err != nil {
						t.Fatal(err)
					}
					wire, err := json.Marshal(chunk)
					if err != nil || len(wire) > 24576 {
						t.Fatal("large success chunk budget exceeded")
					}
					assembled = append(assembled, chunk.Data...)
					if chunk.NextOffset == -1 {
						break
					}
					if chunk.NextOffset != offset+len(chunk.Data) || chunk.NextOffset <= offset {
						t.Fatal("large success chunk skipped/stalled")
					}
					offset = chunk.NextOffset
				}
				if !bytes.Equal(assembled, want) {
					t.Fatal("large success raw bytes lost")
				}
				if got := independentOutcomePut(t, st, o); got != key {
					t.Fatal("large success replay drift")
				}
				return
			}
			if err == nil || key != "" || !errors.Is(err, errIdentityOutcome) {
				t.Fatalf("missing safe static outcome classification: %v", err)
			}
			wantError := errIdentityOutcome.Error()
			if mode == "transaction-size-refusal" {
				if !errors.Is(err, badger.ErrTxnTooBig) {
					t.Fatal("real transaction-size classification lost")
				}
				wantError += "\n" + badger.ErrTxnTooBig.Error()
			} else if errors.Is(err, badger.ErrTxnTooBig) {
				t.Fatal("value limit mislabeled transaction size")
			}
			if err.Error() != wantError {
				t.Fatalf("error includes nonallowlisted detail: %q", err.Error())
			}
			if !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
				t.Fatal("refusal changed bytes")
			}
			// A failed transaction does not break a later independent valid write.
			o.Materialization = nil
			key = independentOutcomePut(t, st, o)
			if _, err := readIdentityOutcomeChunk(st, input.EpisodeID, key, 0, 512); err != nil {
				t.Fatal(err)
			}
		})
	}
}
