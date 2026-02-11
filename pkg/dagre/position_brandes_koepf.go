package dagre

import (
	"math"
	"sort"
	"strings"
)

// positionX computes horizontal coordinates using Brandes-Koepf.
func positionX(g *Graph) map[string]float64 {
	layering := buildLayerMatrix(g)
	conflicts := findType1Conflicts(g, layering)
	for v, ws := range findType2Conflicts(g, layering) {
		conflicts[v] = ws
	}

	xss := map[string]map[string]float64{}
	for _, vert := range []string{"u", "d"} {
		adjustedLayering := copyLayeringMatrix(layering)
		if vert == "d" {
			reverseLayers(adjustedLayering)
		}

		for _, horiz := range []string{"l", "r"} {
			workingLayering := copyLayeringMatrix(adjustedLayering)
			if horiz == "r" {
				for i := range workingLayering {
					reverseStrings(workingLayering[i])
				}
			}

			neighborFn := g.Predecessors
			if vert != "u" {
				neighborFn = g.Successors
			}

			root, align := verticalAlignment(g, workingLayering, conflicts, neighborFn)
			xs := horizontalCompaction(g, workingLayering, root, align, horiz == "r")
			if horiz == "r" {
				for v, x := range xs {
					xs[v] = -x
				}
			}
			xss[vert+horiz] = xs
		}
	}

	alignKey, alignTo := findSmallestWidthAlignment(g, xss)
	alignCoordinates(xss, alignKey, alignTo)
	return balance(xss, g.Align)
}

func findType1Conflicts(g *Graph, layering [][]string) map[string]map[string]bool {
	conflicts := map[string]map[string]bool{}
	if len(layering) == 0 {
		return conflicts
	}

	for i := 1; i < len(layering); i++ {
		prevLayer := layering[i-1]
		layer := layering[i]
		if len(layer) == 0 {
			continue
		}

		k0 := 0
		scanPos := 0
		prevLayerLength := len(prevLayer)
		lastNode := layer[len(layer)-1]

		for idx, v := range layer {
			w := findOtherInnerSegmentNode(g, v)
			k1 := prevLayerLength
			if w != "" {
				if wn := g.Node(w); wn != nil {
					k1 = wn.Order
				}
			}

			if w != "" || v == lastNode {
				for _, scanNode := range layer[scanPos : idx+1] {
					for _, u := range g.Predecessors(scanNode) {
						uLabel := g.Node(u)
						scanLabel := g.Node(scanNode)
						if uLabel == nil || scanLabel == nil {
							continue
						}
						uPos := uLabel.Order
						if (uPos < k0 || k1 < uPos) && !(uLabel.Dummy != "" && scanLabel.Dummy != "") {
							addConflict(conflicts, u, scanNode)
						}
					}
				}
				scanPos = idx + 1
				k0 = k1
			}
		}
	}

	return conflicts
}

func findType2Conflicts(g *Graph, layering [][]string) map[string]map[string]bool {
	conflicts := map[string]map[string]bool{}
	if len(layering) == 0 {
		return conflicts
	}

	for i := 1; i < len(layering); i++ {
		north := layering[i-1]
		south := layering[i]

		prevNorthPos := -1
		nextNorthPos := 0
		hasNextNorthPos := false
		southPos := 0

		for southLookahead, v := range south {
			node := g.Node(v)
			if node != nil && node.Dummy == "border" {
				predecessors := g.Predecessors(v)
				if len(predecessors) > 0 {
					if pred := g.Node(predecessors[0]); pred != nil {
						nextNorthPos = pred.Order
						hasNextNorthPos = true
						scanType2Conflicts(g, conflicts, south, southPos, southLookahead, prevNorthPos, true, nextNorthPos, true)
						southPos = southLookahead
						prevNorthPos = nextNorthPos
					}
				}
			}

			scanType2Conflicts(g, conflicts, south, southPos, len(south), nextNorthPos, hasNextNorthPos, len(north), true)
		}
	}

	return conflicts
}

func scanType2Conflicts(
	g *Graph,
	conflicts map[string]map[string]bool,
	south []string,
	southPos, southEnd int,
	prevNorthBorder int,
	hasPrevNorthBorder bool,
	nextNorthBorder int,
	hasNextNorthBorder bool,
) {
	for i := southPos; i < southEnd; i++ {
		v := south[i]
		vNode := g.Node(v)
		if vNode == nil || vNode.Dummy == "" {
			continue
		}
		for _, u := range g.Predecessors(v) {
			uNode := g.Node(u)
			if uNode == nil || uNode.Dummy == "" {
				continue
			}
			if (hasPrevNorthBorder && uNode.Order < prevNorthBorder) ||
				(hasNextNorthBorder && uNode.Order > nextNorthBorder) {
				addConflict(conflicts, u, v)
			}
		}
	}
}

func findOtherInnerSegmentNode(g *Graph, v string) string {
	node := g.Node(v)
	if node == nil || node.Dummy == "" {
		return ""
	}
	for _, u := range g.Predecessors(v) {
		uNode := g.Node(u)
		if uNode != nil && uNode.Dummy != "" {
			return u
		}
	}
	return ""
}

func addConflict(conflicts map[string]map[string]bool, v, w string) {
	if v > w {
		v, w = w, v
	}
	conflictsV := conflicts[v]
	if conflictsV == nil {
		conflictsV = map[string]bool{}
		conflicts[v] = conflictsV
	}
	conflictsV[w] = true
}

func hasConflict(conflicts map[string]map[string]bool, v, w string) bool {
	if v > w {
		v, w = w, v
	}
	conflictsV := conflicts[v]
	if conflictsV == nil {
		return false
	}
	return conflictsV[w]
}

func verticalAlignment(
	g *Graph,
	layering [][]string,
	conflicts map[string]map[string]bool,
	neighborFn func(string) []string,
) (map[string]string, map[string]string) {
	root := map[string]string{}
	align := map[string]string{}
	pos := map[string]int{}

	for _, layer := range layering {
		for order, v := range layer {
			root[v] = v
			align[v] = v
			pos[v] = order
		}
	}

	for _, layer := range layering {
		prevIdx := -1
		for _, v := range layer {
			ws := append([]string(nil), neighborFn(v)...)
			if len(ws) == 0 {
				continue
			}
			sort.Slice(ws, func(i, j int) bool {
				return pos[ws[i]] < pos[ws[j]]
			})
			mp := float64(len(ws)-1) / 2.0
			for i := int(math.Floor(mp)); i <= int(math.Ceil(mp)); i++ {
				w := ws[i]
				if align[v] == v && prevIdx < pos[w] && !hasConflict(conflicts, v, w) {
					align[w] = v
					root[v] = root[w]
					align[v] = root[v]
					prevIdx = pos[w]
				}
			}
		}
	}

	return root, align
}

func horizontalCompaction(
	g *Graph,
	layering [][]string,
	root map[string]string,
	align map[string]string,
	reverseSep bool,
) map[string]float64 {
	xs := map[string]float64{}
	blockG := buildBlockGraph(g, layering, root, reverseSep)
	borderType := "borderRight"
	if reverseSep {
		borderType = "borderLeft"
	}

	iterate := func(setXsFunc func(string), nextNodesFunc func(string) []string) {
		stack := append([]string(nil), blockG.Nodes()...)
		if len(stack) == 0 {
			return
		}
		visited := map[string]bool{}
		elem := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for {
			if visited[elem] {
				setXsFunc(elem)
			} else {
				visited[elem] = true
				stack = append(stack, elem)
				stack = append(stack, nextNodesFunc(elem)...)
			}

			if len(stack) == 0 {
				break
			}
			elem = stack[len(stack)-1]
			stack = stack[:len(stack)-1]
		}
	}

	pass1 := func(elem string) {
		best := 0.0
		for _, e := range blockG.InEdges(elem) {
			edge := blockG.EdgeWithName(e.V, e.W, e.Name)
			if edge == nil {
				continue
			}
			candidate := xs[e.V] + edge.Weight
			if candidate > best {
				best = candidate
			}
		}
		xs[elem] = best
	}

	pass2 := func(elem string) {
		minVal := math.Inf(1)
		for _, e := range blockG.OutEdges(elem) {
			edge := blockG.EdgeWithName(e.V, e.W, e.Name)
			if edge == nil {
				continue
			}
			candidate := xs[e.W] - edge.Weight
			if candidate < minVal {
				minVal = candidate
			}
		}

		node := g.Node(elem)
		if !math.IsInf(minVal, 1) && (node == nil || node.BorderType != borderType) {
			xs[elem] = max(xs[elem], minVal)
		}
	}

	iterate(pass1, blockG.Predecessors)
	iterate(pass2, blockG.Successors)

	for v := range align {
		xs[v] = xs[root[v]]
	}

	return xs
}

func buildBlockGraph(g *Graph, layering [][]string, root map[string]string, reverseSep bool) *Graph {
	blockGraph := NewGraphWithOptions(GraphOptions{Directed: true, Multigraph: false, Compound: false})
	sepFn := sep(g.NodeSep, g.EdgeSep, reverseSep)

	for _, layer := range layering {
		var u string
		hasU := false
		for _, v := range layer {
			vRoot := root[v]
			if vRoot == "" {
				vRoot = v
			}
			if !blockGraph.HasNode(vRoot) {
				blockGraph.SetNode(vRoot, &NodeData{})
			}

			if hasU {
				uRoot := root[u]
				if uRoot == "" {
					uRoot = u
				}
				prevMax := 0.0
				if edge := blockGraph.Edge(uRoot, vRoot); edge != nil {
					prevMax = edge.Weight
				}
				sepVal := sepFn(g, v, u)
				if prevMax > sepVal {
					sepVal = prevMax
				}
				blockGraph.SetEdge(uRoot, vRoot, &EdgeData{Weight: sepVal})
			}

			u = v
			hasU = true
		}
	}

	return blockGraph
}

func findSmallestWidthAlignment(g *Graph, xss map[string]map[string]float64) (string, map[string]float64) {
	bestKey := ""
	bestWidth := math.Inf(1)
	var best map[string]float64

	for key, xs := range xss {
		maxX := math.Inf(-1)
		minX := math.Inf(1)
		for v, x := range xs {
			halfWidth := nodeWidth(g, v) / 2
			maxX = max(maxX, x+halfWidth)
			minX = min(minX, x-halfWidth)
		}
		w := maxX - minX
		if w < bestWidth {
			bestWidth = w
			bestKey = key
			best = xs
		}
	}

	return bestKey, best
}

func alignCoordinates(xss map[string]map[string]float64, alignKey string, alignTo map[string]float64) {
	if len(alignTo) == 0 {
		return
	}
	alignToMin := mapMin(alignTo)
	alignToMax := mapMax(alignTo)

	for _, vert := range []string{"u", "d"} {
		for _, horiz := range []string{"l", "r"} {
			alignment := vert + horiz
			xs := xss[alignment]
			if alignment == alignKey || len(xs) == 0 {
				continue
			}

			delta := alignToMin - mapMin(xs)
			if horiz != "l" {
				delta = alignToMax - mapMax(xs)
			}
			if delta != 0 {
				for v, x := range xs {
					xs[v] = x + delta
				}
			}
		}
	}
}

func balance(xss map[string]map[string]float64, align string) map[string]float64 {
	result := map[string]float64{}
	ul := xss["ul"]
	if len(ul) == 0 {
		return result
	}

	if align != "" {
		alignKey := strings.ToLower(align)
		if xs, ok := xss[alignKey]; ok {
			for v := range ul {
				result[v] = xs[v]
			}
			return result
		}
	}

	for v := range ul {
		vals := make([]float64, 0, 4)
		for _, key := range []string{"ul", "ur", "dl", "dr"} {
			vals = append(vals, xss[key][v])
		}
		sort.Float64s(vals)
		result[v] = (vals[1] + vals[2]) / 2
	}
	return result
}

func sep(nodeSep, edgeSep float64, reverseSep bool) func(*Graph, string, string) float64 {
	return func(g *Graph, v, w string) float64 {
		vLabel := g.Node(v)
		wLabel := g.Node(w)
		if vLabel == nil {
			vLabel = &NodeData{}
		}
		if wLabel == nil {
			wLabel = &NodeData{}
		}

		sum := 0.0
		delta := 0.0

		sum += vLabel.Width / 2
		switch strings.ToLower(vLabel.LabelPos) {
		case "l":
			delta = -vLabel.Width / 2
		case "r":
			delta = vLabel.Width / 2
		}
		if delta != 0 {
			if reverseSep {
				sum += delta
			} else {
				sum -= delta
			}
		}
		delta = 0

		if vLabel.Dummy != "" {
			sum += edgeSep / 2
		} else {
			sum += nodeSep / 2
		}
		if wLabel.Dummy != "" {
			sum += edgeSep / 2
		} else {
			sum += nodeSep / 2
		}

		sum += wLabel.Width / 2
		switch strings.ToLower(wLabel.LabelPos) {
		case "l":
			delta = wLabel.Width / 2
		case "r":
			delta = -wLabel.Width / 2
		}
		if delta != 0 {
			if reverseSep {
				sum += delta
			} else {
				sum -= delta
			}
		}

		return sum
	}
}

func nodeWidth(g *Graph, v string) float64 {
	n := g.Node(v)
	if n == nil {
		return 0
	}
	return n.Width
}

func copyLayeringMatrix(layering [][]string) [][]string {
	out := make([][]string, len(layering))
	for i := range layering {
		out[i] = append([]string(nil), layering[i]...)
	}
	return out
}

func reverseLayers(layering [][]string) {
	for i, j := 0, len(layering)-1; i < j; i, j = i+1, j-1 {
		layering[i], layering[j] = layering[j], layering[i]
	}
}

func reverseStrings(values []string) {
	for i, j := 0, len(values)-1; i < j; i, j = i+1, j-1 {
		values[i], values[j] = values[j], values[i]
	}
}

func mapMin(xs map[string]float64) float64 {
	m := math.Inf(1)
	for _, x := range xs {
		if x < m {
			m = x
		}
	}
	if math.IsInf(m, 1) {
		return 0
	}
	return m
}

func mapMax(xs map[string]float64) float64 {
	m := math.Inf(-1)
	for _, x := range xs {
		if x > m {
			m = x
		}
	}
	if math.IsInf(m, -1) {
		return 0
	}
	return m
}
