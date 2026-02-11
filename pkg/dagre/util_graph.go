package dagre

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"sync/atomic"
)

var uniqueNodeCounter atomic.Int64

func uniqueID(prefix string) string {
	if prefix == "" {
		prefix = "_"
	}
	n := uniqueNodeCounter.Add(1)
	return fmt.Sprintf("%s%d", prefix, n)
}

func resetUniqueIDCounter() {
	uniqueNodeCounter.Store(0)
}

func addDummyNode(
	g *Graph,
	dummyType string,
	width, height float64,
	rank int,
	edgeSource, edgeTarget, edgeName, prefix string,
) string {
	if prefix == "" {
		prefix = "_d"
	}
	id := uniqueID(prefix)
	g.SetNode(id, &NodeData{
		Width:      width,
		Height:     height,
		Dummy:      dummyType,
		Rank:       rank,
		EdgeSource: edgeSource,
		EdgeTarget: edgeTarget,
		EdgeName:   edgeName,
	})
	return id
}

func addBorderNode(g *Graph, prefix string, rank, order *int) string {
	if prefix == "" {
		prefix = "_b"
	}
	id := uniqueID(prefix)
	n := &NodeData{Dummy: "border"}
	if rank != nil {
		n.Rank = *rank
	}
	if order != nil {
		n.Order = *order
	}
	g.SetNode(id, n)
	return id
}

func simplify(g *Graph) *Graph {
	simplified := NewGraphWithOptions(GraphOptions{
		Directed:   g.IsDirected(),
		Multigraph: false,
		Compound:   false,
	})
	simplified.SetLabel(g.Label())

	for _, v := range g.Nodes() {
		simplified.SetNode(v, g.Node(v))
	}

	for _, e := range g.Edges() {
		orig := g.EdgeWithName(e.V, e.W, e.Name)
		if orig == nil {
			continue
		}
		if existing := simplified.Edge(e.V, e.W); existing != nil {
			existing.Weight += orig.Weight
			if orig.MinLen > existing.MinLen {
				existing.MinLen = orig.MinLen
			}
			continue
		}
		ed := &EdgeData{MinLen: orig.MinLen, Weight: orig.Weight}
		simplified.SetEdge(e.V, e.W, ed)
	}

	return simplified
}

func asNonCompoundGraph(g *Graph) *Graph {
	result := NewGraphWithOptions(GraphOptions{
		Directed:   g.IsDirected(),
		Multigraph: g.IsMultigraph(),
		Compound:   false,
	})
	result.SetLabel(g.Label())

	for _, v := range g.Nodes() {
		if g.IsLeaf(v) {
			result.SetNode(v, g.Node(v))
		}
	}

	for _, e := range g.Edges() {
		if result.HasNode(e.V) && result.HasNode(e.W) {
			result.SetEdgeWithName(e.V, e.W, e.Name, g.EdgeWithName(e.V, e.W, e.Name))
		}
	}

	return result
}

func buildLayerMatrix(g *Graph) [][]string {
	maxRank := maxRank(g)
	if maxRank < 0 {
		return nil
	}

	layers := make([][]string, maxRank+1)
	for _, v := range g.Nodes() {
		n := g.Node(v)
		if n == nil {
			continue
		}
		if n.Rank < 0 || n.Rank >= len(layers) {
			continue
		}
		layers[n.Rank] = append(layers[n.Rank], v)
	}

	for i := range layers {
		sort.Slice(layers[i], func(a, b int) bool {
			na := g.Node(layers[i][a])
			nb := g.Node(layers[i][b])
			if na == nil || nb == nil {
				return layers[i][a] < layers[i][b]
			}
			if na.Order == nb.Order {
				return layers[i][a] < layers[i][b]
			}
			return na.Order < nb.Order
		})
	}

	return layers
}

func maxRank(g *Graph) int {
	max := -1
	for _, v := range g.Nodes() {
		if n := g.Node(v); n != nil && n.Rank > max {
			max = n.Rank
		}
	}
	return max
}

func normalizeRanks(g *Graph) {
	minRank := math.MaxInt
	for _, v := range g.Nodes() {
		n := g.Node(v)
		if n == nil {
			continue
		}
		if n.Dummy == "" || n.Dummy == "edge" || n.Dummy == "edge-label" {
			if n.Rank < minRank {
				minRank = n.Rank
			}
		}
	}

	if minRank == math.MaxInt {
		return
	}

	for _, v := range g.Nodes() {
		n := g.Node(v)
		if n != nil {
			n.Rank -= minRank
		}
	}
}

func removeEmptyRanks(g *Graph) {
	if g.IsCompound() {
		for _, v := range g.Nodes() {
			if children := g.Children(v); len(children) > 0 {
				return
			}
		}
	}

	nodeRanks := make([]int, 0, g.NodeCount())
	for _, v := range g.Nodes() {
		if n := g.Node(v); n != nil {
			nodeRanks = append(nodeRanks, n.Rank)
		}
	}
	if len(nodeRanks) == 0 {
		return
	}

	offset := nodeRanks[0]
	for _, r := range nodeRanks[1:] {
		if r < offset {
			offset = r
		}
	}

	layers := map[int][]string{}
	maxLayer := 0
	for _, v := range g.Nodes() {
		n := g.Node(v)
		if n == nil {
			continue
		}
		rank := n.Rank - offset
		if rank < 0 {
			continue
		}
		layers[rank] = append(layers[rank], v)
		if rank > maxLayer {
			maxLayer = rank
		}
	}

	nodeRankFactor := 1
	if opts, ok := g.Label().(*LayoutOptions); ok && opts.NodeRankFactor > 0 {
		nodeRankFactor = opts.NodeRankFactor
	}

	delta := 0
	for i := 0; i <= maxLayer; i++ {
		vs, ok := layers[i]
		if !ok {
			if nodeRankFactor != 0 && i%nodeRankFactor != 0 {
				delta--
			}
			continue
		}
		if delta == 0 {
			continue
		}
		for _, v := range vs {
			if n := g.Node(v); n != nil {
				n.Rank += delta
			}
		}
	}
}

func successorWeights(g *Graph) map[string]map[string]float64 {
	result := make(map[string]map[string]float64, g.NodeCount())
	for _, v := range g.Nodes() {
		result[v] = map[string]float64{}
		for _, edge := range g.OutEdges(v) {
			weight := 1.0
			if label := g.EdgeWithName(edge.V, edge.W, edge.Name); label != nil {
				weight = label.Weight
			}
			result[v][edge.W] += weight
		}
	}
	return result
}

func predecessorWeights(g *Graph) map[string]map[string]float64 {
	result := make(map[string]map[string]float64, g.NodeCount())
	for _, v := range g.Nodes() {
		result[v] = map[string]float64{}
		for _, edge := range g.InEdges(v) {
			weight := 1.0
			if label := g.EdgeWithName(edge.V, edge.W, edge.Name); label != nil {
				weight = label.Weight
			}
			result[v][edge.V] += weight
		}
	}
	return result
}

func intersectRect(x, y, width, height float64, point Point) (Point, error) {
	dx := point.X - x
	dy := point.Y - y
	w := width / 2
	h := height / 2

	if dx == 0 && dy == 0 {
		return Point{}, errors.New("not possible to find intersection inside of the rectangle")
	}

	var sx, sy float64
	if math.Abs(dy)*w > math.Abs(dx)*h {
		if dy < 0 {
			h = -h
		}
		sx = h * dx / dy
		sy = h
	} else {
		if dx < 0 {
			w = -w
		}
		sx = w
		sy = w * dy / dx
	}

	return Point{X: x + sx, Y: y + sy}, nil
}
