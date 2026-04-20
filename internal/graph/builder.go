// Package graph builds a unified cross-domain graph from scry's code, git,
// schema, and HTTP indexes.
package graph

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	gitstore "github.com/jeffdhooton/scry/internal/git/store"
	graphstore "github.com/jeffdhooton/scry/internal/graph/store"
	httpstore "github.com/jeffdhooton/scry/internal/http/store"
	"github.com/jeffdhooton/scry/internal/schema"
	schemastore "github.com/jeffdhooton/scry/internal/schema/store"
	codestore "github.com/jeffdhooton/scry/internal/store"
)

type RepoLayout struct {
	RepoPath     string
	StorageDir   string
	BadgerDir    string
	ManifestPath string
}

type Manifest struct {
	SchemaVersion int       `json:"schema_version"`
	RepoPath      string    `json:"repo_path"`
	IndexedAt     time.Time `json:"indexed_at"`
	Status        string    `json:"status"`
	NodeCount     int       `json:"node_count"`
	EdgeCount     int       `json:"edge_count"`
	Communities   int       `json:"communities"`
	ElapsedMs     int64     `json:"elapsed_ms"`
}

func Layout(scryHome, repoPath string) RepoLayout {
	hash := sha256.Sum256([]byte(repoPath))
	short := hex.EncodeToString(hash[:])[:16]
	storage := filepath.Join(scryHome, "repos", short, "graph")
	return RepoLayout{
		RepoPath:     repoPath,
		StorageDir:   storage,
		BadgerDir:    filepath.Join(storage, "index.db"),
		ManifestPath: filepath.Join(storage, "manifest.json"),
	}
}

func LoadManifest(layout RepoLayout) (*Manifest, error) {
	b, err := os.ReadFile(layout.ManifestPath)
	if err != nil {
		return nil, err
	}
	var m Manifest
	return &m, json.Unmarshal(b, &m)
}

// Sources holds references to the domain stores that the builder reads from.
// Any field may be nil if that domain isn't indexed.
type Sources struct {
	Code   *codestore.Store
	Git    *gitstore.Store
	Schema *schemastore.Store
	HTTP   *httpstore.Store
}

// Build constructs the unified graph from all available domain indexes.
func Build(scryHome, repoPath string, src Sources) (*Manifest, error) {
	layout := Layout(scryHome, repoPath)
	if err := os.MkdirAll(layout.StorageDir, 0o755); err != nil {
		return nil, fmt.Errorf("create storage dir: %w", err)
	}

	st, err := graphstore.Open(layout.BadgerDir)
	if err != nil {
		return nil, fmt.Errorf("open graph store: %w", err)
	}
	defer st.Close()

	if err := st.Reset(); err != nil {
		return nil, fmt.Errorf("reset graph store: %w", err)
	}
	if err := st.SetMeta("schema_version", graphstore.SchemaVersion); err != nil {
		return nil, err
	}

	start := time.Now()

	w := st.NewWriter()
	nodeSet := map[string]bool{}

	if src.Code != nil {
		if err := extractCodeNodes(src.Code, w, nodeSet); err != nil {
			return nil, fmt.Errorf("extract code nodes: %w", err)
		}
		if err := extractCallEdges(src.Code, w, nodeSet); err != nil {
			return nil, fmt.Errorf("extract call edges: %w", err)
		}
		if err := extractImplEdges(src.Code, w, nodeSet); err != nil {
			return nil, fmt.Errorf("extract impl edges: %w", err)
		}
	}

	if src.Schema != nil {
		if err := extractSchemaNodes(src.Schema, w, nodeSet); err != nil {
			return nil, fmt.Errorf("extract schema nodes: %w", err)
		}
	}

	if src.Git != nil {
		if err := extractGitNodes(src.Git, w, nodeSet); err != nil {
			return nil, fmt.Errorf("extract git nodes: %w", err)
		}
		if err := extractCochangeEdges(src.Git, w, nodeSet); err != nil {
			return nil, fmt.Errorf("extract cochange edges: %w", err)
		}
	}

	if src.HTTP != nil {
		if err := extractHTTPNodes(src.HTTP, w, nodeSet); err != nil {
			return nil, fmt.Errorf("extract http nodes: %w", err)
		}
	}

	if err := w.Flush(); err != nil {
		return nil, fmt.Errorf("flush graph: %w", err)
	}

	// Community detection and report generation
	communities, err := detectCommunities(st)
	if err != nil {
		return nil, fmt.Errorf("community detection: %w", err)
	}

	w2 := st.NewWriter()
	for i := range communities {
		if err := w2.PutCommunity(&communities[i]); err != nil {
			return nil, err
		}
	}

	report, err := generateReport(st, communities)
	if err != nil {
		return nil, fmt.Errorf("generate report: %w", err)
	}
	if err := w2.PutReport(report); err != nil {
		return nil, err
	}
	if err := w2.Flush(); err != nil {
		return nil, fmt.Errorf("flush report: %w", err)
	}

	elapsed := time.Since(start)
	manifest := &Manifest{
		SchemaVersion: graphstore.SchemaVersion,
		RepoPath:      repoPath,
		IndexedAt:     time.Now(),
		Status:        "ready",
		NodeCount:     st.NodeCount(),
		EdgeCount:     st.EdgeCount(),
		Communities:   len(communities),
		ElapsedMs:     elapsed.Milliseconds(),
	}

	b, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(layout.ManifestPath, b, 0o644); err != nil {
		return nil, err
	}

	return manifest, nil
}

func extractCodeNodes(s *codestore.Store, w *graphstore.Writer, nodeSet map[string]bool) error {
	return s.IterateAllSymbols(func(sym *codestore.SymbolRecord) error {
		nodeType := classifySymbolKind(sym.Kind)
		if nodeType == "" {
			return nil
		}
		node := &graphstore.NodeRecord{
			Type: nodeType,
			ID:   sym.Symbol,
			Name: sym.DisplayName,
			Metadata: map[string]any{
				"kind": sym.Kind,
			},
		}
		// Get first def location
		_ = s.IterateDefs(sym.Symbol, func(occ *codestore.OccurrenceRecord) error {
			node.File = occ.File
			node.Line = occ.Line
			return stopIter
		})
		key := node.Key()
		if nodeSet[key] {
			return nil
		}
		nodeSet[key] = true
		return w.PutNode(node)
	})
}

func extractCallEdges(s *codestore.Store, w *graphstore.Writer, nodeSet map[string]bool) error {
	return s.IterateAllSymbols(func(sym *codestore.SymbolRecord) error {
		callerKey := classifySymbolKind(sym.Kind) + ":" + sym.Symbol
		if !nodeSet[callerKey] {
			return nil
		}
		return s.IterateCallees(sym.Symbol, func(occ *codestore.OccurrenceRecord) error {
			calleeSym, _ := s.GetSymbol(occ.Symbol)
			if calleeSym == nil {
				return nil
			}
			calleeType := classifySymbolKind(calleeSym.Kind)
			if calleeType == "" {
				return nil
			}
			calleeKey := calleeType + ":" + occ.Symbol
			if !nodeSet[calleeKey] {
				return nil
			}
			return w.PutEdge(&graphstore.EdgeRecord{
				Type:         "calls",
				SrcKey:       callerKey,
				DstKey:       calleeKey,
				Confidence:   1.0,
				SourceDomain: "code",
			})
		})
	})
}

func extractImplEdges(s *codestore.Store, w *graphstore.Writer, nodeSet map[string]bool) error {
	return s.IterateAllSymbols(func(sym *codestore.SymbolRecord) error {
		baseType := classifySymbolKind(sym.Kind)
		if baseType == "" {
			return nil
		}
		baseKey := baseType + ":" + sym.Symbol
		if !nodeSet[baseKey] {
			return nil
		}
		implIDs, err := s.IterateImpls(sym.Symbol)
		if err != nil {
			return err
		}
		for _, implID := range implIDs {
			implSym, _ := s.GetSymbol(implID)
			if implSym == nil {
				continue
			}
			implType := classifySymbolKind(implSym.Kind)
			if implType == "" {
				continue
			}
			implKey := implType + ":" + implID
			if !nodeSet[implKey] {
				continue
			}
			if err := w.PutEdge(&graphstore.EdgeRecord{
				Type:         "implements",
				SrcKey:       implKey,
				DstKey:       baseKey,
				Confidence:   1.0,
				SourceDomain: "code",
			}); err != nil {
				return err
			}
		}
		return nil
	})
}

func extractSchemaNodes(s *schemastore.Store, w *graphstore.Writer, nodeSet map[string]bool) error {
	tables, err := s.ListTables()
	if err != nil {
		return err
	}
	for _, name := range tables {
		data, err := s.GetTable(name)
		if err != nil {
			continue
		}
		var table schema.TableRecord
		if err := json.Unmarshal(data, &table); err != nil {
			continue
		}

		node := &graphstore.NodeRecord{
			Type: "table",
			ID:   name,
			Name: name,
			Metadata: map[string]any{
				"type":         table.Type,
				"row_estimate": table.RowEstimate,
				"columns":      len(table.Columns),
			},
		}
		nodeSet[node.Key()] = true
		if err := w.PutNode(node); err != nil {
			return err
		}

		// FK edges
		for _, fk := range table.ForeignKeys {
			dstKey := "table:" + fk.ReferencedTable
			if err := w.PutEdge(&graphstore.EdgeRecord{
				Type:         "fk",
				SrcKey:       node.Key(),
				DstKey:       dstKey,
				Confidence:   1.0,
				SourceDomain: "schema",
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

func extractGitNodes(s *gitstore.Store, w *graphstore.Writer, nodeSet map[string]bool) error {
	contribs, err := s.GetContributors("")
	if err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, c := range contribs {
		if seen[c.Author] {
			continue
		}
		seen[c.Author] = true
		node := &graphstore.NodeRecord{
			Type: "author",
			ID:   c.Author,
			Name: c.Author,
			Metadata: map[string]any{
				"commits":       c.CommitCount,
				"lines_added":   c.LinesAdded,
				"lines_removed": c.LinesRemoved,
			},
		}
		nodeSet[node.Key()] = true
		if err := w.PutNode(node); err != nil {
			return err
		}
	}
	return nil
}

func extractCochangeEdges(s *gitstore.Store, w *graphstore.Writer, nodeSet map[string]bool) error {
	// Get hotspots to know which files exist
	hotspots, err := s.GetHotspots(500)
	if err != nil {
		return err
	}
	for _, hs := range hotspots {
		cochanges, err := s.GetCochange(hs.Path, 5)
		if err != nil {
			continue
		}
		for _, cc := range cochanges {
			if cc.Count < 3 {
				continue
			}
			srcKey := "file:" + hs.Path
			dstKey := "file:" + cc.FileB
			// Ensure file nodes exist
			if !nodeSet[srcKey] {
				nodeSet[srcKey] = true
				_ = w.PutNode(&graphstore.NodeRecord{Type: "file", ID: hs.Path, Name: hs.Path})
			}
			if !nodeSet[dstKey] {
				nodeSet[dstKey] = true
				_ = w.PutNode(&graphstore.NodeRecord{Type: "file", ID: cc.FileB, Name: cc.FileB})
			}
			confidence := 0.5 + float64(cc.Count)*0.05
			if confidence > 1.0 {
				confidence = 1.0
			}
			if err := w.PutEdge(&graphstore.EdgeRecord{
				Type:         "changed_with",
				SrcKey:       srcKey,
				DstKey:       dstKey,
				Confidence:   confidence,
				SourceDomain: "git",
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

func extractHTTPNodes(s *httpstore.Store, w *graphstore.Writer, nodeSet map[string]bool) error {
	reqs, err := s.List(httpstore.ListFilter{Limit: 1000})
	if err != nil {
		return err
	}
	// Deduplicate by method+path pattern
	seen := map[string]bool{}
	for _, req := range reqs {
		pattern := req.Method + " " + normalizePath(req.Path)
		if seen[pattern] {
			continue
		}
		seen[pattern] = true
		node := &graphstore.NodeRecord{
			Type: "endpoint",
			ID:   pattern,
			Name: pattern,
			Metadata: map[string]any{
				"method":      req.Method,
				"path":        req.Path,
				"status_code": req.StatusCode,
			},
		}
		nodeSet[node.Key()] = true
		if err := w.PutNode(node); err != nil {
			return err
		}
	}
	return nil
}

func normalizePath(path string) string {
	parts := strings.Split(path, "/")
	for i, p := range parts {
		if isNumeric(p) || isUUID(p) {
			parts[i] = ":id"
		}
	}
	return strings.Join(parts, "/")
}

func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func isUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	return s[8] == '-' && s[13] == '-' && s[18] == '-' && s[23] == '-'
}

func classifySymbolKind(kind string) string {
	switch strings.ToLower(kind) {
	case "function", "method":
		return "function"
	case "class", "struct", "type":
		return "class"
	case "interface", "trait", "protocol":
		return "interface"
	case "module", "package", "namespace":
		return "module"
	default:
		return ""
	}
}

var stopIter = fmt.Errorf("stop")

// --- Community Detection (Louvain) ---

func detectCommunities(st *graphstore.Store) ([]graphstore.CommunityRecord, error) {
	nodes, err := st.AllNodes()
	if err != nil {
		return nil, err
	}
	edges, err := st.AllEdges()
	if err != nil {
		return nil, err
	}

	if len(nodes) == 0 {
		return nil, nil
	}

	// Build adjacency for Louvain
	nodeIdx := map[string]int{}
	for i, n := range nodes {
		nodeIdx[n.Key()] = i
	}

	type weightedEdge struct {
		i, j   int
		weight float64
	}
	var wEdges []weightedEdge
	for _, e := range edges {
		si, ok1 := nodeIdx[e.SrcKey]
		di, ok2 := nodeIdx[e.DstKey]
		if ok1 && ok2 && si != di {
			wEdges = append(wEdges, weightedEdge{si, di, e.Confidence})
		}
	}

	// Simple Louvain: assign each node to its own community, then greedily
	// merge into neighbor communities that maximize modularity gain.
	n := len(nodes)
	community := make([]int, n)
	for i := range community {
		community[i] = i
	}

	// Build adjacency lists with weights
	adj := make([][]struct{ node int; weight float64 }, n)
	totalWeight := 0.0
	for _, e := range wEdges {
		adj[e.i] = append(adj[e.i], struct{ node int; weight float64 }{e.j, e.weight})
		adj[e.j] = append(adj[e.j], struct{ node int; weight float64 }{e.i, e.weight})
		totalWeight += e.weight
	}

	if totalWeight == 0 {
		// No edges — each node is its own community, skip detection
		return buildCommunityRecords(nodes, community), nil
	}

	// Node strength (sum of edge weights)
	strength := make([]float64, n)
	for i := range adj {
		for _, nb := range adj[i] {
			strength[i] += nb.weight
		}
	}

	// Iterate until no improvement
	for pass := 0; pass < 20; pass++ {
		changed := false
		for i := 0; i < n; i++ {
			bestComm := community[i]
			bestGain := 0.0

			// Sum of weights to each neighboring community
			commWeight := map[int]float64{}
			for _, nb := range adj[i] {
				commWeight[community[nb.node]] += nb.weight
			}

			currentComm := community[i]
			kiIn := commWeight[currentComm]
			ki := strength[i]

			// Community total strength
			commStrength := map[int]float64{}
			for j := 0; j < n; j++ {
				commStrength[community[j]] += strength[j]
			}

			for c, wic := range commWeight {
				if c == currentComm {
					continue
				}
				// Modularity gain from moving node i to community c
				sigmaTot := commStrength[c]
				gain := (wic - kiIn) - ki*(sigmaTot-commStrength[currentComm]+ki)/(2*totalWeight)
				if gain > bestGain {
					bestGain = gain
					bestComm = c
				}
			}

			if bestComm != currentComm {
				community[i] = bestComm
				changed = true
			}
		}
		if !changed {
			break
		}
	}

	return buildCommunityRecords(nodes, community), nil
}

func buildCommunityRecords(nodes []graphstore.NodeRecord, community []int) []graphstore.CommunityRecord {
	// Group nodes by community
	groups := map[int][]string{}
	for i, c := range community {
		groups[c] = append(groups[c], nodes[i].Key())
	}

	// Filter out single-node communities
	var communities []graphstore.CommunityRecord
	id := 0
	for _, members := range groups {
		if len(members) < 2 {
			continue
		}
		label := inferCommunityLabel(members)
		communities = append(communities, graphstore.CommunityRecord{
			ID:       id,
			Nodes:    members,
			Label:    label,
			Cohesion: float64(len(members)),
		})
		id++
	}

	sort.Slice(communities, func(i, j int) bool {
		return len(communities[i].Nodes) > len(communities[j].Nodes)
	})
	return communities
}

func inferCommunityLabel(members []string) string {
	// Use the most common path prefix among file nodes, or the most common
	// node type otherwise
	typeCounts := map[string]int{}
	var filePaths []string
	for _, key := range members {
		parts := strings.SplitN(key, ":", 2)
		if len(parts) == 2 {
			typeCounts[parts[0]]++
			if parts[0] == "file" || parts[0] == "function" || parts[0] == "class" {
				filePaths = append(filePaths, parts[1])
			}
		}
	}

	// Try common path prefix
	if len(filePaths) > 1 {
		prefix := commonPathPrefix(filePaths)
		if prefix != "" && prefix != "/" {
			return prefix
		}
	}

	// Fall back to dominant type + count
	bestType := ""
	bestCount := 0
	for t, c := range typeCounts {
		if c > bestCount {
			bestType = t
			bestCount = c
		}
	}
	return fmt.Sprintf("%s cluster (%d nodes)", bestType, len(members))
}

func commonPathPrefix(paths []string) string {
	if len(paths) == 0 {
		return ""
	}
	prefix := paths[0]
	for _, p := range paths[1:] {
		for !strings.HasPrefix(p, prefix) {
			lastSlash := strings.LastIndex(prefix, "/")
			if lastSlash < 0 {
				return ""
			}
			prefix = prefix[:lastSlash]
		}
	}
	return prefix
}

// --- Report Generation ---

func generateReport(st *graphstore.Store, communities []graphstore.CommunityRecord) (*graphstore.GraphReport, error) {
	nodeCount := st.NodeCount()
	edgeCount := st.EdgeCount()

	degrees, err := st.NodeDegrees()
	if err != nil {
		return nil, err
	}

	// God nodes: top 10 by degree
	type nd struct {
		key    string
		degree int
	}
	var ranked []nd
	for k, d := range degrees {
		ranked = append(ranked, nd{k, d})
	}
	sort.Slice(ranked, func(i, j int) bool { return ranked[i].degree > ranked[j].degree })

	var godNodes []graphstore.GodNode
	limit := 10
	if len(ranked) < limit {
		limit = len(ranked)
	}
	for _, r := range ranked[:limit] {
		node, _ := st.GetNode(r.key)
		if node == nil {
			continue
		}
		godNodes = append(godNodes, graphstore.GodNode{
			Node:   *node,
			Degree: r.degree,
		})
	}

	// Surprising edges: cross-domain with high confidence
	edges, _ := st.AllEdges()
	var surprising []graphstore.SurprisingEdge
	for _, e := range edges {
		srcDomain := strings.SplitN(e.SrcKey, ":", 2)[0]
		dstDomain := strings.SplitN(e.DstKey, ":", 2)[0]
		if srcDomain != dstDomain && e.Confidence >= 0.8 {
			surprising = append(surprising, graphstore.SurprisingEdge{
				Edge: e,
				Why:  fmt.Sprintf("cross-domain edge: %s → %s (%s)", srcDomain, dstDomain, e.Type),
			})
		}
	}
	sort.Slice(surprising, func(i, j int) bool {
		return surprising[i].Edge.Confidence > surprising[j].Edge.Confidence
	})
	if len(surprising) > 20 {
		surprising = surprising[:20]
	}

	// Suggested queries
	var queries []string
	if nodeCount > 0 {
		queries = append(queries, "What are the most connected functions?")
	}
	if len(communities) > 1 {
		queries = append(queries, "What connects community 0 to community 1?")
	}
	for _, gn := range godNodes {
		if gn.Node.Type == "function" {
			queries = append(queries, fmt.Sprintf("What does %s depend on?", gn.Node.Name))
			break
		}
	}
	for _, gn := range godNodes {
		if gn.Node.Type == "table" {
			queries = append(queries, fmt.Sprintf("What code touches the %s table?", gn.Node.Name))
			break
		}
	}

	return &graphstore.GraphReport{
		NodeCount:        nodeCount,
		EdgeCount:        edgeCount,
		CommunityCount:   len(communities),
		GodNodes:         godNodes,
		SurprisingEdges:  surprising,
		Communities:      communities,
		SuggestedQueries: queries,
	}, nil
}
