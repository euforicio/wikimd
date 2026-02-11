package dagre

import "testing"

func newAcyclicTestGraph() *Graph {
	return NewGraphWithOptions(GraphOptions{Directed: true, Multigraph: true, Compound: false})
}

func TestRunAcyclicDFSNoCycle(t *testing.T) {
	g := newAcyclicTestGraph()
	g.Acyclicer = "dfs"
	g.SetNode("A", &NodeData{})
	g.SetNode("B", &NodeData{})
	g.SetNode("C", &NodeData{})
	g.SetEdgeWithName("A", "B", "ab", &EdgeData{})
	g.SetEdgeWithName("B", "C", "bc", &EdgeData{})

	runAcyclic(g)

	if !g.HasEdgeWithName("A", "B", "ab") || !g.HasEdgeWithName("B", "C", "bc") {
		t.Fatalf("expected original DAG edges to remain")
	}

	for _, e := range g.Edges() {
		if ed := g.EdgeWithName(e.V, e.W, e.Name); ed != nil && ed.Reversed {
			t.Fatalf("expected no reversed edges in DAG")
		}
	}
}

func TestRunAcyclicDFSCycleMarksReversed(t *testing.T) {
	g := newAcyclicTestGraph()
	g.Acyclicer = "dfs"
	g.SetNode("A", &NodeData{})
	g.SetNode("B", &NodeData{})
	g.SetNode("C", &NodeData{})
	g.SetEdgeWithName("A", "B", "ab", &EdgeData{})
	g.SetEdgeWithName("B", "C", "bc", &EdgeData{})
	g.SetEdgeWithName("C", "A", "ca", &EdgeData{})

	runAcyclic(g)

	reversed := 0
	for _, e := range g.Edges() {
		ed := g.EdgeWithName(e.V, e.W, e.Name)
		if ed != nil && ed.Reversed {
			reversed++
			if ed.ForwardName == "" {
				t.Fatalf("expected ForwardName on reversed edge")
			}
		}
	}
	if reversed == 0 {
		t.Fatalf("expected at least one reversed edge")
	}
}

func TestRunAcyclicGreedyCycleMarksReversed(t *testing.T) {
	g := newAcyclicTestGraph()
	g.Acyclicer = "greedy"
	g.SetNode("A", &NodeData{})
	g.SetNode("B", &NodeData{})
	g.SetNode("C", &NodeData{})
	g.SetNode("D", &NodeData{})
	g.SetEdgeWithName("A", "B", "ab", &EdgeData{})
	g.SetEdgeWithName("B", "C", "bc", &EdgeData{})
	g.SetEdgeWithName("C", "D", "cd", &EdgeData{})
	g.SetEdgeWithName("D", "A", "da", &EdgeData{})

	runAcyclic(g)

	reversed := 0
	for _, e := range g.Edges() {
		if ed := g.EdgeWithName(e.V, e.W, e.Name); ed != nil && ed.Reversed {
			reversed++
		}
	}
	if reversed == 0 {
		t.Fatalf("expected greedy FAS to reverse at least one edge")
	}
}

func TestRunAcyclicMultigraphSafeReverseNaming(t *testing.T) {
	g := newAcyclicTestGraph()
	g.Acyclicer = "dfs"
	g.SetNode("A", &NodeData{})
	g.SetNode("B", &NodeData{})

	g.SetEdgeWithName("A", "B", "ab", &EdgeData{})
	g.SetEdgeWithName("B", "A", "ba", &EdgeData{})

	runAcyclic(g)

	if g.EdgeCount() != 2 {
		t.Fatalf("expected both directional edges to remain after reversal, got %d", g.EdgeCount())
	}

	reversedWithRevName := 0
	for _, e := range g.Edges() {
		ed := g.EdgeWithName(e.V, e.W, e.Name)
		if ed != nil && ed.Reversed {
			if len(e.Name) >= 3 && e.Name[:3] == "rev" {
				reversedWithRevName++
			}
		}
	}
	if reversedWithRevName == 0 {
		t.Fatalf("expected reversed edge to use unique rev* name")
	}
}

func TestUndoAcyclicRestoresOriginalNamedEdges(t *testing.T) {
	g := newAcyclicTestGraph()
	g.Acyclicer = "dfs"
	g.SetNode("A", &NodeData{})
	g.SetNode("B", &NodeData{})

	g.SetEdgeWithName("A", "B", "ab", &EdgeData{})
	g.SetEdgeWithName("B", "A", "ba", &EdgeData{})

	runAcyclic(g)
	undoAcyclic(g)

	if !g.HasEdgeWithName("A", "B", "ab") {
		t.Fatalf("expected original edge A->B:ab restored")
	}
	if !g.HasEdgeWithName("B", "A", "ba") {
		t.Fatalf("expected original edge B->A:ba restored")
	}

	for _, e := range g.Edges() {
		if ed := g.EdgeWithName(e.V, e.W, e.Name); ed != nil && ed.Reversed {
			t.Fatalf("expected no reversed flags after undo")
		}
	}
}
