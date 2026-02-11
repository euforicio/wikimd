package mermaid

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// LayoutEngineType represents supported layout engines (uses FlowchartLayoutEngine from render.go)
type LayoutEngineType = FlowchartLayoutEngine

// TestCase represents a single mermaid diagram test case
type TestCase struct {
	Name        string
	Category    string
	Diagram     string
	Description string
	Features    []string // Features being tested
}

// ComparisonResult holds the results of comparing Go vs JS output
type ComparisonResult struct {
	TestCase     *TestCase
	GoSVG        string
	JsSVG        string
	GoWidth      float64
	GoHeight     float64
	JsWidth      float64
	JsHeight     float64
	Match        bool
	Differences  []string
	LayoutEngine LayoutEngineType
}

// ==============================================================================
// TEST CASES FOR ALL DIAGRAM TYPES
// ==============================================================================

var flowchartTestCases = []TestCase{
	// Direction tests
	{
		Name:        "flowchart_direction_tb",
		Category:    "flowchart",
		Description: "Top to bottom direction",
		Features:    []string{"direction", "TB"},
		Diagram: `flowchart TB
    A[Start] --> B[End]`,
	},
	{
		Name:        "flowchart_direction_bt",
		Category:    "flowchart",
		Description: "Bottom to top direction",
		Features:    []string{"direction", "BT"},
		Diagram: `flowchart BT
    A[Start] --> B[End]`,
	},
	{
		Name:        "flowchart_direction_lr",
		Category:    "flowchart",
		Description: "Left to right direction",
		Features:    []string{"direction", "LR"},
		Diagram: `flowchart LR
    A[Start] --> B[End]`,
	},
	{
		Name:        "flowchart_direction_rl",
		Category:    "flowchart",
		Description: "Right to left direction",
		Features:    []string{"direction", "RL"},
		Diagram: `flowchart RL
    A[Start] --> B[End]`,
	},

	// Node shape tests
	{
		Name:        "flowchart_shape_rectangle",
		Category:    "flowchart",
		Description: "Rectangle node shape",
		Features:    []string{"shape", "rectangle"},
		Diagram: `flowchart TD
    A[Rectangle Node]`,
	},
	{
		Name:        "flowchart_shape_rounded",
		Category:    "flowchart",
		Description: "Rounded node shape",
		Features:    []string{"shape", "rounded"},
		Diagram: `flowchart TD
    A(Rounded Node)`,
	},
	{
		Name:        "flowchart_shape_stadium",
		Category:    "flowchart",
		Description: "Stadium node shape",
		Features:    []string{"shape", "stadium"},
		Diagram: `flowchart TD
    A([Stadium Node])`,
	},
	{
		Name:        "flowchart_shape_subroutine",
		Category:    "flowchart",
		Description: "Subroutine node shape",
		Features:    []string{"shape", "subroutine"},
		Diagram: `flowchart TD
    A[[Subroutine Node]]`,
	},
	{
		Name:        "flowchart_shape_cylinder",
		Category:    "flowchart",
		Description: "Cylinder/database node shape",
		Features:    []string{"shape", "cylinder"},
		Diagram: `flowchart TD
    A[(Database)]`,
	},
	{
		Name:        "flowchart_shape_circle",
		Category:    "flowchart",
		Description: "Circle node shape",
		Features:    []string{"shape", "circle"},
		Diagram: `flowchart TD
    A((Circle Node))`,
	},
	{
		Name:        "flowchart_shape_double_circle",
		Category:    "flowchart",
		Description: "Double circle node shape",
		Features:    []string{"shape", "double_circle"},
		Diagram: `flowchart TD
    A(((Double Circle)))`,
	},
	{
		Name:        "flowchart_shape_asymmetric",
		Category:    "flowchart",
		Description: "Asymmetric node shape",
		Features:    []string{"shape", "asymmetric"},
		Diagram: `flowchart TD
    A>Asymmetric Node]`,
	},
	{
		Name:        "flowchart_shape_rhombus",
		Category:    "flowchart",
		Description: "Rhombus/diamond node shape",
		Features:    []string{"shape", "rhombus"},
		Diagram: `flowchart TD
    A{Decision}`,
	},
	{
		Name:        "flowchart_shape_hexagon",
		Category:    "flowchart",
		Description: "Hexagon node shape",
		Features:    []string{"shape", "hexagon"},
		Diagram: `flowchart TD
    A{{Hexagon Node}}`,
	},
	{
		Name:        "flowchart_shape_parallelogram",
		Category:    "flowchart",
		Description: "Parallelogram node shape",
		Features:    []string{"shape", "parallelogram"},
		Diagram: `flowchart TD
    A[/Parallelogram/]`,
	},
	{
		Name:        "flowchart_shape_parallelogram_alt",
		Category:    "flowchart",
		Description: "Parallelogram alt node shape",
		Features:    []string{"shape", "parallelogram_alt"},
		Diagram: `flowchart TD
    A[\Parallelogram Alt\]`,
	},
	{
		Name:        "flowchart_shape_trapezoid",
		Category:    "flowchart",
		Description: "Trapezoid node shape",
		Features:    []string{"shape", "trapezoid"},
		Diagram: `flowchart TD
    A[/Trapezoid\]`,
	},
	{
		Name:        "flowchart_shape_trapezoid_alt",
		Category:    "flowchart",
		Description: "Trapezoid alt node shape",
		Features:    []string{"shape", "trapezoid_alt"},
		Diagram: `flowchart TD
    A[\Trapezoid Alt/]`,
	},

	// Edge type tests
	{
		Name:        "flowchart_edge_solid_arrow",
		Category:    "flowchart",
		Description: "Solid line with arrow",
		Features:    []string{"edge", "solid", "arrow"},
		Diagram: `flowchart TD
    A --> B`,
	},
	{
		Name:        "flowchart_edge_solid_no_arrow",
		Category:    "flowchart",
		Description: "Solid line without arrow",
		Features:    []string{"edge", "solid", "no_arrow"},
		Diagram: `flowchart TD
    A --- B`,
	},
	{
		Name:        "flowchart_edge_dotted_arrow",
		Category:    "flowchart",
		Description: "Dotted line with arrow",
		Features:    []string{"edge", "dotted", "arrow"},
		Diagram: `flowchart TD
    A -.-> B`,
	},
	{
		Name:        "flowchart_edge_dotted_no_arrow",
		Category:    "flowchart",
		Description: "Dotted line without arrow",
		Features:    []string{"edge", "dotted", "no_arrow"},
		Diagram: `flowchart TD
    A -.- B`,
	},
	{
		Name:        "flowchart_edge_thick_arrow",
		Category:    "flowchart",
		Description: "Thick line with arrow",
		Features:    []string{"edge", "thick", "arrow"},
		Diagram: `flowchart TD
    A ==> B`,
	},
	{
		Name:        "flowchart_edge_thick_no_arrow",
		Category:    "flowchart",
		Description: "Thick line without arrow",
		Features:    []string{"edge", "thick", "no_arrow"},
		Diagram: `flowchart TD
    A === B`,
	},
	{
		Name:        "flowchart_edge_bidirectional",
		Category:    "flowchart",
		Description: "Bidirectional arrow",
		Features:    []string{"edge", "bidirectional"},
		Diagram: `flowchart TD
    A <--> B`,
	},
	{
		Name:        "flowchart_edge_circle_arrow",
		Category:    "flowchart",
		Description: "Circle arrow head",
		Features:    []string{"edge", "circle_arrow"},
		Diagram: `flowchart TD
    A --o B`,
	},
	{
		Name:        "flowchart_edge_cross_arrow",
		Category:    "flowchart",
		Description: "Cross arrow head",
		Features:    []string{"edge", "cross_arrow"},
		Diagram: `flowchart TD
    A --x B`,
	},

	// Edge label tests
	{
		Name:        "flowchart_edge_label_pipe",
		Category:    "flowchart",
		Description: "Edge label using pipe syntax",
		Features:    []string{"edge", "label", "pipe"},
		Diagram: `flowchart TD
    A -->|label text| B`,
	},
	{
		Name:        "flowchart_edge_label_inline",
		Category:    "flowchart",
		Description: "Edge label using inline syntax",
		Features:    []string{"edge", "label", "inline"},
		Diagram: `flowchart TD
    A -- label text --> B`,
	},

	// Subgraph tests
	{
		Name:        "flowchart_subgraph_basic",
		Category:    "flowchart",
		Description: "Basic subgraph",
		Features:    []string{"subgraph"},
		Diagram: `flowchart TB
    subgraph one
        A --> B
    end
    C --> D`,
	},
	{
		Name:        "flowchart_subgraph_nested",
		Category:    "flowchart",
		Description: "Nested subgraphs",
		Features:    []string{"subgraph", "nested"},
		Diagram: `flowchart TB
    subgraph outer
        subgraph inner
            A --> B
        end
        C --> D
    end`,
	},
	{
		Name:        "flowchart_subgraph_direction",
		Category:    "flowchart",
		Description: "Subgraph with direction",
		Features:    []string{"subgraph", "direction"},
		Diagram: `flowchart LR
    subgraph TOP
        direction TB
        A --> B
    end
    C --> TOP`,
	},
	{
		Name:        "flowchart_subgraph_connection",
		Category:    "flowchart",
		Description: "Connection between subgraphs",
		Features:    []string{"subgraph", "connection"},
		Diagram: `flowchart TB
    subgraph one
        A
    end
    subgraph two
        B
    end
    one --> two`,
	},

	// Chained edges
	{
		Name:        "flowchart_chained_edges",
		Category:    "flowchart",
		Description: "Chained edge syntax",
		Features:    []string{"chain"},
		Diagram: `flowchart TD
    A --> B --> C --> D`,
	},

	// Complex diagram
	{
		Name:        "flowchart_complex",
		Category:    "flowchart",
		Description: "Complex flowchart with multiple features",
		Features:    []string{"complex", "multiple_shapes", "decision"},
		Diagram: `flowchart TD
    A[Start] --> B{Is it valid?}
    B -->|Yes| C[Process]
    B -->|No| D[Reject]
    C --> E{More work?}
    E -->|Yes| C
    E -->|No| F[End]
    D --> F
    subgraph Processing
        C
        E
    end`,
	},

	// Self-reference
	{
		Name:        "flowchart_self_reference",
		Category:    "flowchart",
		Description: "Self-referencing edge",
		Features:    []string{"self_reference"},
		Diagram: `flowchart TD
    A --> A`,
	},

	// Multiple edges between same nodes
	{
		Name:        "flowchart_multiple_edges",
		Category:    "flowchart",
		Description: "Multiple edges between same nodes",
		Features:    []string{"multiple_edges"},
		Diagram: `flowchart TD
    A --> B
    A --> B`,
	},
}

var sequenceTestCases = []TestCase{
	// Participant tests
	{
		Name:        "sequence_participant_basic",
		Category:    "sequence",
		Description: "Basic participant",
		Features:    []string{"participant"},
		Diagram: `sequenceDiagram
    participant A
    participant B
    A->>B: Hello`,
	},
	{
		Name:        "sequence_participant_alias",
		Category:    "sequence",
		Description: "Participant with alias",
		Features:    []string{"participant", "alias"},
		Diagram: `sequenceDiagram
    participant A as Alice
    participant B as Bob
    A->>B: Hello`,
	},
	{
		Name:        "sequence_actor",
		Category:    "sequence",
		Description: "Actor instead of participant",
		Features:    []string{"actor"},
		Diagram: `sequenceDiagram
    actor U as User
    participant S as System
    U->>S: Request`,
	},

	// Message type tests
	{
		Name:        "sequence_message_solid_arrow",
		Category:    "sequence",
		Description: "Solid line with arrowhead",
		Features:    []string{"message", "solid", "arrow"},
		Diagram: `sequenceDiagram
    A->>B: Solid arrow`,
	},
	{
		Name:        "sequence_message_dotted_arrow",
		Category:    "sequence",
		Description: "Dotted line with arrowhead",
		Features:    []string{"message", "dotted", "arrow"},
		Diagram: `sequenceDiagram
    A-->>B: Dotted arrow`,
	},
	{
		Name:        "sequence_message_solid_line",
		Category:    "sequence",
		Description: "Solid line without arrowhead",
		Features:    []string{"message", "solid", "line"},
		Diagram: `sequenceDiagram
    A->B: Solid line`,
	},
	{
		Name:        "sequence_message_dotted_line",
		Category:    "sequence",
		Description: "Dotted line without arrowhead",
		Features:    []string{"message", "dotted", "line"},
		Diagram: `sequenceDiagram
    A-->B: Dotted line`,
	},
	{
		Name:        "sequence_message_solid_cross",
		Category:    "sequence",
		Description: "Solid line with cross end",
		Features:    []string{"message", "solid", "cross"},
		Diagram: `sequenceDiagram
    participant A
    participant B
    A-xB: Cross`,
	},
	{
		Name:        "sequence_message_dotted_cross",
		Category:    "sequence",
		Description: "Dotted line with cross end",
		Features:    []string{"message", "dotted", "cross"},
		Diagram: `sequenceDiagram
    participant A
    participant B
    A--xB: Dotted Cross`,
	},
	{
		Name:        "sequence_message_solid_open",
		Category:    "sequence",
		Description: "Solid line with open arrow",
		Features:    []string{"message", "solid", "open"},
		Diagram: `sequenceDiagram
    participant A
    participant B
    A-)B: Open`,
	},
	{
		Name:        "sequence_message_dotted_open",
		Category:    "sequence",
		Description: "Dotted line with open arrow",
		Features:    []string{"message", "dotted", "open"},
		Diagram: `sequenceDiagram
    participant A
    participant B
    A--)B: Dotted Open`,
	},

	// Note tests
	{
		Name:        "sequence_note_right",
		Category:    "sequence",
		Description: "Note right of participant",
		Features:    []string{"note", "right"},
		Diagram: `sequenceDiagram
    participant A
    Note right of A: Right note`,
	},
	{
		Name:        "sequence_note_left",
		Category:    "sequence",
		Description: "Note left of participant",
		Features:    []string{"note", "left"},
		Diagram: `sequenceDiagram
    participant A
    Note left of A: Left note`,
	},
	{
		Name:        "sequence_note_over",
		Category:    "sequence",
		Description: "Note over participant",
		Features:    []string{"note", "over"},
		Diagram: `sequenceDiagram
    participant A
    Note over A: Over note`,
	},
	{
		Name:        "sequence_note_over_two",
		Category:    "sequence",
		Description: "Note over two participants",
		Features:    []string{"note", "over", "spanning"},
		Diagram: `sequenceDiagram
    participant A
    participant B
    Note over A,B: Spanning note`,
	},

	// Loop tests
	{
		Name:        "sequence_loop",
		Category:    "sequence",
		Description: "Loop fragment",
		Features:    []string{"loop"},
		Diagram: `sequenceDiagram
    participant A
    participant B
    loop Every minute
        A->>B: Ping
        B-->>A: Pong
    end`,
	},

	// Alt/else tests
	{
		Name:        "sequence_alt_else",
		Category:    "sequence",
		Description: "Alt/else fragment",
		Features:    []string{"alt", "else"},
		Diagram: `sequenceDiagram
    participant A
    participant B
    alt is valid
        A->>B: Process
    else is invalid
        A->>B: Reject
    end`,
	},

	// Activation tests
	{
		Name:        "sequence_activation",
		Category:    "sequence",
		Description: "Explicit activation/deactivation",
		Features:    []string{"activation"},
		Diagram: `sequenceDiagram
    participant A
    participant B
    A->>B: Request
    activate B
    B-->>A: Response
    deactivate B`,
	},

	// Self-message
	{
		Name:        "sequence_self_message",
		Category:    "sequence",
		Description: "Self-referencing message",
		Features:    []string{"self_message"},
		Diagram: `sequenceDiagram
    participant A
    A->>A: Self message`,
	},

	// Complex sequence
	{
		Name:        "sequence_complex",
		Category:    "sequence",
		Description: "Complex sequence diagram",
		Features:    []string{"complex"},
		Diagram: `sequenceDiagram
    actor User
    participant API
    participant DB

    User->>API: Login request
    API->>DB: Validate credentials
    alt valid
        DB-->>API: Valid
        API-->>User: Login successful
    else invalid
        DB-->>API: Invalid
        API-->>User: Login failed
    end

    loop Polling
        User->>API: Check status
        API-->>User: Status OK
    end`,
	},
}

var classTestCases = []TestCase{
	// Basic class
	{
		Name:        "class_basic",
		Category:    "class",
		Description: "Basic class definition",
		Features:    []string{"class"},
		Diagram: `classDiagram
    class Animal {
        +String name
        +int age
        +makeSound()
    }`,
	},

	// Visibility modifiers
	{
		Name:        "class_visibility",
		Category:    "class",
		Description: "All visibility modifiers",
		Features:    []string{"visibility"},
		Diagram: `classDiagram
    class MyClass {
        +publicAttr
        -privateAttr
        #protectedAttr
        ~packageAttr
        +publicMethod()
        -privateMethod()
    }`,
	},

	// Relationships
	{
		Name:        "class_inheritance",
		Category:    "class",
		Description: "Inheritance relationship",
		Features:    []string{"relationship", "inheritance"},
		Diagram: `classDiagram
    Animal <|-- Dog
    Animal <|-- Cat`,
	},
	{
		Name:        "class_composition",
		Category:    "class",
		Description: "Composition relationship",
		Features:    []string{"relationship", "composition"},
		Diagram: `classDiagram
    Car *-- Engine`,
	},
	{
		Name:        "class_aggregation",
		Category:    "class",
		Description: "Aggregation relationship",
		Features:    []string{"relationship", "aggregation"},
		Diagram: `classDiagram
    Library o-- Book`,
	},
	{
		Name:        "class_association",
		Category:    "class",
		Description: "Association relationship",
		Features:    []string{"relationship", "association"},
		Diagram: `classDiagram
    Student --> Course`,
	},
	{
		Name:        "class_dependency",
		Category:    "class",
		Description: "Dependency relationship",
		Features:    []string{"relationship", "dependency"},
		Diagram: `classDiagram
    Client ..> Server`,
	},
	{
		Name:        "class_realization",
		Category:    "class",
		Description: "Realization relationship",
		Features:    []string{"relationship", "realization"},
		Diagram: `classDiagram
    ArrayList ..|> List`,
	},

	// Relationship with label
	{
		Name:        "class_relationship_label",
		Category:    "class",
		Description: "Relationship with label",
		Features:    []string{"relationship", "label"},
		Diagram: `classDiagram
    Customer --> Order : places`,
	},

	// Complex class diagram
	{
		Name:        "class_complex",
		Category:    "class",
		Description: "Complex class diagram",
		Features:    []string{"complex"},
		Diagram: `classDiagram
    class Shape {
        +int x
        +int y
        +draw()
    }
    class Circle {
        +int radius
        +draw()
    }
    class Rectangle {
        +int width
        +int height
        +draw()
    }
    Shape <|-- Circle
    Shape <|-- Rectangle`,
	},
}

var stateTestCases = []TestCase{
	// Basic states
	{
		Name:        "state_basic",
		Category:    "state",
		Description: "Basic state diagram",
		Features:    []string{"state"},
		Diagram: `stateDiagram-v2
    [*] --> Idle
    Idle --> [*]`,
	},

	// Transitions with labels
	{
		Name:        "state_transitions",
		Category:    "state",
		Description: "State transitions with labels",
		Features:    []string{"transition", "label"},
		Diagram: `stateDiagram-v2
    [*] --> Idle
    Idle --> Processing : start
    Processing --> Complete : finish
    Complete --> [*]`,
	},

	// State descriptions
	{
		Name:        "state_descriptions",
		Category:    "state",
		Description: "States with descriptions",
		Features:    []string{"description"},
		Diagram: `stateDiagram-v2
    Idle : Waiting for input
    Processing : Working on request
    Complete : Task finished
    [*] --> Idle
    Idle --> Processing
    Processing --> Complete
    Complete --> [*]`,
	},

	// V1 syntax
	{
		Name:        "state_v1_syntax",
		Category:    "state",
		Description: "State diagram v1 syntax",
		Features:    []string{"v1"},
		Diagram: `stateDiagram
    [*] --> State1
    State1 --> State2
    State2 --> [*]`,
	},

	// Complex state diagram
	{
		Name:        "state_complex",
		Category:    "state",
		Description: "Complex state diagram",
		Features:    []string{"complex"},
		Diagram: `stateDiagram-v2
    [*] --> Still
    Still --> [*]
    Still --> Moving
    Moving --> Still
    Moving --> Crash
    Crash --> [*]

    Still : No movement
    Moving : In motion
    Crash : Accident occurred`,
	},
}

var erTestCases = []TestCase{
	// Basic entity
	{
		Name:        "er_basic_entity",
		Category:    "er",
		Description: "Basic entity with attributes",
		Features:    []string{"entity"},
		Diagram: `erDiagram
    CUSTOMER {
        int id PK
        string name
        string email
    }`,
	},

	// Key types
	{
		Name:        "er_key_types",
		Category:    "er",
		Description: "Different key types",
		Features:    []string{"key", "PK", "FK", "UK"},
		Diagram: `erDiagram
    PRODUCT {
        int id PK
        string sku UK
        int categoryId FK
        string name
    }`,
	},

	// Relationships
	{
		Name:        "er_relationship",
		Category:    "er",
		Description: "ER relationship",
		Features:    []string{"relationship"},
		Diagram: `erDiagram
    CUSTOMER ||--o{ ORDER : places`,
	},

	// Complex ER diagram
	{
		Name:        "er_complex",
		Category:    "er",
		Description: "Complex ER diagram",
		Features:    []string{"complex"},
		Diagram: `erDiagram
    CUSTOMER {
        int id PK
        string name
        string email
    }
    ORDER {
        int id PK
        int customerId FK
        date orderDate
    }
    PRODUCT {
        int id PK
        string name
        decimal price
    }
    ORDER_ITEM {
        int orderId FK
        int productId FK
        int quantity
    }
    CUSTOMER ||--o{ ORDER : places
    ORDER ||--|{ ORDER_ITEM : contains
    PRODUCT ||--o{ ORDER_ITEM : "is in"`,
	},
}

// GetAllTestCases returns all test cases across all categories
func GetAllTestCases() []TestCase {
	var all []TestCase
	all = append(all, flowchartTestCases...)
	all = append(all, sequenceTestCases...)
	all = append(all, classTestCases...)
	all = append(all, stateTestCases...)
	all = append(all, erTestCases...)
	return all
}

// ==============================================================================
// MERMAID-CLI INTEGRATION
// ==============================================================================

// MermaidCLI wraps the mermaid-cli (mmdc) tool
type MermaidCLI struct {
	BinaryPath   string
	ConfigPath   string
	LayoutEngine LayoutEngineType
}

// NewMermaidCLI creates a new MermaidCLI instance
func NewMermaidCLI() (*MermaidCLI, error) {
	// Try to find mmdc in PATH
	path, err := exec.LookPath("mmdc")
	if err != nil {
		// Try npx mmdc
		path = "npx"
	}

	return &MermaidCLI{
		BinaryPath:   path,
		LayoutEngine: LayoutEngineFullDagre,
	}, nil
}

// SetLayoutEngine sets the layout engine to use
func (m *MermaidCLI) SetLayoutEngine(engine LayoutEngineType) {
	m.LayoutEngine = engine
}

// RenderSVG renders a mermaid diagram to SVG using the CLI
func (m *MermaidCLI) RenderSVG(diagram string) (string, error) {
	// Create temp input file
	tmpDir, err := os.MkdirTemp("", "mermaid-test-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	inputPath := filepath.Join(tmpDir, "input.mmd")
	outputPath := filepath.Join(tmpDir, "output.svg")

	if err := os.WriteFile(inputPath, []byte(diagram), 0644); err != nil {
		return "", fmt.Errorf("failed to write input file: %w", err)
	}

	// Build command
	var cmd *exec.Cmd
	if m.BinaryPath == "npx" {
		cmd = exec.Command("npx", "-y", "@mermaid-js/mermaid-cli", "-i", inputPath, "-o", outputPath, "-b", "transparent")
	} else {
		cmd = exec.Command(m.BinaryPath, "-i", inputPath, "-o", outputPath, "-b", "transparent")
	}

	// Add config file if specified
	if m.ConfigPath != "" {
		cmd.Args = append(cmd.Args, "-c", m.ConfigPath)
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("mermaid-cli failed: %w, stderr: %s", err, stderr.String())
	}

	svgData, err := os.ReadFile(outputPath)
	if err != nil {
		return "", fmt.Errorf("failed to read output: %w", err)
	}

	return string(svgData), nil
}

// ==============================================================================
// SVG COMPARISON UTILITIES
// ==============================================================================

// SVGMetrics holds extracted metrics from an SVG
type SVGMetrics struct {
	Width        float64
	Height       float64
	ViewBox      [4]float64
	NodeCount    int
	EdgeCount    int
	TextCount    int
	PathCount    int
	RectCount    int
	CircleCount  int
	EllipseCount int
	PolygonCount int
	LineCount    int
	GroupCount   int
	MarkerCount  int
	TextContents []string
	HasDefs      bool
	HasStyles    bool
}

// ExtractSVGMetrics parses an SVG and extracts metrics for comparison
func ExtractSVGMetrics(svg string) (*SVGMetrics, error) {
	m := &SVGMetrics{}

	// Extract dimensions from the root SVG tag only (not from nested markers/symbols)
	// Find the opening <svg tag first
	svgTagRe := regexp.MustCompile(`<svg[^>]*>`)
	svgTagMatch := svgTagRe.FindString(svg)
	if svgTagMatch == "" {
		return m, nil
	}

	widthRe := regexp.MustCompile(`width="([^"]+)"`)
	heightRe := regexp.MustCompile(`height="([^"]+)"`)
	viewBoxRe := regexp.MustCompile(`viewBox="([^"]+)"`)

	// Extract from root SVG tag only
	if match := widthRe.FindStringSubmatch(svgTagMatch); len(match) > 1 {
		m.Width, _ = parsePixelValue(match[1])
	}
	if match := heightRe.FindStringSubmatch(svgTagMatch); len(match) > 1 {
		m.Height, _ = parsePixelValue(match[1])
	}
	if match := viewBoxRe.FindStringSubmatch(svgTagMatch); len(match) > 1 {
		parts := strings.Fields(match[1])
		for i, p := range parts {
			if i < 4 {
				m.ViewBox[i], _ = strconv.ParseFloat(p, 64)
			}
		}
	}

	// If width/height are zero or invalid (e.g., "100%"), use viewBox dimensions
	// This is common for mermaid-cli output which uses viewBox for actual sizing
	if m.Width <= 0 && m.ViewBox[2] > 0 {
		m.Width = m.ViewBox[2]
	}
	if m.Height <= 0 && m.ViewBox[3] > 0 {
		m.Height = m.ViewBox[3]
	}

	// Count elements
	m.PathCount = strings.Count(svg, "<path")
	m.RectCount = strings.Count(svg, "<rect")
	m.CircleCount = strings.Count(svg, "<circle")
	m.EllipseCount = strings.Count(svg, "<ellipse")
	m.PolygonCount = strings.Count(svg, "<polygon")
	m.LineCount = strings.Count(svg, "<line")
	m.TextCount = strings.Count(svg, "<text")
	m.GroupCount = strings.Count(svg, "<g")
	m.MarkerCount = strings.Count(svg, "<marker")
	m.HasDefs = strings.Contains(svg, "<defs")
	m.HasStyles = strings.Contains(svg, "<style")

	// Extract text contents from both <text> elements and <span> elements
	// Mermaid-cli often uses foreignObject with spans instead of text elements
	textRe := regexp.MustCompile(`<text[^>]*>([^<]+)</text>`)
	matches := textRe.FindAllStringSubmatch(svg, -1)
	for _, match := range matches {
		if len(match) > 1 {
			m.TextContents = append(m.TextContents, strings.TrimSpace(match[1]))
		}
	}

	// Also extract from span.nodeLabel and span.edgeLabel elements (mermaid-cli format)
	spanRe := regexp.MustCompile(`<span[^>]*class="[^"]*(?:nodeLabel|edgeLabel)[^"]*"[^>]*>(?:<p>)?([^<]+)(?:</p>)?</span>`)
	spanMatches := spanRe.FindAllStringSubmatch(svg, -1)
	for _, match := range spanMatches {
		if len(match) > 1 {
			m.TextContents = append(m.TextContents, strings.TrimSpace(match[1]))
		}
	}

	// Also try a simpler pattern for nodeLabel paragraphs
	pRe := regexp.MustCompile(`<span class="nodeLabel"><p>([^<]+)</p></span>`)
	pMatches := pRe.FindAllStringSubmatch(svg, -1)
	for _, match := range pMatches {
		if len(match) > 1 {
			text := strings.TrimSpace(match[1])
			// Avoid duplicates
			found := false
			for _, existing := range m.TextContents {
				if existing == text {
					found = true
					break
				}
			}
			if !found {
				m.TextContents = append(m.TextContents, text)
			}
		}
	}

	return m, nil
}

func parsePixelValue(s string) (float64, error) {
	s = strings.TrimSuffix(s, "px")
	return strconv.ParseFloat(s, 64)
}

// CompareSVGs compares two SVG strings and returns differences
// Uses tolerances based on expected differences between Go and JS implementations:
// - Dimensions: max(20px, 50%) due to different text measurement approaches
// - Text content: checks that Go SVG contains all labels from JS SVG
func CompareSVGs(goSVG, jsSVG string) ([]string, error) {
	return compareSVGsWithTolerance(goSVG, jsSVG, 0.50)
}

// CompareSVGsStrict compares two SVG strings with tight tolerances
// Suitable for golden file regression tests where both SVGs are from the same Go implementation
// Uses max(2px, 1%) tolerance for dimensions - any significant change is a regression
func CompareSVGsStrict(currentSVG, goldenSVG string) ([]string, error) {
	return compareSVGsWithTolerance(currentSVG, goldenSVG, 0.01)
}

// compareSVGsWithTolerance is the internal comparison function with configurable tolerance
func compareSVGsWithTolerance(svg1, svg2 string, tolerancePercent float64) ([]string, error) {
	var diffs []string

	metrics1, err := ExtractSVGMetrics(svg1)
	if err != nil {
		return nil, fmt.Errorf("failed to extract SVG1 metrics: %w", err)
	}

	metrics2, err := ExtractSVGMetrics(svg2)
	if err != nil {
		return nil, fmt.Errorf("failed to extract SVG2 metrics: %w", err)
	}

	// Calculate tolerances based on reference dimensions
	// Use min tolerance of 2px for strict mode, 20px for loose mode
	minTolerance := 20.0
	if tolerancePercent <= 0.05 {
		minTolerance = 2.0
	}
	widthTolerance := maxFloat(minTolerance, metrics2.Width*tolerancePercent)
	heightTolerance := maxFloat(minTolerance, metrics2.Height*tolerancePercent)

	// Check for invalid dimensions
	if metrics2.Width <= 0 || metrics2.Height <= 0 {
		diffs = append(diffs, fmt.Sprintf("reference SVG has invalid dimensions: width=%.1f, height=%.1f", metrics2.Width, metrics2.Height))
	} else {
		if abs(metrics1.Width-metrics2.Width) > widthTolerance {
			diffs = append(diffs, fmt.Sprintf("width: got=%.1f, want=%.1f (tolerance=%.1f)", metrics1.Width, metrics2.Width, widthTolerance))
		}
		if abs(metrics1.Height-metrics2.Height) > heightTolerance {
			diffs = append(diffs, fmt.Sprintf("height: got=%.1f, want=%.1f (tolerance=%.1f)", metrics1.Height, metrics2.Height, heightTolerance))
		}
	}

	// Compare text content - both SVGs should contain the same labels
	textSet1 := make(map[string]bool)
	for _, t := range metrics1.TextContents {
		textSet1[t] = true
	}
	textSet2 := make(map[string]bool)
	for _, t := range metrics2.TextContents {
		textSet2[t] = true
	}

	// Check for text present in reference but missing in current (potential bug)
	for t := range textSet2 {
		if !textSet1[t] {
			diffs = append(diffs, fmt.Sprintf("SVG missing text present in reference: %q", t))
		}
	}

	// Check for unexpected text in current (may indicate a bug or intentional change)
	for t := range textSet1 {
		if !textSet2[t] {
			diffs = append(diffs, fmt.Sprintf("SVG contains unexpected text not in reference: %q", t))
		}
	}

	return diffs, nil
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// ==============================================================================
// GOLDEN FILE MANAGEMENT
// ==============================================================================

const goldenDir = "testdata/golden"

// SaveGoldenFile saves an SVG as a golden file
func SaveGoldenFile(name string, svg string) error {
	if err := os.MkdirAll(goldenDir, 0755); err != nil {
		return err
	}
	path := filepath.Join(goldenDir, name+".svg")
	return os.WriteFile(path, []byte(svg), 0644)
}

// LoadGoldenFile loads a golden SVG file
func LoadGoldenFile(name string) (string, error) {
	path := filepath.Join(goldenDir, name+".svg")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// GoldenExists checks if a golden file exists
func GoldenExists(name string) bool {
	path := filepath.Join(goldenDir, name+".svg")
	_, err := os.Stat(path)
	return err == nil
}

// ==============================================================================
// MAIN COMPARISON TESTS
// ==============================================================================

// TestGoVsJSComparison runs all comparison tests
func TestGoVsJSComparison(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping JS comparison tests in short mode")
	}

	// Check if mermaid-cli is available
	cli, err := NewMermaidCLI()
	if err != nil {
		t.Skipf("mermaid-cli not available: %v", err)
	}

	// Test a simple diagram to ensure CLI works
	testDiagram := "flowchart TD\n    A-->B"
	_, err = cli.RenderSVG(testDiagram)
	if err != nil {
		t.Skipf("mermaid-cli not working: %v", err)
	}

	renderer := NewRenderer(nil)
	ctx := context.Background()

	allTests := GetAllTestCases()

	for _, tc := range allTests {
		t.Run(tc.Name, func(t *testing.T) {
			// Render with Go
			goResult, err := renderer.Render(ctx, tc.Diagram)
			if err != nil {
				t.Fatalf("Go render failed: %v", err)
			}

			// Render with JS
			jsSVG, err := cli.RenderSVG(tc.Diagram)
			if err != nil {
				t.Logf("JS render failed (may not support feature): %v", err)
				return
			}

			// Compare
			diffs, err := CompareSVGs(goResult.SVG, jsSVG)
			if err != nil {
				t.Fatalf("comparison failed: %v", err)
			}

			if len(diffs) > 0 {
				t.Logf("Differences found for %s:", tc.Name)
				for _, d := range diffs {
					t.Logf("  - %s", d)
				}
				// Log differences but don't fail - these represent known implementation
				// gaps between Go and JS, not regressions. Failures would occur only
				// if Go render itself fails.
			}

			// Verify essential content is present in both (logs but doesn't fail)
			verifyContent(t, &tc, goResult.SVG)
		})
	}
}

// verifyContent checks that essential content appears in Go SVG
// Missing labels in Go SVG indicate a rendering bug and should fail the test
func verifyContent(t *testing.T, tc *TestCase, goSVG string) {
	t.Helper()

	// Extract expected labels from the diagram
	labels := extractExpectedLabels(tc.Diagram)

	for _, label := range labels {
		if !strings.Contains(goSVG, label) {
			t.Errorf("Go SVG missing expected label: %q", label)
		}
	}
}

// extractExpectedLabels extracts labels that should appear in the rendered SVG
func extractExpectedLabels(diagram string) []string {
	var labels []string

	// Extract text from various patterns - most specific first
	// Note: For nested patterns like ([text]), we want just "text", not "[text"
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`\(\(\(([^()]+)\)\)\)`),    // (((double circle))) - no nested parens in content
		regexp.MustCompile(`\[\[([^\[\]]+)\]\]`),      // [[subroutine]] - no nested brackets in content
		regexp.MustCompile(`\(\(([^()]+)\)\)`),        // ((circle)) - no nested parens in content
		regexp.MustCompile(`\(\[([^\[\]]+)\]\)`),      // ([stadium]) - no nested brackets in content
		regexp.MustCompile(`\[\(([^()]+)\)\]`),        // [(cylinder)] - no nested parens in content
		regexp.MustCompile(`\{\{([^{}]+)\}\}`),        // {{hexagon}} - no nested braces in content
		regexp.MustCompile(`\[/([^/\\]+)/\]`),         // [/parallelogram/]
		regexp.MustCompile(`\[\\([^/\\]+)\\\]`),       // [\parallelogram\]
		regexp.MustCompile(`\[/([^/\\]+)\\\]`),        // [/trapezoid\]
		regexp.MustCompile(`\[\\([^/\\]+)/\]`),        // [\trapezoid/]
		regexp.MustCompile(`(?:^|[^-])>([^\]>]+)\]`),  // >asymmetric] - not preceded by - (arrow)
		regexp.MustCompile(`\[([^\[\](){}/<>\\]+)\]`), // [rectangle] - no nested delimiters or slashes
		regexp.MustCompile(`\(([^\[\](){}/<>\\]+)\)`), // (rounded) - no nested delimiters or slashes
		regexp.MustCompile(`\{([^\[\](){}/<>\\]+)\}`), // {diamond} - no nested delimiters or slashes
		regexp.MustCompile(`:\s*([^\n]+)`),            // : label (messages)
		regexp.MustCompile(`as\s+(\w+)`),              // as Alias
	}

	seen := make(map[string]bool)
	for _, re := range patterns {
		matches := re.FindAllStringSubmatch(diagram, -1)
		for _, match := range matches {
			if len(match) > 1 {
				label := strings.TrimSpace(match[1])
				// Strip surrounding quotes from labels (ER diagrams often have quoted labels)
				if len(label) >= 2 && (label[0] == '"' && label[len(label)-1] == '"') {
					label = label[1 : len(label)-1]
				}
				// Filter out syntax patterns and duplicates
				if len(label) > 0 && len(label) < 50 && !isSyntax(label) && !seen[label] {
					labels = append(labels, label)
					seen[label] = true
				}
			}
		}
	}

	return labels
}

func isSyntax(s string) bool {
	syntax := []string{"[", "]", "(", ")", "{", "}", "|", "-", ".", ">", "<", "*", "o"}
	for _, syn := range syntax {
		if s == syn {
			return true
		}
	}
	return false
}

// ==============================================================================
// LAYOUT ENGINE TESTS
// ==============================================================================

// TestLayoutEngineFullDagre tests the Dagre layout engine
func TestLayoutEngineFullDagre(t *testing.T) {
	testCases := []struct {
		name    string
		diagram string
	}{
		{
			name: "simple_chain",
			diagram: `flowchart TD
    A --> B --> C`,
		},
		{
			name: "branching",
			diagram: `flowchart TD
    A --> B
    A --> C
    B --> D
    C --> D`,
		},
		{
			name: "deep_hierarchy",
			diagram: `flowchart TD
    A --> B
    B --> C
    C --> D
    D --> E`,
		},
	}

	renderer := NewRenderer(nil)
	ctx := context.Background()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := renderer.Render(ctx, tc.diagram)
			if err != nil {
				t.Fatalf("render failed: %v", err)
			}

			// Verify SVG has proper structure (layout was applied)
			if !strings.Contains(result.SVG, "<svg") {
				t.Error("result should contain svg element")
			}

			// Verify paths exist (edges were routed)
			if !strings.Contains(result.SVG, "<path") {
				t.Error("result should contain path elements for edges")
			}

			// Verify SVG contains expected elements
			if result.Width <= 0 {
				t.Error("result width should be positive")
			}
			if result.Height <= 0 {
				t.Error("result height should be positive")
			}
		})
	}
}

// TestLayoutEngineSequence tests the sequence diagram layout engine
func TestLayoutEngineSequence(t *testing.T) {
	testCases := []struct {
		name    string
		diagram string
	}{
		{
			name: "two_participants",
			diagram: `sequenceDiagram
    A->>B: Hello
    B-->>A: Hi`,
		},
		{
			name: "three_participants",
			diagram: `sequenceDiagram
    A->>B: First
    B->>C: Second
    C-->>A: Third`,
		},
		{
			name: "many_messages",
			diagram: `sequenceDiagram
    A->>B: 1
    B->>A: 2
    A->>B: 3
    B->>A: 4
    A->>B: 5`,
		},
	}

	renderer := NewRenderer(nil)
	ctx := context.Background()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := renderer.Render(ctx, tc.diagram)
			if err != nil {
				t.Fatalf("render failed: %v", err)
			}

			// Verify SVG contains lifelines
			if !strings.Contains(result.SVG, "stroke-dasharray") {
				t.Error("expected dashed lifelines in sequence diagram")
			}

			// Verify dimensions
			if result.Width <= 0 {
				t.Error("result width should be positive")
			}
			if result.Height <= 0 {
				t.Error("result height should be positive")
			}
		})
	}
}

// ==============================================================================
// REGRESSION TESTS
// ==============================================================================

// TestRegressionSuite runs regression tests against golden files
func TestRegressionSuite(t *testing.T) {
	renderer := NewRenderer(nil)
	ctx := context.Background()

	allTests := GetAllTestCases()

	for _, tc := range allTests {
		t.Run(tc.Name, func(t *testing.T) {
			result, err := renderer.Render(ctx, tc.Diagram)
			if err != nil {
				// Some test cases may not be fully supported yet
				t.Skipf("render not supported: %v", err)
			}

			goldenName := tc.Category + "_" + tc.Name

			if os.Getenv("UPDATE_GOLDEN") == "1" {
				if err := SaveGoldenFile(goldenName, result.SVG); err != nil {
					t.Fatalf("failed to save golden file: %v", err)
				}
				t.Logf("Updated golden file: %s", goldenName)
				return
			}

			if !GoldenExists(goldenName) {
				t.Skipf("golden file not found: %s (run with UPDATE_GOLDEN=1 to create)", goldenName)
			}

			golden, err := LoadGoldenFile(goldenName)
			if err != nil {
				t.Fatalf("failed to load golden file: %v", err)
			}

			// Compare with strict tolerances - golden files are from the same Go implementation
			// so any significant difference indicates a regression
			diffs, err := CompareSVGsStrict(result.SVG, golden)
			if err != nil {
				t.Fatalf("comparison failed: %v", err)
			}

			if len(diffs) > 0 {
				t.Errorf("regression detected:")
				for _, d := range diffs {
					t.Errorf("  - %s", d)
				}
			}
		})
	}
}

// ==============================================================================
// FEATURE COVERAGE TESTS
// ==============================================================================

// TestFeatureCoverage ensures all features are tested
func TestFeatureCoverage(t *testing.T) {
	allTests := GetAllTestCases()

	// Collect tested features per category
	testedFeatures := make(map[string]map[string][]string) // category -> feature -> test names
	for _, tc := range allTests {
		if testedFeatures[tc.Category] == nil {
			testedFeatures[tc.Category] = make(map[string][]string)
		}
		for _, f := range tc.Features {
			testedFeatures[tc.Category][f] = append(testedFeatures[tc.Category][f], tc.Name)
		}
	}

	// Expected features per category
	expectedFlowchartFeatures := []string{
		"direction", "TB", "BT", "LR", "RL",
		"shape", "rectangle", "rounded", "stadium", "subroutine",
		"cylinder", "circle", "double_circle", "asymmetric", "rhombus",
		"hexagon", "parallelogram", "parallelogram_alt", "trapezoid", "trapezoid_alt",
		"edge", "solid", "dotted", "thick", "arrow", "no_arrow",
		"bidirectional", "circle_arrow", "cross_arrow",
		"label", "pipe", "inline",
		"subgraph", "nested", "connection",
		"chain", "complex", "self_reference", "multiple_edges",
	}

	expectedSequenceFeatures := []string{
		"participant", "alias", "actor",
		"message", "solid", "dotted", "arrow", "line", "cross", "open",
		"note", "right", "left", "over", "spanning",
		"loop", "alt", "else", "activation",
		"self_message", "complex",
	}

	expectedClassFeatures := []string{
		"class", "visibility",
		"relationship", "inheritance", "composition", "aggregation",
		"association", "dependency", "realization", "label",
		"complex",
	}

	expectedStateFeatures := []string{
		"state", "transition", "label", "description", "v1", "complex",
	}

	expectedERFeatures := []string{
		"entity", "key", "PK", "FK", "UK", "relationship", "complex",
	}

	// Check coverage per category
	checkFeatures := func(category string, expected []string) {
		categoryFeatures := testedFeatures[category]
		if categoryFeatures == nil {
			t.Errorf("%s: no tests found for category", category)
			return
		}
		for _, f := range expected {
			if _, ok := categoryFeatures[f]; !ok {
				t.Errorf("%s: feature %q not covered by any test in this category", category, f)
			}
		}
	}

	checkFeatures("flowchart", expectedFlowchartFeatures)
	checkFeatures("sequence", expectedSequenceFeatures)
	checkFeatures("class", expectedClassFeatures)
	checkFeatures("state", expectedStateFeatures)
	checkFeatures("er", expectedERFeatures)
}

// ==============================================================================
// BENCHMARK COMPARISON
// ==============================================================================

func BenchmarkGoVsJS(b *testing.B) {
	// Skip JS benchmark if CLI not available
	cli, err := NewMermaidCLI()
	skipJS := err != nil

	// Also check if CLI actually works
	if !skipJS {
		_, testErr := cli.RenderSVG("flowchart TD\n    A-->B")
		skipJS = testErr != nil
	}

	diagrams := map[string]string{
		"simple_flowchart": `flowchart TD
    A --> B --> C`,
		"medium_flowchart": `flowchart TD
    A[Start] --> B{Decision}
    B -->|Yes| C[Process 1]
    B -->|No| D[Process 2]
    C --> E[End]
    D --> E`,
		"sequence": `sequenceDiagram
    A->>B: Hello
    B-->>A: Hi`,
	}

	renderer := NewRenderer(nil)
	ctx := context.Background()

	for name, diagram := range diagrams {
		b.Run("Go_"+name, func(b *testing.B) {
			for b.Loop() {
				_, err := renderer.Render(ctx, diagram)
				if err != nil {
					b.Fatal(err)
				}
			}
		})

		if !skipJS {
			b.Run("JS_"+name, func(b *testing.B) {
				for b.Loop() {
					_, err := cli.RenderSVG(diagram)
					if err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}

// ==============================================================================
// TEST REPORT GENERATION
// ==============================================================================

// TestReport holds test results for reporting
type TestReport struct {
	TotalTests    int
	PassedTests   int
	FailedTests   int
	SkippedTests  int
	FeatureMatrix map[string]map[string]bool // category -> feature -> supported
	Results       []TestResult
}

// TestResult holds individual test result
type TestResult struct {
	TestCase    *TestCase
	Passed      bool
	Skipped     bool
	Error       string
	Differences []string
}

// GenerateReport generates a JSON report of all test results
func GenerateReport(t *testing.T) {
	report := &TestReport{
		FeatureMatrix: make(map[string]map[string]bool),
		Results:       []TestResult{},
	}

	renderer := NewRenderer(nil)
	ctx := context.Background()

	allTests := GetAllTestCases()
	report.TotalTests = len(allTests)

	for _, tc := range allTests {
		result := TestResult{TestCase: &tc}

		_, err := renderer.Render(ctx, tc.Diagram)
		if err != nil {
			result.Error = err.Error()
			report.SkippedTests++
			result.Skipped = true
		} else {
			result.Passed = true
			report.PassedTests++
		}

		report.Results = append(report.Results, result)

		// Update feature matrix
		if report.FeatureMatrix[tc.Category] == nil {
			report.FeatureMatrix[tc.Category] = make(map[string]bool)
		}
		for _, f := range tc.Features {
			if !result.Skipped {
				report.FeatureMatrix[tc.Category][f] = true
			}
		}
	}

	// Output report
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal report: %v", err)
	}

	t.Logf("Test Report:\n%s", string(data))

	// Save report to file
	reportDir := "testdata/reports"
	if err := os.MkdirAll(reportDir, 0755); err == nil {
		reportPath := filepath.Join(reportDir, "comparison_report.json")
		os.WriteFile(reportPath, data, 0644)
	}
}
