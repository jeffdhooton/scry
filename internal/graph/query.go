package graph

import (
	"strings"

	graphstore "github.com/jeffdhooton/scry/internal/graph/store"
)

type PathResult struct {
	From     string   `json:"from"`
	To       string   `json:"to"`
	Path     []string `json:"path"`
	Edges    []string `json:"edges"`
	Distance int      `json:"distance"`
	Found    bool     `json:"found"`
}

// FindPath performs BFS to find the shortest path between two nodes.
// fromQuery and toQuery are searched by name substring across all node types.
func FindPath(st *graphstore.Store, fromQuery, toQuery string) (*PathResult, error) {
	fromNodes, err := st.SearchNodes(fromQuery)
	if err != nil {
		return nil, err
	}
	toNodes, err := st.SearchNodes(toQuery)
	if err != nil {
		return nil, err
	}

	if len(fromNodes) == 0 || len(toNodes) == 0 {
		return &PathResult{From: fromQuery, To: toQuery, Found: false}, nil
	}

	// Build target set
	targetKeys := map[string]bool{}
	for _, n := range toNodes {
		targetKeys[n.Key()] = true
	}

	// BFS from all source nodes
	type bfsEntry struct {
		key  string
		prev string
	}
	visited := map[string]string{} // key -> previous key (for path reconstruction)
	var queue []bfsEntry

	for _, n := range fromNodes {
		k := n.Key()
		visited[k] = ""
		queue = append(queue, bfsEntry{k, ""})
	}

	var foundTarget string
	maxDepth := 10

	for depth := 0; depth < maxDepth && len(queue) > 0; depth++ {
		var nextQueue []bfsEntry
		for _, entry := range queue {
			if targetKeys[entry.key] {
				foundTarget = entry.key
				goto done
			}
			neighbors, err := st.GetNeighbors(entry.key)
			if err != nil {
				continue
			}
			for _, nb := range neighbors {
				if _, seen := visited[nb]; !seen {
					visited[nb] = entry.key
					nextQueue = append(nextQueue, bfsEntry{nb, entry.key})
				}
			}
		}
		queue = nextQueue
	}

done:
	if foundTarget == "" {
		return &PathResult{From: fromQuery, To: toQuery, Found: false}, nil
	}

	// Reconstruct path
	var path []string
	for cur := foundTarget; cur != ""; cur = visited[cur] {
		path = append(path, cur)
	}
	// Reverse
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}

	// Resolve node names for display
	var displayPath []string
	for _, key := range path {
		node, _ := st.GetNode(key)
		if node != nil {
			displayPath = append(displayPath, node.Name+" ("+node.Type+")")
		} else {
			displayPath = append(displayPath, key)
		}
	}

	// Infer edge types between consecutive nodes
	var edgeTypes []string
	for i := 0; i < len(path)-1; i++ {
		edgeType := inferEdgeType(path[i], path[i+1])
		edgeTypes = append(edgeTypes, edgeType)
	}

	return &PathResult{
		From:     fromQuery,
		To:       toQuery,
		Path:     displayPath,
		Edges:    edgeTypes,
		Distance: len(path) - 1,
		Found:    true,
	}, nil
}

func inferEdgeType(srcKey, dstKey string) string {
	srcType := strings.SplitN(srcKey, ":", 2)[0]
	dstType := strings.SplitN(dstKey, ":", 2)[0]

	switch {
	case srcType == "function" && dstType == "function":
		return "calls"
	case srcType == "class" && dstType == "interface":
		return "implements"
	case srcType == "table" && dstType == "table":
		return "fk"
	case srcType == "file" && dstType == "file":
		return "changed_with"
	case srcType == "function" && dstType == "table":
		return "queries"
	default:
		return "connected"
	}
}
