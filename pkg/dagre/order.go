package dagre

import "sort"

func runOrder(g *Graph) {
	if opts, ok := g.Label().(*LayoutOptions); ok && opts != nil && opts.CustomOrder != nil {
		_ = opts.CustomOrder(g, buildLayerMatrix(g))
		return
	}

	maxRank := maxRank(g)
	if maxRank < 0 {
		return
	}

	downLayerGraphs := buildLayerGraphs(g, intRange(1, maxRank+1, 1), "inEdges")
	upLayerGraphs := buildLayerGraphs(g, intRange(maxRank-1, -1, -1), "outEdges")

	layering := initOrder(g)
	assignOrder(g, layering)

	bestCC := 1e308
	best := layering
	lastBest := 0

	for i := 0; lastBest < 4; i++ {
		lastBest++
		if i%2 == 1 {
			sweepLayerGraphs(downLayerGraphs, i%4 >= 2)
		} else {
			sweepLayerGraphs(upLayerGraphs, i%4 >= 2)
		}

		layering = buildLayerMatrix(g)
		cc := crossCount(g, layering)
		if cc < bestCC {
			lastBest = 0
			bestCC = cc
			best = copyLayering(layering)
		}
	}

	assignOrder(g, best)
}

func initOrder(g *Graph) [][]string {
	visited := map[string]bool{}
	simpleNodes := make([]string, 0)
	for _, v := range g.Nodes() {
		if len(g.Children(v)) == 0 {
			simpleNodes = append(simpleNodes, v)
		}
	}

	max := -1
	for _, v := range simpleNodes {
		n := g.Node(v)
		if n != nil && n.Rank > max {
			max = n.Rank
		}
	}
	if max < 0 {
		return nil
	}

	layers := make([][]string, max+1)

	var dfs func(string)
	dfs = func(v string) {
		if visited[v] {
			return
		}
		visited[v] = true
		node := g.Node(v)
		if node != nil && node.Rank >= 0 && node.Rank < len(layers) {
			layers[node.Rank] = append(layers[node.Rank], v)
		}
		for _, w := range g.Successors(v) {
			dfs(w)
		}
	}

	sort.Slice(simpleNodes, func(i, j int) bool {
		ni := g.Node(simpleNodes[i])
		nj := g.Node(simpleNodes[j])
		if ni == nil || nj == nil {
			return simpleNodes[i] < simpleNodes[j]
		}
		if ni.Rank == nj.Rank {
			return simpleNodes[i] < simpleNodes[j]
		}
		return ni.Rank < nj.Rank
	})

	for _, v := range simpleNodes {
		dfs(v)
	}

	return layers
}

func buildLayerGraphs(g *Graph, ranks []int, relationship string) []*Graph {
	result := make([]*Graph, 0, len(ranks))
	for _, rank := range ranks {
		result = append(result, buildLayerGraph(g, rank, relationship))
	}
	return result
}

func sweepLayerGraphs(layerGraphs []*Graph, biasRight bool) {
	cg := NewGraphWithOptions(GraphOptions{Directed: true, Multigraph: false, Compound: false})

	for _, lg := range layerGraphs {
		if lg == nil {
			continue
		}
		root := layerGraphRoot(lg)
		if root == "" {
			continue
		}
		sorted := sortSubgraph(lg, root, cg, biasRight)
		for i, v := range sorted.VS {
			if n := lg.Node(v); n != nil {
				n.Order = i
			}
		}
		addSubgraphConstraints(lg, cg, sorted.VS)
	}
}

func assignOrder(g *Graph, layering [][]string) {
	for _, layer := range layering {
		for i, v := range layer {
			if n := g.Node(v); n != nil {
				n.Order = i
			}
		}
	}
}

func copyLayering(layering [][]string) [][]string {
	copy := make([][]string, len(layering))
	for i := range layering {
		copy[i] = append([]string(nil), layering[i]...)
	}
	return copy
}

func intRange(start, limit, step int) []int {
	if step == 0 {
		return nil
	}
	result := []int{}
	if step > 0 {
		for i := start; i < limit; i += step {
			result = append(result, i)
		}
		return result
	}
	for i := start; i > limit; i += step {
		result = append(result, i)
	}
	return result
}
