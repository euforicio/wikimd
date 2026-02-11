package dagre

import "testing"

func TestBuildLayerGraphAggregatesAndPreservesHierarchy(t *testing.T) {
	g := NewMultiGraph()
	g.SetNode("cluster", &NodeData{MinRank: 0, MaxRank: 2})
	g.SetNode("a", &NodeData{Rank: 1})
	g.SetNode("x", &NodeData{Rank: 0})
	if err := g.SetParent("a", "cluster"); err != nil {
		t.Fatalf("SetParent(a, cluster) error: %v", err)
	}

	g.SetEdgeWithName("x", "a", "e1", &EdgeData{Weight: 2})
	g.SetEdgeWithName("x", "a", "e2", &EdgeData{Weight: 3})

	lg := buildLayerGraph(g, 1, "inEdges")
	root := layerGraphRoot(lg)
	if root == "" {
		t.Fatal("expected layer graph root")
	}

	if !lg.HasNode("a") || !lg.HasNode("cluster") || !lg.HasNode("x") {
		t.Fatalf("missing nodes in layer graph: has a=%v cluster=%v x=%v", lg.HasNode("a"), lg.HasNode("cluster"), lg.HasNode("x"))
	}

	if got := lg.Parent("a"); got != "cluster" {
		t.Fatalf("Parent(a)=%q, want %q", got, "cluster")
	}
	if got := lg.Parent("cluster"); got != root {
		t.Fatalf("Parent(cluster)=%q, want root %q", got, root)
	}

	edge := lg.Edge("x", "a")
	if edge == nil {
		t.Fatal("expected aggregated edge x->a")
	}
	if edge.Weight != 5 {
		t.Fatalf("edge weight=%v, want 5", edge.Weight)
	}
}

func TestAddSubgraphConstraintsAddsOrderingEdges(t *testing.T) {
	g := NewGraph()
	g.SetNode("sg1", &NodeData{})
	g.SetNode("sg2", &NodeData{})
	g.SetNode("a", &NodeData{})
	g.SetNode("b", &NodeData{})

	if err := g.SetParent("a", "sg1"); err != nil {
		t.Fatalf("SetParent(a, sg1): %v", err)
	}
	if err := g.SetParent("b", "sg2"); err != nil {
		t.Fatalf("SetParent(b, sg2): %v", err)
	}

	cg := NewGraphWithOptions(GraphOptions{Directed: true, Multigraph: false, Compound: false})
	addSubgraphConstraints(g, cg, []string{"a", "b"})

	if !cg.HasEdge("sg1", "sg2") {
		t.Fatalf("expected ordering edge sg1->sg2, got edges=%v", cg.Edges())
	}
}

func TestSortSubgraphRespectsConstraintGraph(t *testing.T) {
	g := NewGraphWithOptions(GraphOptions{Directed: true, Multigraph: false, Compound: true})
	g.SetNode("root", &NodeData{})
	g.SetNode("u0", &NodeData{Order: 0})
	g.SetNode("u1", &NodeData{Order: 1})
	g.SetNode("v1", &NodeData{})
	g.SetNode("v2", &NodeData{})

	if err := g.SetParent("v1", "root"); err != nil {
		t.Fatalf("SetParent(v1, root): %v", err)
	}
	if err := g.SetParent("v2", "root"); err != nil {
		t.Fatalf("SetParent(v2, root): %v", err)
	}

	// Unconstrained barycenters would place v2 before v1.
	g.SetEdge("u1", "v1", &EdgeData{Weight: 1})
	g.SetEdge("u0", "v2", &EdgeData{Weight: 1})

	cg := NewGraphWithOptions(GraphOptions{Directed: true, Multigraph: false, Compound: false})
	cg.SetEdge("v1", "v2", nil)

	result := sortSubgraph(g, "root", cg, false)
	if len(result.VS) != 2 || result.VS[0] != "v1" || result.VS[1] != "v2" {
		t.Fatalf("unexpected constrained order: %v", result.VS)
	}
}

func TestCrossCountWeighted(t *testing.T) {
	g := NewGraphWithOptions(GraphOptions{Directed: true, Multigraph: false, Compound: false})
	g.SetNode("a", &NodeData{})
	g.SetNode("b", &NodeData{})
	g.SetNode("c", &NodeData{})
	g.SetNode("d", &NodeData{})

	g.SetEdge("a", "d", &EdgeData{Weight: 2})
	g.SetEdge("b", "c", &EdgeData{Weight: 3})

	layering := [][]string{{"a", "b"}, {"c", "d"}}
	if got := crossCount(g, layering); got != 6 {
		t.Fatalf("crossCount=%v, want 6", got)
	}
}
