// Package dagre implements a Go port of the dagre JavaScript library
// for directed graph layout using the Sugiyama algorithm.
package dagre

import "math"

// LayoutOptions configures the layout algorithm.
type LayoutOptions struct {
	RankDir   string  // "TB", "BT", "LR", "RL" - direction of layout
	Align     string  // "UL", "UR", "DL", "DR" - alignment within rank
	NodeSep   float64 // Separation between nodes on same rank
	EdgeSep   float64 // Minimum separation between edges
	RankSep   float64 // Separation between ranks
	MarginX   float64 // Horizontal margin
	MarginY   float64 // Vertical margin
	Acyclicer string  // "greedy" or "dfs" for cycle removal
	Ranker    string  // "network-simplex", "tight-tree", or "longest-path"
	Width     float64 // Output graph width
	Height    float64 // Output graph height

	// Internal parity fields.
	NodeRankFactor int
	CustomRanker   func(*Graph) error
	CustomOrder    func(*Graph, [][]string) error

	pipeline *layoutPipelineState
}

type layoutPipelineState struct {
	dummyChains []string
	nestingRoot string
	selfEdges   map[string][]selfEdgeRecord
	maxRank     int
}

type selfEdgeRecord struct {
	edge  EdgeObj
	label *EdgeData
}

func (opts *LayoutOptions) resetPipelineState() {
	opts.pipeline = &layoutPipelineState{
		selfEdges: make(map[string][]selfEdgeRecord),
		maxRank:   -1,
	}
}

func layoutOptionsForGraph(g *Graph) *LayoutOptions {
	opts, ok := g.Label().(*LayoutOptions)
	if !ok || opts == nil {
		return nil
	}
	if opts.pipeline == nil {
		opts.resetPipelineState()
	}
	return opts
}

// DefaultLayoutOptions returns sensible default options.
func DefaultLayoutOptions() *LayoutOptions {
	return &LayoutOptions{
		RankDir:   "TB",
		NodeSep:   50,
		EdgeSep:   10,
		RankSep:   50,
		MarginX:   20,
		MarginY:   20,
		Acyclicer: "dfs",
		Ranker:    "network-simplex",
	}
}

// Layout performs the complete layout algorithm on a graph.
func Layout(g *Graph, opts *LayoutOptions) error {
	if opts == nil {
		opts = DefaultLayoutOptions()
	}
	opts.resetPipelineState()

	g.RankDir = opts.RankDir
	g.Align = opts.Align
	g.NodeSep = opts.NodeSep
	g.EdgeSep = opts.EdgeSep
	g.RankSep = opts.RankSep
	g.MarginX = opts.MarginX
	g.MarginY = opts.MarginY
	g.Acyclicer = opts.Acyclicer
	g.Ranker = opts.Ranker
	g.SetLabel(opts)

	if g.NodeCount() == 0 {
		return nil
	}

	initializeEdgeLayoutDefaults(g)
	return runLayout(g, opts)
}

func initializeEdgeLayoutDefaults(g *Graph) {
	for _, e := range g.Edges() {
		ed := g.EdgeWithName(e.V, e.W, e.Name)
		if ed == nil {
			continue
		}
		if ed.LabelPos == "" {
			ed.LabelPos = "r"
		}
		if ed.MinLen <= 0 {
			ed.MinLen = 1
		}
		if ed.Weight == 0 {
			ed.Weight = 1
		}
		if ed.LabelOffset == 0 {
			ed.LabelOffset = 10
		}
		ed.HasLabelPosition = false
		ed.Points = nil
	}
}

// GraphSize returns the dimensions of the laid out graph.
func GraphSize(g *Graph) (width, height float64) {
	minX, minY := 1e99, 1e99
	maxX, maxY := -1e99, -1e99

	for _, v := range g.Nodes() {
		n := g.Node(v)
		if n == nil {
			continue
		}

		left := n.X - n.Width/2
		right := n.X + n.Width/2
		top := n.Y - n.Height/2
		bottom := n.Y + n.Height/2

		if left < minX {
			minX = left
		}
		if right > maxX {
			maxX = right
		}
		if top < minY {
			minY = top
		}
		if bottom > maxY {
			maxY = bottom
		}
	}

	if minX > maxX {
		return 0, 0
	}

	return maxX - minX, maxY - minY
}

// RouteEdges creates edge routing paths between nodes.
func RouteEdges(g *Graph) {
	for _, e := range g.Edges() {
		ed := g.EdgeWithName(e.V, e.W, e.Name)
		if ed == nil || len(ed.Points) > 0 {
			continue
		}

		vn := g.Node(e.V)
		wn := g.Node(e.W)
		if vn == nil || wn == nil {
			continue
		}

		ed.Points = createEdgePath(vn, wn, g.RankDir)
	}
}

func createEdgePath(from, to *NodeData, rankDir string) []Point {
	var startPoint, endPoint Point

	switch rankDir {
	case "LR":
		startPoint = Point{X: from.X + from.Width/2, Y: from.Y}
		endPoint = Point{X: to.X - to.Width/2, Y: to.Y}
	case "RL":
		startPoint = Point{X: from.X - from.Width/2, Y: from.Y}
		endPoint = Point{X: to.X + to.Width/2, Y: to.Y}
	case "BT":
		startPoint = Point{X: from.X, Y: from.Y - from.Height/2}
		endPoint = Point{X: to.X, Y: to.Y + to.Height/2}
	default:
		startPoint = Point{X: from.X, Y: from.Y + from.Height/2}
		endPoint = Point{X: to.X, Y: to.Y - to.Height/2}
	}

	dx := endPoint.X - startPoint.X
	dy := endPoint.Y - startPoint.Y

	switch rankDir {
	case "LR", "RL":
		if math.Abs(dy) < 1 {
			return []Point{startPoint, endPoint}
		}
		midX := (startPoint.X + endPoint.X) / 2
		return []Point{
			startPoint,
			{X: midX, Y: startPoint.Y},
			{X: midX, Y: endPoint.Y},
			endPoint,
		}
	default:
		if math.Abs(dx) < 1 {
			return []Point{startPoint, endPoint}
		}
		midY := (startPoint.Y + endPoint.Y) / 2
		return []Point{
			startPoint,
			{X: startPoint.X, Y: midY},
			{X: endPoint.X, Y: midY},
			endPoint,
		}
	}
}
