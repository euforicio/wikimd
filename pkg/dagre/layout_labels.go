package dagre

import "strings"

func makeSpaceForEdgeLabels(g *Graph) {
	g.RankSep /= 2
	rankDir := strings.ToUpper(g.RankDir)

	for _, e := range g.Edges() {
		ed := g.EdgeWithName(e.V, e.W, e.Name)
		if ed == nil {
			continue
		}

		ed.MinLen *= 2
		if ed.MinLen <= 0 {
			ed.MinLen = 1
		}

		if strings.ToLower(ed.LabelPos) == "c" {
			continue
		}

		if rankDir == "TB" || rankDir == "BT" {
			ed.Width += ed.LabelOffset
		} else {
			ed.Height += ed.LabelOffset
		}
	}
}

func injectEdgeLabelProxies(g *Graph) {
	for _, e := range g.Edges() {
		ed := g.EdgeWithName(e.V, e.W, e.Name)
		if ed == nil || ed.Width == 0 || ed.Height == 0 {
			continue
		}

		v := g.Node(e.V)
		w := g.Node(e.W)
		if v == nil || w == nil {
			continue
		}

		rank := (w.Rank-v.Rank)/2 + v.Rank
		dummy := addDummyNode(g, "edge-proxy", 0, 0, rank, e.V, e.W, e.Name, "_ep")
		if dn := g.Node(dummy); dn != nil {
			eCopy := e
			dn.EdgeObj = &eCopy
		}
	}
}

func removeEdgeLabelProxies(g *Graph) {
	for _, v := range g.Nodes() {
		n := g.Node(v)
		if n == nil || n.Dummy != "edge-proxy" {
			continue
		}

		ed := g.EdgeWithName(n.EdgeSource, n.EdgeTarget, n.EdgeName)
		if ed != nil {
			ed.LabelRank = n.Rank
		}
		g.RemoveNode(v)
	}
}

func fixupEdgeLabelCoords(g *Graph) {
	for _, e := range g.Edges() {
		ed := g.EdgeWithName(e.V, e.W, e.Name)
		if ed == nil || !ed.HasLabelPosition {
			continue
		}

		labelPos := strings.ToLower(ed.LabelPos)
		if labelPos == "l" || labelPos == "r" {
			ed.Width -= ed.LabelOffset
			if ed.Width < 0 {
				ed.Width = 0
			}
		}

		switch labelPos {
		case "l":
			ed.X -= ed.Width/2 + ed.LabelOffset
		case "r":
			ed.X += ed.Width/2 + ed.LabelOffset
		}
	}
}
