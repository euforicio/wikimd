package mermaid

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// =============================================================================
// Test Fixtures
// =============================================================================

// generateFlowchartSmall creates a small flowchart with 3 nodes.
func generateFlowchartSmall() string {
	return `flowchart TD
    A[Start] --> B{Decision}
    B --> C[End]`
}

// generateFlowchartMedium creates a medium flowchart with 20 nodes.
func generateFlowchartMedium() string {
	var sb strings.Builder
	sb.WriteString("flowchart TD\n")

	for i := 0; i < 20; i++ {
		nodeID := fmt.Sprintf("N%d", i)
		sb.WriteString(fmt.Sprintf("    %s[Node %d]\n", nodeID, i))
	}

	// Create a branching structure
	for i := 0; i < 19; i++ {
		sb.WriteString(fmt.Sprintf("    N%d --> N%d\n", i, i+1))
	}

	// Add some cross-connections for complexity
	sb.WriteString("    N5 --> N10\n")
	sb.WriteString("    N10 --> N15\n")
	sb.WriteString("    N3 --> N8\n")
	sb.WriteString("    N12 --> N18\n")

	return sb.String()
}

// generateFlowchartLarge creates a large flowchart with 100 nodes.
func generateFlowchartLarge() string {
	var sb strings.Builder
	sb.WriteString("flowchart TD\n")

	for i := 0; i < 100; i++ {
		nodeID := fmt.Sprintf("N%d", i)
		shape := "[%s]"
		switch i % 4 {
		case 1:
			shape = "(%s)"
		case 2:
			shape = "{%s}"
		case 3:
			shape = "([%s])"
		}
		sb.WriteString(fmt.Sprintf("    %s"+shape+"\n", nodeID, fmt.Sprintf("Node %d", i)))
	}

	// Create a tree-like structure with multiple branches
	for i := 0; i < 99; i++ {
		sb.WriteString(fmt.Sprintf("    N%d --> N%d\n", i, i+1))
	}

	// Add branch points for complexity
	for i := 0; i < 90; i += 10 {
		sb.WriteString(fmt.Sprintf("    N%d --> N%d\n", i, i+5))
		if i+15 < 100 {
			sb.WriteString(fmt.Sprintf("    N%d --> N%d\n", i, i+15))
		}
	}

	return sb.String()
}

// generateSequenceDiagramComplex creates a complex sequence diagram.
// 10 participants, 50 messages
func generateSequenceDiagramComplex() string {
	var sb strings.Builder
	sb.WriteString("sequenceDiagram\n")

	// Define 10 participants
	participants := []string{"Client", "Gateway", "Auth", "UserSvc", "OrderSvc", "ProductSvc", "PaymentSvc", "NotifySvc", "CacheSvc", "Database"}
	for _, p := range participants {
		sb.WriteString(fmt.Sprintf("    participant %s\n", p))
	}
	sb.WriteString("\n")

	// Generate 50 messages creating a realistic microservice flow
	messages := []struct {
		from, to, msg string
		arrow         string
	}{
		{"Client", "Gateway", "HTTP Request", "->>"},
		{"Gateway", "Auth", "Validate Token", "->>"},
		{"Auth", "CacheSvc", "Check Cache", "->>"},
		{"CacheSvc", "Auth", "Cache Miss", "-->>"},
		{"Auth", "Database", "Query User", "->>"},
		{"Database", "Auth", "User Data", "-->>"},
		{"Auth", "CacheSvc", "Store Cache", "->>"},
		{"Auth", "Gateway", "Token Valid", "-->>"},
		{"Gateway", "UserSvc", "Get Profile", "->>"},
		{"UserSvc", "Database", "Query Profile", "->>"},
		{"Database", "UserSvc", "Profile Data", "-->>"},
		{"UserSvc", "Gateway", "Profile", "-->>"},
		{"Gateway", "OrderSvc", "Create Order", "->>"},
		{"OrderSvc", "ProductSvc", "Check Stock", "->>"},
		{"ProductSvc", "CacheSvc", "Get Stock", "->>"},
		{"CacheSvc", "ProductSvc", "Stock Level", "-->>"},
		{"ProductSvc", "OrderSvc", "Stock OK", "-->>"},
		{"OrderSvc", "PaymentSvc", "Process Payment", "->>"},
		{"PaymentSvc", "Database", "Store Transaction", "->>"},
		{"Database", "PaymentSvc", "Stored", "-->>"},
		{"PaymentSvc", "OrderSvc", "Payment OK", "-->>"},
		{"OrderSvc", "Database", "Store Order", "->>"},
		{"Database", "OrderSvc", "Stored", "-->>"},
		{"OrderSvc", "NotifySvc", "Send Confirmation", "->>"},
		{"NotifySvc", "OrderSvc", "Sent", "-->>"},
		{"OrderSvc", "Gateway", "Order Created", "-->>"},
		{"Gateway", "Client", "HTTP Response", "-->>"},
		{"Client", "Gateway", "Get Orders", "->>"},
		{"Gateway", "Auth", "Validate", "->>"},
		{"Auth", "Gateway", "Valid", "-->>"},
		{"Gateway", "OrderSvc", "List Orders", "->>"},
		{"OrderSvc", "Database", "Query Orders", "->>"},
		{"Database", "OrderSvc", "Orders", "-->>"},
		{"OrderSvc", "Gateway", "Orders List", "-->>"},
		{"Gateway", "Client", "Orders", "-->>"},
		{"Client", "Gateway", "Cancel Order", "->>"},
		{"Gateway", "OrderSvc", "Cancel", "->>"},
		{"OrderSvc", "PaymentSvc", "Refund", "->>"},
		{"PaymentSvc", "Database", "Update", "->>"},
		{"Database", "PaymentSvc", "Done", "-->>"},
		{"PaymentSvc", "OrderSvc", "Refunded", "-->>"},
		{"OrderSvc", "Database", "Update Status", "->>"},
		{"Database", "OrderSvc", "Updated", "-->>"},
		{"OrderSvc", "NotifySvc", "Send Cancel Notice", "->>"},
		{"NotifySvc", "OrderSvc", "Sent", "-->>"},
		{"OrderSvc", "Gateway", "Canceled", "-->>"},
		{"Gateway", "Client", "Success", "-->>"},
		{"Client", "Gateway", "Logout", "->>"},
		{"Gateway", "Auth", "Invalidate", "->>"},
		{"Auth", "CacheSvc", "Clear", "->>"},
	}

	for _, m := range messages {
		sb.WriteString(fmt.Sprintf("    %s%s%s: %s\n", m.from, m.arrow, m.to, m.msg))
	}

	return sb.String()
}

// generateClassDiagramLarge creates a large class diagram.
// 20 classes with relationships
func generateClassDiagramLarge() string {
	var sb strings.Builder
	sb.WriteString("classDiagram\n")

	classes := []struct {
		name       string
		annotation string
		attrs      []string
		methods    []string
	}{
		{"User", "", []string{"+int id", "+string name", "+string email", "-string password"}, []string{"+login()", "+logout()", "+updateProfile()"}},
		{"Admin", "", []string{"+int level"}, []string{"+manageUsers()", "+viewLogs()"}},
		{"Customer", "", []string{"+string address", "+string phone"}, []string{"+placeOrder()", "+viewOrders()"}},
		{"Order", "", []string{"+int id", "+date createdAt", "+string status"}, []string{"+calculate()", "+ship()", "+cancel()"}},
		{"OrderItem", "", []string{"+int quantity", "+float price"}, []string{"+getSubtotal()"}},
		{"Product", "", []string{"+int id", "+string name", "+float price", "+int stock"}, []string{"+updateStock()", "+getDetails()"}},
		{"Category", "", []string{"+int id", "+string name"}, []string{"+getProducts()"}},
		{"Payment", "", []string{"+int id", "+float amount", "+string method"}, []string{"+process()", "+refund()"}},
		{"PaymentMethod", "<<interface>>", []string{}, []string{"+charge()", "+validate()"}},
		{"CreditCard", "", []string{"+string number", "+string expiry"}, []string{"+charge()", "+validate()"}},
		{"PayPal", "", []string{"+string email"}, []string{"+charge()", "+validate()"}},
		{"Notification", "", []string{"+int id", "+string message", "+date sentAt"}, []string{"+send()"}},
		{"EmailNotification", "", []string{"+string subject"}, []string{"+send()"}},
		{"SMSNotification", "", []string{"+string phoneNumber"}, []string{"+send()"}},
		{"Inventory", "", []string{"+int warehouseId"}, []string{"+checkStock()", "+reserve()", "+release()"}},
		{"Warehouse", "", []string{"+int id", "+string location"}, []string{"+getInventory()"}},
		{"Shipping", "", []string{"+string trackingNumber", "+string carrier"}, []string{"+track()", "+updateStatus()"}},
		{"Address", "", []string{"+string street", "+string city", "+string zip"}, []string{"+validate()", "+format()"}},
		{"Review", "", []string{"+int rating", "+string comment"}, []string{"+submit()", "+moderate()"}},
		{"Discount", "", []string{"+float percentage", "+date validUntil"}, []string{"+apply()", "+isValid()"}},
	}

	for _, c := range classes {
		if c.annotation != "" {
			sb.WriteString(fmt.Sprintf("    class %s {\n        %s\n", c.name, c.annotation))
		} else {
			sb.WriteString(fmt.Sprintf("    class %s {\n", c.name))
		}
		for _, attr := range c.attrs {
			sb.WriteString(fmt.Sprintf("        %s\n", attr))
		}
		for _, method := range c.methods {
			sb.WriteString(fmt.Sprintf("        %s\n", method))
		}
		sb.WriteString("    }\n")
	}

	// Define relationships
	relationships := []string{
		"User <|-- Admin",
		"User <|-- Customer",
		"Customer \"1\" --> \"*\" Order",
		"Order \"1\" --> \"*\" OrderItem",
		"OrderItem \"*\" --> \"1\" Product",
		"Product \"*\" --> \"1\" Category",
		"Order \"1\" --> \"1\" Payment",
		"PaymentMethod <|.. CreditCard",
		"PaymentMethod <|.. PayPal",
		"Payment --> PaymentMethod",
		"Notification <|-- EmailNotification",
		"Notification <|-- SMSNotification",
		"Product \"*\" --> \"*\" Inventory",
		"Inventory \"*\" --> \"1\" Warehouse",
		"Order \"1\" --> \"0..1\" Shipping",
		"Customer \"1\" --> \"*\" Address",
		"Order \"1\" --> \"1\" Address",
		"Product \"1\" --> \"*\" Review",
		"Customer \"1\" --> \"*\" Review",
		"Order \"*\" --> \"0..1\" Discount",
	}

	sb.WriteString("\n")
	for _, rel := range relationships {
		sb.WriteString(fmt.Sprintf("    %s\n", rel))
	}

	return sb.String()
}

// =============================================================================
// Benchmark Fixtures - Pre-parsed for component benchmarks
// =============================================================================

func createFlowchartDiagramSmall() *Diagram {
	return &Diagram{
		Type:      DiagramFlowchart,
		Direction: DirectionTB,
		Nodes: []*Node{
			{ID: "A", Label: "Start", Shape: ShapeRectangle},
			{ID: "B", Label: "Decision", Shape: ShapeRhombus},
			{ID: "C", Label: "End", Shape: ShapeRectangle},
		},
		Edges: []*Edge{
			{From: "A", To: "B", ArrowEnd: ArrowNormal},
			{From: "B", To: "C", ArrowEnd: ArrowNormal},
		},
	}
}

func createFlowchartDiagramMedium() *Diagram {
	diagram := &Diagram{
		Type:      DiagramFlowchart,
		Direction: DirectionTB,
	}

	for i := 0; i < 20; i++ {
		nodeID := fmt.Sprintf("N%d", i)
		shape := ShapeRectangle
		switch i % 4 {
		case 1:
			shape = ShapeRounded
		case 2:
			shape = ShapeRhombus
		case 3:
			shape = ShapeStadium
		}
		diagram.Nodes = append(diagram.Nodes, &Node{
			ID:    nodeID,
			Label: fmt.Sprintf("Node %d", i),
			Shape: shape,
		})
	}

	for i := 0; i < 19; i++ {
		diagram.Edges = append(diagram.Edges, &Edge{
			From:     fmt.Sprintf("N%d", i),
			To:       fmt.Sprintf("N%d", i+1),
			ArrowEnd: ArrowNormal,
		})
	}

	// Add cross-connections
	diagram.Edges = append(diagram.Edges,
		&Edge{From: "N5", To: "N10", ArrowEnd: ArrowNormal},
		&Edge{From: "N10", To: "N15", ArrowEnd: ArrowNormal},
		&Edge{From: "N3", To: "N8", ArrowEnd: ArrowNormal},
		&Edge{From: "N12", To: "N18", ArrowEnd: ArrowNormal},
	)

	return diagram
}

func createFlowchartDiagramLarge() *Diagram {
	diagram := &Diagram{
		Type:      DiagramFlowchart,
		Direction: DirectionTB,
	}

	for i := 0; i < 100; i++ {
		nodeID := fmt.Sprintf("N%d", i)
		shape := ShapeRectangle
		switch i % 4 {
		case 1:
			shape = ShapeRounded
		case 2:
			shape = ShapeRhombus
		case 3:
			shape = ShapeStadium
		}
		diagram.Nodes = append(diagram.Nodes, &Node{
			ID:    nodeID,
			Label: fmt.Sprintf("Node %d", i),
			Shape: shape,
		})
	}

	for i := 0; i < 99; i++ {
		diagram.Edges = append(diagram.Edges, &Edge{
			From:     fmt.Sprintf("N%d", i),
			To:       fmt.Sprintf("N%d", i+1),
			ArrowEnd: ArrowNormal,
		})
	}

	for i := 0; i < 90; i += 10 {
		diagram.Edges = append(diagram.Edges, &Edge{
			From:     fmt.Sprintf("N%d", i),
			To:       fmt.Sprintf("N%d", i+5),
			ArrowEnd: ArrowNormal,
		})
		if i+15 < 100 {
			diagram.Edges = append(diagram.Edges, &Edge{
				From:     fmt.Sprintf("N%d", i),
				To:       fmt.Sprintf("N%d", i+15),
				ArrowEnd: ArrowNormal,
			})
		}
	}

	return diagram
}

func createSequenceDiagramComplex() *Diagram {
	diagram := &Diagram{
		Type: DiagramSequence,
	}

	participants := []string{"Client", "Gateway", "Auth", "UserSvc", "OrderSvc", "ProductSvc", "PaymentSvc", "NotifySvc", "CacheSvc", "Database"}
	for _, p := range participants {
		diagram.Participants = append(diagram.Participants, &Participant{
			ID:   p,
			Type: ParticipantDefault,
		})
	}

	messages := []struct {
		from, to, msg string
		msgType       MessageType
	}{
		{"Client", "Gateway", "HTTP Request", MessageSolidArrow},
		{"Gateway", "Auth", "Validate Token", MessageSolidArrow},
		{"Auth", "CacheSvc", "Check Cache", MessageSolidArrow},
		{"CacheSvc", "Auth", "Cache Miss", MessageDottedArrow},
		{"Auth", "Database", "Query User", MessageSolidArrow},
		{"Database", "Auth", "User Data", MessageDottedArrow},
		{"Auth", "CacheSvc", "Store Cache", MessageSolidArrow},
		{"Auth", "Gateway", "Token Valid", MessageDottedArrow},
		{"Gateway", "UserSvc", "Get Profile", MessageSolidArrow},
		{"UserSvc", "Database", "Query Profile", MessageSolidArrow},
	}

	// Add 40 more messages for a total of 50
	for i := 0; i < 40; i++ {
		from := participants[i%len(participants)]
		to := participants[(i+1)%len(participants)]
		messages = append(messages, struct {
			from, to, msg string
			msgType       MessageType
		}{from, to, fmt.Sprintf("Message %d", i+10), MessageSolidArrow})
	}

	for _, m := range messages {
		diagram.Messages = append(diagram.Messages, &Message{
			From:  m.from,
			To:    m.to,
			Label: m.msg,
			Type:  m.msgType,
		})
	}

	return diagram
}

// =============================================================================
// Parsing Benchmarks
// =============================================================================

func BenchmarkParse(b *testing.B) {
	b.Run("Flowchart/Small", func(b *testing.B) {
		input := generateFlowchartSmall()
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_, err := Parse(input)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Flowchart/Medium", func(b *testing.B) {
		input := generateFlowchartMedium()
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_, err := Parse(input)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Flowchart/Large", func(b *testing.B) {
		input := generateFlowchartLarge()
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_, err := Parse(input)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Sequence/Complex", func(b *testing.B) {
		input := generateSequenceDiagramComplex()
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_, err := Parse(input)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Class/Large", func(b *testing.B) {
		input := generateClassDiagramLarge()
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_, err := Parse(input)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// =============================================================================
// Layout Engine Benchmarks
// =============================================================================

func BenchmarkLayout(b *testing.B) {
	b.Run("Dagre/Small", func(b *testing.B) {
		layout := NewDagreLayout(nil)
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			diagram := createFlowchartDiagramSmall()
			err := layout.Layout(diagram)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Dagre/Medium", func(b *testing.B) {
		layout := NewDagreLayout(nil)
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			diagram := createFlowchartDiagramMedium()
			err := layout.Layout(diagram)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Dagre/Large", func(b *testing.B) {
		layout := NewDagreLayout(nil)
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			diagram := createFlowchartDiagramLarge()
			err := layout.Layout(diagram)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Sequence/Complex", func(b *testing.B) {
		layout := NewSequenceLayout(nil)
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			diagram := createSequenceDiagramComplex()
			err := layout.Layout(diagram)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// =============================================================================
// SVG Generation Benchmarks
// =============================================================================

func BenchmarkSVGGeneration(b *testing.B) {
	b.Run("Small", func(b *testing.B) {
		diagram := createFlowchartDiagramSmall()
		layout := NewDagreLayout(nil)
		layout.Layout(diagram)
		size := GetDagreLayoutSize(diagram, nil)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			writer := NewSVGWriter(size.Width, size.Height, nil)
			writer.Start()
			for _, edge := range diagram.Edges {
				writer.DrawEdge(edge)
			}
			for _, node := range diagram.Nodes {
				writer.DrawNode(node)
			}
			_ = writer.End()
		}
	})

	b.Run("Medium", func(b *testing.B) {
		diagram := createFlowchartDiagramMedium()
		layout := NewDagreLayout(nil)
		layout.Layout(diagram)
		size := GetDagreLayoutSize(diagram, nil)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			writer := NewSVGWriter(size.Width, size.Height, nil)
			writer.Start()
			for _, edge := range diagram.Edges {
				writer.DrawEdge(edge)
			}
			for _, node := range diagram.Nodes {
				writer.DrawNode(node)
			}
			_ = writer.End()
		}
	})

	b.Run("Large", func(b *testing.B) {
		diagram := createFlowchartDiagramLarge()
		layout := NewDagreLayout(nil)
		layout.Layout(diagram)
		size := GetDagreLayoutSize(diagram, nil)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			writer := NewSVGWriter(size.Width, size.Height, nil)
			writer.Start()
			for _, edge := range diagram.Edges {
				writer.DrawEdge(edge)
			}
			for _, node := range diagram.Nodes {
				writer.DrawNode(node)
			}
			_ = writer.End()
		}
	})
}

// =============================================================================
// Full Render Pipeline Benchmarks
// =============================================================================

func BenchmarkFullRender(b *testing.B) {
	renderer := NewRenderer(nil)
	ctx := context.Background()

	b.Run("Flowchart/Small", func(b *testing.B) {
		input := generateFlowchartSmall()
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_, err := renderer.Render(ctx, input)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Flowchart/Medium", func(b *testing.B) {
		input := generateFlowchartMedium()
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_, err := renderer.Render(ctx, input)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Flowchart/Large", func(b *testing.B) {
		input := generateFlowchartLarge()
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_, err := renderer.Render(ctx, input)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Sequence/Complex", func(b *testing.B) {
		input := generateSequenceDiagramComplex()
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_, err := renderer.Render(ctx, input)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Class/Large", func(b *testing.B) {
		input := generateClassDiagramLarge()
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_, err := renderer.Render(ctx, input)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// =============================================================================
// Component Benchmarks - Isolated operations
// =============================================================================

func BenchmarkLexer(b *testing.B) {
	b.Run("Small", func(b *testing.B) {
		input := generateFlowchartSmall()
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			lexer := NewLexer(input)
			_ = lexer.Tokenize()
		}
	})

	b.Run("Large", func(b *testing.B) {
		input := generateFlowchartLarge()
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			lexer := NewLexer(input)
			_ = lexer.Tokenize()
		}
	})
}

func BenchmarkNodeDrawing(b *testing.B) {
	shapes := []struct {
		name  string
		shape NodeShape
	}{
		{"Rectangle", ShapeRectangle},
		{"Rounded", ShapeRounded},
		{"Circle", ShapeCircle},
		{"Diamond", ShapeRhombus},
		{"Hexagon", ShapeHexagon},
		{"Stadium", ShapeStadium},
		{"Cylinder", ShapeCylinder},
	}

	for _, s := range shapes {
		b.Run(s.name, func(b *testing.B) {
			node := &Node{
				ID:       "test",
				Label:    "Test Node Label",
				Shape:    s.shape,
				Position: Position{X: 100, Y: 100},
				Size:     Size{Width: 120, Height: 40},
			}
			writer := NewSVGWriter(400, 300, nil)
			writer.Start()

			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				writer.DrawNode(node)
			}
		})
	}
}

func BenchmarkEdgeDrawing(b *testing.B) {
	edgeTypes := []struct {
		name     string
		edgeType EdgeType
		arrow    ArrowType
	}{
		{"Solid/Arrow", EdgeSolid, ArrowNormal},
		{"Dotted/Arrow", EdgeDotted, ArrowNormal},
		{"Thick/Arrow", EdgeThick, ArrowNormal},
		{"Solid/Circle", EdgeSolid, ArrowCircle},
		{"Solid/Cross", EdgeSolid, ArrowCross},
	}

	for _, e := range edgeTypes {
		b.Run(e.name, func(b *testing.B) {
			edge := &Edge{
				From:     "A",
				To:       "B",
				Label:    "Edge Label",
				Type:     e.edgeType,
				ArrowEnd: e.arrow,
				Points: []Position{
					{X: 100, Y: 100},
					{X: 200, Y: 100},
					{X: 200, Y: 200},
					{X: 300, Y: 200},
				},
			}
			writer := NewSVGWriter(400, 300, nil)
			writer.Start()

			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				writer.DrawEdge(edge)
			}
		})
	}
}

// =============================================================================
// Memory Allocation Focused Benchmarks
// =============================================================================

func BenchmarkAllocs(b *testing.B) {
	b.Run("ParseMediumFlowchart", func(b *testing.B) {
		input := generateFlowchartMedium()
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_, _ = Parse(input)
		}
	})

	b.Run("LayoutMediumDiagram", func(b *testing.B) {
		layout := NewDagreLayout(nil)
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			diagram := createFlowchartDiagramMedium()
			_ = layout.Layout(diagram)
		}
	})

	b.Run("SVGMediumDiagram", func(b *testing.B) {
		diagram := createFlowchartDiagramMedium()
		layout := NewDagreLayout(nil)
		layout.Layout(diagram)
		size := GetDagreLayoutSize(diagram, nil)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			writer := NewSVGWriter(size.Width, size.Height, nil)
			writer.Start()
			for _, e := range diagram.Edges {
				writer.DrawEdge(e)
			}
			for _, n := range diagram.Nodes {
				writer.DrawNode(n)
			}
			_ = writer.End()
		}
	})
}

// =============================================================================
// Complexity Scaling Benchmarks
// =============================================================================

func BenchmarkScaling(b *testing.B) {
	nodeCounts := []int{5, 10, 25, 50, 100}

	for _, count := range nodeCounts {
		b.Run(fmt.Sprintf("Parse/Nodes_%d", count), func(b *testing.B) {
			input := generateFlowchartWithNodes(count)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				_, _ = Parse(input)
			}
		})
	}

	for _, count := range nodeCounts {
		b.Run(fmt.Sprintf("Layout/Nodes_%d", count), func(b *testing.B) {
			layout := NewDagreLayout(nil)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				diagram := createFlowchartWithNodes(count)
				_ = layout.Layout(diagram)
			}
		})
	}

	for _, count := range nodeCounts {
		b.Run(fmt.Sprintf("FullRender/Nodes_%d", count), func(b *testing.B) {
			input := generateFlowchartWithNodes(count)
			renderer := NewRenderer(nil)
			ctx := context.Background()
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				_, _ = renderer.Render(ctx, input)
			}
		})
	}
}

func generateFlowchartWithNodes(n int) string {
	var sb strings.Builder
	sb.WriteString("flowchart TD\n")

	for i := 0; i < n; i++ {
		sb.WriteString(fmt.Sprintf("    N%d[Node %d]\n", i, i))
	}

	for i := 0; i < n-1; i++ {
		sb.WriteString(fmt.Sprintf("    N%d --> N%d\n", i, i+1))
	}

	return sb.String()
}

func createFlowchartWithNodes(n int) *Diagram {
	diagram := &Diagram{
		Type:      DiagramFlowchart,
		Direction: DirectionTB,
	}

	for i := 0; i < n; i++ {
		diagram.Nodes = append(diagram.Nodes, &Node{
			ID:    fmt.Sprintf("N%d", i),
			Label: fmt.Sprintf("Node %d", i),
			Shape: ShapeRectangle,
		})
	}

	for i := 0; i < n-1; i++ {
		diagram.Edges = append(diagram.Edges, &Edge{
			From:     fmt.Sprintf("N%d", i),
			To:       fmt.Sprintf("N%d", i+1),
			ArrowEnd: ArrowNormal,
		})
	}

	return diagram
}

// =============================================================================
// Concurrent Rendering Benchmarks
// =============================================================================

func BenchmarkConcurrentRender(b *testing.B) {
	input := generateFlowchartMedium()
	ctx := context.Background()

	b.Run("Sequential", func(b *testing.B) {
		renderer := NewRenderer(nil)
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_, _ = renderer.Render(ctx, input)
		}
	})

	b.Run("Parallel", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			renderer := NewRenderer(nil)
			for pb.Next() {
				_, _ = renderer.Render(ctx, input)
			}
		})
	})
}

// =============================================================================
// Theme Switching Benchmarks
// =============================================================================

func BenchmarkThemes(b *testing.B) {
	input := generateFlowchartMedium()
	ctx := context.Background()

	b.Run("DefaultTheme", func(b *testing.B) {
		renderer := NewRenderer(&RenderOptions{Theme: DefaultTheme()})
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_, _ = renderer.Render(ctx, input)
		}
	})

	b.Run("LightTheme", func(b *testing.B) {
		renderer := NewRenderer(&RenderOptions{Theme: LightTheme()})
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_, _ = renderer.Render(ctx, input)
		}
	})
}
