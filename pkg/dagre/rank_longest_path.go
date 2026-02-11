package dagre

import "math"

// longestPath assigns ranks using the longest path algorithm.
//
// This implementation mirrors dagre's rank/util longestPath behavior:
// it computes ranks from sources, and does not normalize internally.
func longestPath(g *Graph) {
	visited := make(map[string]bool, g.NodeCount())

	var dfs func(v string) int
	dfs = func(v string) int {
		label := g.Node(v)
		if label == nil {
			return 0
		}

		if visited[v] {
			return label.Rank
		}
		visited[v] = true

		minSuccRank := math.MaxInt
		for _, edge := range g.OutEdges(v) {
			succRank := dfs(edge.W)
			minLen := 1
			if edgeLabel := g.EdgeWithName(edge.V, edge.W, edge.Name); edgeLabel != nil {
				minLen = edgeLabel.MinLen
			}
			if succRank-minLen < minSuccRank {
				minSuccRank = succRank - minLen
			}
		}

		rank := 0
		if minSuccRank != math.MaxInt {
			rank = minSuccRank
		}
		label.Rank = rank
		return rank
	}

	for _, v := range g.Sources() {
		_ = dfs(v)
	}
}

func slack(g *Graph, edge EdgeObj) int {
	vLabel := g.Node(edge.V)
	wLabel := g.Node(edge.W)
	if vLabel == nil || wLabel == nil {
		return 0
	}

	edgeLabel := g.EdgeWithName(edge.V, edge.W, edge.Name)
	if edgeLabel == nil {
		return 0
	}

	return wLabel.Rank - vLabel.Rank - edgeLabel.MinLen
}

func isTight(g *Graph, edge EdgeObj) bool {
	return slack(g, edge) == 0
}
