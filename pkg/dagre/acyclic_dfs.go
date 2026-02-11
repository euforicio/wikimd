package dagre

// findFeedbackArcSetDFS finds a feedback arc set via DFS back-edge detection.
func findFeedbackArcSetDFS(g *Graph) []EdgeObj {
	fas := make([]EdgeObj, 0)
	visited := make(map[string]bool, g.NodeCount())
	stack := make(map[string]bool, g.NodeCount())

	var dfs func(v string)
	dfs = func(v string) {
		if visited[v] {
			return
		}

		visited[v] = true
		stack[v] = true

		for _, edge := range g.OutEdges(v) {
			if stack[edge.W] {
				fas = append(fas, edge)
			} else {
				dfs(edge.W)
			}
		}

		delete(stack, v)
	}

	for _, v := range g.Nodes() {
		dfs(v)
	}

	return fas
}
