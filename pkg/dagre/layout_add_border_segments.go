package dagre

func addBorderSegments(g *Graph) {
	var dfs func(v string)
	dfs = func(v string) {
		children := g.Children(v)
		n := g.Node(v)
		if n == nil {
			return
		}

		for _, child := range children {
			dfs(child)
		}

		if n.MinRank < 0 || n.MaxRank < n.MinRank {
			return
		}

		n.BorderLeft = make(map[int]string)
		n.BorderRight = make(map[int]string)
		for rank := n.MinRank; rank <= n.MaxRank; rank++ {
			addBorderSegmentNode(g, "borderLeft", "_bl", v, n, rank)
			addBorderSegmentNode(g, "borderRight", "_br", v, n, rank)
		}
	}

	for _, v := range g.Children("") {
		dfs(v)
	}
}

func addBorderSegmentNode(g *Graph, prop, prefix, sg string, sgNode *NodeData, rank int) {
	curr := addDummyNode(g, "border", 0, 0, rank, "", "", "", prefix)
	n := g.Node(curr)
	if n != nil {
		n.BorderType = prop
	}

	_ = g.SetParent(curr, sg)

	var prev string
	if prop == "borderLeft" {
		prev = sgNode.BorderLeft[rank-1]
		sgNode.BorderLeft[rank] = curr
	} else {
		prev = sgNode.BorderRight[rank-1]
		sgNode.BorderRight[rank] = curr
	}

	if prev != "" {
		g.SetEdge(prev, curr, &EdgeData{Weight: 1, MinLen: 1})
	}
}
