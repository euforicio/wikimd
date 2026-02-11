// Package mermaid provides a native Go Mermaid diagram renderer.
// It parses Mermaid syntax and renders diagrams to SVG without
// requiring JavaScript or Node.js.
package mermaid

// DiagramType identifies the type of Mermaid diagram.
type DiagramType int

// DiagramType values.
const (
	DiagramUnknown DiagramType = iota
	DiagramFlowchart
	DiagramSequence
	DiagramClass
	DiagramState
	DiagramER
	DiagramGantt
	DiagramPie
)

const diagramFlowchartName = "flowchart"

func (dt DiagramType) String() string {
	switch dt {
	case DiagramFlowchart:
		return diagramFlowchartName
	case DiagramSequence:
		return "sequenceDiagram"
	case DiagramClass:
		return "classDiagram"
	case DiagramState:
		return "stateDiagram"
	case DiagramER:
		return "erDiagram"
	case DiagramGantt:
		return "gantt"
	case DiagramPie:
		return "pie"
	default:
		return "unknown"
	}
}

// Direction specifies the layout direction for flowcharts.
type Direction int

// Direction values.
const (
	DirectionTB Direction = iota // Top to bottom (default)
	DirectionBT                  // Bottom to top
	DirectionLR                  // Left to right
	DirectionRL                  // Right to left
)

func (d Direction) String() string {
	switch d {
	case DirectionTB:
		return "TB"
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

// NodeShape defines the shape of a flowchart node.
type NodeShape int

// NodeShape values.
const (
	ShapeRectangle        NodeShape = iota // [text]
	ShapeRounded                           // (text)
	ShapeStadium                           // ([text])
	ShapeSubroutine                        // [[text]]
	ShapeCylinder                          // [(text)]
	ShapeCircle                            // ((text))
	ShapeAsymmetric                        // >text]
	ShapeRhombus                           // {text}
	ShapeHexagon                           // {{text}}
	ShapeParallelogram                     // [/text/]
	ShapeParallelogramAlt                  // [\text\]
	ShapeTrapezoid                         // [/text\]
	ShapeTrapezoidAlt                      // [\text/]
	ShapeDoubleCircle                      // (((text)))
)

func (s NodeShape) String() string {
	shapes := []string{
		"rectangle", "rounded", "stadium", "subroutine", "cylinder",
		"circle", "asymmetric", "rhombus", "hexagon", "parallelogram",
		"parallelogramAlt", "trapezoid", "trapezoidAlt", "doubleCircle",
	}
	if int(s) < len(shapes) {
		return shapes[s]
	}
	return "rectangle"
}

// EdgeType defines the type of connection between nodes.
type EdgeType int

// EdgeType values.
const (
	EdgeSolid     EdgeType = iota // ---
	EdgeDotted                    // -.-
	EdgeThick                     // ===
	EdgeInvisible                 // ~~~
)

// ArrowType defines the arrowhead style.
type ArrowType int

// ArrowType values.
const (
	ArrowNone   ArrowType = iota // No arrow
	ArrowNormal                  // >
	ArrowCircle                  // o
	ArrowCross                   // x
)

// Diagram represents a parsed Mermaid diagram.
type Diagram struct {
	Type      DiagramType
	Direction Direction
	Title     string
	Nodes     []*Node
	Edges     []*Edge
	Subgraphs []*Subgraph

	// Sequence diagram specific
	Participants []*Participant
	Messages     []*Message
	Activations  []*Activation
	Notes        []*Note
	Loops        []*LoopBlock
	AltBlocks    []*AltBlock

	// Class diagram specific
	Classes       []*Class
	Interfaces    []*Interface
	Relationships []*Relationship

	// State diagram specific
	States      []*State
	Transitions []*Transition

	// ER diagram specific
	Entities    []*Entity
	ERRelations []*ERRelation

	// Pie chart specific
	PieSlices []*PieSlice
	ShowData  bool // Whether to show percentages
}

// Node represents a node in a flowchart.
type Node struct {
	ID       string
	Label    string
	Shape    NodeShape
	Style    *NodeStyle
	Position Position // Set after layout
	Size     Size     // Set after layout
}

// NodeStyle defines visual styling for a node.
type NodeStyle struct {
	Fill        string
	Stroke      string
	StrokeWidth float64
	FontSize    float64
	FontFamily  string
	Color       string
}

// Edge represents a connection between nodes.
type Edge struct {
	From       string
	To         string
	Label      string
	Type       EdgeType
	ArrowStart ArrowType
	ArrowEnd   ArrowType
	Style      *EdgeStyle
	Points     []Position // Set after layout/routing
	LabelX     float64
	LabelY     float64
	// HasLabelPosition reports whether LabelX/LabelY were explicitly set by layout.
	HasLabelPosition bool
}

// EdgeStyle defines visual styling for an edge.
type EdgeStyle struct {
	Stroke      string
	StrokeWidth float64
	StrokeDash  string
}

// Subgraph represents a grouped set of nodes.
type Subgraph struct {
	ID        string
	Label     string
	ParentID  string
	Nodes     []string // Node IDs contained in this subgraph
	Direction Direction
	Style     *NodeStyle
	Position  Position
	Size      Size
}

// Position represents x,y coordinates.
type Position struct {
	X float64
	Y float64
}

// Size represents width and height.
type Size struct {
	Width  float64
	Height float64
}

// Participant represents an actor/participant in a sequence diagram.
type Participant struct {
	ID    string
	Alias string
	Type  ParticipantType
}

// ParticipantType distinguishes actors from participants.
type ParticipantType int

// ParticipantType values.
const (
	ParticipantDefault ParticipantType = iota
	ParticipantActor
)

// Message represents a message in a sequence diagram.
type Message struct {
	From       string
	To         string
	Label      string
	Type       MessageType
	Activate   bool
	Deactivate bool
}

// MessageType defines the arrow style for sequence messages.
type MessageType int

// MessageType values.
const (
	MessageSolid       MessageType = iota // ->
	MessageDotted                         // -->
	MessageSolidArrow                     // ->>
	MessageDottedArrow                    // -->>
	MessageSolidCross                     // -x
	MessageDottedCross                    // --x
	MessageSolidOpen                      // -)
	MessageDottedOpen                     // --)
)

// Activation represents an activation bar on a lifeline.
type Activation struct {
	Participant   string
	StartY        float64
	EndY          float64
	StartMsgIndex int // Message index when activation starts (-1 if not set)
	EndMsgIndex   int // Message index when activation ends (-1 if not set)
}

// Note represents a note in a sequence diagram.
type Note struct {
	Position     NotePosition
	Participant  string // For single participant notes
	Participant2 string // For notes over two participants
	Text         string
}

// NotePosition defines where a note appears.
type NotePosition int

// NotePosition values.
const (
	NoteRightOf NotePosition = iota
	NoteLeftOf
	NoteOver
)

// LoopBlock represents a loop fragment in a sequence diagram.
type LoopBlock struct {
	Label    string
	Messages []*Message
}

// AltBlock represents an alt/else fragment in a sequence diagram.
type AltBlock struct {
	Condition string
	Messages  []*Message
	Else      *AltBlock
}

// Class represents a class in a class diagram.
type Class struct {
	Name       string
	Annotation string // e.g., <<interface>>, <<abstract>>
	Attributes []ClassMember
	Methods    []ClassMember
}

// ClassMember represents an attribute or method.
type ClassMember struct {
	Visibility Visibility
	Name       string
	Type       string
	Parameters string // For methods
}

// Visibility defines member visibility.
type Visibility int

// Visibility values.
const (
	VisibilityPublic    Visibility = iota // +
	VisibilityPrivate                     // -
	VisibilityProtected                   // #
	VisibilityPackage                     // ~
)

// Interface represents an interface in a class diagram.
type Interface struct {
	Name    string
	Methods []ClassMember
}

// Relationship represents a relationship between classes.
type Relationship struct {
	From  string
	To    string
	Label string
	Type  RelationType
}

// RelationType defines the type of class relationship.
type RelationType int

// RelationType values.
const (
	RelationInheritance RelationType = iota // <|--
	RelationComposition                     // *--
	RelationAggregation                     // o--
	RelationAssociation                     // -->
	RelationDependency                      // ..>
	RelationRealization                     // ..|>
)

// State represents a state in a state diagram.
type State struct {
	ID          string
	Label       string
	Type        StateType
	SubStates   []*State
	Description string
}

// StateType defines special state types.
type StateType int

// StateType values.
const (
	StateNormal StateType = iota
	StateStart            // [*]
	StateEnd              // [*]
	StateFork
	StateJoin
	StateChoice
	StateComposite
)

const stateMarker = "[*]"

// Transition represents a state transition.
type Transition struct {
	From  string
	To    string
	Label string
	Guard string
}

// Entity represents an entity in an ER diagram.
type Entity struct {
	Name       string
	Attributes []ERAttribute
}

// ERAttribute represents an entity attribute.
type ERAttribute struct {
	Type    string
	Name    string
	Key     ERKeyType
	Comment string
}

// ERKeyType defines attribute key types.
type ERKeyType int

// ERKeyType values.
const (
	KeyNone    ERKeyType = iota
	KeyPrimary           // PK
	KeyForeign           // FK
	KeyUnique            // UK
)

// ERRelation represents a relationship between entities.
type ERRelation struct {
	EntityA      string
	EntityB      string
	CardinalityA Cardinality
	CardinalityB Cardinality
	Label        string
	Identifying  bool
}

// Cardinality defines relationship cardinality.
type Cardinality int

// Cardinality values.
const (
	CardZeroOrOne  Cardinality = iota // |o or o|
	CardExactlyOne                    // ||
	CardZeroOrMore                    // }o or o{
	CardOneOrMore                     // }| or |{
)

// PieSlice represents a slice in a pie chart.
type PieSlice struct {
	Label string
	Value float64
}
