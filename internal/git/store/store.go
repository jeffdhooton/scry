// Package store wraps the BadgerDB-backed git index for one repo.
//
// Key prefixes:
//
//	meta:<key>                                   metadata (schema_version, repo path, timestamps)
//	blame:<rel_path>:<line_08d>                  BlameRecord per line
//	commit:<desc_timestamp>:<hash>               CommitRecord (newest-first iteration)
//	commit_by_hash:<hash>                        CommitRecord (hash lookup for intent)
//	file_commit:<rel_path>:<desc_ts>:<hash>      secondary index for file-history
//	cochange:<path_a>:<path_b>                   CochangeRecord (canonical a < b)
//	churn:<rel_path>                             ChurnRecord
//	contrib:<rel_path>:<author_email>            ContribRecord
//
// All values are JSON. Schema version 1.
package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"

	"github.com/dgraph-io/badger/v4"
	"github.com/dgraph-io/badger/v4/options"
)

const (
	SchemaVersion = 1

	prefixMeta       = "meta:"
	prefixBlame      = "blame:"
	prefixCommit     = "commit:"
	prefixCommitHash = "commit_by_hash:"
	prefixFileCommit = "file_commit:"
	prefixCochange   = "cochange:"
	prefixChurn      = "churn:"
	prefixContrib    = "contrib:"
)

type BlameRecord struct {
	Author  string `json:"author"`
	Email   string `json:"email"`
	Commit  string `json:"commit"`
	Date    int64  `json:"date"`
	Summary string `json:"summary"`
	Line    int    `json:"line"`
	Content string `json:"content"`
}

type CommitRecord struct {
	Hash    string       `json:"hash"`
	Author  string       `json:"author"`
	Email   string       `json:"email"`
	Date    int64        `json:"date"`
	Message string       `json:"message"`
	Files   []FileChange `json:"files"`
}

type FileChange struct {
	Path    string `json:"path"`
	Added   int    `json:"added"`
	Removed int    `json:"removed"`
	Status  string `json:"status"`
}

type CochangeRecord struct {
	FileA       string `json:"file_a"`
	FileB       string `json:"file_b"`
	Count       int    `json:"count"`
	LastChanged int64  `json:"last_changed"`
}

type ChurnRecord struct {
	Path         string `json:"path"`
	CommitCount  int    `json:"commit_count"`
	LinesAdded   int    `json:"lines_added"`
	LinesRemoved int    `json:"lines_removed"`
	LastChanged  int64  `json:"last_changed"`
}

type ContribRecord struct {
	Author       string `json:"author"`
	Email        string `json:"email"`
	CommitCount  int    `json:"commit_count"`
	LinesAdded   int    `json:"lines_added"`
	LinesRemoved int    `json:"lines_removed"`
}

type Store struct {
	db *badger.DB
}

func Open(dir string) (*Store, error) {
	opts := badger.DefaultOptions(dir).
		WithLogger(nil).
		WithCompression(options.Snappy)
	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("open badger at %q: %w", dir, err)
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Reset() error {
	return s.db.DropAll()
}

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

func (s *Store) SetMeta(key string, value any) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(prefixMeta+key), b)
	})
}

func DescTimestamp(ts int64) string {
	return fmt.Sprintf("%020d", math.MaxInt64-ts)
}

type Writer struct {
	wb *badger.WriteBatch
}

func (s *Store) NewWriter() *Writer {
	return &Writer{wb: s.db.NewWriteBatch()}
}

func (w *Writer) PutBlame(relPath string, rec *BlameRecord) error {
	b, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	key := fmt.Sprintf("%s%s:%08d", prefixBlame, relPath, rec.Line)
	return w.wb.Set([]byte(key), b)
}

func (w *Writer) PutCommit(rec *CommitRecord) error {
	b, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	descTs := DescTimestamp(rec.Date)
	key := fmt.Sprintf("%s%s:%s", prefixCommit, descTs, rec.Hash)
	if err := w.wb.Set([]byte(key), b); err != nil {
		return err
	}
	hashKey := prefixCommitHash + rec.Hash
	if err := w.wb.Set([]byte(hashKey), b); err != nil {
		return err
	}
	for _, f := range rec.Files {
		fileKey := fmt.Sprintf("%s%s:%s:%s", prefixFileCommit, f.Path, descTs, rec.Hash)
		if err := w.wb.Set([]byte(fileKey), nil); err != nil {
			return err
		}
	}
	return nil
}

func (w *Writer) PutCochange(pathA, pathB string, rec *CochangeRecord) error {
	if pathA > pathB {
		pathA, pathB = pathB, pathA
	}
	b, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	key := fmt.Sprintf("%s%s:%s", prefixCochange, pathA, pathB)
	return w.wb.Set([]byte(key), b)
}

func (w *Writer) PutChurn(relPath string, rec *ChurnRecord) error {
	b, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	key := prefixChurn + relPath
	return w.wb.Set([]byte(key), b)
}

func (w *Writer) PutContrib(relPath, email string, rec *ContribRecord) error {
	b, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	key := fmt.Sprintf("%s%s:%s", prefixContrib, relPath, email)
	return w.wb.Set([]byte(key), b)
}

func (w *Writer) Flush() error {
	return w.wb.Flush()
}

// --- Read operations ---

func (s *Store) GetBlame(relPath string, startLine, endLine int) ([]BlameRecord, error) {
	prefix := []byte(prefixBlame + relPath + ":")
	var recs []BlameRecord
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 64
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			err := it.Item().Value(func(val []byte) error {
				var rec BlameRecord
				if err := json.Unmarshal(val, &rec); err != nil {
					return err
				}
				if startLine > 0 && rec.Line < startLine {
					return nil
				}
				if endLine > 0 && rec.Line > endLine {
					return nil
				}
				recs = append(recs, rec)
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	return recs, err
}

func (s *Store) GetCommitByHash(hash string) (*CommitRecord, error) {
	var rec *CommitRecord
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(prefixCommitHash + hash))
		if errors.Is(err, badger.ErrKeyNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			rec = &CommitRecord{}
			return json.Unmarshal(val, rec)
		})
	})
	return rec, err
}

func (s *Store) GetRecentCommits(limit int) ([]CommitRecord, error) {
	prefix := []byte(prefixCommit)
	var recs []CommitRecord
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 64
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			if limit > 0 && len(recs) >= limit {
				break
			}
			err := it.Item().Value(func(val []byte) error {
				var rec CommitRecord
				if err := json.Unmarshal(val, &rec); err != nil {
					return err
				}
				recs = append(recs, rec)
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	return recs, err
}

func (s *Store) GetFileCommits(relPath string, limit int) ([]CommitRecord, error) {
	prefix := []byte(prefixFileCommit + relPath + ":")
	var hashes []string
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			if limit > 0 && len(hashes) >= limit {
				break
			}
			k := string(it.Item().KeyCopy(nil))
			parts := splitKeyFromEnd(k, 1)
			if len(parts) == 2 {
				hashes = append(hashes, parts[1])
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	var recs []CommitRecord
	for _, h := range hashes {
		rec, err := s.GetCommitByHash(h)
		if err != nil {
			return nil, err
		}
		if rec != nil {
			recs = append(recs, *rec)
		}
	}
	return recs, nil
}

func splitKeyFromEnd(s string, n int) []string {
	parts := make([]string, 0, n+1)
	for i := 0; i < n; i++ {
		idx := lastIndex(s, ':')
		if idx < 0 {
			return nil
		}
		parts = append(parts, s[idx+1:])
		s = s[:idx]
	}
	result := make([]string, n+1)
	result[0] = s
	for i, p := range parts {
		result[n-i] = p
	}
	return result
}

func lastIndex(s string, b byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == b {
			return i
		}
	}
	return -1
}

func (s *Store) GetCochange(relPath string, limit int) ([]CochangeRecord, error) {
	prefix := []byte(prefixCochange)
	var recs []CochangeRecord
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 64
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			err := it.Item().Value(func(val []byte) error {
				var rec CochangeRecord
				if err := json.Unmarshal(val, &rec); err != nil {
					return err
				}
				if rec.FileA == relPath || rec.FileB == relPath {
					recs = append(recs, rec)
				}
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(recs, func(i, j int) bool {
		return recs[i].Count > recs[j].Count
	})
	if limit > 0 && len(recs) > limit {
		recs = recs[:limit]
	}
	return recs, nil
}

func (s *Store) GetHotspots(limit int) ([]ChurnRecord, error) {
	prefix := []byte(prefixChurn)
	var recs []ChurnRecord
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 64
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			err := it.Item().Value(func(val []byte) error {
				var rec ChurnRecord
				if err := json.Unmarshal(val, &rec); err != nil {
					return err
				}
				recs = append(recs, rec)
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(recs, func(i, j int) bool {
		return recs[i].CommitCount > recs[j].CommitCount
	})
	if limit > 0 && len(recs) > limit {
		recs = recs[:limit]
	}
	return recs, nil
}

func (s *Store) GetContributors(relPath string) ([]ContribRecord, error) {
	var prefix []byte
	if relPath != "" {
		prefix = []byte(prefixContrib + relPath + ":")
	} else {
		prefix = []byte(prefixContrib)
	}
	agg := map[string]*ContribRecord{}
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 64
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			err := it.Item().Value(func(val []byte) error {
				var rec ContribRecord
				if err := json.Unmarshal(val, &rec); err != nil {
					return err
				}
				if existing, ok := agg[rec.Email]; ok {
					existing.CommitCount += rec.CommitCount
					existing.LinesAdded += rec.LinesAdded
					existing.LinesRemoved += rec.LinesRemoved
				} else {
					copy := rec
					agg[rec.Email] = &copy
				}
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	recs := make([]ContribRecord, 0, len(agg))
	for _, r := range agg {
		recs = append(recs, *r)
	}
	sort.Slice(recs, func(i, j int) bool {
		return recs[i].CommitCount > recs[j].CommitCount
	})
	return recs, nil
}

func (s *Store) CountKeys(prefix string) (int, error) {
	pb := []byte(prefix)
	count := 0
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Seek(pb); it.ValidForPrefix(pb); it.Next() {
			count++
		}
		return nil
	})
	return count, err
}

func PrefixBlame() string    { return prefixBlame }
func PrefixCommit() string   { return prefixCommit }
func PrefixCochange() string { return prefixCochange }
func PrefixChurn() string    { return prefixChurn }
func PrefixContrib() string  { return prefixContrib }
