package dagre

import (
	"math"
	"slices"
	"strings"
)

func runLayout(g *Graph, opts *LayoutOptions) error {
	makeSpaceForEdgeLabels(g)
	removeSelfEdges(g)
	runAcyclic(g)
	runNestingGraph(g)
	runRank(asNonCompoundGraph(g))
	injectEdgeLabelProxies(g)
	removeEmptyRanksWithNodeRankFactor(g)
	cleanupNestingGraph(g)
	normalizeRanks(g)
	assignRankMinMax(g)
	removeEdgeLabelProxies(g)
	runNormalize(g)
	parentDummyChains(g)
	addBorderSegments(g)
	runOrder(g)
	insertSelfEdges(g)
	adjustCoordinateSystem(g)
	runPosition(g)
	positionSelfEdges(g)
	removeBorderNodes(g)
	undoNormalize(g)
	fixupEdgeLabelCoords(g)
	undoCoordinateSystem(g)
	translateGraph(g)
	assignNodeIntersects(g)
	reversePointsForReversedEdges(g)
	undoAcyclic(g)
	compactOutputRanks(g)
	return nil
}

func removeEmptyRanksWithNodeRankFactor(g *Graph) {
	nodeRanks := make([]int, 0, g.NodeCount())
	for _, v := range g.Nodes() {
		n := g.Node(v)
		if n != nil {
			nodeRanks = append(nodeRanks, n.Rank)
		}
	}
	if len(nodeRanks) == 0 {
		return
	}

	offset := nodeRanks[0]
	for _, r := range nodeRanks[1:] {
		offset = min(offset, r)
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
		maxLayer = max(maxLayer, rank)
	}

	nodeRankFactor := 1
	if opts := layoutOptionsForGraph(g); opts != nil && opts.NodeRankFactor > 0 {
		nodeRankFactor = opts.NodeRankFactor
	}

	delta := 0
	for i := 0; i <= maxLayer; i++ {
		vs, ok := layers[i]
		if !ok {
			if i%nodeRankFactor != 0 {
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

func assignRankMinMax(g *Graph) {
	maxRank := 0
	for _, v := range g.Nodes() {
		n := g.Node(v)
		if n != nil {
			n.MinRank = -1
			n.MaxRank = -1
		}
	}
	for _, v := range g.Nodes() {
		n := g.Node(v)
		if n == nil || n.BorderTop == "" || n.BorderBottom == "" {
			continue
		}

		top := g.Node(n.BorderTop)
		bottom := g.Node(n.BorderBottom)
		if top == nil || bottom == nil {
			continue
		}

		n.MinRank = top.Rank
		n.MaxRank = bottom.Rank
		maxRank = max(maxRank, n.MaxRank)
	}

	if opts := layoutOptionsForGraph(g); opts != nil {
		opts.pipeline.maxRank = maxRank
	}
}

func removeSelfEdges(g *Graph) {
	opts := layoutOptionsForGraph(g)
	if opts == nil {
		return
	}

	for _, e := range slices.Clone(g.Edges()) {
		if e.V != e.W {
			continue
		}
		ed := g.EdgeWithName(e.V, e.W, e.Name)
		if ed == nil {
			continue
		}

		opts.pipeline.selfEdges[e.V] = append(opts.pipeline.selfEdges[e.V], selfEdgeRecord{
			edge:  e,
			label: cloneEdgeData(ed),
		})
		g.RemoveEdgeWithName(e.V, e.W, e.Name)
	}
}

func insertSelfEdges(g *Graph) {
	opts := layoutOptionsForGraph(g)
	if opts == nil {
		return
	}

	layers := buildLayerMatrix(g)
	for _, layer := range layers {
		orderShift := 0
		for i, v := range layer {
			n := g.Node(v)
			if n == nil {
				continue
			}

			n.Order = i + orderShift
			selfEdges := opts.pipeline.selfEdges[v]
			for _, rec := range selfEdges {
				dummy := addDummyNode(
					g,
					"selfedge",
					rec.label.Width,
					rec.label.Height,
					n.Rank,
					rec.edge.V,
					rec.edge.W,
					rec.edge.Name,
					"_se",
				)
				dn := g.Node(dummy)
				if dn == nil {
					continue
				}
				edgeObj := rec.edge
				dn.Order = i + 1 + orderShift
				dn.EdgeObj = &edgeObj
				dn.EdgeLabel = rec.label
				orderShift++
			}

			delete(opts.pipeline.selfEdges, v)
		}
	}
}

func positionSelfEdges(g *Graph) {
	for _, v := range slices.Clone(g.Nodes()) {
		n := g.Node(v)
		if n == nil || n.Dummy != "selfedge" || n.EdgeObj == nil || n.EdgeLabel == nil {
			continue
		}

		selfNode := g.Node(n.EdgeObj.V)
		if selfNode == nil {
			continue
		}

		x := selfNode.X + selfNode.Width/2
		y := selfNode.Y
		dx := n.X - x
		dy := selfNode.Height / 2

		label := n.EdgeLabel
		g.SetEdgeWithName(n.EdgeObj.V, n.EdgeObj.W, n.EdgeObj.Name, label)
		g.RemoveNode(v)

		label.Points = []Point{
			{X: x + 2*dx/3, Y: y - dy},
			{X: x + 5*dx/6, Y: y - dy},
			{X: x + dx, Y: y},
			{X: x + 5*dx/6, Y: y + dy},
			{X: x + 2*dx/3, Y: y + dy},
		}
		label.X = n.X
		label.Y = n.Y
		label.HasLabelPosition = true
	}
}

func adjustCoordinateSystem(g *Graph) {
	rankDir := strings.ToUpper(g.RankDir)
	if rankDir == "LR" || rankDir == "RL" {
		swapWidthHeight(g)
	}
}

func undoCoordinateSystem(g *Graph) {
	rankDir := strings.ToUpper(g.RankDir)
	if rankDir == "BT" || rankDir == "RL" {
		reverseY(g)
	}
	if rankDir == "LR" || rankDir == "RL" {
		swapXY(g)
		swapWidthHeight(g)
	}
}

func swapWidthHeight(g *Graph) {
	for _, v := range g.Nodes() {
		n := g.Node(v)
		if n == nil {
			continue
		}
		n.Width, n.Height = n.Height, n.Width
	}

	for _, e := range g.Edges() {
		ed := g.EdgeWithName(e.V, e.W, e.Name)
		if ed == nil {
			continue
		}
		ed.Width, ed.Height = ed.Height, ed.Width
	}
}

func reverseY(g *Graph) {
	for _, v := range g.Nodes() {
		n := g.Node(v)
		if n != nil {
			n.Y = -n.Y
		}
	}
	for _, e := range g.Edges() {
		ed := g.EdgeWithName(e.V, e.W, e.Name)
		if ed == nil {
			continue
		}
		for i := range ed.Points {
			ed.Points[i].Y = -ed.Points[i].Y
		}
		if ed.HasLabelPosition {
			ed.Y = -ed.Y
		}
	}
}

func swapXY(g *Graph) {
	for _, v := range g.Nodes() {
		n := g.Node(v)
		if n == nil {
			continue
		}
		n.X, n.Y = n.Y, n.X
	}
	for _, e := range g.Edges() {
		ed := g.EdgeWithName(e.V, e.W, e.Name)
		if ed == nil {
			continue
		}
		for i := range ed.Points {
			ed.Points[i].X, ed.Points[i].Y = ed.Points[i].Y, ed.Points[i].X
		}
		if ed.HasLabelPosition {
			ed.X, ed.Y = ed.Y, ed.X
		}
	}
}

func translateGraph(g *Graph) {
	minX := math.Inf(1)
	maxX := 0.0
	minY := math.Inf(1)
	maxY := 0.0

	getExtremes := func(x, y, width, height float64) {
		minX = min(minX, x-width/2)
		maxX = max(maxX, x+width/2)
		minY = min(minY, y-height/2)
		maxY = max(maxY, y+height/2)
	}

	for _, v := range g.Nodes() {
		n := g.Node(v)
		if n == nil {
			continue
		}
		getExtremes(n.X, n.Y, n.Width, n.Height)
	}

	for _, e := range g.Edges() {
		ed := g.EdgeWithName(e.V, e.W, e.Name)
		if ed == nil || !ed.HasLabelPosition {
			continue
		}
		getExtremes(ed.X, ed.Y, ed.Width, ed.Height)
	}

	if math.IsInf(minX, 1) {
		minX = 0
		minY = 0
	}

	marginX := g.MarginX
	marginY := g.MarginY
	minX -= marginX
	minY -= marginY

	for _, v := range g.Nodes() {
		n := g.Node(v)
		if n == nil {
			continue
		}
		n.X -= minX
		n.Y -= minY
	}

	for _, e := range g.Edges() {
		ed := g.EdgeWithName(e.V, e.W, e.Name)
		if ed == nil {
			continue
		}
		for i := range ed.Points {
			ed.Points[i].X -= minX
			ed.Points[i].Y -= minY
		}
		if ed.HasLabelPosition {
			ed.X -= minX
			ed.Y -= minY
		}
	}

	if opts := layoutOptionsForGraph(g); opts != nil {
		opts.Width = maxX - minX + marginX
		opts.Height = maxY - minY + marginY
	}
}

func assignNodeIntersects(g *Graph) {
	for _, e := range g.Edges() {
		ed := g.EdgeWithName(e.V, e.W, e.Name)
		nodeV := g.Node(e.V)
		nodeW := g.Node(e.W)
		if ed == nil || nodeV == nil || nodeW == nil {
			continue
		}

		var p1, p2 Point
		if len(ed.Points) == 0 {
			ed.Points = []Point{}
			p1 = Point{X: nodeW.X, Y: nodeW.Y}
			p2 = Point{X: nodeV.X, Y: nodeV.Y}
		} else {
			p1 = ed.Points[0]
			p2 = ed.Points[len(ed.Points)-1]
		}

		head, err := intersectRect(nodeV.X, nodeV.Y, nodeV.Width, nodeV.Height, p1)
		if err != nil {
			head = Point{X: nodeV.X, Y: nodeV.Y}
		}
		tail, err := intersectRect(nodeW.X, nodeW.Y, nodeW.Width, nodeW.Height, p2)
		if err != nil {
			tail = Point{X: nodeW.X, Y: nodeW.Y}
		}

		points := make([]Point, 0, len(ed.Points)+2)
		points = append(points, head)
		points = append(points, ed.Points...)
		points = append(points, tail)
		ed.Points = points
	}
}

func reversePointsForReversedEdges(g *Graph) {
	for _, e := range g.Edges() {
		ed := g.EdgeWithName(e.V, e.W, e.Name)
		if ed == nil || !ed.Reversed {
			continue
		}
		slices.Reverse(ed.Points)
	}
}

func removeBorderNodes(g *Graph) {
	for _, v := range g.Nodes() {
		if len(g.Children(v)) == 0 {
			continue
		}

		n := g.Node(v)
		if n == nil || n.BorderTop == "" || n.BorderBottom == "" || len(n.BorderLeft) == 0 || len(n.BorderRight) == 0 {
			continue
		}

		t := g.Node(n.BorderTop)
		b := g.Node(n.BorderBottom)
		if t == nil || b == nil {
			continue
		}

		leftRank, ok := pickLastRank(n.BorderLeft, n.MaxRank)
		if !ok {
			continue
		}
		rightRank, ok := pickLastRank(n.BorderRight, n.MaxRank)
		if !ok {
			continue
		}

		l := g.Node(n.BorderLeft[leftRank])
		r := g.Node(n.BorderRight[rightRank])
		if l == nil || r == nil {
			continue
		}

		n.Width = math.Abs(r.X - l.X)
		n.Height = math.Abs(b.Y - t.Y)
		n.X = l.X + n.Width/2
		n.Y = t.Y + n.Height/2
	}

	for _, v := range slices.Clone(g.Nodes()) {
		n := g.Node(v)
		if n != nil && n.Dummy == "border" {
			g.RemoveNode(v)
		}
	}
}

func pickLastRank(border map[int]string, maxRank int) (int, bool) {
	if _, ok := border[maxRank]; ok {
		return maxRank, true
	}

	found := false
	best := 0
	for rank := range border {
		if !found || rank > best {
			best = rank
			found = true
		}
	}
	return best, found
}

func cloneEdgeData(ed *EdgeData) *EdgeData {
	if ed == nil {
		return nil
	}
	clone := *ed
	if len(ed.Points) > 0 {
		clone.Points = append([]Point(nil), ed.Points...)
	} else {
		clone.Points = nil
	}
	return &clone
}

func compactOutputRanks(g *Graph) {
	ranks := map[int]struct{}{}
	for _, v := range g.Nodes() {
		n := g.Node(v)
		if n == nil {
			continue
		}
		ranks[n.Rank] = struct{}{}
	}

	ordered := make([]int, 0, len(ranks))
	for rank := range ranks {
		ordered = append(ordered, rank)
	}
	slices.Sort(ordered)

	remap := make(map[int]int, len(ordered))
	for i, rank := range ordered {
		remap[rank] = i
	}

	for _, v := range g.Nodes() {
		n := g.Node(v)
		if n == nil {
			continue
		}
		n.Rank = remap[n.Rank]
	}
}
