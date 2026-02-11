package dagre

import (
	"fmt"
	"sync/atomic"
)

var reverseEdgeIDCounter atomic.Int64

func uniqueReverseEdgeName() string {
	n := reverseEdgeIDCounter.Add(1)
	return fmt.Sprintf("rev%d", n)
}

// runAcyclic removes cycles from the graph by reversing edges.
func runAcyclic(g *Graph) {
	var fas []EdgeObj

	if g.Acyclicer == "greedy" {
		fas = findFeedbackArcSetGreedy(g)
	} else {
		fas = findFeedbackArcSetDFS(g)
	}

	for _, e := range fas {
		reverseEdge(g, e)
	}
}

func reverseEdge(g *Graph, edge EdgeObj) {
	ed := g.EdgeWithName(edge.V, edge.W, edge.Name)
	if ed == nil {
		return
	}

	g.RemoveEdgeWithName(edge.V, edge.W, edge.Name)

	ed.ForwardName = edge.Name
	ed.Reversed = true

	revName := uniqueReverseEdgeName()
	g.SetEdgeWithName(edge.W, edge.V, revName, ed)
}

// undoAcyclic reverses the edge reversal done by runAcyclic.
func undoAcyclic(g *Graph) {
	for _, e := range g.Edges() {
		ed := g.EdgeWithName(e.V, e.W, e.Name)
		if ed == nil || !ed.Reversed {
			continue
		}

		g.RemoveEdgeWithName(e.V, e.W, e.Name)

		ed.Reversed = false
		forwardName := ed.ForwardName
		ed.ForwardName = ""
		g.SetEdgeWithName(e.W, e.V, forwardName, ed)
	}
}
