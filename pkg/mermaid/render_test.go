package mermaid

import (
	"context"
	"strings"
	"testing"
)

func TestParseFlowchart(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  bool
		wantType DiagramType
		wantDir  Direction
		nodes    int
		edges    int
	}{
		{
			name:     "simple flowchart",
			input:    "flowchart TD\n  A-->B",
			wantType: DiagramFlowchart,
			wantDir:  DirectionTB,
			nodes:    2,
			edges:    1,
		},
		{
			name:     "flowchart with labels",
			input:    "flowchart LR\n  A[Start]-->B{Decision}\n  B-->|Yes|C[End]",
			wantType: DiagramFlowchart,
			wantDir:  DirectionLR,
			nodes:    3,
			edges:    2,
		},
		{
			name:     "graph keyword",
			input:    "graph TB\n  A-->B-->C",
			wantType: DiagramFlowchart,
			wantDir:  DirectionTB,
			nodes:    3,
			edges:    2,
		},
		{
			name:    "empty input",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid type",
			input:   "invalid\n  A-->B",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diagram, err := Parse(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if diagram.Type != tt.wantType {
				t.Errorf("Type = %v, want %v", diagram.Type, tt.wantType)
			}

			if diagram.Direction != tt.wantDir {
				t.Errorf("Direction = %v, want %v", diagram.Direction, tt.wantDir)
			}

			if len(diagram.Nodes) != tt.nodes {
				t.Errorf("Nodes = %d, want %d", len(diagram.Nodes), tt.nodes)
			}

			if len(diagram.Edges) != tt.edges {
				t.Errorf("Edges = %d, want %d", len(diagram.Edges), tt.edges)
			}
		})
	}
}

func TestParseEdgeLabels(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantLabel string
	}{
		{
			name:      "single word label",
			input:     "flowchart TD\n  A-->|Yes|B",
			wantLabel: "Yes",
		},
		{
			name:      "multi word label",
			input:     "flowchart TD\n  A-->|this is a label|B",
			wantLabel: "this is a label",
		},
		{
			name:      "label with numbers",
			input:     "flowchart TD\n  A-->|step 1|B",
			wantLabel: "step 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diagram, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			if len(diagram.Edges) != 1 {
				t.Fatalf("expected 1 edge, got %d", len(diagram.Edges))
			}

			if diagram.Edges[0].Label != tt.wantLabel {
				t.Errorf("Label = %q, want %q", diagram.Edges[0].Label, tt.wantLabel)
			}
		})
	}
}

func TestParseNodeShapes(t *testing.T) {
	tests := []struct {
		input     string
		nodeID    string
		wantShape NodeShape
		wantLabel string
	}{
		{"flowchart TD\n  A[Rectangle]", "A", ShapeRectangle, "Rectangle"},
		{"flowchart TD\n  A(Rounded)", "A", ShapeRounded, "Rounded"},
		{"flowchart TD\n  A{Diamond}", "A", ShapeRhombus, "Diamond"},
		{"flowchart TD\n  A((Circle))", "A", ShapeCircle, "Circle"},
		{"flowchart TD\n  A([Stadium])", "A", ShapeStadium, "Stadium"},
		{"flowchart TD\n  A[[Subroutine]]", "A", ShapeSubroutine, "Subroutine"},
		{"flowchart TD\n  A{{Hexagon}}", "A", ShapeHexagon, "Hexagon"},
		{"flowchart TD\n  A[(Cylinder)]", "A", ShapeCylinder, "Cylinder"},
	}

	for _, tt := range tests {
		t.Run(tt.wantShape.String(), func(t *testing.T) {
			diagram, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			var node *Node
			for _, n := range diagram.Nodes {
				if n.ID == tt.nodeID {
					node = n
					break
				}
			}

			if node == nil {
				t.Fatalf("node %q not found", tt.nodeID)
			}

			if node.Shape != tt.wantShape {
				t.Errorf("Shape = %v, want %v", node.Shape, tt.wantShape)
			}

			if node.Label != tt.wantLabel {
				t.Errorf("Label = %q, want %q", node.Label, tt.wantLabel)
			}
		})
	}
}

func TestParseSequenceDiagram(t *testing.T) {
	input := `sequenceDiagram
    participant A as Alice
    participant B as Bob
    A->>B: Hello
    B-->>A: Hi there
    Note right of A: This is a note`

	diagram, err := Parse(input)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if diagram.Type != DiagramSequence {
		t.Errorf("Type = %v, want DiagramSequence", diagram.Type)
	}

	if len(diagram.Participants) != 2 {
		t.Errorf("Participants = %d, want 2", len(diagram.Participants))
	}

	if len(diagram.Messages) != 2 {
		t.Errorf("Messages = %d, want 2", len(diagram.Messages))
	}

	if len(diagram.Notes) != 1 {
		t.Errorf("Notes = %d, want 1", len(diagram.Notes))
	}

	// Check message types
	if diagram.Messages[0].Type != MessageSolidArrow {
		t.Errorf("Message[0].Type = %v, want MessageSolidArrow", diagram.Messages[0].Type)
	}

	if diagram.Messages[1].Type != MessageDottedArrow {
		t.Errorf("Message[1].Type = %v, want MessageDottedArrow", diagram.Messages[1].Type)
	}
}

func TestParseClassDiagram(t *testing.T) {
	input := `classDiagram
    class Animal {
        +String name
        +int age
        +makeSound()
    }
    class Dog {
        +bark()
    }
    Dog <|-- Animal`

	diagram, err := Parse(input)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if diagram.Type != DiagramClass {
		t.Errorf("Type = %v, want DiagramClass", diagram.Type)
	}

	if len(diagram.Classes) < 2 {
		t.Errorf("Classes = %d, want >= 2", len(diagram.Classes))
	}

	// Find Animal class
	var animal *Class
	for _, c := range diagram.Classes {
		if c.Name == "Animal" {
			animal = c
			break
		}
	}

	if animal == nil {
		t.Fatal("Animal class not found")
	}

	if len(animal.Attributes) != 2 {
		t.Errorf("Animal.Attributes = %d, want 2", len(animal.Attributes))
	}

	if len(animal.Methods) != 1 {
		t.Errorf("Animal.Methods = %d, want 1", len(animal.Methods))
	}
}

func TestParseStateDiagram(t *testing.T) {
	input := `stateDiagram-v2
    [*] --> Idle
    Idle --> Processing : start
    Processing --> Done : complete
    Done --> [*]`

	diagram, err := Parse(input)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if diagram.Type != DiagramState {
		t.Errorf("Type = %v, want DiagramState", diagram.Type)
	}

	if len(diagram.States) < 3 {
		t.Errorf("States = %d, want >= 3", len(diagram.States))
	}

	if len(diagram.Transitions) != 4 {
		t.Errorf("Transitions = %d, want 4", len(diagram.Transitions))
	}
}

func TestParseERDiagram(t *testing.T) {
	input := `erDiagram
    CUSTOMER {
        int id PK
        string name
    }
    ORDER {
        int id PK
        int customerId FK
    }
    CUSTOMER ||--o{ ORDER : places`

	diagram, err := Parse(input)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if diagram.Type != DiagramER {
		t.Errorf("Type = %v, want DiagramER", diagram.Type)
	}

	if len(diagram.Entities) < 2 {
		t.Errorf("Entities = %d, want >= 2", len(diagram.Entities))
	}
}

func TestRenderFlowchart(t *testing.T) {
	input := `flowchart TD
    A[Start] --> B{Is it?}
    B -->|Yes| C[OK]
    B -->|No| D[End]`

	renderer := NewRenderer(nil)
	result, err := renderer.Render(context.Background(), input)

	if err != nil {
		t.Fatalf("render error: %v", err)
	}

	if result.SVG == "" {
		t.Error("SVG output is empty")
	}

	// Check for SVG structure
	if !strings.HasPrefix(result.SVG, "<svg") {
		t.Error("SVG doesn't start with <svg tag")
	}

	if !strings.HasSuffix(result.SVG, "</svg>") {
		t.Error("SVG doesn't end with </svg> tag")
	}

	// Check for nodes
	if !strings.Contains(result.SVG, "Start") {
		t.Error("SVG doesn't contain 'Start' label")
	}

	if result.Width <= 0 {
		t.Errorf("Width = %v, want > 0", result.Width)
	}

	if result.Height <= 0 {
		t.Errorf("Height = %v, want > 0", result.Height)
	}
}

func TestRenderSequenceDiagram(t *testing.T) {
	input := `sequenceDiagram
    Alice->>Bob: Hello Bob
    Bob-->>Alice: Hello Alice`

	renderer := NewRenderer(nil)
	result, err := renderer.Render(context.Background(), input)

	if err != nil {
		t.Fatalf("render error: %v", err)
	}

	// Check for participants
	if !strings.Contains(result.SVG, "Alice") {
		t.Error("SVG doesn't contain 'Alice' participant")
	}

	if !strings.Contains(result.SVG, "Bob") {
		t.Error("SVG doesn't contain 'Bob' participant")
	}

	// Check for lifelines (dashed lines)
	if !strings.Contains(result.SVG, "stroke-dasharray") {
		t.Error("SVG doesn't contain dashed lifelines")
	}
}

func TestRenderWithTheme(t *testing.T) {
	input := `flowchart TD
    A-->B`

	// Test with light theme
	lightRenderer := NewRenderer(&RenderOptions{Theme: LightTheme()})
	lightResult, err := lightRenderer.Render(context.Background(), input)
	if err != nil {
		t.Fatalf("render error with light theme: %v", err)
	}

	// Test with dark theme
	darkRenderer := NewRenderer(&RenderOptions{Theme: DefaultTheme()})
	darkResult, err := darkRenderer.Render(context.Background(), input)
	if err != nil {
		t.Fatalf("render error with dark theme: %v", err)
	}

	// SVGs should be different due to theme colors
	if lightResult.SVG == darkResult.SVG {
		t.Error("Light and dark theme produced identical SVG")
	}
}

func TestRenderEmptyInput(t *testing.T) {
	renderer := NewRenderer(nil)
	_, err := renderer.Render(context.Background(), "")

	if err != ErrEmptyDiagram {
		t.Errorf("expected ErrEmptyDiagram, got %v", err)
	}
}

func TestRenderUnsupportedType(t *testing.T) {
	renderer := NewRenderer(nil)
	_, err := renderer.Render(context.Background(), "pie\n    title Test")

	// Pie charts not fully supported yet
	if err == nil {
		t.Error("expected error for unsupported diagram type")
	}
}

func TestLayoutDagre(t *testing.T) {
	diagram := &Diagram{
		Type:      DiagramFlowchart,
		Direction: DirectionTB,
		Nodes: []*Node{
			{ID: "A", Label: "Node A"},
			{ID: "B", Label: "Node B"},
			{ID: "C", Label: "Node C"},
		},
		Edges: []*Edge{
			{From: "A", To: "B"},
			{From: "A", To: "C"},
		},
	}

	layout := NewDagreLayout(nil)
	err := layout.Layout(diagram)

	if err != nil {
		t.Fatalf("layout error: %v", err)
	}

	// Check that nodes have positions
	for _, node := range diagram.Nodes {
		if node.Position.X == 0 && node.Position.Y == 0 && node.ID != "A" {
			t.Errorf("Node %s has no position", node.ID)
		}
	}

	// Check that edges have points
	for _, edge := range diagram.Edges {
		if len(edge.Points) == 0 {
			t.Errorf("Edge %s->%s has no points", edge.From, edge.To)
		}
	}

	// Check that A is above B and C (TB direction)
	nodeMap := make(map[string]*Node)
	for _, n := range diagram.Nodes {
		nodeMap[n.ID] = n
	}

	if nodeMap["A"].Position.Y >= nodeMap["B"].Position.Y {
		t.Error("Node A should be above Node B in TB direction")
	}
}

func TestLayoutSequence(t *testing.T) {
	diagram := &Diagram{
		Type: DiagramSequence,
		Participants: []*Participant{
			{ID: "A", Alias: "Alice"},
			{ID: "B", Alias: "Bob"},
			{ID: "C", Alias: "Charlie"},
		},
		Messages: []*Message{
			{From: "A", To: "B", Label: "Hello"},
			{From: "B", To: "C", Label: "Hi"},
		},
	}

	layout := NewSequenceLayout(nil)
	err := layout.Layout(diagram)

	if err != nil {
		t.Fatalf("layout error: %v", err)
	}

	// Check that participants have X positions
	for _, p := range diagram.Participants {
		x := layout.GetParticipantX(p.ID)
		if x <= 0 {
			t.Errorf("Participant %s has invalid X position: %v", p.ID, x)
		}
	}

	// Check ordering: A < B < C
	xA := layout.GetParticipantX("A")
	xB := layout.GetParticipantX("B")
	xC := layout.GetParticipantX("C")

	if xA >= xB || xB >= xC {
		t.Errorf("Participants not ordered correctly: A=%v, B=%v, C=%v", xA, xB, xC)
	}
}

func TestLexer(t *testing.T) {
	input := `flowchart TD
    A[Start] --> B{Decision}
    B -->|Yes| C[End]`

	lexer := NewLexer(input)
	tokens := lexer.Tokenize()

	// Should have tokens for: flowchart, TD, newline, A, [Start], -->, B, {Decision}, etc.
	if len(tokens) < 10 {
		t.Errorf("expected at least 10 tokens, got %d", len(tokens))
	}

	// First token should be keyword
	if tokens[0].Type != TokenKeyword || tokens[0].Value != "flowchart" {
		t.Errorf("first token = %v %q, want keyword 'flowchart'", tokens[0].Type, tokens[0].Value)
	}

	// Check for direction token
	found := false
	for _, tok := range tokens {
		if tok.Type == TokenDirection && tok.Value == "TD" {
			found = true
			break
		}
	}
	if !found {
		t.Error("direction token TD not found")
	}
}

func TestExtractLabel(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"[Hello]", "Hello"},
		{"(World)", "World"},
		{"{Diamond}", "Diamond"},
		{"[[Subroutine]]", "Subroutine"},
		{"((Circle))", "Circle"},
		{"([Stadium])", "Stadium"},
		{">Asymmetric]", "Asymmetric"},
		{"{{Hexagon}}", "Hexagon"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := extractLabel(tt.input)
			if got != tt.want {
				t.Errorf("extractLabel(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestDetectShape(t *testing.T) {
	tests := []struct {
		input string
		want  NodeShape
	}{
		{"[Hello]", ShapeRectangle},
		{"(World)", ShapeRounded},
		{"{Diamond}", ShapeRhombus},
		{"((Circle))", ShapeCircle},
		{"([Stadium])", ShapeStadium},
		{"[[Subroutine]]", ShapeSubroutine},
		{"{{Hexagon}}", ShapeHexagon},
		{"[(Cylinder)]", ShapeCylinder},
		{">Asymmetric]", ShapeAsymmetric},
		{"(((Double)))", ShapeDoubleCircle},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := detectShape(tt.input)
			if got != tt.want {
				t.Errorf("detectShape(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestSVGWriter(t *testing.T) {
	writer := NewSVGWriter(400, 300, nil)
	writer.Start()

	// Draw a simple node
	node := &Node{
		ID:       "test",
		Label:    "Test Node",
		Shape:    ShapeRectangle,
		Position: Position{X: 50, Y: 50},
		Size:     Size{Width: 100, Height: 40},
	}
	writer.DrawNode(node)

	// Draw a simple edge
	edge := &Edge{
		From:     "test",
		To:       "other",
		ArrowEnd: ArrowNormal,
		Points: []Position{
			{X: 150, Y: 70},
			{X: 250, Y: 70},
		},
	}
	writer.DrawEdge(edge)

	svg := writer.End()

	// Validate SVG output
	if !strings.Contains(svg, "<svg") {
		t.Error("missing svg element")
	}

	if !strings.Contains(svg, "Test Node") {
		t.Error("missing node label")
	}

	if !strings.Contains(svg, "<path") {
		t.Error("missing edge path")
	}

	if !strings.Contains(svg, "<marker") {
		t.Error("missing arrow marker definition")
	}
}

func BenchmarkParseFlowchart(b *testing.B) {
	input := `flowchart TD
    A[Start] --> B{Decision}
    B -->|Yes| C[Process 1]
    B -->|No| D[Process 2]
    C --> E[End]
    D --> E`

	b.ResetTimer()
	for b.Loop() {
		_, err := Parse(input)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRenderFlowchart(b *testing.B) {
	input := `flowchart TD
    A[Start] --> B{Decision}
    B -->|Yes| C[Process 1]
    B -->|No| D[Process 2]
    C --> E[End]
    D --> E`

	renderer := NewRenderer(nil)
	ctx := context.Background()

	b.ResetTimer()
	for b.Loop() {
		_, err := renderer.Render(ctx, input)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLayoutDagre(b *testing.B) {
	diagram := &Diagram{
		Type:      DiagramFlowchart,
		Direction: DirectionTB,
	}

	// Create a moderately complex graph
	for i := 0; i < 20; i++ {
		diagram.Nodes = append(diagram.Nodes, &Node{
			ID:    string(rune('A' + i)),
			Label: string(rune('A' + i)),
		})
	}

	// Add edges
	for i := 0; i < 19; i++ {
		diagram.Edges = append(diagram.Edges, &Edge{
			From: string(rune('A' + i)),
			To:   string(rune('A' + i + 1)),
		})
	}

	layout := NewDagreLayout(nil)

	b.ResetTimer()
	for b.Loop() {
		// Reset positions
		for _, n := range diagram.Nodes {
			n.Position = Position{}
			n.Size = Size{}
		}
		for _, e := range diagram.Edges {
			e.Points = nil
		}

		err := layout.Layout(diagram)
		if err != nil {
			b.Fatal(err)
		}
	}
}
