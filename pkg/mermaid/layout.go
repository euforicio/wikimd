package mermaid

import (
	"fmt"
	"math"
	"strings"

	"github.com/euforicio/wikimd/pkg/dagre"
)

// LayoutEngine defines the interface for diagram layout algorithms.
type LayoutEngine interface {
	Layout(diagram *Diagram) error
}

// LayoutOptions configures the layout algorithm.
type LayoutOptions struct {
	NodeWidth   float64
	NodeHeight  float64
	NodePadding float64
	RankSep     float64 // Vertical separation between ranks
	NodeSep     float64 // Horizontal separation between nodes
	EdgeSep     float64 // Separation between edges
	MarginX     float64
	MarginY     float64
}

// DefaultLayoutOptions returns sensible defaults.
func DefaultLayoutOptions() *LayoutOptions {
	return &LayoutOptions{
		NodeWidth:   120,
		NodeHeight:  40,
		NodePadding: 10,
		RankSep:     60,
		NodeSep:     40,
		EdgeSep:     10,
		MarginX:     40,
		MarginY:     40,
	}
}

// DagreLayout implements a Sugiyama-style layered layout algorithm
// using the full dagre port with network simplex ranking.
type DagreLayout struct {
	opts *LayoutOptions
}

// NewDagreLayout creates a new Dagre-style layout engine.
func NewDagreLayout(opts *LayoutOptions) *DagreLayout {
	if opts == nil {
		opts = DefaultLayoutOptions()
	}
	return &DagreLayout{opts: opts}
}

// Layout performs the layered layout algorithm on a diagram.
func (d *DagreLayout) Layout(diagram *Diagram) error {
	// Convert diagram to dagre graph
	g, err := d.diagramToGraph(diagram)
	if err != nil {
		return err
	}

	// Run dagre layout
	dagreOpts := &dagre.LayoutOptions{
		RankDir:   directionToRankDir(diagram.Direction),
		NodeSep:   d.opts.NodeSep,
		EdgeSep:   d.opts.EdgeSep,
		RankSep:   d.opts.RankSep,
		MarginX:   d.opts.MarginX,
		MarginY:   d.opts.MarginY,
		Ranker:    "network-simplex",
		Acyclicer: "dfs",
	}
	if err := dagre.Layout(g, dagreOpts); err != nil {
		return err
	}

	// Transfer results back to diagram
	d.graphToDiagram(g, diagram)

	return nil
}

// diagramToGraph converts a mermaid Diagram to a dagre Graph.
func (d *DagreLayout) diagramToGraph(diagram *Diagram) (*dagre.Graph, error) {
	g := dagre.NewGraphWithOptions(dagre.GraphOptions{
		Directed:   true,
		Multigraph: true,
		Compound:   true,
	})
	subgraphsByID := make(map[string]*Subgraph, len(diagram.Subgraphs))

	// Add subgraph nodes first so node parent assignments can reference them.
	for _, sg := range diagram.Subgraphs {
		if sg.ID == "" {
			continue
		}
		subgraphsByID[sg.ID] = sg
		size := sg.Size
		if size.Width == 0 {
			size.Width = math.Max(120, float64(len(sg.Label))*7+24)
		}
		if size.Height == 0 {
			size.Height = 60
		}
		g.SetNode(sg.ID, &dagre.NodeData{
			Label:  sg.Label,
			Width:  size.Width,
			Height: size.Height,
		})
	}

	// Add nodes
	for _, node := range diagram.Nodes {
		if _, isSubgraph := subgraphsByID[node.ID]; isSubgraph {
			continue
		}
		size := node.Size
		if size.Width == 0 || size.Height == 0 {
			size = calculateNodeSizeStatic(node, d.opts)
		}

		g.SetNode(node.ID, &dagre.NodeData{
			Label:  node.Label,
			Width:  size.Width,
			Height: size.Height,
		})
	}
	for _, sg := range diagram.Subgraphs {
		if sg.ID == "" {
			continue
		}
		if sg.ParentID != "" && g.HasNode(sg.ParentID) {
			if err := g.SetParent(sg.ID, sg.ParentID); err != nil {
				return nil, err
			}
		}
		for _, memberID := range sg.Nodes {
			if memberID == sg.ID || !g.HasNode(memberID) {
				continue
			}
			if err := g.SetParent(memberID, sg.ID); err != nil {
				return nil, err
			}
		}
	}

	// Add edges
	for i, edge := range diagram.Edges {
		ed := &dagre.EdgeData{
			Label:       edge.Label,
			MinLen:      1,
			Weight:      1,
			LabelOffset: 10,
		}

		// Calculate label size if present
		if edge.Label != "" {
			ed.Width = float64(len(edge.Label))*7 + 16
			ed.Height = 20
		}

		edgeName := fmt.Sprintf("e%d", i)
		g.SetEdgeWithName(edge.From, edge.To, edgeName, ed)
	}

	return g, nil
}

// graphToDiagram transfers layout results from dagre Graph to mermaid Diagram.
func (d *DagreLayout) graphToDiagram(g *dagre.Graph, diagram *Diagram) {
	// Create node map for quick lookup
	nodeMap := make(map[string]*Node)
	for _, node := range diagram.Nodes {
		nodeMap[node.ID] = node
	}

	// Transfer node positions
	for _, v := range g.Nodes() {
		n := g.Node(v)
		if n == nil {
			continue
		}

		node := nodeMap[v]
		if node == nil {
			continue
		}

		// dagre uses center coordinates, convert to top-left
		node.Position = Position{
			X: n.X - n.Width/2,
			Y: n.Y - n.Height/2,
		}
		node.Size = Size{
			Width:  n.Width,
			Height: n.Height,
		}
	}

	// Transfer compound/subgraph bounds from dagre layout results.
	for _, sg := range diagram.Subgraphs {
		if sg.ID == "" {
			continue
		}
		if n := g.Node(sg.ID); n != nil && n.Width > 0 && n.Height > 0 {
			sg.Position = Position{
				X: n.X - n.Width/2,
				Y: n.Y - n.Height/2,
			}
			sg.Size = Size{
				Width:  n.Width,
				Height: n.Height,
			}
		}
	}

	// Transfer edge routes
	for i, edge := range diagram.Edges {
		ed := g.EdgeWithName(edge.From, edge.To, fmt.Sprintf("e%d", i))
		if ed == nil {
			ed = g.Edge(edge.From, edge.To)
		}
		if ed == nil || len(ed.Points) == 0 {
			// Create simple route
			edge.Points = d.createSimpleRoute(nodeMap, edge, diagram.Direction)
			if edge.Label != "" && len(edge.Points) > 0 {
				mid := edge.Points[len(edge.Points)/2]
				edge.LabelX = mid.X
				edge.LabelY = mid.Y
				edge.HasLabelPosition = true
			}
			continue
		}

		// Convert dagre points to mermaid positions
		edge.Points = make([]Position, len(ed.Points))
		for i, p := range ed.Points {
			edge.Points[i] = Position{X: p.X, Y: p.Y}
		}
		edge.LabelX = ed.X
		edge.LabelY = ed.Y
		edge.HasLabelPosition = ed.HasLabelPosition || ed.X != 0 || ed.Y != 0
	}

	// Fallback subgraph bounds from contained nodes when dagre did not set dimensions.
	for _, sg := range diagram.Subgraphs {
		if sg.Size.Width == 0 || sg.Size.Height == 0 {
			calculateSubgraphBounds(sg, nodeMap)
		}
	}
}

// createSimpleRoute creates a simple edge route between two nodes.
func (d *DagreLayout) createSimpleRoute(nodeMap map[string]*Node, edge *Edge, direction Direction) []Position {
	fromNode := nodeMap[edge.From]
	toNode := nodeMap[edge.To]

	if fromNode == nil || toNode == nil {
		return nil
	}

	var startPoint, endPoint Position

	switch direction {
	case DirectionLR:
		startPoint = Position{
			X: fromNode.Position.X + fromNode.Size.Width,
			Y: fromNode.Position.Y + fromNode.Size.Height/2,
		}
		endPoint = Position{
			X: toNode.Position.X,
			Y: toNode.Position.Y + toNode.Size.Height/2,
		}
	case DirectionRL:
		startPoint = Position{
			X: fromNode.Position.X,
			Y: fromNode.Position.Y + fromNode.Size.Height/2,
		}
		endPoint = Position{
			X: toNode.Position.X + toNode.Size.Width,
			Y: toNode.Position.Y + toNode.Size.Height/2,
		}
	case DirectionBT:
		startPoint = Position{
			X: fromNode.Position.X + fromNode.Size.Width/2,
			Y: fromNode.Position.Y,
		}
		endPoint = Position{
			X: toNode.Position.X + toNode.Size.Width/2,
			Y: toNode.Position.Y + toNode.Size.Height,
		}
	default: // TB
		startPoint = Position{
			X: fromNode.Position.X + fromNode.Size.Width/2,
			Y: fromNode.Position.Y + fromNode.Size.Height,
		}
		endPoint = Position{
			X: toNode.Position.X + toNode.Size.Width/2,
			Y: toNode.Position.Y,
		}
	}

	// Check if we need intermediate points
	dx := endPoint.X - startPoint.X
	dy := endPoint.Y - startPoint.Y

	switch direction {
	case DirectionLR, DirectionRL:
		if math.Abs(dy) < 1 {
			return []Position{startPoint, endPoint}
		}
		midX := (startPoint.X + endPoint.X) / 2
		return []Position{
			startPoint,
			{X: midX, Y: startPoint.Y},
			{X: midX, Y: endPoint.Y},
			endPoint,
		}
	default: // TB, BT
		if math.Abs(dx) < 1 {
			return []Position{startPoint, endPoint}
		}
		midY := (startPoint.Y + endPoint.Y) / 2
		return []Position{
			startPoint,
			{X: startPoint.X, Y: midY},
			{X: endPoint.X, Y: midY},
			endPoint,
		}
	}
}

// SequenceLayout implements layout for sequence diagrams.
type SequenceLayout struct {
	opts         *SequenceLayoutOptions
	diagram      *Diagram
	participantX map[string]float64
	currentY     float64
	messageIndex int
}

// SequenceLayoutOptions configures sequence diagram layout.
type SequenceLayoutOptions struct {
	ParticipantWidth  float64
	ParticipantHeight float64
	ParticipantGap    float64
	MessageHeight     float64
	ActivationWidth   float64
	NoteWidth         float64
	NotePadding       float64
	MarginX           float64
	MarginY           float64
	LifelineExtend    float64
}

// DefaultSequenceLayoutOptions returns sensible defaults.
func DefaultSequenceLayoutOptions() *SequenceLayoutOptions {
	return &SequenceLayoutOptions{
		ParticipantWidth:  100,
		ParticipantHeight: 40,
		ParticipantGap:    80,
		MessageHeight:     40,
		ActivationWidth:   10,
		NoteWidth:         120,
		NotePadding:       10,
		MarginX:           40,
		MarginY:           40,
		LifelineExtend:    20,
	}
}

// NewSequenceLayout creates a new sequence diagram layout engine.
func NewSequenceLayout(opts *SequenceLayoutOptions) *SequenceLayout {
	if opts == nil {
		opts = DefaultSequenceLayoutOptions()
	}
	return &SequenceLayout{opts: opts}
}

// Layout performs sequence diagram layout.
func (s *SequenceLayout) Layout(diagram *Diagram) error {
	s.diagram = diagram
	s.participantX = make(map[string]float64)

	s.layoutParticipants()
	s.layoutMessages()

	return nil
}

func (s *SequenceLayout) layoutParticipants() {
	x := s.opts.MarginX

	for _, p := range s.diagram.Participants {
		s.participantX[p.ID] = x
		x += s.opts.ParticipantWidth + s.opts.ParticipantGap
	}
}

func (s *SequenceLayout) layoutMessages() {
	s.currentY = s.opts.MarginY + s.opts.ParticipantHeight + 20
	s.messageIndex = 0

	activeActivations := make(map[string]int)

	for i, msg := range s.diagram.Messages {
		s.currentY += s.opts.MessageHeight
		s.messageIndex = i

		if msg.Activate {
			actIdx := s.findNextInactiveActivation(msg.To)
			if actIdx >= 0 {
				s.diagram.Activations[actIdx].StartY = s.currentY
				activeActivations[msg.To] = actIdx
			}
		}

		if msg.Deactivate {
			if actIdx, ok := activeActivations[msg.To]; ok {
				if actIdx < len(s.diagram.Activations) {
					s.diagram.Activations[actIdx].EndY = s.currentY
				}
				delete(activeActivations, msg.To)
			}
		}
	}

	s.processExplicitActivations()

	for range s.diagram.Notes {
		s.currentY += s.opts.MessageHeight
	}

	s.currentY += s.opts.LifelineExtend

	for _, act := range s.diagram.Activations {
		if act.StartY >= 0 && act.EndY < 0 {
			act.EndY = s.currentY
		}
	}
}

func (s *SequenceLayout) findNextInactiveActivation(participant string) int {
	for i, act := range s.diagram.Activations {
		if act.Participant == participant && act.StartY < 0 {
			return i
		}
	}
	act := &Activation{
		Participant: participant,
		StartY:      -1,
		EndY:        -1,
	}
	s.diagram.Activations = append(s.diagram.Activations, act)
	return len(s.diagram.Activations) - 1
}

func (s *SequenceLayout) processExplicitActivations() {
	if len(s.diagram.Activations) == 0 {
		return
	}

	messageY := s.opts.MarginY + s.opts.ParticipantHeight + 20

	for _, act := range s.diagram.Activations {
		if act.StartMsgIndex >= 0 && act.StartY == -1 {
			act.StartY = messageY + float64(act.StartMsgIndex)*s.opts.MessageHeight
		}

		if act.EndMsgIndex >= 0 && act.EndY == -2 {
			act.EndY = messageY + float64(act.EndMsgIndex)*s.opts.MessageHeight
		}
	}
}

// GetParticipantX returns the X coordinate for a participant.
func (s *SequenceLayout) GetParticipantX(id string) float64 {
	if x, ok := s.participantX[id]; ok {
		return x + s.opts.ParticipantWidth/2
	}
	return 0
}

// GetDiagramSize returns the total diagram dimensions.
func (s *SequenceLayout) GetDiagramSize() Size {
	width := s.opts.MarginX*2 + float64(len(s.diagram.Participants))*(s.opts.ParticipantWidth+s.opts.ParticipantGap)
	if len(s.diagram.Participants) > 0 {
		width -= s.opts.ParticipantGap
	}
	return Size{Width: width, Height: s.currentY + s.opts.MarginY}
}

// GetDagreLayoutSize returns the total diagram dimensions after layout.
func GetDagreLayoutSize(diagram *Diagram, opts *LayoutOptions) Size {
	if opts == nil {
		opts = DefaultLayoutOptions()
	}

	maxX := 0.0
	maxY := 0.0

	for _, node := range diagram.Nodes {
		right := node.Position.X + node.Size.Width
		bottom := node.Position.Y + node.Size.Height
		if right > maxX {
			maxX = right
		}
		if bottom > maxY {
			maxY = bottom
		}
	}

	for _, edge := range diagram.Edges {
		if edge.Label == "" || len(edge.Points) < 2 {
			continue
		}
		midIdx := len(edge.Points) / 2
		midPoint := edge.Points[midIdx]

		labelWidth := float64(len(edge.Label))*7 + 16
		labelHeight := 20.0

		labelRight := midPoint.X + labelWidth/2
		labelBottom := midPoint.Y + labelHeight/2
		if labelRight > maxX {
			maxX = labelRight
		}
		if labelBottom > maxY {
			maxY = labelBottom
		}
	}

	for _, sg := range diagram.Subgraphs {
		right := sg.Position.X + sg.Size.Width
		bottom := sg.Position.Y + sg.Size.Height
		if right > maxX {
			maxX = right
		}
		if bottom > maxY {
			maxY = bottom
		}
	}

	return Size{
		Width:  maxX + opts.MarginX,
		Height: maxY + opts.MarginY,
	}
}

// FullDagreLayout implements the dagre-style layout algorithm.
// This is an alias to DagreLayout for backwards compatibility.
type FullDagreLayout struct {
	*DagreLayout
}

// NewFullDagreLayout creates a new dagre layout engine.
func NewFullDagreLayout(opts *LayoutOptions) *FullDagreLayout {
	return &FullDagreLayout{DagreLayout: NewDagreLayout(opts)}
}

// calculateSubgraphBounds computes subgraph bounds from its children.
func calculateSubgraphBounds(sg *Subgraph, nodeMap map[string]*Node) {
	if len(sg.Nodes) == 0 {
		return
	}

	minX, minY := math.MaxFloat64, math.MaxFloat64
	maxX, maxY := -math.MaxFloat64, -math.MaxFloat64

	for _, nodeID := range sg.Nodes {
		node := nodeMap[nodeID]
		if node == nil {
			continue
		}

		left := node.Position.X
		right := node.Position.X + node.Size.Width
		top := node.Position.Y
		bottom := node.Position.Y + node.Size.Height

		if left < minX {
			minX = left
		}
		if right > maxX {
			maxX = right
		}
		if top < minY {
			minY = top
		}
		if bottom > maxY {
			maxY = bottom
		}
	}

	padding := 20.0
	labelHeight := 20.0

	sg.Position = Position{
		X: minX - padding,
		Y: minY - padding - labelHeight,
	}
	sg.Size = Size{
		Width:  maxX - minX + padding*2,
		Height: maxY - minY + padding*2 + labelHeight,
	}
}

// directionToRankDir converts a mermaid Direction to dagre RankDir string.
func directionToRankDir(d Direction) string {
	switch d {
	case DirectionBT:
		return "BT"
	case DirectionLR:
		return "LR"
	case DirectionRL:
		return "RL"
	default:
		return "TB"
	}
}

// calculateNodeSizeStatic estimates node dimensions based on label text.
func calculateNodeSizeStatic(node *Node, opts *LayoutOptions) Size {
	label := node.Label
	if label == "" {
		label = node.ID
	}

	lines := strings.Split(label, "\n")
	maxLineLen := 0
	for _, line := range lines {
		if len(line) > maxLineLen {
			maxLineLen = len(line)
		}
	}

	charWidth := 7.2
	lineHeight := 18.0
	padding := opts.NodePadding * 2

	width := float64(maxLineLen)*charWidth + padding
	if width < opts.NodeWidth {
		width = opts.NodeWidth
	}

	height := lineHeight*float64(len(lines)) + padding
	if height < opts.NodeHeight {
		height = opts.NodeHeight
	}

	switch node.Shape {
	case ShapeCircle, ShapeDoubleCircle:
		diameter := math.Max(width, height)
		return Size{Width: diameter, Height: diameter}
	case ShapeRhombus:
		return Size{Width: width * 1.4, Height: height * 1.4}
	case ShapeHexagon:
		return Size{Width: width * 1.2, Height: height}
	default:
		return Size{Width: width, Height: height}
	}
}
