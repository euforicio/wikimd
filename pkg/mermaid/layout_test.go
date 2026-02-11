package mermaid

import "testing"

func TestDagreLayout_SubgraphCompoundBoundsAndHierarchy(t *testing.T) {
	diagram, err := Parse(`flowchart TB
    subgraph outer
        subgraph inner
            A --> B
        end
        C --> D
    end
    outer --> E`)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	engine := NewDagreLayout(nil)
	if err := engine.Layout(diagram); err != nil {
		t.Fatalf("Layout() error: %v", err)
	}

	outer := findSubgraph(diagram, "outer")
	if outer == nil {
		t.Fatal("outer subgraph not found")
	}
	inner := findSubgraph(diagram, "inner")
	if inner == nil {
		t.Fatal("inner subgraph not found")
	}
	if inner.ParentID != "outer" {
		t.Fatalf("inner.ParentID = %q, want %q", inner.ParentID, "outer")
	}

	if outer.Size.Width <= 0 || outer.Size.Height <= 0 {
		t.Fatalf("outer subgraph size invalid: %+v", outer.Size)
	}
	if inner.Size.Width <= 0 || inner.Size.Height <= 0 {
		t.Fatalf("inner subgraph size invalid: %+v", inner.Size)
	}

	if intersects(outer.Position, outer.Size, inner.Position, inner.Size) {
		if !(inner.Position.X >= outer.Position.X &&
			inner.Position.Y >= outer.Position.Y &&
			inner.Position.X+inner.Size.Width <= outer.Position.X+outer.Size.Width &&
			inner.Position.Y+inner.Size.Height <= outer.Position.Y+outer.Size.Height) {
			t.Fatal("inner subgraph should be fully contained within outer bounds")
		}
	}
}

func TestDagreLayout_EdgeLabelCoordinates(t *testing.T) {
	diagram, err := Parse(`flowchart LR
    A -->|edge label| B`)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	engine := NewDagreLayout(nil)
	if err := engine.Layout(diagram); err != nil {
		t.Fatalf("Layout() error: %v", err)
	}

	if len(diagram.Edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(diagram.Edges))
	}
	edge := diagram.Edges[0]
	if !edge.HasLabelPosition {
		t.Fatal("expected explicit edge label coordinates")
	}
}

func intersects(p1 Position, s1 Size, p2 Position, s2 Size) bool {
	return p1.X < p2.X+s2.Width &&
		p1.X+s1.Width > p2.X &&
		p1.Y < p2.Y+s2.Height &&
		p1.Y+s1.Height > p2.Y
}
