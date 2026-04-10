// Package store wraps the BadgerDB-backed index for one repo.
//
// Key prefixes:
//
//	meta:<key>                  metadata (schema_version, project_root, indexed_at)
//	sym:<symbol_id>             SymbolRecord (one per defined symbol)
//	def:<symbol_id>:<seq>       OccurrenceRecord with role=def
//	ref:<symbol_id>:<seq>       OccurrenceRecord with role=ref
//	name:<lower_name>:<sym_id>  empty value, secondary index for symbol-by-name lookup
//	fsym:<rel_path>:<sym_id>    empty value, secondary index for symbols-in-file
//
// All values are JSON. Schema version 1.
package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/dgraph-io/badger/v4"
)

const (
	// SchemaVersion is bumped whenever the on-disk layout changes.
	// On mismatch the index is wiped and rebuilt from scratch.
	SchemaVersion = 1

	prefixMeta   = "meta:"
	prefixSym    = "sym:"
	prefixDef    = "def:"
	prefixRef    = "ref:"
	prefixName   = "name:"
	prefixFSym   = "fsym:"
	prefixCallee = "callee:" // callee:<caller_id>:<seq> -> OccurrenceRecord (the callee)
	prefixImpl   = "impl:"   // impl:<base_id>:<impl_id> -> empty (impl_id implements base_id)
)

// SymbolRecord is one defined symbol. The Symbol field is the SCIP-formatted
// symbol id (long structured string); DisplayName is the human-readable name
// pulled from SymbolInformation.
type SymbolRecord struct {
	Symbol        string `json:"symbol"`
	DisplayName   string `json:"display_name"`
	Kind          string `json:"kind"`
	Documentation string `json:"documentation,omitempty"`
}

// OccurrenceRecord is one ref or def site for a symbol.
type OccurrenceRecord struct {
	Symbol           string `json:"symbol"`
	File             string `json:"file"`
	Line             int    `json:"line"`
	Column           int    `json:"column"`
	EndLine          int    `json:"end_line"`
	EndColumn        int    `json:"end_column"`
	Context          string `json:"context,omitempty"`
	ContainingSymbol string `json:"containing_symbol,omitempty"`
	IsDefinition     bool   `json:"is_definition,omitempty"`
}

// Store is an open BadgerDB-backed index for one repo.
type Store struct {
	db *badger.DB
}

// Open opens (or creates) a Store at dir.
func Open(dir string) (*Store, error) {
	opts := badger.DefaultOptions(dir).
		WithLogger(nil).
		WithCompression(0)
	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("open badger at %q: %w", dir, err)
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

// Reset wipes every key. Used when bumping schema version.
func (s *Store) Reset() error {
	return s.db.DropAll()
}

// SchemaVersionOnDisk returns the schema version recorded in metadata,
// or 0 if no metadata is present (a fresh store).
func (s *Store) SchemaVersionOnDisk() (int, error) {
	v, err := s.GetMeta("schema_version")
	if errors.Is(err, badger.ErrKeyNotFound) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	var n int
	if err := json.Unmarshal(v, &n); err != nil {
		return 0, fmt.Errorf("decode schema_version: %w", err)
	}
	return n, nil
}

// GetMeta returns a metadata value as raw bytes.
func (s *Store) GetMeta(key string) ([]byte, error) {
	var out []byte
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(prefixMeta + key))
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			out = append([]byte{}, val...)
			return nil
		})
	})
	return out, err
}

// SetMeta stores a JSON-encoded metadata value.
func (s *Store) SetMeta(key string, value any) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(prefixMeta+key), b)
	})
}

// Writer is a batched writer for bulk indexing.
type Writer struct {
	wb *badger.WriteBatch
	// monotonic counters for occurrence sequence numbers, keyed by symbol id
	defSeq    map[string]int
	refSeq    map[string]int
	calleeSeq map[string]int
}

func (s *Store) NewWriter() *Writer {
	return &Writer{
		wb:        s.db.NewWriteBatch(),
		defSeq:    map[string]int{},
		refSeq:    map[string]int{},
		calleeSeq: map[string]int{},
	}
}

func (w *Writer) PutSymbol(rec *SymbolRecord) error {
	b, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	if err := w.wb.Set([]byte(prefixSym+rec.Symbol), b); err != nil {
		return err
	}
	if rec.DisplayName != "" {
		key := prefixName + strings.ToLower(rec.DisplayName) + ":" + rec.Symbol
		if err := w.wb.Set([]byte(key), nil); err != nil {
			return err
		}
	}
	return nil
}

// PutFileSymbol records that symbolID is defined inside relPath. Used by
// `scry symbols <file>` (P0+ — written ahead so the index is ready when
// the query lands).
func (w *Writer) PutFileSymbol(relPath, symbolID string) error {
	key := prefixFSym + relPath + ":" + symbolID
	return w.wb.Set([]byte(key), nil)
}

func (w *Writer) PutOccurrence(rec *OccurrenceRecord) error {
	b, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	var key string
	if rec.IsDefinition {
		seq := w.defSeq[rec.Symbol]
		w.defSeq[rec.Symbol] = seq + 1
		key = fmt.Sprintf("%s%s:%08d", prefixDef, rec.Symbol, seq)
	} else {
		seq := w.refSeq[rec.Symbol]
		w.refSeq[rec.Symbol] = seq + 1
		key = fmt.Sprintf("%s%s:%08d", prefixRef, rec.Symbol, seq)
	}
	return w.wb.Set([]byte(key), b)
}

// PutCalleeEdge records that callerID calls calleeID at one specific
// occurrence (the OccurrenceRecord). The callee's display name is on the
// occurrence's Symbol field; resolving its SymbolRecord at query time gives
// the human-readable name.
func (w *Writer) PutCalleeEdge(callerID string, occ *OccurrenceRecord) error {
	b, err := json.Marshal(occ)
	if err != nil {
		return err
	}
	seq := w.calleeSeq[callerID]
	w.calleeSeq[callerID] = seq + 1
	key := fmt.Sprintf("%s%s:%08d", prefixCallee, callerID, seq)
	return w.wb.Set([]byte(key), b)
}

// PutImplEdge records that implID implements baseID. Both are SCIP symbol ids.
func (w *Writer) PutImplEdge(baseID, implID string) error {
	key := prefixImpl + baseID + ":" + implID
	return w.wb.Set([]byte(key), nil)
}

func (w *Writer) Flush() error {
	return w.wb.Flush()
}

// LookupSymbolsByName returns every symbol id whose display_name (case-
// insensitive) equals name.
func (s *Store) LookupSymbolsByName(name string) ([]string, error) {
	prefix := []byte(prefixName + strings.ToLower(name) + ":")
	var ids []string
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			k := it.Item().KeyCopy(nil)
			ids = append(ids, string(k[len(prefix):]))
		}
		return nil
	})
	return ids, err
}

// GetSymbol returns the SymbolRecord for one symbol id, or nil if not found.
func (s *Store) GetSymbol(symbolID string) (*SymbolRecord, error) {
	var rec *SymbolRecord
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(prefixSym + symbolID))
		if errors.Is(err, badger.ErrKeyNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			rec = &SymbolRecord{}
			return json.Unmarshal(val, rec)
		})
	})
	return rec, err
}

// IterateRefs streams every reference occurrence for symbolID through fn.
// Stops early if fn returns a non-nil error.
func (s *Store) IterateRefs(symbolID string, fn func(*OccurrenceRecord) error) error {
	return s.iterateOccurrences(prefixRef+symbolID+":", fn)
}

// IterateDefs streams every definition occurrence for symbolID through fn.
func (s *Store) IterateDefs(symbolID string, fn func(*OccurrenceRecord) error) error {
	return s.iterateOccurrences(prefixDef+symbolID+":", fn)
}

// IterateCallees streams every callee edge for callerID through fn. Each
// record's Symbol field is the called symbol id; resolve it via GetSymbol for
// display.
func (s *Store) IterateCallees(callerID string, fn func(*OccurrenceRecord) error) error {
	return s.iterateOccurrences(prefixCallee+callerID+":", fn)
}

// IterateImpls returns every implementation symbol id for baseID.
func (s *Store) IterateImpls(baseID string) ([]string, error) {
	prefix := []byte(prefixImpl + baseID + ":")
	var ids []string
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			k := it.Item().KeyCopy(nil)
			ids = append(ids, string(k[len(prefix):]))
		}
		return nil
	})
	return ids, err
}

func (s *Store) iterateOccurrences(prefix string, fn func(*OccurrenceRecord) error) error {
	pb := []byte(prefix)
	return s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 64
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Seek(pb); it.ValidForPrefix(pb); it.Next() {
			err := it.Item().Value(func(val []byte) error {
				rec := &OccurrenceRecord{}
				if err := json.Unmarshal(val, rec); err != nil {
					return err
				}
				return fn(rec)
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
}
