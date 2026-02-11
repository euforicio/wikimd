package dagre

// orderLayerGraphLabel carries root node metadata for layer graphs.
type orderLayerGraphLabel struct {
	Root string
}

// buildLayerGraph constructs the movable/non-movable view for a single rank.
func buildLayerGraph(g *Graph, rank int, relationship string) *Graph {
	root := createLayerGraphRoot(g)
	result := NewGraphWithOptions(GraphOptions{Directed: true, Multigraph: false, Compound: true})
	result.SetLabel(orderLayerGraphLabel{Root: root})
	result.SetNode(root, &NodeData{})

	for _, v := range g.Nodes() {
		node := g.Node(v)
		if node == nil {
			continue
		}

		if !nodeInLayer(g, v, node, rank) {
			continue
		}

		copyNodeWithAncestors(result, g, v, root)
		for _, e := range layerGraphRelationshipEdges(g, v, relationship) {
			u := e.V
			if u == v {
				u = e.W
			}
			edge := g.EdgeWithName(e.V, e.W, e.Name)
			weight := 1.0
			if edge != nil && edge.Weight > 0 {
				weight = edge.Weight
			}
			existing := result.Edge(u, v)
			aggregated := weight
			if existing != nil {
				aggregated += existing.Weight
			}
			result.SetEdge(u, v, &EdgeData{Weight: aggregated})
		}

		if len(g.Children(v)) > 0 && node.BorderLeft != nil && node.BorderRight != nil {
			left := node.BorderLeft[rank]
			right := node.BorderRight[rank]
			result.SetNode(v, &NodeData{
				BorderLeft:  map[int]string{rank: left},
				BorderRight: map[int]string{rank: right},
			})
		}
	}

	return result
}

func createLayerGraphRoot(g *Graph) string {
	for {
		v := uniqueID("_root")
		if !g.HasNode(v) {
			return v
		}
	}
}

func layerGraphRoot(g *Graph) string {
	if label, ok := g.Label().(orderLayerGraphLabel); ok {
		return label.Root
	}
	if label, ok := g.Label().(*orderLayerGraphLabel); ok && label != nil {
		return label.Root
	}
	return ""
}

func nodeInLayer(g *Graph, v string, node *NodeData, rank int) bool {
	if node.Rank == rank {
		return true
	}
	if len(g.Children(v)) == 0 {
		return false
	}
	return node.MinRank <= rank && rank <= node.MaxRank
}

func copyNodeWithAncestors(result, src *Graph, v, root string) {
	if result.HasNode(v) {
		return
	}

	node := src.Node(v)
	if node != nil {
		result.SetNode(v, node)
	} else {
		result.SetNode(v, &NodeData{})
	}

	parent := src.Parent(v)
	if parent == "" {
		_ = result.SetParent(v, root)
		return
	}

	copyNodeWithAncestors(result, src, parent, root)
	_ = result.SetParent(v, parent)
}

func layerGraphRelationshipEdges(g *Graph, v, relationship string) []EdgeObj {
	switch relationship {
	case "inEdges":
		return g.InEdges(v)
	case "outEdges":
		return g.OutEdges(v)
	default:
		return nil
	}
}
