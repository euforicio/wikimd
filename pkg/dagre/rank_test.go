package dagre

import "testing"

func newRankTestGraph() *Graph {
	return NewGraphWithOptions(GraphOptions{Directed: true, Multigraph: true, Compound: false})
}

func TestLongestPathLinearChain(t *testing.T) {
	g := newRankTestGraph()
	g.SetNode("A", &NodeData{})
	g.SetNode("B", &NodeData{})
	g.SetNode("C", &NodeData{})
	g.SetNode("D", &NodeData{})
	g.SetEdgeWithName("A", "B", "ab", &EdgeData{})
	g.SetEdgeWithName("B", "C", "bc", &EdgeData{})
	g.SetEdgeWithName("C", "D", "cd", &EdgeData{})

	longestPath(g)
	normalizeRanks(g)

	if g.Node("A").Rank != 0 || g.Node("B").Rank != 1 || g.Node("C").Rank != 2 || g.Node("D").Rank != 3 {
		t.Fatalf("unexpected normalized ranks: A=%d B=%d C=%d D=%d", g.Node("A").Rank, g.Node("B").Rank, g.Node("C").Rank, g.Node("D").Rank)
	}
}

func TestLongestPathMinLenConstraint(t *testing.T) {
	g := newRankTestGraph()
	g.SetNode("A", &NodeData{})
	g.SetNode("B", &NodeData{})
	g.SetEdgeWithName("A", "B", "ab", &EdgeData{MinLen: 2, Weight: 1})

	longestPath(g)

	rankA := g.Node("A").Rank
	rankB := g.Node("B").Rank
	if rankB-rankA != 2 {
		t.Fatalf("expected rank(B)-rank(A)=2, got %d", rankB-rankA)
	}
}

func TestBuildFeasibleTreeUsesTightEdges(t *testing.T) {
	g := newRankTestGraph()
	g.SetNode("A", &NodeData{})
	g.SetNode("B", &NodeData{})
	g.SetNode("C", &NodeData{})
	g.SetEdgeWithName("A", "B", "ab", &EdgeData{})
	g.SetEdgeWithName("B", "C", "bc", &EdgeData{})

	longestPath(g)
	tree := buildFeasibleTree(g)

	if tree == nil {
		t.Fatalf("expected feasible tree")
	}
	if tree.NodeCount() != g.NodeCount() {
		t.Fatalf("expected tree to include all nodes, got %d/%d", tree.NodeCount(), g.NodeCount())
	}

	for _, e := range tree.Edges() {
		if s := slack(g, e); s != 0 {
			t.Fatalf("expected tight tree edge slack=0, got %d for %s->%s", s, e.V, e.W)
		}
	}
}

func TestNetworkSimplexDiamond(t *testing.T) {
	g := newRankTestGraph()
	g.SetNode("A", &NodeData{})
	g.SetNode("B", &NodeData{})
	g.SetNode("C", &NodeData{})
	g.SetNode("D", &NodeData{})
	g.SetEdgeWithName("A", "B", "ab", &EdgeData{})
	g.SetEdgeWithName("A", "C", "ac", &EdgeData{})
	g.SetEdgeWithName("B", "D", "bd", &EdgeData{})
	g.SetEdgeWithName("C", "D", "cd", &EdgeData{})

	networkSimplex(g)
	normalizeRanks(g)

	if g.Node("A").Rank != 0 || g.Node("B").Rank != 1 || g.Node("C").Rank != 1 || g.Node("D").Rank != 2 {
		t.Fatalf("unexpected ranks: A=%d B=%d C=%d D=%d", g.Node("A").Rank, g.Node("B").Rank, g.Node("C").Rank, g.Node("D").Rank)
	}
}

func TestNetworkSimplexRespectsMinLen(t *testing.T) {
	g := newRankTestGraph()
	g.SetNode("A", &NodeData{})
	g.SetNode("B", &NodeData{})
	g.SetNode("C", &NodeData{})
	g.SetEdgeWithName("A", "B", "ab", &EdgeData{MinLen: 2, Weight: 1})
	g.SetEdgeWithName("B", "C", "bc", &EdgeData{MinLen: 1, Weight: 1})
	g.SetEdgeWithName("A", "C", "ac", &EdgeData{MinLen: 1, Weight: 1})

	networkSimplex(g)

	for _, e := range g.Edges() {
		ed := g.EdgeWithName(e.V, e.W, e.Name)
		if ed == nil {
			t.Fatalf("missing edge label for %s->%s", e.V, e.W)
		}
		rv := g.Node(e.V).Rank
		rw := g.Node(e.W).Rank
		if rw-rv < ed.MinLen {
			t.Fatalf("edge %s->%s violated minlen: rank diff=%d minlen=%d", e.V, e.W, rw-rv, ed.MinLen)
		}
	}
}

func TestNetworkSimplexLowLimAndCutValuesInitialized(t *testing.T) {
	g := newRankTestGraph()
	g.SetNode("A", &NodeData{})
	g.SetNode("B", &NodeData{})
	g.SetNode("C", &NodeData{})
	g.SetEdgeWithName("A", "B", "ab", &EdgeData{})
	g.SetEdgeWithName("B", "C", "bc", &EdgeData{})

	simplified := simplify(g)
	longestPath(simplified)
	tree := buildFeasibleTree(simplified)
	if tree == nil {
		t.Fatalf("expected feasible tree")
	}

	initLowLimValues(tree, "")
	for _, v := range tree.Nodes() {
		n := tree.Node(v)
		if n == nil {
			t.Fatalf("missing tree node label for %s", v)
		}
		if n.Low <= 0 || n.Lim < n.Low {
			t.Fatalf("invalid low/lim for %s: low=%d lim=%d", v, n.Low, n.Lim)
		}
	}

	initCutValues(tree, simplified)
	if tree.EdgeCount() == 0 {
		t.Fatalf("expected tree edges")
	}

	for _, e := range tree.Edges() {
		ed := tree.EdgeWithName(e.V, e.W, e.Name)
		if ed == nil {
			t.Fatalf("missing tree edge label")
		}
		_ = ed.Cutvalue
	}
}

func TestRunRankDispatchesAlgorithms(t *testing.T) {
	g := newRankTestGraph()
	g.Ranker = "longest-path"
	g.SetNode("A", &NodeData{})
	g.SetNode("B", &NodeData{})
	g.SetEdgeWithName("A", "B", "ab", &EdgeData{})

	runRank(g)

	if g.Node("B").Rank-g.Node("A").Rank != 1 {
		t.Fatalf("expected runRank(longest-path) to assign rank diff 1, got %d", g.Node("B").Rank-g.Node("A").Rank)
	}
}
