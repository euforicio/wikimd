package mermaid

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"html"
	"math"
	"strings"
)

// SVGWriter generates SVG output from positioned diagram elements.
type SVGWriter struct {
	buf    strings.Builder
	theme  *Theme
	width  float64
	height float64
	defs   strings.Builder
	defIDs map[string]bool
	id     string // Unique ID for CSS scoping and marker prefixing
}

// Theme defines colors and styles for diagram rendering.
type Theme struct {
	Background      string
	NodeFill        string
	NodeStroke      string
	NodeText        string
	EdgeStroke      string
	EdgeText        string
	FontFamily      string
	FontSize        float64
	NodeStrokeWidth float64
	EdgeStrokeWidth float64
}

// DefaultTheme returns a dark theme matching the project style.
func DefaultTheme() *Theme {
	return &Theme{
		Background:      "transparent",
		NodeFill:        "#1e1e2e",
		NodeStroke:      "#6c7086",
		NodeText:        "#cdd6f4",
		EdgeStroke:      "#6c7086",
		EdgeText:        "#a6adc8",
		FontFamily:      "ui-monospace, SFMono-Regular, SF Mono, Menlo, monospace",
		FontSize:        14,
		NodeStrokeWidth: 1.5,
		EdgeStrokeWidth: 1.5,
	}
}

// LightTheme returns a light theme.
func LightTheme() *Theme {
	return &Theme{
		Background:      "transparent",
		NodeFill:        "#eff1f5",
		NodeStroke:      "#7c7f93",
		NodeText:        "#4c4f69",
		EdgeStroke:      "#7c7f93",
		EdgeText:        "#5c5f77",
		FontFamily:      "ui-monospace, SFMono-Regular, SF Mono, Menlo, monospace",
		FontSize:        14,
		NodeStrokeWidth: 1.5,
		EdgeStrokeWidth: 1.5,
	}
}

// generateSVGID creates a short unique identifier for SVG scoping.
func generateSVGID() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		// Fallback to a simple counter-based ID if crypto/rand fails
		return "mmd-0"
	}
	return "mmd-" + hex.EncodeToString(b)
}

// NewSVGWriter creates a new SVG writer with the given dimensions.
func NewSVGWriter(width, height float64, theme *Theme) *SVGWriter {
	if theme == nil {
		theme = DefaultTheme()
	}
	return &SVGWriter{
		width:  width,
		height: height,
		theme:  theme,
		defIDs: make(map[string]bool),
		id:     generateSVGID(),
	}
}

// NewSVGWriterWithID creates a new SVG writer with a specific ID for deterministic output.
func NewSVGWriterWithID(width, height float64, theme *Theme, id string) *SVGWriter {
	if theme == nil {
		theme = DefaultTheme()
	}
	if id == "" {
		id = generateSVGID()
	}
	return &SVGWriter{
		width:  width,
		height: height,
		theme:  theme,
		defIDs: make(map[string]bool),
		id:     id,
	}
}

// Start begins the SVG document.
func (w *SVGWriter) Start() {
	w.buf.WriteString(fmt.Sprintf(
		`<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" id="%s" viewBox="0 0 %.1f %.1f" width="%.1f" height="%.1f" role="img">`,
		w.id, w.width, w.height, w.width, w.height,
	))
	w.buf.WriteString("\n")

	// Add scoped styles (prefixed with SVG ID to avoid global leakage)
	w.buf.WriteString("<style>\n")
	w.buf.WriteString(fmt.Sprintf(
		`#%s .node-text { font-family: %s; font-size: %.1fpx; fill: %s; dominant-baseline: middle; text-anchor: middle; }`,
		w.id, w.theme.FontFamily, w.theme.FontSize, w.theme.NodeText,
	))
	w.buf.WriteString("\n")
	w.buf.WriteString(fmt.Sprintf(
		`#%s .edge-text { font-family: %s; font-size: %.1fpx; fill: %s; dominant-baseline: middle; text-anchor: middle; }`,
		w.id, w.theme.FontFamily, w.theme.FontSize-2, w.theme.EdgeText,
	))
	w.buf.WriteString("\n")
	w.buf.WriteString(fmt.Sprintf(
		`#%s .participant-text { font-family: %s; font-size: %.1fpx; fill: %s; dominant-baseline: middle; text-anchor: middle; font-weight: 500; }`,
		w.id, w.theme.FontFamily, w.theme.FontSize, w.theme.NodeText,
	))
	w.buf.WriteString("\n")
	w.buf.WriteString("</style>\n")
}

// ID returns the unique identifier for this SVG.
func (w *SVGWriter) ID() string {
	return w.id
}

// End closes the SVG document.
func (w *SVGWriter) End() string {
	// Insert defs if any were created
	if w.defs.Len() > 0 {
		w.buf.WriteString("<defs>\n")
		w.buf.WriteString(w.defs.String())
		w.buf.WriteString("</defs>\n")
	}

	w.buf.WriteString("</svg>")
	return w.buf.String()
}

// AddArrowMarker adds an arrowhead marker definition.
func (w *SVGWriter) AddArrowMarker(id string, arrowType ArrowType) {
	if w.defIDs[id] {
		return
	}
	w.defIDs[id] = true

	switch arrowType {
	case ArrowNormal:
		w.defs.WriteString(fmt.Sprintf(
			`<marker id="%s" markerWidth="10" markerHeight="7" refX="9" refY="3.5" orient="auto" markerUnits="strokeWidth">
  <polygon points="0 0, 10 3.5, 0 7" fill="%s"/>
</marker>`,
			id, w.theme.EdgeStroke,
		))
	case ArrowCircle:
		w.defs.WriteString(fmt.Sprintf(
			`<marker id="%s" markerWidth="10" markerHeight="10" refX="5" refY="5" orient="auto" markerUnits="strokeWidth">
  <circle cx="5" cy="5" r="3" fill="none" stroke="%s" stroke-width="1.5"/>
</marker>`,
			id, w.theme.EdgeStroke,
		))
	case ArrowCross:
		w.defs.WriteString(fmt.Sprintf(
			`<marker id="%s" markerWidth="10" markerHeight="10" refX="5" refY="5" orient="auto" markerUnits="strokeWidth">
  <line x1="2" y1="2" x2="8" y2="8" stroke="%s" stroke-width="1.5"/>
  <line x1="8" y1="2" x2="2" y2="8" stroke="%s" stroke-width="1.5"/>
</marker>`,
			id, w.theme.EdgeStroke, w.theme.EdgeStroke,
		))
	}
	w.defs.WriteString("\n")
}

// DrawNode renders a node with its shape and label.
//
//nolint:gocyclo // rendering intentionally branches by node shape.
func (w *SVGWriter) DrawNode(node *Node) {
	x := node.Position.X
	y := node.Position.Y
	width := node.Size.Width
	height := node.Size.Height
	label := html.EscapeString(node.Label)

	fill := w.theme.NodeFill
	stroke := w.theme.NodeStroke
	strokeWidth := w.theme.NodeStrokeWidth

	if node.Style != nil {
		if node.Style.Fill != "" {
			fill = node.Style.Fill
		}
		if node.Style.Stroke != "" {
			stroke = node.Style.Stroke
		}
		if node.Style.StrokeWidth > 0 {
			strokeWidth = node.Style.StrokeWidth
		}
	}

	w.buf.WriteString(fmt.Sprintf(`<g id="node-%s">`, html.EscapeString(node.ID)))
	w.buf.WriteString("\n")

	switch node.Shape {
	case ShapeRounded:
		w.drawRoundedRect(x, y, width, height, 8, fill, stroke, strokeWidth)
	case ShapeCircle:
		radius := math.Min(width, height) / 2
		cx := x + width/2
		cy := y + height/2
		w.drawCircle(cx, cy, radius, fill, stroke, strokeWidth)
	case ShapeDoubleCircle:
		radius := math.Min(width, height) / 2
		cx := x + width/2
		cy := y + height/2
		w.drawCircle(cx, cy, radius, fill, stroke, strokeWidth)
		w.drawCircle(cx, cy, radius-4, fill, stroke, strokeWidth)
	case ShapeRhombus:
		w.drawDiamond(x, y, width, height, fill, stroke, strokeWidth)
	case ShapeHexagon:
		w.drawHexagon(x, y, width, height, fill, stroke, strokeWidth)
	case ShapeStadium:
		w.drawStadium(x, y, width, height, fill, stroke, strokeWidth)
	case ShapeCylinder:
		w.drawCylinder(x, y, width, height, fill, stroke, strokeWidth)
	case ShapeParallelogram:
		w.drawParallelogram(x, y, width, height, false, fill, stroke, strokeWidth)
	case ShapeParallelogramAlt:
		w.drawParallelogram(x, y, width, height, true, fill, stroke, strokeWidth)
	case ShapeTrapezoid:
		w.drawTrapezoid(x, y, width, height, false, fill, stroke, strokeWidth)
	case ShapeTrapezoidAlt:
		w.drawTrapezoid(x, y, width, height, true, fill, stroke, strokeWidth)
	case ShapeSubroutine:
		w.drawSubroutine(x, y, width, height, fill, stroke, strokeWidth)
	case ShapeAsymmetric:
		w.drawAsymmetric(x, y, width, height, fill, stroke, strokeWidth)
	default: // Rectangle
		w.drawRect(x, y, width, height, fill, stroke, strokeWidth)
	}

	// Draw label
	textX := x + width/2
	textY := y + height/2
	w.buf.WriteString(fmt.Sprintf(
		`  <text x="%.1f" y="%.1f" class="node-text">%s</text>`,
		textX, textY, label,
	))
	w.buf.WriteString("\n")

	w.buf.WriteString("</g>\n")
}

func (w *SVGWriter) drawRect(x, y, width, height float64, fill, stroke string, strokeWidth float64) {
	w.buf.WriteString(fmt.Sprintf(
		`  <rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="%s" stroke="%s" stroke-width="%.1f"/>`,
		x, y, width, height, fill, stroke, strokeWidth,
	))
	w.buf.WriteString("\n")
}

func (w *SVGWriter) drawRoundedRect(x, y, width, height, radius float64, fill, stroke string, strokeWidth float64) {
	w.buf.WriteString(fmt.Sprintf(
		`  <rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" rx="%.1f" fill="%s" stroke="%s" stroke-width="%.1f"/>`,
		x, y, width, height, radius, fill, stroke, strokeWidth,
	))
	w.buf.WriteString("\n")
}

func (w *SVGWriter) drawCircle(cx, cy, radius float64, fill, stroke string, strokeWidth float64) {
	w.buf.WriteString(fmt.Sprintf(
		`  <circle cx="%.1f" cy="%.1f" r="%.1f" fill="%s" stroke="%s" stroke-width="%.1f"/>`,
		cx, cy, radius, fill, stroke, strokeWidth,
	))
	w.buf.WriteString("\n")
}

func (w *SVGWriter) drawDiamond(x, y, width, height float64, fill, stroke string, strokeWidth float64) {
	cx := x + width/2
	cy := y + height/2
	points := fmt.Sprintf("%.1f,%.1f %.1f,%.1f %.1f,%.1f %.1f,%.1f",
		cx, y,
		x+width, cy,
		cx, y+height,
		x, cy,
	)
	w.buf.WriteString(fmt.Sprintf(
		`  <polygon points="%s" fill="%s" stroke="%s" stroke-width="%.1f"/>`,
		points, fill, stroke, strokeWidth,
	))
	w.buf.WriteString("\n")
}

func (w *SVGWriter) drawHexagon(x, y, width, height float64, fill, stroke string, strokeWidth float64) {
	offset := width * 0.15
	points := fmt.Sprintf("%.1f,%.1f %.1f,%.1f %.1f,%.1f %.1f,%.1f %.1f,%.1f %.1f,%.1f",
		x+offset, y,
		x+width-offset, y,
		x+width, y+height/2,
		x+width-offset, y+height,
		x+offset, y+height,
		x, y+height/2,
	)
	w.buf.WriteString(fmt.Sprintf(
		`  <polygon points="%s" fill="%s" stroke="%s" stroke-width="%.1f"/>`,
		points, fill, stroke, strokeWidth,
	))
	w.buf.WriteString("\n")
}

func (w *SVGWriter) drawStadium(x, y, width, height float64, fill, stroke string, strokeWidth float64) {
	radius := height / 2
	w.buf.WriteString(fmt.Sprintf(
		`  <rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" rx="%.1f" ry="%.1f" fill="%s" stroke="%s" stroke-width="%.1f"/>`,
		x, y, width, height, radius, radius, fill, stroke, strokeWidth,
	))
	w.buf.WriteString("\n")
}

func (w *SVGWriter) drawCylinder(x, y, width, height float64, fill, stroke string, strokeWidth float64) {
	ellipseHeight := height * 0.15
	bodyHeight := height - ellipseHeight

	// Top ellipse
	w.buf.WriteString(fmt.Sprintf(
		`  <ellipse cx="%.1f" cy="%.1f" rx="%.1f" ry="%.1f" fill="%s" stroke="%s" stroke-width="%.1f"/>`,
		x+width/2, y+ellipseHeight/2, width/2, ellipseHeight/2, fill, stroke, strokeWidth,
	))
	w.buf.WriteString("\n")

	// Body rectangle
	w.buf.WriteString(fmt.Sprintf(
		`  <rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="%s" stroke="none"/>`,
		x, y+ellipseHeight/2, width, bodyHeight, fill,
	))
	w.buf.WriteString("\n")

	// Side lines
	w.buf.WriteString(fmt.Sprintf(
		`  <line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="%s" stroke-width="%.1f"/>`,
		x, y+ellipseHeight/2, x, y+height-ellipseHeight/2, stroke, strokeWidth,
	))
	w.buf.WriteString("\n")
	w.buf.WriteString(fmt.Sprintf(
		`  <line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="%s" stroke-width="%.1f"/>`,
		x+width, y+ellipseHeight/2, x+width, y+height-ellipseHeight/2, stroke, strokeWidth,
	))
	w.buf.WriteString("\n")

	// Bottom ellipse
	w.buf.WriteString(fmt.Sprintf(
		`  <ellipse cx="%.1f" cy="%.1f" rx="%.1f" ry="%.1f" fill="%s" stroke="%s" stroke-width="%.1f"/>`,
		x+width/2, y+height-ellipseHeight/2, width/2, ellipseHeight/2, fill, stroke, strokeWidth,
	))
	w.buf.WriteString("\n")
}

func (w *SVGWriter) drawParallelogram(x, y, width, height float64, alt bool, fill, stroke string, strokeWidth float64) {
	offset := width * 0.2
	var points string
	if alt {
		points = fmt.Sprintf("%.1f,%.1f %.1f,%.1f %.1f,%.1f %.1f,%.1f",
			x+offset, y,
			x+width, y,
			x+width-offset, y+height,
			x, y+height,
		)
	} else {
		points = fmt.Sprintf("%.1f,%.1f %.1f,%.1f %.1f,%.1f %.1f,%.1f",
			x, y,
			x+width-offset, y,
			x+width, y+height,
			x+offset, y+height,
		)
	}
	w.buf.WriteString(fmt.Sprintf(
		`  <polygon points="%s" fill="%s" stroke="%s" stroke-width="%.1f"/>`,
		points, fill, stroke, strokeWidth,
	))
	w.buf.WriteString("\n")
}

func (w *SVGWriter) drawTrapezoid(x, y, width, height float64, alt bool, fill, stroke string, strokeWidth float64) {
	offset := width * 0.15
	var points string
	if alt {
		points = fmt.Sprintf("%.1f,%.1f %.1f,%.1f %.1f,%.1f %.1f,%.1f",
			x, y,
			x+width, y,
			x+width-offset, y+height,
			x+offset, y+height,
		)
	} else {
		points = fmt.Sprintf("%.1f,%.1f %.1f,%.1f %.1f,%.1f %.1f,%.1f",
			x+offset, y,
			x+width-offset, y,
			x+width, y+height,
			x, y+height,
		)
	}
	w.buf.WriteString(fmt.Sprintf(
		`  <polygon points="%s" fill="%s" stroke="%s" stroke-width="%.1f"/>`,
		points, fill, stroke, strokeWidth,
	))
	w.buf.WriteString("\n")
}

func (w *SVGWriter) drawSubroutine(x, y, width, height float64, fill, stroke string, strokeWidth float64) {
	inset := 8.0
	// Main rectangle
	w.drawRect(x, y, width, height, fill, stroke, strokeWidth)
	// Left line
	w.buf.WriteString(fmt.Sprintf(
		`  <line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="%s" stroke-width="%.1f"/>`,
		x+inset, y, x+inset, y+height, stroke, strokeWidth,
	))
	w.buf.WriteString("\n")
	// Right line
	w.buf.WriteString(fmt.Sprintf(
		`  <line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="%s" stroke-width="%.1f"/>`,
		x+width-inset, y, x+width-inset, y+height, stroke, strokeWidth,
	))
	w.buf.WriteString("\n")
}

func (w *SVGWriter) drawAsymmetric(x, y, width, height float64, fill, stroke string, strokeWidth float64) {
	offset := width * 0.15
	points := fmt.Sprintf("%.1f,%.1f %.1f,%.1f %.1f,%.1f %.1f,%.1f %.1f,%.1f",
		x+offset, y,
		x+width, y,
		x+width, y+height,
		x+offset, y+height,
		x, y+height/2,
	)
	w.buf.WriteString(fmt.Sprintf(
		`  <polygon points="%s" fill="%s" stroke="%s" stroke-width="%.1f"/>`,
		points, fill, stroke, strokeWidth,
	))
	w.buf.WriteString("\n")
}

// DrawEdge renders an edge with its path and optional label.
func (w *SVGWriter) DrawEdge(edge *Edge) {
	if len(edge.Points) < 2 {
		return
	}

	stroke := w.theme.EdgeStroke
	strokeWidth := w.theme.EdgeStrokeWidth
	strokeDash := ""

	if edge.Style != nil {
		if edge.Style.Stroke != "" {
			stroke = edge.Style.Stroke
		}
		if edge.Style.StrokeWidth > 0 {
			strokeWidth = edge.Style.StrokeWidth
		}
		if edge.Style.StrokeDash != "" {
			strokeDash = edge.Style.StrokeDash
		}
	}

	// Apply edge type styling
	switch edge.Type {
	case EdgeDotted:
		strokeDash = "5,5"
	case EdgeThick:
		strokeWidth *= 2
	case EdgeInvisible:
		stroke = "none"
	}

	// Add arrow markers (prefixed with SVG ID to avoid collisions)
	markerStart := ""
	markerEnd := ""

	if edge.ArrowStart != ArrowNone {
		markerID := fmt.Sprintf("%s-arrow-start-%d", w.id, edge.ArrowStart)
		w.AddArrowMarker(markerID, edge.ArrowStart)
		markerStart = fmt.Sprintf(` marker-start="url(#%s)"`, markerID)
	}

	if edge.ArrowEnd != ArrowNone {
		markerID := fmt.Sprintf("%s-arrow-end-%d", w.id, edge.ArrowEnd)
		w.AddArrowMarker(markerID, edge.ArrowEnd)
		markerEnd = fmt.Sprintf(` marker-end="url(#%s)"`, markerID)
	}

	// Build path
	var pathData strings.Builder
	pathData.WriteString(fmt.Sprintf("M %.1f %.1f", edge.Points[0].X, edge.Points[0].Y))

	if len(edge.Points) == 2 {
		// Straight line
		pathData.WriteString(fmt.Sprintf(" L %.1f %.1f", edge.Points[1].X, edge.Points[1].Y))
	} else {
		// Use smooth curves through intermediate points
		for i := 1; i < len(edge.Points); i++ {
			pathData.WriteString(fmt.Sprintf(" L %.1f %.1f", edge.Points[i].X, edge.Points[i].Y))
		}
	}

	dashAttr := ""
	if strokeDash != "" {
		dashAttr = fmt.Sprintf(` stroke-dasharray="%s"`, strokeDash)
	}

	w.buf.WriteString(fmt.Sprintf(
		`<path d="%s" fill="none" stroke="%s" stroke-width="%.1f"%s%s%s/>`,
		pathData.String(), stroke, strokeWidth, dashAttr, markerStart, markerEnd,
	))
	w.buf.WriteString("\n")

	// Draw label if present
	if edge.Label != "" {
		labelPoint := Position{}
		if edge.HasLabelPosition {
			labelPoint = Position{X: edge.LabelX, Y: edge.LabelY}
		} else {
			// Position label at midpoint
			midIdx := len(edge.Points) / 2
			labelPoint = edge.Points[midIdx]
		}

		// Add background for readability
		label := html.EscapeString(edge.Label)
		labelWidth := float64(len(edge.Label)) * 7
		labelHeight := 16.0

		w.buf.WriteString(fmt.Sprintf(
			`<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="%s" rx="3"/>`,
			labelPoint.X-labelWidth/2-4, labelPoint.Y-labelHeight/2,
			labelWidth+8, labelHeight,
			w.theme.NodeFill,
		))
		w.buf.WriteString("\n")

		w.buf.WriteString(fmt.Sprintf(
			`<text x="%.1f" y="%.1f" class="edge-text">%s</text>`,
			labelPoint.X, labelPoint.Y, label,
		))
		w.buf.WriteString("\n")
	}
}

// DrawSubgraph renders a subgraph container.
func (w *SVGWriter) DrawSubgraph(sg *Subgraph) {
	x := sg.Position.X
	y := sg.Position.Y
	width := sg.Size.Width
	height := sg.Size.Height
	label := html.EscapeString(sg.Label)

	fill := "rgba(0,0,0,0.1)"
	stroke := w.theme.NodeStroke

	if sg.Style != nil {
		if sg.Style.Fill != "" {
			fill = sg.Style.Fill
		}
		if sg.Style.Stroke != "" {
			stroke = sg.Style.Stroke
		}
	}

	w.buf.WriteString(fmt.Sprintf(`<g id="subgraph-%s">`, html.EscapeString(sg.ID)))
	w.buf.WriteString("\n")

	// Draw container
	w.buf.WriteString(fmt.Sprintf(
		`  <rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="%s" stroke="%s" stroke-width="1" stroke-dasharray="4,2" rx="4"/>`,
		x, y, width, height, fill, stroke,
	))
	w.buf.WriteString("\n")

	// Draw label at top
	if label != "" {
		w.buf.WriteString(fmt.Sprintf(
			`  <text x="%.1f" y="%.1f" class="node-text" style="font-size: 12px; text-anchor: start;">%s</text>`,
			x+8, y+16, label,
		))
		w.buf.WriteString("\n")
	}

	w.buf.WriteString("</g>\n")
}

// DrawParticipant renders a sequence diagram participant.
func (w *SVGWriter) DrawParticipant(p *Participant, x, y, width, height, lifelineEnd float64) {
	label := p.Alias
	if label == "" {
		label = p.ID
	}
	label = html.EscapeString(label)

	fill := w.theme.NodeFill
	stroke := w.theme.NodeStroke

	w.buf.WriteString(fmt.Sprintf(`<g id="participant-%s">`, html.EscapeString(p.ID)))
	w.buf.WriteString("\n")

	// Draw participant box
	if p.Type == ParticipantActor {
		// Draw stick figure for actors
		cx := x + width/2
		headRadius := 10.0
		bodyTop := y + headRadius*2 + 5
		bodyBottom := y + height - 10

		// Head
		w.drawCircle(cx, y+headRadius+5, headRadius, "none", stroke, w.theme.NodeStrokeWidth)
		// Body
		w.buf.WriteString(fmt.Sprintf(
			`  <line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="%s" stroke-width="%.1f"/>`,
			cx, bodyTop, cx, bodyBottom, stroke, w.theme.NodeStrokeWidth,
		))
		w.buf.WriteString("\n")
		// Arms
		w.buf.WriteString(fmt.Sprintf(
			`  <line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="%s" stroke-width="%.1f"/>`,
			cx-15, bodyTop+10, cx+15, bodyTop+10, stroke, w.theme.NodeStrokeWidth,
		))
		w.buf.WriteString("\n")
		// Legs
		w.buf.WriteString(fmt.Sprintf(
			`  <line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="%s" stroke-width="%.1f"/>`,
			cx, bodyBottom, cx-12, bodyBottom+15, stroke, w.theme.NodeStrokeWidth,
		))
		w.buf.WriteString("\n")
		w.buf.WriteString(fmt.Sprintf(
			`  <line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="%s" stroke-width="%.1f"/>`,
			cx, bodyBottom, cx+12, bodyBottom+15, stroke, w.theme.NodeStrokeWidth,
		))
		w.buf.WriteString("\n")
	} else {
		w.drawRoundedRect(x, y, width, height, 4, fill, stroke, w.theme.NodeStrokeWidth)
	}

	// Draw label
	w.buf.WriteString(fmt.Sprintf(
		`  <text x="%.1f" y="%.1f" class="participant-text">%s</text>`,
		x+width/2, y+height/2, label,
	))
	w.buf.WriteString("\n")

	// Draw lifeline
	lifelineX := x + width/2
	lifelineY := y + height
	w.buf.WriteString(fmt.Sprintf(
		`  <line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="%s" stroke-width="1" stroke-dasharray="4,4"/>`,
		lifelineX, lifelineY, lifelineX, lifelineEnd, w.theme.EdgeStroke,
	))
	w.buf.WriteString("\n")

	w.buf.WriteString("</g>\n")
}

// DrawMessage renders a sequence diagram message.
func (w *SVGWriter) DrawMessage(msg *Message, fromX, toX, y float64) {
	stroke := w.theme.EdgeStroke
	strokeWidth := w.theme.EdgeStrokeWidth
	strokeDash := ""

	// Determine arrow style
	switch msg.Type {
	case MessageDotted, MessageDottedArrow, MessageDottedCross, MessageDottedOpen:
		strokeDash = "5,5"
	}

	// Draw the line
	dashAttr := ""
	if strokeDash != "" {
		dashAttr = fmt.Sprintf(` stroke-dasharray="%s"`, strokeDash)
	}

	// Add arrow marker (prefixed with SVG ID to avoid collisions)
	markerID := fmt.Sprintf("%s-msg-arrow", w.id)
	switch msg.Type {
	case MessageSolidCross, MessageDottedCross:
		markerID = fmt.Sprintf("%s-msg-cross", w.id)
		w.AddArrowMarker(markerID, ArrowCross)
	case MessageSolidOpen, MessageDottedOpen:
		markerID = fmt.Sprintf("%s-msg-open", w.id)
		// Open arrow is just no marker
	default:
		w.AddArrowMarker(markerID, ArrowNormal)
	}

	markerAttr := fmt.Sprintf(` marker-end="url(#%s)"`, markerID)
	if msg.Type == MessageSolidOpen || msg.Type == MessageDottedOpen {
		markerAttr = ""
	}

	w.buf.WriteString(fmt.Sprintf(
		`<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="%s" stroke-width="%.1f"%s%s/>`,
		fromX, y, toX, y, stroke, strokeWidth, dashAttr, markerAttr,
	))
	w.buf.WriteString("\n")

	// Draw label
	if msg.Label != "" {
		label := html.EscapeString(msg.Label)
		midX := (fromX + toX) / 2
		w.buf.WriteString(fmt.Sprintf(
			`<text x="%.1f" y="%.1f" class="edge-text">%s</text>`,
			midX, y-8, label,
		))
		w.buf.WriteString("\n")
	}
}

// DrawNote renders a sequence diagram note.
func (w *SVGWriter) DrawNote(note *Note, x, y, width, height float64) {
	fill := "#fffacd" // Light yellow
	stroke := w.theme.NodeStroke
	foldSize := 10.0

	// Draw note shape with folded corner
	path := fmt.Sprintf(
		"M %.1f %.1f L %.1f %.1f L %.1f %.1f L %.1f %.1f L %.1f %.1f Z",
		x, y,
		x+width-foldSize, y,
		x+width, y+foldSize,
		x+width, y+height,
		x, y+height,
	)

	w.buf.WriteString(fmt.Sprintf(
		`<path d="%s" fill="%s" stroke="%s" stroke-width="1"/>`,
		path, fill, stroke,
	))
	w.buf.WriteString("\n")

	// Draw fold
	foldPath := fmt.Sprintf(
		"M %.1f %.1f L %.1f %.1f L %.1f %.1f",
		x+width-foldSize, y,
		x+width-foldSize, y+foldSize,
		x+width, y+foldSize,
	)
	w.buf.WriteString(fmt.Sprintf(
		`<path d="%s" fill="none" stroke="%s" stroke-width="1"/>`,
		foldPath, stroke,
	))
	w.buf.WriteString("\n")

	// Draw text
	text := html.EscapeString(note.Text)
	w.buf.WriteString(fmt.Sprintf(
		`<text x="%.1f" y="%.1f" class="edge-text" style="fill: #333;">%s</text>`,
		x+width/2, y+height/2, text,
	))
	w.buf.WriteString("\n")
}

// DrawActivation renders an activation bar on a lifeline.
func (w *SVGWriter) DrawActivation(x, startY, endY, width float64) {
	w.buf.WriteString(fmt.Sprintf(
		`<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="%s" stroke="%s" stroke-width="1"/>`,
		x-width/2, startY, width, endY-startY, w.theme.NodeFill, w.theme.NodeStroke,
	))
	w.buf.WriteString("\n")
}
