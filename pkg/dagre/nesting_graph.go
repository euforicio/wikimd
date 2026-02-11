package dagre

func runNestingGraph(g *Graph) {
	opts := layoutOptionsForGraph(g)
	if opts == nil {
		return
	}

	root := addDummyNode(g, "root", 0, 0, 0, "", "", "", "_root")
	depths := treeDepths(g)
	height := maxTreeDepth(depths) - 1
	nodeSep := 2*height + 1
	if nodeSep < 1 {
		nodeSep = 1
	}

	opts.pipeline.nestingRoot = root

	for _, e := range g.Edges() {
		ed := g.EdgeWithName(e.V, e.W, e.Name)
		if ed == nil {
			continue
		}
		ed.MinLen *= nodeSep
		if ed.MinLen <= 0 {
			ed.MinLen = nodeSep
		}
	}

	weight := sumWeights(g) + 1
	for _, child := range g.Children("") {
		nestingDFS(g, root, nodeSep, weight, height, depths, child)
	}

	opts.NodeRankFactor = nodeSep
}

func nestingDFS(g *Graph, root string, nodeSep int, weight float64, height int, depths map[string]int, v string) {
	children := g.Children(v)
	if len(children) == 0 {
		if v != root {
			g.SetEdge(root, v, &EdgeData{Weight: 0, MinLen: nodeSep})
		}
		return
	}

	top := addBorderNode(g, "_bt", nil, nil)
	bottom := addBorderNode(g, "_bb", nil, nil)
	label := g.Node(v)
	if label == nil {
		return
	}

	_ = g.SetParent(top, v)
	label.BorderTop = top
	_ = g.SetParent(bottom, v)
	label.BorderBottom = bottom

	for _, child := range children {
		nestingDFS(g, root, nodeSep, weight, height, depths, child)

		childNode := g.Node(child)
		if childNode == nil {
			continue
		}

		childTop := child
		if childNode.BorderTop != "" {
			childTop = childNode.BorderTop
		}
		childBottom := child
		if childNode.BorderBottom != "" {
			childBottom = childNode.BorderBottom
		}

		thisWeight := 2 * weight
		if childNode.BorderTop != "" {
			thisWeight = weight
		}
		minLen := height - depths[v] + 1
		if childTop != childBottom {
			minLen = 1
		}
		if minLen < 1 {
			minLen = 1
		}

		g.SetEdge(top, childTop, &EdgeData{Weight: thisWeight, MinLen: minLen, NestingEdge: true})
		g.SetEdge(childBottom, bottom, &EdgeData{Weight: thisWeight, MinLen: minLen, NestingEdge: true})
	}

	if g.Parent(v) == "" {
		minLen := height + depths[v]
		if minLen < 1 {
			minLen = 1
		}
		g.SetEdge(root, top, &EdgeData{Weight: 0, MinLen: minLen})
	}
}

func cleanupNestingGraph(g *Graph) {
	opts := layoutOptionsForGraph(g)
	if opts == nil {
		return
	}

	if opts.pipeline.nestingRoot != "" {
		g.RemoveNode(opts.pipeline.nestingRoot)
		opts.pipeline.nestingRoot = ""
	}

	for _, e := range g.Edges() {
		ed := g.EdgeWithName(e.V, e.W, e.Name)
		if ed != nil && ed.NestingEdge {
			g.RemoveEdgeWithName(e.V, e.W, e.Name)
		}
	}
}

func treeDepths(g *Graph) map[string]int {
	depths := make(map[string]int)
	var dfs func(v string, depth int)
	dfs = func(v string, depth int) {
		children := g.Children(v)
		if len(children) > 0 {
			for _, child := range children {
				dfs(child, depth+1)
			}
		}
		depths[v] = depth
	}

	for _, v := range g.Children("") {
		dfs(v, 1)
	}
	return depths
}

func maxTreeDepth(depths map[string]int) int {
	maxDepth := 0
	for _, depth := range depths {
		if depth > maxDepth {
			maxDepth = depth
		}
	}
	if maxDepth == 0 {
		maxDepth = 1
	}
	return maxDepth
}

func sumWeights(g *Graph) float64 {
	total := 0.0
	for _, e := range g.Edges() {
		ed := g.EdgeWithName(e.V, e.W, e.Name)
		if ed != nil {
			total += ed.Weight
		}
	}
	return total
}
