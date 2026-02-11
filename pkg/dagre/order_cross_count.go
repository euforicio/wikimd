package dagre

import "sort"

// crossCount returns the weighted edge crossing count for a full layering.
func crossCount(g *Graph, layering [][]string) float64 {
	cc := 0.0
	for i := 1; i < len(layering); i++ {
		cc += twoLayerCrossCount(g, layering[i-1], layering[i])
	}
	return cc
}

// twoLayerCrossCount implements Barth et al. bilayer crossing counting.
func twoLayerCrossCount(g *Graph, northLayer, southLayer []string) float64 {
	if len(northLayer) == 0 || len(southLayer) == 0 {
		return 0
	}

	southPos := make(map[string]int, len(southLayer))
	for i, v := range southLayer {
		southPos[v] = i
	}

	type southEntry struct {
		pos    int
		weight float64
	}
	southEntries := make([]southEntry, 0)

	for _, v := range northLayer {
		edges := g.OutEdges(v)
		tmp := make([]southEntry, 0, len(edges))
		for _, e := range edges {
			pos, ok := southPos[e.W]
			if !ok {
				continue
			}
			edge := g.EdgeWithName(e.V, e.W, e.Name)
			weight := 1.0
			if edge != nil && edge.Weight > 0 {
				weight = edge.Weight
			}
			tmp = append(tmp, southEntry{pos: pos, weight: weight})
		}
		sort.Slice(tmp, func(i, j int) bool { return tmp[i].pos < tmp[j].pos })
		southEntries = append(southEntries, tmp...)
	}

	firstIndex := 1
	for firstIndex < len(southLayer) {
		firstIndex <<= 1
	}
	treeSize := 2*firstIndex - 1
	firstIndex--
	tree := make([]float64, treeSize)

	cc := 0.0
	for _, entry := range southEntries {
		index := entry.pos + firstIndex
		tree[index] += entry.weight
		weightSum := 0.0
		for index > 0 {
			if index%2 == 1 {
				weightSum += tree[index+1]
			}
			index = (index - 1) >> 1
			tree[index] += entry.weight
		}
		cc += entry.weight * weightSum
	}

	return cc
}
