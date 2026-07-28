// Package store is the BadgerDB-backed store for scry's global episodic
// temporal knowledge graph (the memory domain). Unlike the per-repo indexes
// under internal/*/store, this store is global, incremental, and additive:
// episodes, entities, and facts accumulate over time rather than being wiped
// and rebuilt on every run. The only automatic wipe is on schema mismatch.
//
// Key prefixes:
//
//	meta:schema_version                                → int
//	ep:<id>                                            → Episode
//	en:<slug>                                          → Entity
//	al:<normalized-name>                               → slug (raw string)
//	fa:<src>:<relation>:<dst>:<validfrom-unixnano>      → Fact
//	adj:<dst>:<src>:<relation>:<validfrom-unixnano>     → empty (reverse index for FactsAbout)
//	cur:<sha256(path)>                                  → Cursor
//
// All values are JSON (except al: values, which are raw slug strings, and
// adj: values, which are empty). Schema version 1.
package store

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/dgraph-io/badger/v4"
)

// SchemaVersion is bumped whenever the on-disk layout changes. On mismatch
// the store is wiped and rebuilt from scratch.
const SchemaVersion = 1

const (
	prefixMeta    = "meta:"
	prefixEpisode = "ep:"
	prefixEntity  = "en:"
	prefixAlias   = "al:"
	prefixFact    = "fa:"
	prefixAdj     = "adj:"
	prefixCursor  = "cur:"

	keySchemaVersion = prefixMeta + "schema_version"
)

// ErrNotFound is returned by single-item getters when the key does not exist.
var ErrNotFound = errors.New("memory: not found")

// Episode is one ingested slice of source material (a session transcript
// span, a loom run, a manually seeded fact, etc.) that facts and entities
// can trace their provenance back to.
type Episode struct {
	ID         string    `json:"id"`         // sha256 hex of source_ref
	Source     string    `json:"source"`     // claude-session | codex-session | loom-run | seed | manual
	SourceRef  string    `json:"source_ref"` // path + byte/line span, e.g. "/path/file.jsonl#L120-L340"
	Summary    string    `json:"summary"`
	OccurredAt time.Time `json:"occurred_at"`
	IngestedAt time.Time `json:"ingested_at"`
}

// Entity is a named node in the knowledge graph: a project, service,
// machine, tool, person, decision, runbook, or concept.
type Entity struct {
	Slug        string    `json:"slug"`
	Name        string    `json:"name"`
	Type        string    `json:"type"` // project|service|machine|tool|person|decision|runbook|concept
	Description string    `json:"description"`
	Aliases     []string  `json:"aliases,omitempty"`
	RepoRefs    []string  `json:"repo_refs,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	LastSeen    time.Time `json:"last_seen"`
}

// Fact is one temporal edge between two entities. A given (Src, Relation,
// Dst) triple may have multiple Facts, one per ValidFrom — invalidating a
// fact sets InvalidAt rather than deleting it, preserving history.
type Fact struct {
	Src        string     `json:"src"`
	Relation   string     `json:"relation"`
	Dst        string     `json:"dst"`
	Fact       string     `json:"fact"` // one-sentence natural language
	ValidFrom  time.Time  `json:"valid_from"`
	InvalidAt  *time.Time `json:"invalid_at,omitempty"` // nil = current
	Confidence float64    `json:"confidence"`
	Episodes   []string   `json:"episodes"` // provenance episode IDs
}

// Cursor tracks ingestion progress through one source file, so re-runs can
// resume from where they left off.
type Cursor struct {
	Path           string    `json:"path"`
	Size           int64     `json:"size"`
	ModTime        time.Time `json:"mod_time"`
	ProcessedBytes int64     `json:"processed_bytes"`
}

// Store is an open BadgerDB-backed handle on the global memory store.
type Store struct {
	db *badger.DB
}

// Open opens (creating if necessary) the store at dir. If the on-disk schema
// version does not match SchemaVersion, the store is wiped and reinitialized
// at the current version.
func Open(dir string) (*Store, error) {
	opts := badger.DefaultOptions(dir).
		WithLogger(nil).
		WithCompression(0)
	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("open badger at %q: %w", dir, err)
	}
	s := &Store{db: db}
	if err := s.ensureSchema(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

// Close closes the underlying database.
func (s *Store) Close() error { return s.db.Close() }

func (s *Store) ensureSchema() error {
	disk, err := s.schemaVersionOnDisk()
	if err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}
	if disk != 0 && disk != SchemaVersion {
		if err := s.db.DropAll(); err != nil {
			return fmt.Errorf("wipe stale schema (disk=%d, want=%d): %w", disk, SchemaVersion, err)
		}
	}
	if disk != SchemaVersion {
		if err := s.db.Update(func(txn *badger.Txn) error {
			b, err := json.Marshal(SchemaVersion)
			if err != nil {
				return err
			}
			return txn.Set([]byte(keySchemaVersion), b)
		}); err != nil {
			return fmt.Errorf("write schema version: %w", err)
		}
	}
	return nil
}

func (s *Store) schemaVersionOnDisk() (int, error) {
	var v int
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(keySchemaVersion))
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &v)
		})
	})
	if errors.Is(err, badger.ErrKeyNotFound) {
		return 0, nil
	}
	return v, err
}

// --- Episodes ---

func (s *Store) PutEpisode(e Episode) error {
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(prefixEpisode+e.ID), b)
	})
}

func (s *Store) GetEpisode(id string) (Episode, error) {
	var e Episode
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(prefixEpisode + id))
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &e)
		})
	})
	if errors.Is(err, badger.ErrKeyNotFound) {
		return Episode{}, ErrNotFound
	}
	return e, err
}

func (s *Store) HasEpisode(id string) (bool, error) {
	found := false
	err := s.db.View(func(txn *badger.Txn) error {
		_, err := txn.Get([]byte(prefixEpisode + id))
		if errors.Is(err, badger.ErrKeyNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		found = true
		return nil
	})
	return found, err
}

// --- Entities ---

// PutEntity writes e and (re)indexes al: keys for its Name and every alias.
// If an entity already exists at e.Slug, any al: key the old version owned
// that the new version no longer claims is deleted — but only when that
// al: key still points at e.Slug, so a alias another entity has since
// claimed for itself is never clobbered.
func (s *Store) PutEntity(e Entity) error {
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	newNorms := normalizedNameSet(e.Name, e.Aliases)
	return s.db.Update(func(txn *badger.Txn) error {
		prev, err := getEntityTxn(txn, e.Slug)
		if err != nil && !errors.Is(err, ErrNotFound) {
			return err
		}
		if err == nil {
			for norm := range normalizedNameSet(prev.Name, prev.Aliases) {
				if newNorms[norm] {
					continue
				}
				if err := deleteAliasIfOwnedBy(txn, norm, e.Slug); err != nil {
					return err
				}
			}
		}

		if err := txn.Set([]byte(prefixEntity+e.Slug), b); err != nil {
			return err
		}
		for norm := range newNorms {
			if err := txn.Set([]byte(prefixAlias+norm), []byte(e.Slug)); err != nil {
				return err
			}
		}
		return nil
	})
}

// normalizedNameSet returns the set of non-empty Normalize()d forms of name
// and aliases.
func normalizedNameSet(name string, aliases []string) map[string]bool {
	set := make(map[string]bool, 1+len(aliases))
	for _, n := range append([]string{name}, aliases...) {
		if norm := Normalize(n); norm != "" {
			set[norm] = true
		}
	}
	return set
}

// getEntityTxn reads an Entity within an existing transaction, returning
// ErrNotFound if it does not exist.
func getEntityTxn(txn *badger.Txn, slug string) (Entity, error) {
	var e Entity
	item, err := txn.Get([]byte(prefixEntity + slug))
	if errors.Is(err, badger.ErrKeyNotFound) {
		return Entity{}, ErrNotFound
	}
	if err != nil {
		return Entity{}, err
	}
	err = item.Value(func(val []byte) error {
		return json.Unmarshal(val, &e)
	})
	return e, err
}

// deleteAliasIfOwnedBy removes al:<norm> only if it currently maps to slug,
// so pruning a stale alias never clobbers a mapping another entity has
// since claimed for itself.
func deleteAliasIfOwnedBy(txn *badger.Txn, norm, slug string) error {
	aliasKey := []byte(prefixAlias + norm)
	item, err := txn.Get(aliasKey)
	if errors.Is(err, badger.ErrKeyNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	var owner string
	if err := item.Value(func(val []byte) error {
		owner = string(val)
		return nil
	}); err != nil {
		return err
	}
	if owner != slug {
		return nil
	}
	return txn.Delete(aliasKey)
}

func (s *Store) GetEntity(slug string) (Entity, error) {
	var e Entity
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(prefixEntity + slug))
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &e)
		})
	})
	if errors.Is(err, badger.ErrKeyNotFound) {
		return Entity{}, ErrNotFound
	}
	return e, err
}

// ResolveAlias looks up Normalize(name) in the alias index and returns the
// slug it maps to, if any.
func (s *Store) ResolveAlias(name string) (string, bool, error) {
	norm := Normalize(name)
	var slug string
	found := false
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(prefixAlias + norm))
		if errors.Is(err, badger.ErrKeyNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			slug = string(val)
			found = true
			return nil
		})
	})
	return slug, found, err
}

// Entities returns every entity, sorted by slug.
func (s *Store) Entities() ([]Entity, error) {
	var entities []Entity
	pb := []byte(prefixEntity)
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 256
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Seek(pb); it.ValidForPrefix(pb); it.Next() {
			err := it.Item().Value(func(val []byte) error {
				var e Entity
				if err := json.Unmarshal(val, &e); err != nil {
					return err
				}
				entities = append(entities, e)
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	sort.Slice(entities, func(i, j int) bool { return entities[i].Slug < entities[j].Slug })
	return entities, err
}

// EntitiesByRepoRef returns every entity whose RepoRefs contains repoPath
// exactly. Scans the full entity set (small scale).
func (s *Store) EntitiesByRepoRef(repoPath string) ([]Entity, error) {
	all, err := s.Entities()
	if err != nil {
		return nil, err
	}
	var matches []Entity
	for _, e := range all {
		for _, ref := range e.RepoRefs {
			if ref == repoPath {
				matches = append(matches, e)
				break
			}
		}
	}
	return matches, nil
}

// --- Facts ---

func factKey(src, relation, dst string, validFrom time.Time) []byte {
	return []byte(fmt.Sprintf("%s%s:%s:%s:%d", prefixFact, src, relation, dst, validFrom.UnixNano()))
}

func adjKey(dst, src, relation string, validFrom time.Time) []byte {
	return []byte(fmt.Sprintf("%s%s:%s:%s:%d", prefixAdj, dst, src, relation, validFrom.UnixNano()))
}

func (s *Store) PutFact(f Fact) error {
	b, err := json.Marshal(f)
	if err != nil {
		return err
	}
	return s.db.Update(func(txn *badger.Txn) error {
		if err := txn.Set(factKey(f.Src, f.Relation, f.Dst, f.ValidFrom), b); err != nil {
			return err
		}
		return txn.Set(adjKey(f.Dst, f.Src, f.Relation, f.ValidFrom), nil)
	})
}

// FactsFrom returns every fact with Src == slug.
func (s *Store) FactsFrom(slug string, includeInvalid bool) ([]Fact, error) {
	var facts []Fact
	pb := []byte(prefixFact + slug + ":")
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 256
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Seek(pb); it.ValidForPrefix(pb); it.Next() {
			err := it.Item().Value(func(val []byte) error {
				var f Fact
				if err := json.Unmarshal(val, &f); err != nil {
					return err
				}
				if !includeInvalid && f.InvalidAt != nil {
					return nil
				}
				facts = append(facts, f)
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	return facts, err
}

// FactsAbout returns every fact with Src == slug or Dst == slug. Facts where
// slug is the Dst are found via the adj: reverse index rather than a full
// scan of fa:.
func (s *Store) FactsAbout(slug string, includeInvalid bool) ([]Fact, error) {
	facts, err := s.FactsFrom(slug, includeInvalid)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool, len(facts))
	for _, f := range facts {
		seen[string(factKey(f.Src, f.Relation, f.Dst, f.ValidFrom))] = true
	}

	pb := []byte(prefixAdj + slug + ":")
	err = s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Seek(pb); it.ValidForPrefix(pb); it.Next() {
			k := it.Item().KeyCopy(nil)
			rest := string(k[len(pb):]) // "<src>:<relation>:<validfrom-unixnano>"
			parts := strings.SplitN(rest, ":", 3)
			if len(parts) != 3 {
				continue
			}
			src, relation, validFromNano := parts[0], parts[1], parts[2]
			fk := prefixFact + src + ":" + relation + ":" + slug + ":" + validFromNano
			if seen[fk] {
				continue
			}
			item, err := txn.Get([]byte(fk))
			if errors.Is(err, badger.ErrKeyNotFound) {
				continue
			}
			if err != nil {
				return err
			}
			var f Fact
			if err := item.Value(func(val []byte) error {
				return json.Unmarshal(val, &f)
			}); err != nil {
				return err
			}
			if !includeInvalid && f.InvalidAt != nil {
				continue
			}
			seen[fk] = true
			facts = append(facts, f)
		}
		return nil
	})
	return facts, err
}

// InvalidateFact locates the exact fact identified by (src, relation, dst,
// validFrom) and sets its InvalidAt timestamp.
func (s *Store) InvalidateFact(src, relation, dst string, validFrom, at time.Time) error {
	key := factKey(src, relation, dst, validFrom)
	return s.db.Update(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if errors.Is(err, badger.ErrKeyNotFound) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		var f Fact
		if err := item.Value(func(val []byte) error {
			return json.Unmarshal(val, &f)
		}); err != nil {
			return err
		}
		atCopy := at
		f.InvalidAt = &atCopy
		b, err := json.Marshal(f)
		if err != nil {
			return err
		}
		return txn.Set(key, b)
	})
}

// DeleteFact removes the exact fact identified by (src, relation, dst,
// validFrom) — both its fa: record and its adj: reverse-index mirror.
// Unlike InvalidateFact (which preserves history by marking a fact
// invalid), DeleteFact erases it outright; callers that need to relocate a
// fact to a different ValidFrom (a different fa:/adj: key) should read the
// value first, DeleteFact the old key, then PutFact the new one. Returns
// ErrNotFound if no such fact exists.
func (s *Store) DeleteFact(src, relation, dst string, validFrom time.Time) error {
	key := factKey(src, relation, dst, validFrom)
	return s.db.Update(func(txn *badger.Txn) error {
		if _, err := txn.Get(key); err != nil {
			if errors.Is(err, badger.ErrKeyNotFound) {
				return ErrNotFound
			}
			return err
		}
		if err := txn.Delete(key); err != nil {
			return err
		}
		return txn.Delete(adjKey(dst, src, relation, validFrom))
	})
}

// --- Cursors ---

func cursorKey(path string) []byte {
	sum := sha256.Sum256([]byte(path))
	return []byte(prefixCursor + hex.EncodeToString(sum[:]))
}

func (s *Store) GetCursor(path string) (Cursor, bool, error) {
	var c Cursor
	found := false
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(cursorKey(path))
		if errors.Is(err, badger.ErrKeyNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			if err := json.Unmarshal(val, &c); err != nil {
				return err
			}
			found = true
			return nil
		})
	})
	return c, found, err
}

func (s *Store) PutCursor(c Cursor) error {
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set(cursorKey(c.Path), b)
	})
}

// Cursors returns every cursor, sorted by path.
func (s *Store) Cursors() ([]Cursor, error) {
	var cursors []Cursor
	pb := []byte(prefixCursor)
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 256
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Seek(pb); it.ValidForPrefix(pb); it.Next() {
			err := it.Item().Value(func(val []byte) error {
				var c Cursor
				if err := json.Unmarshal(val, &c); err != nil {
					return err
				}
				cursors = append(cursors, c)
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	sort.Slice(cursors, func(i, j int) bool { return cursors[i].Path < cursors[j].Path })
	return cursors, err
}

// --- Counts ---

func (s *Store) Counts() (episodes, entities, facts int, err error) {
	episodes = s.countPrefix(prefixEpisode)
	entities = s.countPrefix(prefixEntity)
	facts = s.countPrefix(prefixFact)
	return episodes, entities, facts, nil
}

func (s *Store) countPrefix(prefix string) int {
	var n int
	pb := []byte(prefix)
	_ = s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Seek(pb); it.ValidForPrefix(pb); it.Next() {
			n++
		}
		return nil
	})
	return n
}

// --- Normalization ---

var collapseRunsRE = regexp.MustCompile(`[ _]+`)

// Normalize lowercases, trims, and collapses runs of spaces/underscores into
// a single hyphen.
func Normalize(name string) string {
	n := strings.ToLower(strings.TrimSpace(name))
	return collapseRunsRE.ReplaceAllString(n, "-")
}

var nonSlugRE = regexp.MustCompile(`[^a-z0-9-]`)

// Slugify is Normalize followed by stripping any rune outside [a-z0-9-].
func Slugify(name string) string {
	return nonSlugRE.ReplaceAllString(Normalize(name), "")
}
