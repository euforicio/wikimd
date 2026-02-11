package dagre

// networkSimplex runs the network simplex rank optimizer.
func networkSimplex(g *Graph) {
	simplified := simplify(g)

	longestPath(simplified)
	tree := buildFeasibleTree(simplified)
	if tree == nil || tree.NodeCount() == 0 {
		return
	}

	initLowLimValues(tree, "")
	initCutValues(tree, simplified)

	for {
		leave := leaveEdge(tree)
		if leave == nil {
			break
		}

		enter := enterEdge(tree, simplified, *leave)
		if enter == nil {
			break
		}

		exchangeEdges(tree, simplified, *leave, *enter)
	}

	for _, v := range g.Nodes() {
		orig := g.Node(v)
		s := simplified.Node(v)
		if orig != nil && s != nil {
			orig.Rank = s.Rank
		}
	}
}

func initLowLimValues(tree *Graph, root string) {
	start := root
	if start == "" {
		nodes := tree.Nodes()
		if len(nodes) == 0 {
			return
		}
		start = nodes[0]
	}

	visited := make(map[string]bool, tree.NodeCount())
	counter := 1

	var dfs func(v, parent string) int
	dfs = func(v, parent string) int {
		label := tree.Node(v)
		if label == nil {
			return counter
		}

		low := counter
		visited[v] = true

		for _, w := range tree.Neighbors(v) {
			if !visited[w] {
				counter = dfs(w, v)
			}
		}

		label.Low = low
		label.Lim = counter
		label.Parent = parent
		counter++

		return counter
	}

	_ = dfs(start, "")
}

func initCutValues(tree, g *Graph) {
	nodes := postorder(tree)
	if len(nodes) <= 1 {
		return
	}

	for _, v := range nodes[:len(nodes)-1] {
		assignCutValue(tree, g, v)
	}
}

func assignCutValue(tree, g *Graph, child string) {
	childLabel := tree.Node(child)
	if childLabel == nil || childLabel.Parent == "" {
		return
	}
	parent := childLabel.Parent

	cut := calcCutValue(tree, g, child)
	if edgeLabel := tree.Edge(child, parent); edgeLabel != nil {
		edgeLabel.Cutvalue = cut
		return
	}
	if edgeLabel := tree.Edge(parent, child); edgeLabel != nil {
		edgeLabel.Cutvalue = cut
	}
}

func calcCutValue(tree, g *Graph, child string) int {
	childLabel := tree.Node(child)
	if childLabel == nil || childLabel.Parent == "" {
		return 0
	}
	parent := childLabel.Parent

	childIsTail := true
	graphEdge := g.Edge(child, parent)
	if graphEdge == nil {
		childIsTail = false
		graphEdge = g.Edge(parent, child)
	}
	if graphEdge == nil {
		return 0
	}

	cut := int(graphEdge.Weight)

	for _, e := range g.NodeEdges(child) {
		isOutEdge := e.V == child
		other := e.V
		if isOutEdge {
			other = e.W
		}

		if other == parent {
			continue
		}

		pointsToHead := isOutEdge == childIsTail
		otherWeight := 1
		if otherEdge := g.EdgeWithName(e.V, e.W, e.Name); otherEdge != nil {
			otherWeight = int(otherEdge.Weight)
		}

		if pointsToHead {
			cut += otherWeight
		} else {
			cut -= otherWeight
		}

		if tree.HasEdge(child, other) {
			if otherCut, ok := treeEdgeCutValue(tree, child, other); ok {
				if pointsToHead {
					cut -= otherCut
				} else {
					cut += otherCut
				}
			}
		}
	}

	return cut
}

func treeEdgeCutValue(tree *Graph, u, v string) (int, bool) {
	if label := tree.Edge(u, v); label != nil {
		return label.Cutvalue, true
	}
	if label := tree.Edge(v, u); label != nil {
		return label.Cutvalue, true
	}
	return 0, false
}

func leaveEdge(tree *Graph) *EdgeObj {
	for _, edge := range tree.Edges() {
		edgeLabel := tree.EdgeWithName(edge.V, edge.W, edge.Name)
		if edgeLabel != nil && edgeLabel.Cutvalue < 0 {
			e := edge
			return &e
		}
	}
	return nil
}

func enterEdge(tree, g *Graph, edge EdgeObj) *EdgeObj {
	v := edge.V
	w := edge.W

	if !g.HasEdge(v, w) {
		v, w = w, v
	}

	vLabel := tree.Node(v)
	wLabel := tree.Node(w)
	if vLabel == nil || wLabel == nil {
		return nil
	}

	tailLabel := vLabel
	flip := false
	if vLabel.Lim > wLabel.Lim {
		tailLabel = wLabel
		flip = true
	}

	maxInt := int(^uint(0) >> 1)
	minSlack := maxInt
	var result *EdgeObj

	for _, candidate := range g.Edges() {
		evLabel := tree.Node(candidate.V)
		ewLabel := tree.Node(candidate.W)
		if evLabel == nil || ewLabel == nil {
			continue
		}

		vIsDescendant := isDescendant(evLabel, tailLabel)
		wIsDescendant := isDescendant(ewLabel, tailLabel)
		if flip != vIsDescendant || flip == wIsDescendant {
			continue
		}

		s := slack(g, candidate)
		if s < minSlack {
			minSlack = s
			e := candidate
			result = &e
		}
	}

	return result
}

func isDescendant(vLabel, rootLabel *NodeData) bool {
	return rootLabel.Low <= vLabel.Lim && vLabel.Lim <= rootLabel.Lim
}

func exchangeEdges(tree, g *Graph, leave, enter EdgeObj) {
	tree.RemoveEdgeWithName(leave.V, leave.W, leave.Name)
	tree.RemoveEdgeWithName(leave.W, leave.V, leave.Name)
	tree.SetEdge(enter.V, enter.W, &EdgeData{})

	initLowLimValues(tree, "")
	initCutValues(tree, g)
	updateRanks(tree, g)
}

func updateRanks(tree, g *Graph) {
	root := ""
	for _, v := range tree.Nodes() {
		if label := g.Node(v); label != nil && label.Parent == "" {
			root = v
			break
		}
	}
	if root == "" {
		return
	}

	nodes := preorder(tree, root)
	if len(nodes) <= 1 {
		return
	}

	for _, v := range nodes[1:] {
		tLabel := tree.Node(v)
		if tLabel == nil || tLabel.Parent == "" {
			continue
		}
		parent := tLabel.Parent

		gLabel := g.Node(v)
		parentLabel := g.Node(parent)
		if gLabel == nil || parentLabel == nil {
			continue
		}

		flipped := false
		edge := g.Edge(v, parent)
		if edge == nil {
			edge = g.Edge(parent, v)
			flipped = true
		}
		if edge == nil {
			continue
		}

		if flipped {
			gLabel.Rank = parentLabel.Rank + edge.MinLen
		} else {
			gLabel.Rank = parentLabel.Rank - edge.MinLen
		}
	}
}

func postorder(tree *Graph) []string {
	result := make([]string, 0, tree.NodeCount())
	visited := make(map[string]bool, tree.NodeCount())
	nodes := tree.Nodes()
	if len(nodes) == 0 {
		return result
	}

	var dfs func(v string)
	dfs = func(v string) {
		visited[v] = true
		for _, w := range tree.Neighbors(v) {
			if !visited[w] {
				dfs(w)
			}
		}
		result = append(result, v)
	}

	dfs(nodes[0])
	return result
}

func preorder(tree *Graph, root string) []string {
	result := make([]string, 0, tree.NodeCount())
	visited := make(map[string]bool, tree.NodeCount())

	var dfs func(v string)
	dfs = func(v string) {
		visited[v] = true
		result = append(result, v)
		for _, w := range tree.Neighbors(v) {
			if !visited[w] {
				dfs(w)
			}
		}
	}

	dfs(root)
	return result
}
