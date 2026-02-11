package dagre

import (
	"testing"
)

func TestNewGraph(t *testing.T) {
	g := NewGraph()

	if g == nil {
		t.Fatal("NewGraph returned nil")
	}
	if g.NodeCount() != 0 {
		t.Errorf("expected 0 nodes, got %d", g.NodeCount())
	}
	if g.EdgeCount() != 0 {
		t.Errorf("expected 0 edges, got %d", g.EdgeCount())
	}
}

func TestGraphNodes(t *testing.T) {
	g := NewGraph()

	// Add nodes
	g.SetNode("a", &NodeData{Label: "A", Width: 100, Height: 50})
	g.SetNode("b", &NodeData{Label: "B", Width: 100, Height: 50})
	g.SetNode("c", &NodeData{Label: "C", Width: 100, Height: 50})

	if g.NodeCount() != 3 {
		t.Errorf("expected 3 nodes, got %d", g.NodeCount())
	}

	// Check node exists
	if !g.HasNode("a") {
		t.Error("expected node 'a' to exist")
	}
	if g.HasNode("d") {
		t.Error("expected node 'd' to not exist")
	}

	// Check node data
	nodeA := g.Node("a")
	if nodeA == nil {
		t.Fatal("expected node 'a' data")
	}
	if nodeA.Label != "A" {
		t.Errorf("expected label 'A', got %v", nodeA.Label)
	}

	// Remove node
	g.RemoveNode("a")
	if g.HasNode("a") {
		t.Error("expected node 'a' to be removed")
	}
	if g.NodeCount() != 2 {
		t.Errorf("expected 2 nodes after removal, got %d", g.NodeCount())
	}
}

func TestGraphEdges(t *testing.T) {
	g := NewGraph()

	g.SetNode("a", nil)
	g.SetNode("b", nil)
	g.SetNode("c", nil)

	// Add edges
	g.SetEdge("a", "b", &EdgeData{Weight: 1})
	g.SetEdge("b", "c", &EdgeData{Weight: 2})
	g.SetEdge("a", "c", &EdgeData{Weight: 3})

	if g.EdgeCount() != 3 {
		t.Errorf("expected 3 edges, got %d", g.EdgeCount())
	}

	// Check edge exists
	if !g.HasEdge("a", "b") {
		t.Error("expected edge a->b to exist")
	}
	if g.HasEdge("b", "a") {
		t.Error("expected edge b->a to not exist (directed)")
	}

	// Check edge data
	edgeAB := g.Edge("a", "b")
	if edgeAB == nil {
		t.Fatal("expected edge a->b data")
	}
	if edgeAB.Weight != 1 {
		t.Errorf("expected weight 1, got %v", edgeAB.Weight)
	}

	// Remove edge
	g.RemoveEdge("a", "b")
	if g.HasEdge("a", "b") {
		t.Error("expected edge a->b to be removed")
	}
	if g.EdgeCount() != 2 {
		t.Errorf("expected 2 edges after removal, got %d", g.EdgeCount())
	}
}

func TestGraphAdjacency(t *testing.T) {
	g := NewGraph()

	g.SetNode("a", nil)
	g.SetNode("b", nil)
	g.SetNode("c", nil)

	g.SetEdge("a", "b", nil)
	g.SetEdge("a", "c", nil)
	g.SetEdge("b", "c", nil)

	// Test successors
	succs := g.Successors("a")
	if len(succs) != 2 {
		t.Errorf("expected 2 successors of 'a', got %d", len(succs))
	}

	// Test predecessors
	preds := g.Predecessors("c")
	if len(preds) != 2 {
		t.Errorf("expected 2 predecessors of 'c', got %d", len(preds))
	}

	// Test degree
	if g.OutDegree("a") != 2 {
		t.Errorf("expected out-degree 2 for 'a', got %d", g.OutDegree("a"))
	}
	if g.InDegree("c") != 2 {
		t.Errorf("expected in-degree 2 for 'c', got %d", g.InDegree("c"))
	}

	// Test sources and sinks
	sources := g.Sources()
	if len(sources) != 1 || sources[0] != "a" {
		t.Errorf("expected sources ['a'], got %v", sources)
	}

	sinks := g.Sinks()
	if len(sinks) != 1 || sinks[0] != "c" {
		t.Errorf("expected sinks ['c'], got %v", sinks)
	}
}

func TestCompoundGraph(t *testing.T) {
	g := NewGraph()

	g.SetNode("a", nil)
	g.SetNode("b", nil)
	g.SetNode("c", nil)
	g.SetNode("sg", nil) // subgraph

	// Set parent relationships
	if err := g.SetParent("a", "sg"); err != nil {
		t.Errorf("unexpected error setting parent: %v", err)
	}
	if err := g.SetParent("b", "sg"); err != nil {
		t.Errorf("unexpected error setting parent: %v", err)
	}

	// Check parent
	if g.Parent("a") != "sg" {
		t.Errorf("expected parent 'sg' for 'a', got %q", g.Parent("a"))
	}

	// Check children
	children := g.Children("sg")
	if len(children) != 2 {
		t.Errorf("expected 2 children of 'sg', got %d", len(children))
	}

	// Check compound detection
	if !g.IsCompound() {
		t.Error("expected graph to be compound")
	}

	// Root children
	rootChildren := g.Children("")
	if len(rootChildren) != 2 { // sg and c
		t.Errorf("expected 2 root children, got %d: %v", len(rootChildren), rootChildren)
	}
}

func TestLayoutSimple(t *testing.T) {
	g := NewGraph()

	g.SetNode("a", &NodeData{Label: "A", Width: 100, Height: 40})
	g.SetNode("b", &NodeData{Label: "B", Width: 100, Height: 40})
	g.SetNode("c", &NodeData{Label: "C", Width: 100, Height: 40})

	g.SetEdge("a", "b", nil)
	g.SetEdge("b", "c", nil)

	if err := Layout(g, nil); err != nil {
		t.Fatalf("Layout() error: %v", err)
	}

	// Check that positions were assigned
	nodeA := g.Node("a")
	nodeB := g.Node("b")
	nodeC := g.Node("c")

	if nodeA.X == 0 && nodeA.Y == 0 && nodeB.X == 0 && nodeB.Y == 0 {
		t.Error("expected positions to be assigned")
	}

	// Check rank order (a should be before b, b before c)
	if nodeA.Rank >= nodeB.Rank {
		t.Errorf("expected rank(a) < rank(b), got %d >= %d", nodeA.Rank, nodeB.Rank)
	}
	if nodeB.Rank >= nodeC.Rank {
		t.Errorf("expected rank(b) < rank(c), got %d >= %d", nodeB.Rank, nodeC.Rank)
	}
}

func TestLayoutDirection(t *testing.T) {
	tests := []struct {
		name    string
		rankDir string
	}{
		{"TB", "TB"},
		{"BT", "BT"},
		{"LR", "LR"},
		{"RL", "RL"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewGraph()

			g.SetNode("a", &NodeData{Width: 100, Height: 40})
			g.SetNode("b", &NodeData{Width: 100, Height: 40})

			g.SetEdge("a", "b", nil)

			opts := &LayoutOptions{
				RankDir: tt.rankDir,
				NodeSep: 50,
				RankSep: 50,
				MarginX: 20,
				MarginY: 20,
			}

			if err := Layout(g, opts); err != nil {
				t.Fatalf("Layout() error: %v", err)
			}

			nodeA := g.Node("a")
			nodeB := g.Node("b")

			// Verify direction
			switch tt.rankDir {
			case "TB":
				if nodeA.Y >= nodeB.Y {
					t.Errorf("TB: expected a.Y < b.Y, got %v >= %v", nodeA.Y, nodeB.Y)
				}
			case "BT":
				if nodeA.Y <= nodeB.Y {
					t.Errorf("BT: expected a.Y > b.Y, got %v <= %v", nodeA.Y, nodeB.Y)
				}
			case "LR":
				if nodeA.X >= nodeB.X {
					t.Errorf("LR: expected a.X < b.X, got %v >= %v", nodeA.X, nodeB.X)
				}
			case "RL":
				if nodeA.X <= nodeB.X {
					t.Errorf("RL: expected a.X > b.X, got %v <= %v", nodeA.X, nodeB.X)
				}
			}
		})
	}
}

func TestLayoutCycle(t *testing.T) {
	g := NewGraph()

	g.SetNode("a", &NodeData{Width: 100, Height: 40})
	g.SetNode("b", &NodeData{Width: 100, Height: 40})
	g.SetNode("c", &NodeData{Width: 100, Height: 40})

	// Create a cycle: a -> b -> c -> a
	g.SetEdge("a", "b", nil)
	g.SetEdge("b", "c", nil)
	g.SetEdge("c", "a", nil)

	// Layout should handle cycles without crashing
	if err := Layout(g, nil); err != nil {
		t.Fatalf("Layout() error: %v", err)
	}

	// Check that positions were assigned
	nodeA := g.Node("a")
	nodeB := g.Node("b")
	nodeC := g.Node("c")

	if nodeA.X == 0 && nodeA.Y == 0 && nodeB.X == 0 && nodeB.Y == 0 && nodeC.X == 0 && nodeC.Y == 0 {
		t.Error("expected positions to be assigned even with cycle")
	}
}

func TestLayoutDiamond(t *testing.T) {
	g := NewGraph()

	// Diamond pattern: a -> b, a -> c, b -> d, c -> d
	g.SetNode("a", &NodeData{Width: 100, Height: 40})
	g.SetNode("b", &NodeData{Width: 100, Height: 40})
	g.SetNode("c", &NodeData{Width: 100, Height: 40})
	g.SetNode("d", &NodeData{Width: 100, Height: 40})

	g.SetEdge("a", "b", nil)
	g.SetEdge("a", "c", nil)
	g.SetEdge("b", "d", nil)
	g.SetEdge("c", "d", nil)

	if err := Layout(g, nil); err != nil {
		t.Fatalf("Layout() error: %v", err)
	}

	nodeA := g.Node("a")
	nodeB := g.Node("b")
	nodeC := g.Node("c")
	nodeD := g.Node("d")

	// Check rank structure
	if nodeA.Rank != 0 {
		t.Errorf("expected a at rank 0, got %d", nodeA.Rank)
	}
	if nodeB.Rank != 1 || nodeC.Rank != 1 {
		t.Errorf("expected b and c at rank 1, got b=%d, c=%d", nodeB.Rank, nodeC.Rank)
	}
	if nodeD.Rank != 2 {
		t.Errorf("expected d at rank 2, got %d", nodeD.Rank)
	}

	// b and c should be at same Y (TB layout)
	if abs(nodeB.Y-nodeC.Y) > 1 {
		t.Errorf("expected b and c at same Y, got b.Y=%v, c.Y=%v", nodeB.Y, nodeC.Y)
	}
}

func TestLayoutLongEdge(t *testing.T) {
	g := NewGraph()

	// Chain with a long edge: a -> b -> c, a -> c (skips rank)
	g.SetNode("a", &NodeData{Width: 100, Height: 40})
	g.SetNode("b", &NodeData{Width: 100, Height: 40})
	g.SetNode("c", &NodeData{Width: 100, Height: 40})

	g.SetEdge("a", "b", nil)
	g.SetEdge("b", "c", nil)
	g.SetEdge("a", "c", nil) // Long edge

	if err := Layout(g, nil); err != nil {
		t.Fatalf("Layout() error: %v", err)
	}

	nodeA := g.Node("a")
	nodeC := g.Node("c")

	// a->c edge should have points
	edgeAC := g.Edge("a", "c")
	if edgeAC == nil {
		t.Fatal("expected edge a->c")
	}

	// Long edge should have routing points
	// After denormalization, edge points should include intermediate positions
	t.Logf("Edge a->c has %d points", len(edgeAC.Points))

	// Verify basic structure
	if nodeA.Rank >= nodeC.Rank {
		t.Errorf("expected a before c, got ranks a=%d, c=%d", nodeA.Rank, nodeC.Rank)
	}
}

func TestGraphCopy(t *testing.T) {
	g := NewGraph()

	g.SetNode("a", &NodeData{Label: "A", Width: 100, Height: 40})
	g.SetNode("b", &NodeData{Label: "B", Width: 100, Height: 40})
	g.SetEdge("a", "b", &EdgeData{Weight: 5})

	// Copy
	g2 := g.Copy()

	// Modify original
	g.Node("a").Label = "Modified"
	g.RemoveNode("b")

	// Check copy is independent
	if g2.Node("a").Label != "A" {
		t.Errorf("copy should be independent, got label %v", g2.Node("a").Label)
	}
	if !g2.HasNode("b") {
		t.Error("copy should still have node b")
	}
	if g2.NodeCount() != 2 {
		t.Errorf("copy should have 2 nodes, got %d", g2.NodeCount())
	}
}

func TestMultiGraph(t *testing.T) {
	g := NewMultiGraph()

	g.SetNode("a", nil)
	g.SetNode("b", nil)

	// Add multiple edges between same nodes
	g.SetEdgeWithName("a", "b", "e1", &EdgeData{Weight: 1})
	g.SetEdgeWithName("a", "b", "e2", &EdgeData{Weight: 2})

	if g.EdgeCount() != 2 {
		t.Errorf("expected 2 edges, got %d", g.EdgeCount())
	}

	e1 := g.EdgeWithName("a", "b", "e1")
	e2 := g.EdgeWithName("a", "b", "e2")

	if e1 == nil || e2 == nil {
		t.Fatal("expected both edges to exist")
	}
	if e1.Weight != 1 || e2.Weight != 2 {
		t.Errorf("expected weights 1 and 2, got %v and %v", e1.Weight, e2.Weight)
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// Additional f-test style tests for better error locations

func TestGraphBasicOperations(t *testing.T) {
	// Test SetNode and Node operations
	fSetNode := func(g *Graph, id string, width, height float64) {
		t.Helper()
		g.SetNode(id, &NodeData{Width: width, Height: height})
		if !g.HasNode(id) {
			t.Fatalf("SetNode(%q) failed: node not found", id)
		}
		n := g.Node(id)
		if n.Width != width || n.Height != height {
			t.Fatalf("SetNode(%q) = {Width: %v, Height: %v}, expected {Width: %v, Height: %v}",
				id, n.Width, n.Height, width, height)
		}
	}

	g := NewGraph()
	fSetNode(g, "a", 100, 50)
	fSetNode(g, "b", 200, 75)
	fSetNode(g, "c", 150, 60)

	// Test SetEdge operations
	fSetEdge := func(v, w string, weight float64) {
		t.Helper()
		g.SetEdge(v, w, &EdgeData{Weight: weight})
		if !g.HasEdge(v, w) {
			t.Fatalf("SetEdge(%q, %q) failed: edge not found", v, w)
		}
		e := g.Edge(v, w)
		if e.Weight != weight {
			t.Fatalf("Edge(%q, %q).Weight = %v, expected %v", v, w, e.Weight, weight)
		}
	}

	fSetEdge("a", "b", 1.0)
	fSetEdge("b", "c", 2.0)
	fSetEdge("a", "c", 3.0)
}

func TestLayoutRanks(t *testing.T) {
	// f-test for verifying rank assignments
	fRank := func(g *Graph, node string, expectedRank int) {
		t.Helper()
		n := g.Node(node)
		if n == nil {
			t.Fatalf("Node(%q) = nil", node)
		}
		if n.Rank != expectedRank {
			t.Fatalf("Node(%q).Rank = %d, expected %d", node, n.Rank, expectedRank)
		}
	}

	g := NewGraph()
	g.SetNode("a", &NodeData{Width: 100, Height: 40})
	g.SetNode("b", &NodeData{Width: 100, Height: 40})
	g.SetNode("c", &NodeData{Width: 100, Height: 40})
	g.SetNode("d", &NodeData{Width: 100, Height: 40})

	g.SetEdge("a", "b", nil)
	g.SetEdge("a", "c", nil)
	g.SetEdge("b", "d", nil)
	g.SetEdge("c", "d", nil)

	if err := Layout(g, nil); err != nil {
		t.Fatalf("Layout() error: %v", err)
	}

	fRank(g, "a", 0)
	fRank(g, "b", 1)
	fRank(g, "c", 1)
	fRank(g, "d", 2)
}

func TestIterators(t *testing.T) {
	g := NewGraph()
	g.SetNode("a", &NodeData{Width: 100, Height: 40})
	g.SetNode("b", &NodeData{Width: 100, Height: 40})
	g.SetNode("c", &NodeData{Width: 100, Height: 40})

	g.SetEdge("a", "b", &EdgeData{Weight: 1})
	g.SetEdge("b", "c", &EdgeData{Weight: 2})

	// Test NodesIter
	nodeCount := 0
	for range g.NodesIter() {
		nodeCount++
	}
	if nodeCount != 3 {
		t.Errorf("NodesIter: expected 3 nodes, got %d", nodeCount)
	}

	// Test NodesDataIter
	nodeDataCount := 0
	for id, data := range g.NodesDataIter() {
		nodeDataCount++
		if data == nil {
			t.Errorf("NodesDataIter: nil data for node %q", id)
		}
	}
	if nodeDataCount != 3 {
		t.Errorf("NodesDataIter: expected 3 nodes, got %d", nodeDataCount)
	}

	// Test EdgesIter
	edgeCount := 0
	for range g.EdgesIter() {
		edgeCount++
	}
	if edgeCount != 2 {
		t.Errorf("EdgesIter: expected 2 edges, got %d", edgeCount)
	}

	// Test EdgesDataIter
	edgeDataCount := 0
	for e, data := range g.EdgesDataIter() {
		edgeDataCount++
		if data == nil {
			t.Errorf("EdgesDataIter: nil data for edge %v->%v", e.V, e.W)
		}
	}
	if edgeDataCount != 2 {
		t.Errorf("EdgesDataIter: expected 2 edges, got %d", edgeDataCount)
	}

	// Test early termination
	earlyCount := 0
	for range g.NodesIter() {
		earlyCount++
		if earlyCount == 2 {
			break
		}
	}
	if earlyCount != 2 {
		t.Errorf("NodesIter early termination: expected 2, got %d", earlyCount)
	}
}
