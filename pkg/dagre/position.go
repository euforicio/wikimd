package dagre

// runPosition assigns coordinates using Dagre's position pipeline.
func runPosition(g *Graph) {
	ng := asNonCompoundGraph(g)
	positionY(ng)
	xs := positionX(ng)
	for v, x := range xs {
		if n := ng.Node(v); n != nil {
			n.X = x
		}
	}
}
