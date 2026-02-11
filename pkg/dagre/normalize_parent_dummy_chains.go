package dagre

func parentDummyChains(g *Graph) {
	opts := layoutOptionsForGraph(g)
	if opts == nil {
		return
	}

	postorderNums := postorderNums(g)

	for _, vStart := range opts.pipeline.dummyChains {
		v := vStart
		node := g.Node(v)
		if node == nil || node.EdgeObj == nil {
			continue
		}

		edgeObj := node.EdgeObj
		path, lca := findPathToLCA(g, postorderNums, edgeObj.V, edgeObj.W)
		if len(path) == 0 {
			continue
		}

		pathIdx := 0
		pathV := path[pathIdx]
		ascending := true

		for v != edgeObj.W {
			node = g.Node(v)
			if node == nil {
				break
			}

			if ascending {
				for pathIdx < len(path) && pathV != lca {
					pvNode := g.Node(pathV)
					if pvNode == nil || pvNode.MaxRank >= node.Rank {
						break
					}
					pathIdx++
					if pathIdx >= len(path) {
						break
					}
					pathV = path[pathIdx]
				}

				if pathV == lca {
					ascending = false
				}
			}

			if !ascending {
				for pathIdx < len(path)-1 {
					nextPathV := path[pathIdx+1]
					nextNode := g.Node(nextPathV)
					if nextNode == nil || nextNode.MinRank > node.Rank {
						break
					}
					pathIdx++
					pathV = path[pathIdx]
				}
			}

			_ = g.SetParent(v, pathV)
			succ := g.Successors(v)
			if len(succ) == 0 {
				break
			}
			v = succ[0]
		}
	}
}

type lowLim struct {
	low int
	lim int
}

func findPathToLCA(g *Graph, order map[string]lowLim, v, w string) ([]string, string) {
	vInfo, vok := order[v]
	wInfo, wok := order[w]
	if !vok || !wok {
		return nil, ""
	}

	vPath := []string{}
	wPath := []string{}
	low := min(vInfo.low, wInfo.low)
	lim := max(vInfo.lim, wInfo.lim)

	parent := v
	lca := ""
	for {
		parent = g.Parent(parent)
		if parent == "" {
			break
		}
		vPath = append(vPath, parent)
		info, ok := order[parent]
		if !ok {
			break
		}
		if info.low <= low && lim <= info.lim {
			lca = parent
			break
		}
	}

	if lca == "" {
		return nil, ""
	}

	parent = w
	for {
		parent = g.Parent(parent)
		if parent == "" || parent == lca {
			break
		}
		wPath = append(wPath, parent)
	}

	for i, j := 0, len(wPath)-1; i < j; i, j = i+1, j-1 {
		wPath[i], wPath[j] = wPath[j], wPath[i]
	}

	return append(vPath, wPath...), lca
}

func postorderNums(g *Graph) map[string]lowLim {
	result := map[string]lowLim{}
	lim := 0

	var dfs func(v string)
	dfs = func(v string) {
		low := lim
		for _, child := range g.Children(v) {
			dfs(child)
		}
		result[v] = lowLim{low: low, lim: lim}
		lim++
	}

	for _, v := range g.Children("") {
		dfs(v)
	}

	return result
}
