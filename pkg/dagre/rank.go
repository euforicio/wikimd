package dagre

// runRank assigns ranks to nodes in the graph.
func runRank(g *Graph) {
	if opts, ok := g.Label().(*LayoutOptions); ok && opts != nil && opts.CustomRanker != nil {
		_ = opts.CustomRanker(g)
		return
	}

	switch g.Ranker {
	case "tight-tree":
		longestPath(g)
		_ = buildFeasibleTree(g)
	case "longest-path":
		longestPath(g)
	case "none":
		return
	default:
		networkSimplex(g)
	}

	// The current Go layout pipeline expects non-negative rank indices by this stage.
	normalizeRanks(g)
}
