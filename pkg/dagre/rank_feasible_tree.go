package dagre

// buildFeasibleTree constructs a spanning tree made only of tight edges.
func buildFeasibleTree(g *Graph) *Graph {
	tree := NewGraphWithOptions(GraphOptions{Directed: false, Multigraph: false, Compound: false})

	nodes := g.Nodes()
	if len(nodes) == 0 {
		return tree
	}

	start := nodes[0]
	tree.SetNode(start, &NodeData{})
	size := g.NodeCount()

	for tightTree(tree, g) < size {
		edge := findMinSlackEdge(tree, g)
		if edge == nil {
			break
		}

		delta := slack(g, *edge)
		if tree.HasNode(edge.V) {
			shiftRanks(tree, g, delta)
		} else {
			shiftRanks(tree, g, -delta)
		}
	}

	return tree
}

func tightTree(tree, g *Graph) int {
	visited := make(map[string]bool, tree.NodeCount())

	var dfs func(v string)
	dfs = func(v string) {
		if visited[v] {
			return
		}
		visited[v] = true

		for _, edge := range g.NodeEdges(v) {
			w := edge.V
			if w == v {
				w = edge.W
			}

			if tree.HasNode(w) || !isTight(g, edge) {
				continue
			}

			tree.SetNode(w, &NodeData{})
			tree.SetEdge(v, w, &EdgeData{})
			dfs(w)
		}
	}

	for _, v := range tree.Nodes() {
		dfs(v)
	}

	return tree.NodeCount()
}

func findMinSlackEdge(tree, g *Graph) *EdgeObj {
	minSlack := int(^uint(0) >> 1)
	var minEdge *EdgeObj

	for _, edge := range g.Edges() {
		vInTree := tree.HasNode(edge.V)
		wInTree := tree.HasNode(edge.W)
		if vInTree == wInTree {
			continue
		}

		edgeSlack := slack(g, edge)
		if edgeSlack >= minSlack {
			continue
		}

		minSlack = edgeSlack
		e := edge
		minEdge = &e
	}

	return minEdge
}

func shiftRanks(tree, g *Graph, delta int) {
	for _, v := range tree.Nodes() {
		if label := g.Node(v); label != nil {
			label.Rank += delta
		}
	}
}
