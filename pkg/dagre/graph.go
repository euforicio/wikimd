// Package dagre implements a Go port of the dagre JavaScript library
// for directed graph layout using the Sugiyama algorithm.
package dagre

import (
	"errors"
	"fmt"
	"iter"
	"maps"
	"slices"
)

var (
	// ErrNamedEdgeOnNonMultigraph is returned when using named edges without multigraph mode.
	ErrNamedEdgeOnNonMultigraph = errors.New("cannot set a named edge when isMultigraph = false")
	// ErrParentOnNonCompound is returned when setting parent on a non-compound graph.
	ErrParentOnNonCompound = errors.New("cannot set parent in a non-compound graph")
)

// GraphOptions configures graph construction.
type GraphOptions struct {
	Directed   bool
	Multigraph bool
	Compound   bool
}

// Graph represents a directed graph with optional compound graph support.
// It provides the graph data structure equivalent to graphlib in the JS dagre.
type Graph struct {
	// Node storage
	nodes map[string]*NodeData

	// Edge storage - key is edgeID (v,w,name)
	edges map[string]*EdgeData

	// Adjacency: outEdges[v][w] = list of edge IDs from v to w
	outEdges map[string]map[string][]string
	// inEdges[w][v] = list of edge IDs from v to w
	inEdges map[string]map[string][]string

	// Compound graph support
	parent   map[string]string          // node -> parent node
	children map[string]map[string]bool // node -> set of child nodes

	// Graph-level attributes
	label      interface{}
	directed   bool
	isMulti    bool // Allow multiple edges between same nodes
	isCompound bool // Enable parent/child compound relationships

	// Graph attributes for layout
	RankDir   string  // "TB", "BT", "LR", "RL"
	Align     string  // "UL", "UR", "DL", "DR"
	NodeSep   float64 // Separation between nodes on same rank
	EdgeSep   float64 // Separation between edges
	RankSep   float64 // Separation between ranks
	MarginX   float64
	MarginY   float64
	Acyclicer string // "greedy" or "dfs"
	Ranker    string // "network-simplex", "tight-tree", "longest-path"
}

// NodeData holds all data for a node.
type NodeData struct {
	Label  interface{}
	Width  float64
	Height float64
	X      float64
	Y      float64

	// Layout-computed values
	Rank       int
	Order      int
	Dummy      string // "", "edge", "edge-label", "edge-proxy", "border", "root", "selfedge"
	BorderType string
	LabelPos   string // "l", "c", "r" for label position

	// For edge labels
	E          string // edge ID this dummy represents
	LabelRank  int
	EdgeRef    *EdgeObj
	EdgeObj    *EdgeObj
	EdgeLabel  *EdgeData
	EdgeNext   string
	EdgeSource string
	EdgeTarget string
	EdgeName   string
	Parent     string

	// Compound graph
	BorderTop    string
	BorderBottom string
	BorderLeft   map[int]string
	BorderRight  map[int]string

	// For min/max rank constraints
	MinRank       int
	MaxRank       int
	PaddingLeft   float64
	PaddingRight  float64
	PaddingTop    float64
	PaddingBottom float64
	Low           int
	Lim           int
}

// EdgeData holds all data for an edge.
type EdgeData struct {
	V     string // source node
	W     string // target node
	Name  string // edge name (for multigraph)
	Label interface{}

	MinLen      int     // Minimum length (rank difference)
	Weight      float64 // Weight for crossing minimization
	Width       float64 // Label width
	Height      float64 // Label height
	LabelPos    string  // "l", "c", "r"
	LabelOffset float64

	// Layout-computed
	X                float64 // label X
	Y                float64 // label Y
	HasLabelPosition bool
	Points           []Point // edge routing points

	// For layout algorithm
	LabelRank   int
	Reversed    bool
	ForwardName string
	Next        string
	NestingEdge bool
	Cutvalue    int
}

// Point represents a 2D coordinate.
type Point struct {
	X float64
	Y float64
}

// NewGraphWithOptions creates a graph with explicit construction options.
func NewGraphWithOptions(opts GraphOptions) *Graph {
	return &Graph{
		nodes:      make(map[string]*NodeData),
		edges:      make(map[string]*EdgeData),
		outEdges:   make(map[string]map[string][]string),
		inEdges:    make(map[string]map[string][]string),
		parent:     make(map[string]string),
		children:   make(map[string]map[string]bool),
		directed:   opts.Directed,
		isMulti:    opts.Multigraph,
		isCompound: opts.Compound,
		RankDir:    "TB",
		NodeSep:    50,
		EdgeSep:    10,
		RankSep:    50,
		MarginX:    20,
		MarginY:    20,
		Ranker:     "network-simplex",
	}
}

// NewGraph creates a new empty graph.
func NewGraph() *Graph {
	return NewGraphWithOptions(GraphOptions{
		Directed:   true,
		Multigraph: false,
		Compound:   true,
	})
}

// NewMultiGraph creates a new multigraph (allows multiple edges between nodes).
func NewMultiGraph() *Graph {
	return NewGraphWithOptions(GraphOptions{
		Directed:   true,
		Multigraph: true,
		Compound:   true,
	})
}

// IsMultigraph returns whether this graph allows multiple edges.
func (g *Graph) IsMultigraph() bool {
	return g.isMulti
}

// SetLabel sets the graph-level label.
func (g *Graph) SetLabel(label interface{}) {
	g.label = label
}

// Label returns the graph-level label.
func (g *Graph) Label() interface{} {
	return g.label
}

// edgeID generates a unique edge identifier.
func (g *Graph) edgeID(v, w, name string) string {
	if !g.directed && w < v {
		v, w = w, v
	}
	if g.isMulti {
		return fmt.Sprintf("%s\x00%s\x00%s", v, w, name)
	}
	return fmt.Sprintf("%s\x00%s", v, w)
}

// SetNode adds or updates a node in the graph.
func (g *Graph) SetNode(id string, data *NodeData) {
	if data == nil {
		data = &NodeData{}
	}
	if _, exists := g.nodes[id]; !exists {
		// Initialize adjacency lists
		g.outEdges[id] = make(map[string][]string)
		g.inEdges[id] = make(map[string][]string)
		// Initialize as root in compound hierarchy
		if g.isCompound {
			if g.children[""] == nil {
				g.children[""] = make(map[string]bool)
			}
			g.children[""][id] = true
			g.parent[id] = ""
		}
	}
	g.nodes[id] = data
}

// Node returns the data for a node, or nil if not found.
func (g *Graph) Node(id string) *NodeData {
	return g.nodes[id]
}

// HasNode returns whether the graph contains a node.
func (g *Graph) HasNode(id string) bool {
	_, ok := g.nodes[id]
	return ok
}

// RemoveNode removes a node and all incident edges.
func (g *Graph) RemoveNode(id string) {
	if !g.HasNode(id) {
		return
	}

	// Remove all incident edges
	for w, edgeIDs := range g.outEdges[id] {
		for _, eid := range edgeIDs {
			delete(g.edges, eid)
			g.inEdges[w][id] = removeFromSlice(g.inEdges[w][id], eid)
		}
		delete(g.outEdges[id], w)
	}
	for v, edgeIDs := range g.inEdges[id] {
		for _, eid := range edgeIDs {
			delete(g.edges, eid)
			g.outEdges[v][id] = removeFromSlice(g.outEdges[v][id], eid)
		}
		delete(g.inEdges[id], v)
	}

	// Remove from compound hierarchy
	if g.isCompound {
		if parent := g.parent[id]; parent != "" || g.children[""] != nil {
			delete(g.children[parent], id)
		}
		delete(g.parent, id)

		// Move children to root
		if children := g.children[id]; children != nil {
			for child := range children {
				g.parent[child] = ""
				if g.children[""] == nil {
					g.children[""] = make(map[string]bool)
				}
				g.children[""][child] = true
			}
		}
		delete(g.children, id)
	}

	// Remove node
	delete(g.nodes, id)
	delete(g.outEdges, id)
	delete(g.inEdges, id)
}

func removeFromSlice(slice []string, val string) []string {
	return slices.DeleteFunc(slice, func(v string) bool {
		return v == val
	})
}

// Nodes returns all node IDs in the graph.
func (g *Graph) Nodes() []string {
	return slices.Collect(maps.Keys(g.nodes))
}

// NodeCount returns the number of nodes.
func (g *Graph) NodeCount() int {
	return len(g.nodes)
}

// SetEdge adds or updates an edge in the graph.
func (g *Graph) SetEdge(v, w string, data *EdgeData) {
	g.SetEdgeWithName(v, w, "", data)
}

// SetEdgeWithName adds or updates a named edge (for multigraphs).
func (g *Graph) SetEdgeWithName(v, w, name string, data *EdgeData) {
	if !g.isMulti {
		name = ""
	}
	if !g.directed && w < v {
		v, w = w, v
	}

	// Ensure nodes exist
	if !g.HasNode(v) {
		g.SetNode(v, nil)
	}
	if !g.HasNode(w) {
		g.SetNode(w, nil)
	}

	eid := g.edgeID(v, w, name)

	if data == nil {
		data = &EdgeData{}
	}
	data.V = v
	data.W = w
	data.Name = name

	// Set defaults
	if data.MinLen == 0 {
		data.MinLen = 1
	}
	if data.Weight == 0 {
		data.Weight = 1
	}
	if data.LabelOffset == 0 {
		data.LabelOffset = 10
	}

	if _, exists := g.edges[eid]; !exists {
		// Add to adjacency
		if g.outEdges[v] == nil {
			g.outEdges[v] = make(map[string][]string)
		}
		g.outEdges[v][w] = append(g.outEdges[v][w], eid)

		if g.inEdges[w] == nil {
			g.inEdges[w] = make(map[string][]string)
		}
		g.inEdges[w][v] = append(g.inEdges[w][v], eid)
	}

	g.edges[eid] = data
}

// Edge returns the data for an edge, or nil if not found.
func (g *Graph) Edge(v, w string) *EdgeData {
	return g.EdgeWithName(v, w, "")
}

// EdgeWithName returns the data for a named edge.
func (g *Graph) EdgeWithName(v, w, name string) *EdgeData {
	eid := g.edgeID(v, w, name)
	return g.edges[eid]
}

// HasEdge returns whether an edge exists.
func (g *Graph) HasEdge(v, w string) bool {
	return g.HasEdgeWithName(v, w, "")
}

// HasEdgeWithName returns whether a named edge exists.
func (g *Graph) HasEdgeWithName(v, w, name string) bool {
	eid := g.edgeID(v, w, name)
	_, ok := g.edges[eid]
	return ok
}

// RemoveEdge removes an edge from the graph.
func (g *Graph) RemoveEdge(v, w string) {
	g.RemoveEdgeWithName(v, w, "")
}

// RemoveEdgeWithName removes a named edge from the graph.
func (g *Graph) RemoveEdgeWithName(v, w, name string) {
	eid := g.edgeID(v, w, name)
	if _, exists := g.edges[eid]; !exists {
		return
	}

	delete(g.edges, eid)

	// Remove from adjacency
	if g.outEdges[v] != nil && g.outEdges[v][w] != nil {
		edges := g.outEdges[v][w]
		for i, e := range edges {
			if e == eid {
				g.outEdges[v][w] = append(edges[:i], edges[i+1:]...)
				break
			}
		}
		if len(g.outEdges[v][w]) == 0 {
			delete(g.outEdges[v], w)
		}
	}

	if g.inEdges[w] != nil && g.inEdges[w][v] != nil {
		edges := g.inEdges[w][v]
		for i, e := range edges {
			if e == eid {
				g.inEdges[w][v] = append(edges[:i], edges[i+1:]...)
				break
			}
		}
		if len(g.inEdges[w][v]) == 0 {
			delete(g.inEdges[w], v)
		}
	}
}

// EdgeObj represents an edge as a simple struct.
type EdgeObj struct {
	V    string
	W    string
	Name string
}

// Edges returns all edges in the graph.
func (g *Graph) Edges() []EdgeObj {
	result := make([]EdgeObj, 0, len(g.edges))
	for _, e := range g.edges {
		result = append(result, EdgeObj{V: e.V, W: e.W, Name: e.Name})
	}
	return result
}

// EdgeCount returns the number of edges.
func (g *Graph) EdgeCount() int {
	return len(g.edges)
}

// InEdges returns all edges pointing to node v.
func (g *Graph) InEdges(v string) []EdgeObj {
	var result []EdgeObj
	if g.inEdges[v] == nil {
		return result
	}
	for _, edgeIDs := range g.inEdges[v] {
		for _, eid := range edgeIDs {
			if e := g.edges[eid]; e != nil {
				result = append(result, EdgeObj{V: e.V, W: e.W, Name: e.Name})
			}
		}
	}
	return result
}

// OutEdges returns all edges originating from node v.
func (g *Graph) OutEdges(v string) []EdgeObj {
	var result []EdgeObj
	if g.outEdges[v] == nil {
		return result
	}
	for _, edgeIDs := range g.outEdges[v] {
		for _, eid := range edgeIDs {
			if e := g.edges[eid]; e != nil {
				result = append(result, EdgeObj{V: e.V, W: e.W, Name: e.Name})
			}
		}
	}
	return result
}

// NodeEdges returns all edges incident to node v (both in and out).
func (g *Graph) NodeEdges(v string) []EdgeObj {
	inE := g.InEdges(v)
	outE := g.OutEdges(v)

	// Combine, avoiding duplicates for self-loops
	seen := make(map[string]bool)
	result := make([]EdgeObj, 0, len(inE)+len(outE))

	for _, e := range inE {
		eid := g.edgeID(e.V, e.W, e.Name)
		if !seen[eid] {
			seen[eid] = true
			result = append(result, e)
		}
	}
	for _, e := range outE {
		eid := g.edgeID(e.V, e.W, e.Name)
		if !seen[eid] {
			seen[eid] = true
			result = append(result, e)
		}
	}
	return result
}

// Predecessors returns all nodes with edges pointing to v.
func (g *Graph) Predecessors(v string) []string {
	if g.inEdges[v] == nil {
		return nil
	}
	return slices.Collect(maps.Keys(g.inEdges[v]))
}

// Successors returns all nodes that v has edges pointing to.
func (g *Graph) Successors(v string) []string {
	if g.outEdges[v] == nil {
		return nil
	}
	return slices.Collect(maps.Keys(g.outEdges[v]))
}

// Neighbors returns all nodes adjacent to v (predecessors + successors).
func (g *Graph) Neighbors(v string) []string {
	seen := make(map[string]bool)
	for _, p := range g.Predecessors(v) {
		seen[p] = true
	}
	for _, s := range g.Successors(v) {
		seen[s] = true
	}
	return slices.Collect(maps.Keys(seen))
}

// IsLeaf reports whether a node has no compound children.
func (g *Graph) IsLeaf(v string) bool {
	if !g.isCompound {
		return true
	}
	children := g.Children(v)
	return len(children) == 0
}

// InDegree returns the number of edges pointing to v.
func (g *Graph) InDegree(v string) int {
	count := 0
	if g.inEdges[v] != nil {
		for _, edges := range g.inEdges[v] {
			count += len(edges)
		}
	}
	return count
}

// OutDegree returns the number of edges originating from v.
func (g *Graph) OutDegree(v string) int {
	count := 0
	if g.outEdges[v] != nil {
		for _, edges := range g.outEdges[v] {
			count += len(edges)
		}
	}
	return count
}

// Degree returns the total number of edges incident to v.
func (g *Graph) Degree(v string) int {
	return g.InDegree(v) + g.OutDegree(v)
}

// Sources returns all nodes with no incoming edges.
func (g *Graph) Sources() []string {
	var result []string
	for id := range g.nodes {
		if g.InDegree(id) == 0 {
			result = append(result, id)
		}
	}
	return result
}

// Sinks returns all nodes with no outgoing edges.
func (g *Graph) Sinks() []string {
	var result []string
	for id := range g.nodes {
		if g.OutDegree(id) == 0 {
			result = append(result, id)
		}
	}
	return result
}

// Compound graph operations

// SetParent sets the parent of a node for compound graphs.
func (g *Graph) SetParent(child, parent string) error {
	if !g.isCompound {
		return ErrParentOnNonCompound
	}
	if !g.HasNode(child) {
		return fmt.Errorf("child node %q not in graph", child)
	}
	if parent != "" && !g.HasNode(parent) {
		return fmt.Errorf("parent node %q not in graph", parent)
	}

	// Check for cycle
	if parent != "" {
		for p := parent; p != ""; p = g.parent[p] {
			if p == child {
				return fmt.Errorf("setting parent would create cycle")
			}
		}
	}

	// Remove from old parent
	if oldParent := g.parent[child]; oldParent != "" || g.children[""] != nil {
		delete(g.children[oldParent], child)
	}

	// Add to new parent
	g.parent[child] = parent
	if g.children[parent] == nil {
		g.children[parent] = make(map[string]bool)
	}
	g.children[parent][child] = true

	return nil
}

// Parent returns the parent of a node, or empty string if none.
func (g *Graph) Parent(v string) string {
	if !g.isCompound {
		return ""
	}
	return g.parent[v]
}

// Children returns all direct children of a node.
// Pass empty string to get root nodes.
func (g *Graph) Children(v string) []string {
	if !g.isCompound {
		if v == "" {
			return g.Nodes()
		}
		if g.HasNode(v) {
			return []string{}
		}
		return nil
	}
	if g.children[v] == nil {
		return nil
	}
	return slices.Collect(maps.Keys(g.children[v]))
}

// IsCompound returns whether the graph has any compound structure.
func (g *Graph) IsCompound() bool {
	return g.isCompound
}

// IsDirected returns whether the graph is directed.
func (g *Graph) IsDirected() bool {
	return g.directed
}

// Copy creates a deep copy of the graph.
func (g *Graph) Copy() *Graph {
	ng := NewGraphWithOptions(GraphOptions{
		Directed:   g.directed,
		Multigraph: g.isMulti,
		Compound:   g.isCompound,
	})
	ng.label = g.label
	ng.RankDir = g.RankDir
	ng.Align = g.Align
	ng.NodeSep = g.NodeSep
	ng.EdgeSep = g.EdgeSep
	ng.RankSep = g.RankSep
	ng.MarginX = g.MarginX
	ng.MarginY = g.MarginY
	ng.Ranker = g.Ranker

	// Copy nodes
	for id, data := range g.nodes {
		nd := *data
		ng.SetNode(id, &nd)
	}

	// Copy edges
	for _, e := range g.edges {
		ed := *e
		ng.SetEdgeWithName(e.V, e.W, e.Name, &ed)
	}

	// Copy compound structure
	if g.isCompound {
		for child, parent := range g.parent {
			if parent != "" {
				if err := ng.SetParent(child, parent); err != nil {
					continue
				}
			}
		}
	}

	return ng
}

// Utility functions

// MaxRank returns the maximum rank in the graph.
func (g *Graph) MaxRank() int {
	maxRank := -1
	for _, n := range g.nodes {
		maxRank = max(maxRank, n.Rank)
	}
	return max(maxRank, 0)
}

// NodesByRank returns nodes grouped by their rank.
func (g *Graph) NodesByRank() map[int][]string {
	ranks := make(map[int][]string)
	for id, n := range g.nodes {
		ranks[n.Rank] = append(ranks[n.Rank], id)
	}
	return ranks
}

// FilterNodes returns a new graph with only nodes matching the predicate.
func (g *Graph) FilterNodes(pred func(id string, data *NodeData) bool) *Graph {
	ng := NewGraphWithOptions(GraphOptions{
		Directed:   g.directed,
		Multigraph: g.isMulti,
		Compound:   false,
	})

	for id, data := range g.nodes {
		if pred(id, data) {
			nd := *data
			ng.SetNode(id, &nd)
		}
	}

	for _, e := range g.edges {
		if ng.HasNode(e.V) && ng.HasNode(e.W) {
			ed := *e
			ng.SetEdgeWithName(e.V, e.W, e.Name, &ed)
		}
	}

	return ng
}

// Iterator methods for Go 1.25+ support

// NodesIter returns an iterator over all node IDs in the graph.
// This provides lazy iteration without allocating a slice.
func (g *Graph) NodesIter() iter.Seq[string] {
	return maps.Keys(g.nodes)
}

// NodesDataIter returns an iterator over all nodes with their data.
// This provides lazy iteration without allocating a slice.
func (g *Graph) NodesDataIter() iter.Seq2[string, *NodeData] {
	return maps.All(g.nodes)
}

// EdgesIter returns an iterator over all edges in the graph.
// This provides lazy iteration without allocating a slice.
func (g *Graph) EdgesIter() iter.Seq[EdgeObj] {
	return func(yield func(EdgeObj) bool) {
		for _, e := range g.edges {
			if !yield(EdgeObj{V: e.V, W: e.W, Name: e.Name}) {
				return
			}
		}
	}
}

// EdgesDataIter returns an iterator over all edges with their data.
// This provides lazy iteration without allocating a slice.
func (g *Graph) EdgesDataIter() iter.Seq2[EdgeObj, *EdgeData] {
	return func(yield func(EdgeObj, *EdgeData) bool) {
		for _, e := range g.edges {
			if !yield(EdgeObj{V: e.V, W: e.W, Name: e.Name}, e) {
				return
			}
		}
	}
}
