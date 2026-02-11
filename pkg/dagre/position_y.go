package dagre

func positionY(g *Graph) {
	layering := buildLayerMatrix(g)
	prevY := 0.0
	for _, layer := range layering {
		maxHeight := 0.0
		for _, v := range layer {
			node := g.Node(v)
			if node == nil {
				continue
			}
			if node.Height > maxHeight {
				maxHeight = node.Height
			}
		}
		for _, v := range layer {
			node := g.Node(v)
			if node != nil {
				node.Y = prevY + maxHeight/2
			}
		}
		prevY += maxHeight + g.RankSep
	}
}
