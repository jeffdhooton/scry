// Package store wraps the BadgerDB-backed graph index for one repo.
//
// Key prefixes:
//
//	node:<type>:<id>                              → NodeRecord
//	edge:<src_type>:<src_id>:<edge_type>:<dst_id> → EdgeRecord
//	adj:<node_key>:<edge_type>:<neighbor_key>     → empty (adjacency list)
//	community:<id>                                → CommunityRecord
//	report:latest                                 → GraphReport
//	meta:<key>                                    → metadata
package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/dgraph-io/badger/v4"
)

const SchemaVersion = 1

type NodeRecord struct {
	Type     string         `json:"type"`
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	File     string         `json:"file,omitempty"`
	Line     int            `json:"line,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

func (n *NodeRecord) Key() string {
	return n.Type + ":" + n.ID
}

type EdgeRecord struct {
	Type         string  `json:"type"`
	SrcKey       string  `json:"src"`
	DstKey       string  `json:"dst"`
	Confidence   float64 `json:"confidence"`
	SourceDomain string  `json:"source_domain"`
}

type CommunityRecord struct {
	ID       int      `json:"id"`
	Nodes    []string `json:"nodes"`
	Label    string   `json:"label"`
	Cohesion float64  `json:"cohesion"`
}

type GraphReport struct {
	NodeCount       int              `json:"node_count"`
	EdgeCount       int              `json:"edge_count"`
	CommunityCount  int              `json:"community_count"`
	GodNodes        []GodNode        `json:"god_nodes"`
	SurprisingEdges []SurprisingEdge `json:"surprising_edges"`
	Communities     []CommunityRecord `json:"communities"`
	SuggestedQueries []string        `json:"suggested_queries"`
}

type GodNode struct {
	Node   NodeRecord `json:"node"`
	Degree int        `json:"degree"`
}

type SurprisingEdge struct {
	Edge EdgeRecord `json:"edge"`
	Why  string     `json:"why"`
}

type Store struct {
	db *badger.DB
}

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

func (s *Store) Close() error  { return s.db.Close() }
func (s *Store) Reset() error  { return s.db.DropAll() }

func (s *Store) SetMeta(key string, value any) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte("meta:"+key), b)
	})
}

type Writer struct {
	wb *badger.WriteBatch
}

func (s *Store) NewWriter() *Writer {
	return &Writer{wb: s.db.NewWriteBatch()}
}

func (w *Writer) PutNode(n *NodeRecord) error {
	b, err := json.Marshal(n)
	if err != nil {
		return err
	}
	return w.wb.Set([]byte("node:"+n.Key()), b)
}

func (w *Writer) PutEdge(e *EdgeRecord) error {
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	key := fmt.Sprintf("edge:%s:%s:%s", e.SrcKey, e.Type, e.DstKey)
	if err := w.wb.Set([]byte(key), b); err != nil {
		return err
	}
	adjFwd := fmt.Sprintf("adj:%s:%s:%s", e.SrcKey, e.Type, e.DstKey)
	adjRev := fmt.Sprintf("adj:%s:%s:%s", e.DstKey, e.Type, e.SrcKey)
	if err := w.wb.Set([]byte(adjFwd), nil); err != nil {
		return err
	}
	return w.wb.Set([]byte(adjRev), nil)
}

func (w *Writer) PutCommunity(c *CommunityRecord) error {
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return w.wb.Set([]byte(fmt.Sprintf("community:%d", c.ID)), b)
}

func (w *Writer) PutReport(r *GraphReport) error {
	b, err := json.Marshal(r)
	if err != nil {
		return err
	}
	return w.wb.Set([]byte("report:latest"), b)
}

func (w *Writer) Flush() error { return w.wb.Flush() }

// --- Reads ---

func (s *Store) GetNode(key string) (*NodeRecord, error) {
	var rec NodeRecord
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("node:" + key))
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &rec)
		})
	})
	if errors.Is(err, badger.ErrKeyNotFound) {
		return nil, nil
	}
	return &rec, err
}

func (s *Store) GetNeighbors(nodeKey string) ([]string, error) {
	prefix := []byte("adj:" + nodeKey + ":")
	var neighbors []string
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			k := string(it.Item().KeyCopy(nil))
			rest := k[len("adj:"+nodeKey+":"):]
			// rest is "edge_type:neighbor_key"
			if idx := strings.Index(rest, ":"); idx >= 0 {
				neighbors = append(neighbors, rest[idx+1:])
			}
		}
		return nil
	})
	return neighbors, err
}

func (s *Store) GetEdge(srcKey, edgeType, dstKey string) (*EdgeRecord, error) {
	key := fmt.Sprintf("edge:%s:%s:%s", srcKey, edgeType, dstKey)
	var rec EdgeRecord
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &rec)
		})
	})
	if errors.Is(err, badger.ErrKeyNotFound) {
		return nil, nil
	}
	return &rec, err
}

func (s *Store) GetReport() (*GraphReport, error) {
	var rep GraphReport
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("report:latest"))
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &rep)
		})
	})
	if errors.Is(err, badger.ErrKeyNotFound) {
		return nil, nil
	}
	return &rep, err
}

func (s *Store) NodeCount() int {
	return s.countPrefix("node:")
}

func (s *Store) EdgeCount() int {
	return s.countPrefix("edge:")
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

// NodeDegrees returns degree count for every node (undirected, via adjacency list).
func (s *Store) NodeDegrees() (map[string]int, error) {
	degrees := map[string]int{}
	prefix := []byte("adj:")
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			k := string(it.Item().KeyCopy(nil))
			rest := k[len("adj:"):]
			// rest is "node_key:edge_type:neighbor_key"
			// node_key can contain ":", so we need to find the right split
			// adj entries are: adj:<nodeType>:<nodeID>:<edgeType>:<neighborType>:<neighborID>
			parts := strings.SplitN(rest, ":", 5)
			if len(parts) >= 2 {
				nodeKey := parts[0] + ":" + parts[1]
				degrees[nodeKey]++
			}
		}
		return nil
	})
	return degrees, err
}

// AllNodes returns every node in the store.
func (s *Store) AllNodes() ([]NodeRecord, error) {
	prefix := []byte("node:")
	var nodes []NodeRecord
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 256
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			err := it.Item().Value(func(val []byte) error {
				var n NodeRecord
				if err := json.Unmarshal(val, &n); err != nil {
					return err
				}
				nodes = append(nodes, n)
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	return nodes, err
}

// AllEdges returns every edge in the store.
func (s *Store) AllEdges() ([]EdgeRecord, error) {
	prefix := []byte("edge:")
	var edges []EdgeRecord
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 256
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			err := it.Item().Value(func(val []byte) error {
				var e EdgeRecord
				if err := json.Unmarshal(val, &e); err != nil {
					return err
				}
				edges = append(edges, e)
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	return edges, err
}

// SearchNodes finds nodes whose name contains the query (case-insensitive).
func (s *Store) SearchNodes(query string) ([]NodeRecord, error) {
	lower := strings.ToLower(query)
	nodes, err := s.AllNodes()
	if err != nil {
		return nil, err
	}
	var matches []NodeRecord
	for _, n := range nodes {
		if strings.Contains(strings.ToLower(n.Name), lower) {
			matches = append(matches, n)
		}
	}
	return matches, nil
}

// AllCommunities returns every community in the store.
func (s *Store) AllCommunities() ([]CommunityRecord, error) {
	prefix := []byte("community:")
	var comms []CommunityRecord
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 64
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			err := it.Item().Value(func(val []byte) error {
				var c CommunityRecord
				if err := json.Unmarshal(val, &c); err != nil {
					return err
				}
				comms = append(comms, c)
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	sort.Slice(comms, func(i, j int) bool { return comms[i].ID < comms[j].ID })
	return comms, err
}
