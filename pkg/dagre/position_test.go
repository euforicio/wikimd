package dagre

import "testing"

func TestFindType1Conflicts(t *testing.T) {
	g := NewGraphWithOptions(GraphOptions{Directed: true, Multigraph: false, Compound: false})
	g.SetNode("a", &NodeData{Rank: 0, Order: 0})
	g.SetNode("b", &NodeData{Rank: 0, Order: 1, Dummy: "edge"})
	g.SetNode("c", &NodeData{Rank: 0, Order: 2})
	g.SetNode("d", &NodeData{Rank: 1, Order: 0})
	g.SetNode("e", &NodeData{Rank: 1, Order: 1, Dummy: "edge"})
	g.SetNode("f", &NodeData{Rank: 1, Order: 2})

	g.SetEdge("c", "d", nil)
	g.SetEdge("a", "f", nil)
	g.SetEdge("b", "e", nil)

	layering := [][]string{{"a", "b", "c"}, {"d", "e", "f"}}
	conflicts := findType1Conflicts(g, layering)

	if !hasConflict(conflicts, "c", "d") {
		t.Fatalf("expected conflict between c and d; conflicts=%v", conflicts)
	}
	if !hasConflict(conflicts, "a", "f") {
		t.Fatalf("expected conflict between a and f; conflicts=%v", conflicts)
	}
}

func TestFindType2Conflicts(t *testing.T) {
	g := NewGraphWithOptions(GraphOptions{Directed: true, Multigraph: false, Compound: false})
	g.SetNode("n0", &NodeData{Rank: 0, Order: 0, Dummy: "edge"})
	g.SetNode("n1", &NodeData{Rank: 0, Order: 1, Dummy: "edge"})
	g.SetNode("n2", &NodeData{Rank: 0, Order: 2, Dummy: "edge"})
	g.SetNode("s0", &NodeData{Rank: 1, Order: 0, Dummy: "edge"})
	g.SetNode("br", &NodeData{Rank: 1, Order: 1, Dummy: "border"})
	g.SetNode("s1", &NodeData{Rank: 1, Order: 2, Dummy: "edge"})

	g.SetEdge("n2", "s0", nil)
	g.SetEdge("n1", "br", nil)
	g.SetEdge("n1", "s1", nil)

	layering := [][]string{{"n0", "n1", "n2"}, {"s0", "br", "s1"}}
	conflicts := findType2Conflicts(g, layering)

	if !hasConflict(conflicts, "n2", "s0") {
		t.Fatalf("expected type-2 conflict between n2 and s0; conflicts=%v", conflicts)
	}
}

func TestPositionYUsesMaxHeightPerRank(t *testing.T) {
	g := NewGraphWithOptions(GraphOptions{Directed: true, Multigraph: false, Compound: false})
	g.RankSep = 40
	g.SetNode("a", &NodeData{Rank: 0, Order: 0, Height: 10})
	g.SetNode("b", &NodeData{Rank: 0, Order: 1, Height: 30})
	g.SetNode("c", &NodeData{Rank: 1, Order: 0, Height: 20})

	positionY(g)

	if got := g.Node("a").Y; got != 15 {
		t.Fatalf("a.Y=%v, want 15", got)
	}
	if got := g.Node("b").Y; got != 15 {
		t.Fatalf("b.Y=%v, want 15", got)
	}
	if got := g.Node("c").Y; got != 80 {
		t.Fatalf("c.Y=%v, want 80", got)
	}
}

func TestPositionXRespectsNodeSeparation(t *testing.T) {
	g := NewGraphWithOptions(GraphOptions{Directed: true, Multigraph: false, Compound: false})
	g.NodeSep = 50
	g.EdgeSep = 10
	g.SetNode("a", &NodeData{Rank: 0, Order: 0, Width: 10})
	g.SetNode("b", &NodeData{Rank: 0, Order: 1, Width: 10})

	xs := positionX(g)
	if xs["b"]-xs["a"] < 60 {
		t.Fatalf("expected at least 60 separation, got a=%v b=%v", xs["a"], xs["b"])
	}
}

func TestBalanceUsesRequestedAlignment(t *testing.T) {
	xss := map[string]map[string]float64{
		"ul": {"v": 0},
		"ur": {"v": 10},
		"dl": {"v": 20},
		"dr": {"v": 30},
	}

	aligned := balance(xss, "UR")
	if aligned["v"] != 10 {
		t.Fatalf("balance(UR)=%v, want 10", aligned["v"])
	}

	median := balance(xss, "")
	if median["v"] != 15 {
		t.Fatalf("balance(default)=%v, want 15", median["v"])
	}
}
