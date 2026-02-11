package dagre

func undoNormalize(g *Graph) {
	opts := layoutOptionsForGraph(g)
	if opts == nil {
		return
	}

	for _, vStart := range opts.pipeline.dummyChains {
		v := vStart
		node := g.Node(v)
		if node == nil || node.EdgeObj == nil || node.EdgeLabel == nil {
			continue
		}

		edgeObj := *node.EdgeObj
		origLabel := node.EdgeLabel
		g.SetEdgeWithName(edgeObj.V, edgeObj.W, edgeObj.Name, origLabel)

		for node != nil && node.Dummy != "" {
			succ := g.Successors(v)
			if len(succ) == 0 {
				break
			}
			w := succ[0]
			g.RemoveNode(v)

			origLabel.Points = append(origLabel.Points, Point{X: node.X, Y: node.Y})
			if node.Dummy == "edge-label" {
				origLabel.X = node.X
				origLabel.Y = node.Y
				origLabel.Width = node.Width
				origLabel.Height = node.Height
				origLabel.HasLabelPosition = true
			}

			v = w
			node = g.Node(v)
		}
	}

	opts.pipeline.dummyChains = nil
}
