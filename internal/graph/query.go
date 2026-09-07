package graph

import graphstore "github.com/jeffdhooton/scry/internal/graph/store"

type PathResult struct {
	From     string   `json:"from"`
	To       string   `json:"to"`
	Path     []string `json:"path"`
	Edges    []string `json:"edges"`
	Distance int      `json:"distance"`
	Found    bool     `json:"found"`
	// Nodes attribute the display path to exact symbol IDs and definition sites.
	// Relationships retain stored direction even when BFS traverses in reverse.
	Nodes         []graphstore.NodeRecord `json:"nodes,omitempty"`
	Relationships []graphstore.EdgeRecord `json:"relationships,omitempty"`
}

// FindPath searches names and walks stored relationships, undirected, for at
// most nine hops. Labels come from actual edges, never guessed from node types.
func FindPath(st *graphstore.Store, fromQuery, toQuery string) (*PathResult, error) {
	result := &PathResult{From: fromQuery, To: toQuery}
	fromNodes, err := st.SearchNodes(fromQuery)
	if err != nil {
		return nil, err
	}
	toNodes, err := st.SearchNodes(toQuery)
	if err != nil {
		return nil, err
	}
	if len(fromNodes) == 0 || len(toNodes) == 0 {
		return result, nil
	}
	targetKeys := map[string]bool{}
	for _, n := range toNodes {
		targetKeys[n.Key()] = true
	}
	// The store returns edges in key order, giving deterministic tie breaking.
	edges, err := st.AllEdges()
	if err != nil {
		return nil, err
	}
	type step struct {
		key  string
		edge graphstore.EdgeRecord
	}
	adjacent := map[string][]step{}
	for _, e := range edges {
		adjacent[e.SrcKey] = append(adjacent[e.SrcKey], step{e.DstKey, e})
		adjacent[e.DstKey] = append(adjacent[e.DstKey], step{e.SrcKey, e})
	}
	visited := map[string]string{}
	via := map[string]graphstore.EdgeRecord{}
	var queue []string
	for _, n := range fromNodes {
		k := n.Key()
		visited[k] = ""
		queue = append(queue, k)
	}
	var target string
	for depth := 0; depth < 10 && len(queue) > 0; depth++ {
		var next []string
		for _, key := range queue {
			if targetKeys[key] {
				target = key
				goto done
			}
			for _, nb := range adjacent[key] {
				if _, seen := visited[nb.key]; seen {
					continue
				}
				// Do not return paths with dangling, unattributable endpoints.
				node, err := st.GetNode(nb.key)
				if err != nil {
					return nil, err
				}
				if node == nil {
					continue
				}
				visited[nb.key] = key
				via[nb.key] = nb.edge
				next = append(next, nb.key)
			}
		}
		queue = next
	}
done:
	if target == "" {
		return result, nil
	}
	var keys []string
	for cur := target; cur != ""; cur = visited[cur] {
		keys = append(keys, cur)
	}
	for i, j := 0, len(keys)-1; i < j; i, j = i+1, j-1 {
		keys[i], keys[j] = keys[j], keys[i]
	}
	for i, key := range keys {
		node, err := st.GetNode(key)
		if err != nil {
			return nil, err
		}
		result.Nodes = append(result.Nodes, *node)
		result.Path = append(result.Path, node.Name+" ("+node.Type+")")
		if i > 0 {
			e := via[key]
			result.Edges = append(result.Edges, e.Type)
			result.Relationships = append(result.Relationships, e)
		}
	}
	result.Found = true
	result.Distance = len(keys) - 1
	return result, nil
}
