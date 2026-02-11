package mermaid

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
)

// FlowchartLayoutEngine identifies the layout algorithm for flowcharts.
// Currently only the full dagre algorithm is supported.
type FlowchartLayoutEngine string

const (
	// LayoutEngineFullDagre uses the complete dagre algorithm with all phases:
	// cycle breaking, network simplex ranking, dummy nodes, and Brandes-Kopf positioning.
	LayoutEngineFullDagre FlowchartLayoutEngine = "dagre-full"
)

const (
	stateStartID = stateMarker + "_start"
	stateEndID   = stateMarker + "_end"
)

// Renderer is the main entry point for rendering Mermaid diagrams.
type Renderer struct {
	theme      *Theme
	timeout    time.Duration
	layoutOpts *LayoutOptions
}

// RenderOptions configures rendering behavior.
type RenderOptions struct {
	Theme      *Theme
	Timeout    time.Duration
	LayoutOpts *LayoutOptions
}

// RenderResult contains the rendered SVG and metadata.
type RenderResult struct {
	SVG      string
	Width    float64
	Height   float64
	Duration time.Duration
}

// Errors
var (
	ErrEmptyDiagram    = errors.New("empty diagram source")
	ErrUnsupportedType = errors.New("unsupported diagram type")
	ErrRenderTimeout   = errors.New("render timeout exceeded")
)

// NewRenderer creates a new Mermaid renderer.
func NewRenderer(opts *RenderOptions) *Renderer {
	r := &Renderer{
		theme:      DefaultTheme(),
		timeout:    10 * time.Second,
		layoutOpts: DefaultLayoutOptions(),
	}

	if opts != nil {
		if opts.Theme != nil {
			r.theme = opts.Theme
		}
		if opts.Timeout > 0 {
			r.timeout = opts.Timeout
		}
		if opts.LayoutOpts != nil {
			r.layoutOpts = opts.LayoutOpts
		}
	}

	return r
}

// Render parses and renders a Mermaid diagram to SVG.
func (r *Renderer) Render(ctx context.Context, source string) (*RenderResult, error) {
	if strings.TrimSpace(source) == "" {
		return nil, ErrEmptyDiagram
	}

	start := time.Now()

	// Set up timeout
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	// Parse the diagram
	diagram, err := Parse(source)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	// Render based on diagram type
	var svg string
	var width, height float64

	done := make(chan struct{})
	var renderErr error

	go func() {
		defer close(done)
		switch diagram.Type {
		case DiagramFlowchart:
			svg, width, height, renderErr = r.renderFlowchart(diagram)
		case DiagramSequence:
			svg, width, height, renderErr = r.renderSequence(diagram)
		case DiagramClass:
			svg, width, height, renderErr = r.renderClass(diagram)
		case DiagramState:
			svg, width, height, renderErr = r.renderState(diagram)
		case DiagramER:
			svg, width, height, renderErr = r.renderER(diagram)
		case DiagramPie:
			svg, width, height, renderErr = r.renderPie(diagram)
		default:
			renderErr = ErrUnsupportedType
		}
	}()

	select {
	case <-ctx.Done():
		return nil, ErrRenderTimeout
	case <-done:
		if renderErr != nil {
			return nil, renderErr
		}
	}

	return &RenderResult{
		SVG:      svg,
		Width:    width,
		Height:   height,
		Duration: time.Since(start),
	}, nil
}

// renderFlowchart renders a flowchart diagram.
func (r *Renderer) renderFlowchart(diagram *Diagram) (string, float64, float64, error) {
	// Apply layout using the full dagre engine
	layoutEngine := NewFullDagreLayout(r.layoutOpts)

	if err := layoutEngine.Layout(diagram); err != nil {
		return "", 0, 0, fmt.Errorf("layout error: %w", err)
	}

	// Calculate subgraph bounds
	r.calculateSubgraphBounds(diagram)

	// Get total size
	size := GetDagreLayoutSize(diagram, r.layoutOpts)

	// Create SVG writer
	writer := NewSVGWriter(size.Width, size.Height, r.theme)
	writer.Start()

	// Draw subgraphs first (as background)
	for _, sg := range diagram.Subgraphs {
		writer.DrawSubgraph(sg)
	}

	// Draw edges
	for _, edge := range diagram.Edges {
		writer.DrawEdge(edge)
	}

	// Draw nodes
	for _, node := range diagram.Nodes {
		writer.DrawNode(node)
	}

	svg := writer.End()
	return svg, size.Width, size.Height, nil
}

// calculateSubgraphBounds computes bounding boxes for subgraphs.
//
//nolint:gocognit // complex geometry aggregation mirrors mermaid behavior.
func (r *Renderer) calculateSubgraphBounds(diagram *Diagram) {
	nodeMap := make(map[string]*Node)
	for _, node := range diagram.Nodes {
		nodeMap[node.ID] = node
	}

	for _, sg := range diagram.Subgraphs {
		if len(sg.Nodes) == 0 {
			continue
		}

		minX, minY := 1e9, 1e9
		maxX, maxY := 0.0, 0.0

		for _, nodeID := range sg.Nodes {
			node, ok := nodeMap[nodeID]
			if !ok {
				continue
			}

			if node.Position.X < minX {
				minX = node.Position.X
			}
			if node.Position.Y < minY {
				minY = node.Position.Y
			}
			right := node.Position.X + node.Size.Width
			bottom := node.Position.Y + node.Size.Height
			if right > maxX {
				maxX = right
			}
			if bottom > maxY {
				maxY = bottom
			}
		}

		padding := 20.0
		sg.Position = Position{X: minX - padding, Y: minY - padding - 20} // Extra space for label
		sg.Size = Size{Width: maxX - minX + padding*2, Height: maxY - minY + padding*2 + 20}
	}
}

// renderSequence renders a sequence diagram.
func (r *Renderer) renderSequence(diagram *Diagram) (string, float64, float64, error) {
	opts := DefaultSequenceLayoutOptions()
	layout := NewSequenceLayout(opts)

	if err := layout.Layout(diagram); err != nil {
		return "", 0, 0, fmt.Errorf("layout error: %w", err)
	}

	size := layout.GetDiagramSize()
	writer := NewSVGWriter(size.Width, size.Height, r.theme)
	writer.Start()

	// Calculate lifeline end point
	lifelineEnd := size.Height - opts.MarginY

	// Draw participants
	x := opts.MarginX
	for _, p := range diagram.Participants {
		writer.DrawParticipant(p, x, opts.MarginY, opts.ParticipantWidth, opts.ParticipantHeight, lifelineEnd)
		x += opts.ParticipantWidth + opts.ParticipantGap
	}

	// Draw activations (before messages so messages draw on top)
	for _, act := range diagram.Activations {
		if act.StartY > 0 && act.EndY > 0 {
			actX := layout.GetParticipantX(act.Participant)
			writer.DrawActivation(actX, act.StartY, act.EndY, opts.ActivationWidth)
		}
	}

	// Draw messages
	y := opts.MarginY + opts.ParticipantHeight + 30
	for _, msg := range diagram.Messages {
		fromX := layout.GetParticipantX(msg.From)
		toX := layout.GetParticipantX(msg.To)

		// Handle self-messages
		if msg.From == msg.To {
			// Draw self-referencing arrow
			r.drawSelfMessage(writer, fromX, y, msg)
		} else {
			writer.DrawMessage(msg, fromX, toX, y)
		}
		y += opts.MessageHeight
	}

	// Draw notes
	for _, note := range diagram.Notes {
		noteX := layout.GetParticipantX(note.Participant)
		noteWidth := opts.NoteWidth
		noteHeight := 40.0

		switch note.Position {
		case NoteRightOf:
			noteX += opts.ParticipantWidth/2 + 10
		case NoteLeftOf:
			noteX -= opts.ParticipantWidth/2 + noteWidth + 10
		case NoteOver:
			noteX -= noteWidth / 2
		}

		writer.DrawNote(note, noteX, y, noteWidth, noteHeight)
		y += noteHeight + 10
	}

	svg := writer.End()
	return svg, size.Width, size.Height, nil
}

func (r *Renderer) drawSelfMessage(writer *SVGWriter, x, y float64, msg *Message) {
	// Draw self-referencing message loop
	loopWidth := 30.0
	loopHeight := 20.0

	writer.AddArrowMarker("msg-arrow", ArrowNormal)

	path := fmt.Sprintf(
		"M %.1f %.1f L %.1f %.1f L %.1f %.1f L %.1f %.1f",
		x, y,
		x+loopWidth, y,
		x+loopWidth, y+loopHeight,
		x, y+loopHeight,
	)

	strokeDash := ""
	switch msg.Type {
	case MessageDotted, MessageDottedArrow, MessageDottedCross, MessageDottedOpen:
		strokeDash = ` stroke-dasharray="5,5"`
	}

	writer.buf.WriteString(fmt.Sprintf(
		`<path d="%s" fill="none" stroke="%s" stroke-width="%.1f"%s marker-end="url(#msg-arrow)"/>`,
		path, writer.theme.EdgeStroke, writer.theme.EdgeStrokeWidth, strokeDash,
	))
	writer.buf.WriteString("\n")

	// Draw label
	if msg.Label != "" {
		writer.buf.WriteString(fmt.Sprintf(
			`<text x="%.1f" y="%.1f" class="edge-text" text-anchor="start">%s</text>`,
			x+loopWidth+5, y+loopHeight/2, msg.Label,
		))
		writer.buf.WriteString("\n")
	}
}

// renderClass renders a class diagram.
//
//nolint:gocognit,gocyclo // complex renderer mirrors mermaid behavior.
func (r *Renderer) renderClass(diagram *Diagram) (string, float64, float64, error) {
	// Convert classes to nodes for layout
	for i, class := range diagram.Classes {
		node := &Node{
			ID:    class.Name,
			Label: class.Name,
			Shape: ShapeRectangle,
		}
		// Calculate size based on content
		maxWidth := float64(len(class.Name)) * 8
		for _, attr := range class.Attributes {
			w := float64(len(attr.Name)+len(attr.Type)+4) * 7
			if w > maxWidth {
				maxWidth = w
			}
		}
		for _, method := range class.Methods {
			w := float64(len(method.Name)+len(method.Parameters)+4) * 7
			if w > maxWidth {
				maxWidth = w
			}
		}

		lineHeight := 18.0
		height := lineHeight*2 + float64(len(class.Attributes)+len(class.Methods))*lineHeight + 20

		node.Size = Size{Width: maxWidth + 20, Height: height}

		if i < len(diagram.Nodes) {
			diagram.Nodes[i] = node
		} else {
			diagram.Nodes = append(diagram.Nodes, node)
		}
	}

	// Convert relationships to edges
	for _, rel := range diagram.Relationships {
		edge := &Edge{
			From:  rel.From,
			To:    rel.To,
			Label: rel.Label,
		}
		// Set arrow types based on relationship
		switch rel.Type {
		case RelationInheritance:
			edge.ArrowEnd = ArrowNormal // Will be styled as hollow triangle
		case RelationRealization:
			edge.Type = EdgeDotted
			edge.ArrowEnd = ArrowNormal
		case RelationComposition:
			edge.ArrowStart = ArrowNormal // Filled diamond
		case RelationAggregation:
			edge.ArrowStart = ArrowCircle // Empty diamond
		default:
			edge.ArrowEnd = ArrowNormal
		}
		diagram.Edges = append(diagram.Edges, edge)
	}

	// Use full dagre layout
	layout := NewFullDagreLayout(r.layoutOpts)
	if err := layout.Layout(diagram); err != nil {
		return "", 0, 0, fmt.Errorf("layout error: %w", err)
	}

	size := GetDagreLayoutSize(diagram, r.layoutOpts)
	writer := NewSVGWriter(size.Width, size.Height, r.theme)
	writer.Start()

	// Draw edges first
	for _, edge := range diagram.Edges {
		writer.DrawEdge(edge)
	}

	// Draw class boxes
	for i, node := range diagram.Nodes {
		if i >= len(diagram.Classes) {
			continue
		}
		class := diagram.Classes[i]
		r.drawClassBox(writer, node, class)
	}

	svg := writer.End()
	return svg, size.Width, size.Height, nil
}

func (r *Renderer) drawClassBox(writer *SVGWriter, node *Node, class *Class) {
	x := node.Position.X
	y := node.Position.Y
	width := node.Size.Width
	height := node.Size.Height

	lineHeight := 18.0
	headerHeight := lineHeight * 2

	// Draw main box
	writer.drawRoundedRect(x, y, width, height, 2, writer.theme.NodeFill, writer.theme.NodeStroke, writer.theme.NodeStrokeWidth)

	// Draw class name (header)
	writer.buf.WriteString(fmt.Sprintf(
		`<text x="%.1f" y="%.1f" class="node-text" style="font-weight: bold;">%s</text>`,
		x+width/2, y+lineHeight, class.Name,
	))
	writer.buf.WriteString("\n")

	// Separator line
	sepY := y + headerHeight
	writer.buf.WriteString(fmt.Sprintf(
		`<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="%s" stroke-width="1"/>`,
		x, sepY, x+width, sepY, writer.theme.NodeStroke,
	))
	writer.buf.WriteString("\n")

	// Draw attributes
	attrY := sepY + lineHeight
	for _, attr := range class.Attributes {
		vis := visibilitySymbol(attr.Visibility)
		text := fmt.Sprintf("%s%s", vis, attr.Name)
		if attr.Type != "" {
			text += ": " + attr.Type
		}
		writer.buf.WriteString(fmt.Sprintf(
			`<text x="%.1f" y="%.1f" class="node-text" style="text-anchor: start; font-size: 12px;">%s</text>`,
			x+8, attrY, text,
		))
		writer.buf.WriteString("\n")
		attrY += lineHeight
	}

	// Separator if there are methods
	if len(class.Methods) > 0 {
		writer.buf.WriteString(fmt.Sprintf(
			`<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="%s" stroke-width="1"/>`,
			x, attrY, x+width, attrY, writer.theme.NodeStroke,
		))
		writer.buf.WriteString("\n")
		attrY += 5
	}

	// Draw methods
	for _, method := range class.Methods {
		vis := visibilitySymbol(method.Visibility)
		text := fmt.Sprintf("%s%s()", vis, method.Name)
		if method.Type != "" {
			text += ": " + method.Type
		}
		writer.buf.WriteString(fmt.Sprintf(
			`<text x="%.1f" y="%.1f" class="node-text" style="text-anchor: start; font-size: 12px;">%s</text>`,
			x+8, attrY, text,
		))
		writer.buf.WriteString("\n")
		attrY += lineHeight
	}
}

func visibilitySymbol(v Visibility) string {
	switch v {
	case VisibilityPublic:
		return "+"
	case VisibilityPrivate:
		return "-"
	case VisibilityProtected:
		return "#"
	case VisibilityPackage:
		return "~"
	default:
		return ""
	}
}

// renderState renders a state diagram.
//
//nolint:gocognit,gocyclo // complex renderer mirrors mermaid behavior.
func (r *Renderer) renderState(diagram *Diagram) (string, float64, float64, error) {
	// Convert states to nodes
	for _, state := range diagram.States {
		// Use description if available, otherwise use label or ID
		label := state.Description
		if label == "" {
			label = state.Label
		}
		if label == "" {
			label = state.ID
		}

		node := &Node{
			ID:    state.ID,
			Label: label,
			Shape: ShapeRounded,
		}

		// Special shapes for start/end states
		switch state.Type {
		case StateStart, StateEnd:
			node.Shape = ShapeCircle
			node.Size = Size{Width: 30, Height: 30}
		default:
			// Calculate size based on label
			width := float64(len(label))*8 + 40
			if width < 80 {
				width = 80
			}
			node.Size = Size{Width: width, Height: 40}
		}

		diagram.Nodes = append(diagram.Nodes, node)
	}

	// Handle [*] start/end states
	for _, trans := range diagram.Transitions {
		if trans.From == stateMarker {
			node := &Node{
				ID:    stateStartID,
				Label: "",
				Shape: ShapeCircle,
				Size:  Size{Width: 20, Height: 20},
				Style: &NodeStyle{Fill: r.theme.NodeStroke},
			}
			diagram.Nodes = append(diagram.Nodes, node)
			trans.From = stateStartID
		}
		if trans.To == stateMarker {
			// Check if end state already exists
			found := false
			for _, n := range diagram.Nodes {
				if n.ID == stateEndID {
					found = true
					break
				}
			}
			if !found {
				node := &Node{
					ID:    stateEndID,
					Label: "",
					Shape: ShapeDoubleCircle,
					Size:  Size{Width: 24, Height: 24},
				}
				diagram.Nodes = append(diagram.Nodes, node)
			}
			trans.To = stateEndID
		}
	}

	// Convert transitions to edges
	for _, trans := range diagram.Transitions {
		// Build label with guard if present
		label := trans.Label
		if trans.Guard != "" {
			if label != "" {
				label = label + " [" + trans.Guard + "]"
			} else {
				label = "[" + trans.Guard + "]"
			}
		}

		edge := &Edge{
			From:     trans.From,
			To:       trans.To,
			Label:    label,
			ArrowEnd: ArrowNormal,
		}
		diagram.Edges = append(diagram.Edges, edge)
	}

	// Use full dagre layout
	layout := NewFullDagreLayout(r.layoutOpts)
	if err := layout.Layout(diagram); err != nil {
		return "", 0, 0, fmt.Errorf("layout error: %w", err)
	}

	size := GetDagreLayoutSize(diagram, r.layoutOpts)
	writer := NewSVGWriter(size.Width, size.Height, r.theme)
	writer.Start()

	// Draw edges
	for _, edge := range diagram.Edges {
		writer.DrawEdge(edge)
	}

	// Draw nodes
	for _, node := range diagram.Nodes {
		writer.DrawNode(node)
	}

	svg := writer.End()
	return svg, size.Width, size.Height, nil
}

// renderER renders an entity-relationship diagram.
func (r *Renderer) renderER(diagram *Diagram) (string, float64, float64, error) {
	// Convert entities to nodes
	for _, entity := range diagram.Entities {
		// Calculate size based on attributes
		lineHeight := 18.0
		maxWidth := float64(len(entity.Name)) * 9

		for _, attr := range entity.Attributes {
			w := float64(len(attr.Type)+len(attr.Name)+5) * 7
			if w > maxWidth {
				maxWidth = w
			}
		}

		height := lineHeight*2 + float64(len(entity.Attributes))*lineHeight + 10

		node := &Node{
			ID:    entity.Name,
			Label: entity.Name,
			Shape: ShapeRectangle,
			Size:  Size{Width: maxWidth + 30, Height: height},
		}
		diagram.Nodes = append(diagram.Nodes, node)
	}

	// Convert ER relations to edges
	for _, rel := range diagram.ERRelations {
		edge := &Edge{
			From:  rel.EntityA,
			To:    rel.EntityB,
			Label: rel.Label,
		}

		// Set cardinality indicators (simplified - would need custom rendering for crow's foot)
		if rel.Identifying {
			edge.Type = EdgeSolid
		} else {
			edge.Type = EdgeDotted
		}
		edge.ArrowEnd = ArrowNone
		edge.ArrowStart = ArrowNone

		diagram.Edges = append(diagram.Edges, edge)
	}

	// Apply layout with ER-specific options
	erOpts := &LayoutOptions{
		NodeWidth:   150,
		NodeHeight:  80,
		NodePadding: 15,
		RankSep:     100,
		NodeSep:     60,
		EdgeSep:     20,
		MarginX:     50,
		MarginY:     50,
	}
	layout := NewFullDagreLayout(erOpts)

	if err := layout.Layout(diagram); err != nil {
		return "", 0, 0, fmt.Errorf("layout error: %w", err)
	}

	size := GetDagreLayoutSize(diagram, erOpts)
	writer := NewSVGWriter(size.Width, size.Height, r.theme)
	writer.Start()

	// Draw edges with cardinality labels
	for i, edge := range diagram.Edges {
		writer.DrawEdge(edge)

		// Draw cardinality at endpoints
		if i < len(diagram.ERRelations) {
			rel := diagram.ERRelations[i]
			r.drawCardinality(writer, edge, rel)
		}
	}

	// Draw entity boxes
	for i, node := range diagram.Nodes {
		if i >= len(diagram.Entities) {
			continue
		}
		entity := diagram.Entities[i]
		r.drawEntityBox(writer, node, entity)
	}

	svg := writer.End()
	return svg, size.Width, size.Height, nil
}

func (r *Renderer) drawEntityBox(writer *SVGWriter, node *Node, entity *Entity) {
	x := node.Position.X
	y := node.Position.Y
	width := node.Size.Width
	height := node.Size.Height

	lineHeight := 18.0
	headerHeight := lineHeight * 2

	// Draw main box
	writer.drawRect(x, y, width, height, writer.theme.NodeFill, writer.theme.NodeStroke, writer.theme.NodeStrokeWidth)

	// Draw entity name (header)
	writer.buf.WriteString(fmt.Sprintf(
		`<text x="%.1f" y="%.1f" class="node-text" style="font-weight: bold;">%s</text>`,
		x+width/2, y+lineHeight, entity.Name,
	))
	writer.buf.WriteString("\n")

	// Separator line
	sepY := y + headerHeight
	writer.buf.WriteString(fmt.Sprintf(
		`<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="%s" stroke-width="1"/>`,
		x, sepY, x+width, sepY, writer.theme.NodeStroke,
	))
	writer.buf.WriteString("\n")

	// Draw attributes
	attrY := sepY + lineHeight
	for _, attr := range entity.Attributes {
		keyMark := ""
		switch attr.Key {
		case KeyPrimary:
			keyMark = "PK "
		case KeyForeign:
			keyMark = "FK "
		case KeyUnique:
			keyMark = "UK "
		}

		text := fmt.Sprintf("%s%s %s", keyMark, attr.Type, attr.Name)
		writer.buf.WriteString(fmt.Sprintf(
			`<text x="%.1f" y="%.1f" class="node-text" style="text-anchor: start; font-size: 12px;">%s</text>`,
			x+8, attrY, text,
		))
		writer.buf.WriteString("\n")
		attrY += lineHeight
	}
}

func (r *Renderer) drawCardinality(writer *SVGWriter, edge *Edge, rel *ERRelation) {
	if len(edge.Points) < 2 {
		return
	}

	// Draw cardinality near start
	startLabel := cardinalityLabel(rel.CardinalityA)
	if startLabel != "" {
		p := edge.Points[0]
		writer.buf.WriteString(fmt.Sprintf(
			`<text x="%.1f" y="%.1f" class="edge-text" style="font-size: 11px;">%s</text>`,
			p.X+15, p.Y-8, startLabel,
		))
		writer.buf.WriteString("\n")
	}

	// Draw cardinality near end
	endLabel := cardinalityLabel(rel.CardinalityB)
	if endLabel != "" {
		p := edge.Points[len(edge.Points)-1]
		writer.buf.WriteString(fmt.Sprintf(
			`<text x="%.1f" y="%.1f" class="edge-text" style="font-size: 11px;">%s</text>`,
			p.X-25, p.Y-8, endLabel,
		))
		writer.buf.WriteString("\n")
	}
}

func cardinalityLabel(c Cardinality) string {
	switch c {
	case CardZeroOrOne:
		return "0..1"
	case CardExactlyOne:
		return "1"
	case CardZeroOrMore:
		return "0..*"
	case CardOneOrMore:
		return "1..*"
	default:
		return ""
	}
}

// renderPie renders a pie chart diagram.
func (r *Renderer) renderPie(diagram *Diagram) (string, float64, float64, error) {
	if len(diagram.PieSlices) == 0 {
		return "", 0, 0, errors.New("pie chart has no slices")
	}

	// Calculate total value
	total := 0.0
	for _, slice := range diagram.PieSlices {
		total += slice.Value
	}

	if total == 0 {
		return "", 0, 0, errors.New("pie chart total value is zero")
	}

	// Determine dimensions
	margin := 40.0
	legendWidth := 150.0
	radius := 120.0
	cx := margin + radius
	titleHeight := 0.0
	if diagram.Title != "" {
		titleHeight = 40.0
	}
	cy := margin + radius + titleHeight
	width := cx + radius + margin + legendWidth
	height := cy + radius + margin

	writer := NewSVGWriter(width, height, r.theme)
	writer.Start()

	// Draw title if present
	if diagram.Title != "" {
		writer.buf.WriteString(fmt.Sprintf(
			`<text x="%.1f" y="%.1f" class="node-text" style="font-size: 16px; font-weight: bold;">%s</text>`,
			width/2, margin, diagram.Title,
		))
		writer.buf.WriteString("\n")
	}

	// Draw pie slices
	startAngle := -90.0 // Start from top
	colors := []string{
		"#4e79a7", "#f28e2b", "#e15759", "#76b7b2",
		"#59a14f", "#edc948", "#b07aa1", "#ff9da7",
		"#9c755f", "#bab0ac",
	}

	for i, slice := range diagram.PieSlices {
		percentage := slice.Value / total
		sweepAngle := percentage * 360

		color := colors[i%len(colors)]
		r.drawPieSlice(writer, cx, cy, radius, startAngle, sweepAngle, color)

		// Draw label on slice if showData
		if diagram.ShowData {
			midAngle := startAngle + sweepAngle/2
			labelRadius := radius * 0.7
			labelX := cx + labelRadius*cosD(midAngle)
			labelY := cy + labelRadius*sinD(midAngle)
			writer.buf.WriteString(fmt.Sprintf(
				`<text x="%.1f" y="%.1f" class="node-text" style="font-size: 11px; fill: white;">%.1f%%</text>`,
				labelX, labelY, percentage*100,
			))
			writer.buf.WriteString("\n")
		}

		startAngle += sweepAngle
	}

	// Draw legend
	legendX := cx + radius + margin
	legendY := margin + titleHeight + 20
	for i, slice := range diagram.PieSlices {
		color := colors[i%len(colors)]

		// Color box
		writer.buf.WriteString(fmt.Sprintf(
			`<rect x="%.1f" y="%.1f" width="16" height="16" fill="%s"/>`,
			legendX, legendY, color,
		))
		writer.buf.WriteString("\n")

		// Label
		label := slice.Label
		if diagram.ShowData {
			label = fmt.Sprintf("%s (%.1f%%)", label, (slice.Value/total)*100)
		}
		writer.buf.WriteString(fmt.Sprintf(
			`<text x="%.1f" y="%.1f" class="edge-text" style="text-anchor: start; font-size: 12px;">%s</text>`,
			legendX+24, legendY+12, label,
		))
		writer.buf.WriteString("\n")

		legendY += 24
	}

	svg := writer.End()
	return svg, width, height, nil
}

// drawPieSlice draws a pie slice using SVG arc.
func (r *Renderer) drawPieSlice(writer *SVGWriter, cx, cy, radius, startAngle, sweepAngle float64, fill string) {
	// Convert angles to radians
	startRad := startAngle * 3.14159265359 / 180
	endRad := (startAngle + sweepAngle) * 3.14159265359 / 180

	// Calculate start and end points
	x1 := cx + radius*cosD(startAngle)
	y1 := cy + radius*sinD(startAngle)
	x2 := cx + radius*cosD(startAngle+sweepAngle)
	y2 := cy + radius*sinD(startAngle+sweepAngle)

	// Determine if arc is greater than 180 degrees
	largeArc := 0
	if sweepAngle > 180 {
		largeArc = 1
	}

	// Handle full circle (360 degrees) - need to draw two arcs
	if sweepAngle >= 359.9 {
		// Draw as a full circle
		writer.buf.WriteString(fmt.Sprintf(
			`<circle cx="%.1f" cy="%.1f" r="%.1f" fill="%s" stroke="%s" stroke-width="1"/>`,
			cx, cy, radius, fill, writer.theme.NodeStroke,
		))
		writer.buf.WriteString("\n")
		return
	}

	// Build SVG arc path
	path := fmt.Sprintf(
		"M %.1f %.1f L %.1f %.1f A %.1f %.1f 0 %d 1 %.1f %.1f Z",
		cx, cy, // Move to center
		x1, y1, // Line to start of arc
		radius, radius, // Arc radii
		largeArc, // Large arc flag
		x2, y2,   // Arc end point
	)

	_ = startRad
	_ = endRad

	writer.buf.WriteString(fmt.Sprintf(
		`<path d="%s" fill="%s" stroke="%s" stroke-width="1"/>`,
		path, fill, writer.theme.NodeStroke,
	))
	writer.buf.WriteString("\n")
}

// cosD returns cosine of angle in degrees.
func cosD(degrees float64) float64 {
	return math.Cos(degrees * math.Pi / 180)
}

// sinD returns sine of angle in degrees.
func sinD(degrees float64) float64 {
	return math.Sin(degrees * math.Pi / 180)
}
