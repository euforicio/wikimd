package dagre

func runNormalize(g *Graph) {
	opts := layoutOptionsForGraph(g)
	if opts == nil {
		return
	}
	opts.pipeline.dummyChains = nil

	for _, e := range g.Edges() {
		normalizeEdge(g, e)
	}
}

func normalizeEdge(g *Graph, e EdgeObj) {
	v := e.V
	w := e.W
	name := e.Name

	vNode := g.Node(v)
	wNode := g.Node(w)
	edgeLabel := g.EdgeWithName(v, w, name)
	if vNode == nil || wNode == nil || edgeLabel == nil {
		return
	}

	vRank := vNode.Rank
	wRank := wNode.Rank
	labelRank := edgeLabel.LabelRank
	if wRank == vRank+1 {
		return
	}

	g.RemoveEdgeWithName(v, w, name)

	var dummy string
	for i := 0; vRank+1 < wRank; i++ {
		vRank++
		edgeLabel.Points = nil

		dummy = addDummyNode(g, "edge", 0, 0, vRank, e.V, e.W, e.Name, "_d")
		dummyNode := g.Node(dummy)
		if dummyNode == nil {
			continue
		}
		dummyNode.E = makeEdgeID(e)
		dummyNode.EdgeLabel = edgeLabel
		eCopy := e
		dummyNode.EdgeObj = &eCopy

		if vRank == labelRank {
			dummyNode.Width = edgeLabel.Width
			dummyNode.Height = edgeLabel.Height
			dummyNode.Dummy = "edge-label"
			dummyNode.LabelPos = edgeLabel.LabelPos
		}

		g.SetEdgeWithName(v, dummy, name, &EdgeData{Weight: edgeLabel.Weight, MinLen: 1})
		if i == 0 {
			if opts := layoutOptionsForGraph(g); opts != nil {
				opts.pipeline.dummyChains = append(opts.pipeline.dummyChains, dummy)
			}
		}
		v = dummy
	}

	g.SetEdgeWithName(v, w, name, &EdgeData{Weight: edgeLabel.Weight, MinLen: 1})
}
