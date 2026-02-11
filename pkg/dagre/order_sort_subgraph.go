package dagre

import (
	"sort"
)

type barycenterEntry struct {
	V             string
	VS            []string
	I             int
	Barycenter    float64
	Weight        float64
	HasBarycenter bool
}

type sortResult struct {
	VS            []string
	Barycenter    float64
	Weight        float64
	HasBarycenter bool
}

type conflictEntry struct {
	VS            []string
	I             int
	Barycenter    float64
	Weight        float64
	HasBarycenter bool

	indegree int
	in       []*conflictEntry
	out      []*conflictEntry
	merged   bool
}

func addSubgraphConstraints(g, cg *Graph, vs []string) {
	prev := map[string]string{}
	rootPrev := ""

	for _, v := range vs {
		child := g.Parent(v)
		for child != "" {
			parent := g.Parent(child)

			var prevChild string
			if parent != "" {
				prevChild = prev[parent]
				prev[parent] = child
			} else {
				prevChild = rootPrev
				rootPrev = child
			}

			if prevChild != "" && prevChild != child {
				cg.SetEdge(prevChild, child, nil)
				break
			}

			child = parent
		}
	}
}

func sortSubgraph(g *Graph, v string, cg *Graph, biasRight bool) sortResult {
	movable := append([]string(nil), g.Children(v)...)
	node := g.Node(v)
	bl, br := layerGraphBorders(node)
	subgraphs := map[string]sortResult{}

	if bl != "" {
		filtered := movable[:0]
		for _, w := range movable {
			if w != bl && w != br {
				filtered = append(filtered, w)
			}
		}
		movable = filtered
	}

	barycenters := barycenter(g, movable)
	for i := range barycenters {
		entry := &barycenters[i]
		if len(g.Children(entry.V)) == 0 {
			continue
		}
		subgraphResult := sortSubgraph(g, entry.V, cg, biasRight)
		subgraphs[entry.V] = subgraphResult
		if subgraphResult.HasBarycenter {
			mergeBarycenters(entry, subgraphResult)
		}
	}

	entries := resolveConflicts(barycenters, cg)
	expandSubgraphs(entries, subgraphs)
	result := sortEntries(entries, biasRight)

	if bl != "" {
		newVS := make([]string, 0, len(result.VS)+2)
		newVS = append(newVS, bl)
		newVS = append(newVS, result.VS...)
		newVS = append(newVS, br)
		result.VS = newVS

		blPreds := g.Predecessors(bl)
		brPreds := g.Predecessors(br)
		if len(blPreds) > 0 && len(brPreds) > 0 {
			blPred := g.Node(blPreds[0])
			brPred := g.Node(brPreds[0])
			if blPred != nil && brPred != nil {
				if !result.HasBarycenter {
					result.HasBarycenter = true
					result.Barycenter = 0
					result.Weight = 0
				}
				result.Barycenter =
					(result.Barycenter*result.Weight + float64(blPred.Order+brPred.Order)) /
						(result.Weight + 2)
				result.Weight += 2
			}
		}
	}

	return result
}

func barycenter(g *Graph, movable []string) []barycenterEntry {
	result := make([]barycenterEntry, 0, len(movable))
	for _, v := range movable {
		entry := barycenterEntry{V: v, VS: []string{v}}
		inEdges := g.InEdges(v)
		if len(inEdges) == 0 {
			result = append(result, entry)
			continue
		}

		sum := 0.0
		weight := 0.0
		for _, e := range inEdges {
			edge := g.EdgeWithName(e.V, e.W, e.Name)
			nodeU := g.Node(e.V)
			if edge == nil || nodeU == nil {
				continue
			}
			sum += edge.Weight * float64(nodeU.Order)
			weight += edge.Weight
		}

		if weight > 0 {
			entry.Barycenter = sum / weight
			entry.Weight = weight
			entry.HasBarycenter = true
		}
		result = append(result, entry)
	}
	return result
}

func resolveConflicts(entries []barycenterEntry, cg *Graph) []barycenterEntry {
	mapped := make(map[string]*conflictEntry, len(entries))
	for i, entry := range entries {
		tmp := &conflictEntry{
			indegree:      0,
			in:            nil,
			out:           nil,
			VS:            []string{entry.V},
			I:             i,
			HasBarycenter: entry.HasBarycenter,
			Barycenter:    entry.Barycenter,
			Weight:        entry.Weight,
		}
		mapped[entry.V] = tmp
	}

	for _, e := range cg.Edges() {
		entryV := mapped[e.V]
		entryW := mapped[e.W]
		if entryV == nil || entryW == nil {
			continue
		}
		entryW.indegree++
		entryV.out = append(entryV.out, entryW)
	}

	sourceSet := make([]*conflictEntry, 0, len(mapped))
	for _, entry := range mapped {
		if entry.indegree == 0 {
			sourceSet = append(sourceSet, entry)
		}
	}

	resolved := doResolveConflicts(sourceSet)
	result := make([]barycenterEntry, 0, len(resolved))
	for _, entry := range resolved {
		result = append(result, barycenterEntry{
			VS:            append([]string(nil), entry.VS...),
			I:             entry.I,
			Barycenter:    entry.Barycenter,
			Weight:        entry.Weight,
			HasBarycenter: entry.HasBarycenter,
		})
	}

	return result
}

func doResolveConflicts(sourceSet []*conflictEntry) []*conflictEntry {
	entries := make([]*conflictEntry, 0, len(sourceSet))

	for len(sourceSet) > 0 {
		entry := sourceSet[len(sourceSet)-1]
		sourceSet = sourceSet[:len(sourceSet)-1]
		entries = append(entries, entry)

		for i := len(entry.in) - 1; i >= 0; i-- {
			uEntry := entry.in[i]
			if uEntry.merged {
				continue
			}
			if !uEntry.HasBarycenter || !entry.HasBarycenter || uEntry.Barycenter >= entry.Barycenter {
				mergeEntries(entry, uEntry)
			}
		}

		for _, wEntry := range entry.out {
			wEntry.in = append(wEntry.in, entry)
			wEntry.indegree--
			if wEntry.indegree == 0 {
				sourceSet = append(sourceSet, wEntry)
			}
		}
	}

	result := make([]*conflictEntry, 0, len(entries))
	for _, entry := range entries {
		if !entry.merged {
			result = append(result, entry)
		}
	}
	return result
}

func mergeEntries(target, source *conflictEntry) {
	sum := 0.0
	weight := 0.0

	if target.HasBarycenter && target.Weight > 0 {
		sum += target.Barycenter * target.Weight
		weight += target.Weight
	}
	if source.HasBarycenter && source.Weight > 0 {
		sum += source.Barycenter * source.Weight
		weight += source.Weight
	}

	target.VS = append(append([]string(nil), source.VS...), target.VS...)
	target.I = min(target.I, source.I)
	if weight > 0 {
		target.Barycenter = sum / weight
		target.Weight = weight
		target.HasBarycenter = true
	}
	source.merged = true
}

func expandSubgraphs(entries []barycenterEntry, subgraphs map[string]sortResult) {
	for i := range entries {
		expanded := make([]string, 0, len(entries[i].VS))
		for _, v := range entries[i].VS {
			if subgraph, ok := subgraphs[v]; ok {
				expanded = append(expanded, subgraph.VS...)
			} else {
				expanded = append(expanded, v)
			}
		}
		entries[i].VS = expanded
	}
}

func mergeBarycenters(target *barycenterEntry, other sortResult) {
	if target.HasBarycenter {
		target.Barycenter = (target.Barycenter*target.Weight + other.Barycenter*other.Weight) /
			(target.Weight + other.Weight)
		target.Weight += other.Weight
		return
	}
	target.Barycenter = other.Barycenter
	target.Weight = other.Weight
	target.HasBarycenter = true
}

func sortEntries(entries []barycenterEntry, biasRight bool) sortResult {
	sortable := make([]barycenterEntry, 0, len(entries))
	unsortable := make([]barycenterEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.HasBarycenter {
			sortable = append(sortable, entry)
		} else {
			unsortable = append(unsortable, entry)
		}
	}

	sort.Slice(unsortable, func(i, j int) bool {
		return unsortable[i].I > unsortable[j].I
	})
	sort.Slice(sortable, func(i, j int) bool {
		if sortable[i].Barycenter < sortable[j].Barycenter {
			return true
		}
		if sortable[i].Barycenter > sortable[j].Barycenter {
			return false
		}
		if biasRight {
			return sortable[i].I > sortable[j].I
		}
		return sortable[i].I < sortable[j].I
	})

	vs := make([]string, 0, len(entries))
	sum := 0.0
	weight := 0.0
	vsIndex := 0

	vsIndex = consumeUnsortable(&vs, &unsortable, vsIndex)

	for _, entry := range sortable {
		vsIndex += len(entry.VS)
		vs = append(vs, entry.VS...)
		sum += entry.Barycenter * entry.Weight
		weight += entry.Weight
		vsIndex = consumeUnsortable(&vs, &unsortable, vsIndex)
	}

	result := sortResult{VS: vs}
	if weight > 0 {
		result.Barycenter = sum / weight
		result.Weight = weight
		result.HasBarycenter = true
	}
	return result
}

func consumeUnsortable(vs *[]string, unsortable *[]barycenterEntry, index int) int {
	for len(*unsortable) > 0 {
		last := (*unsortable)[len(*unsortable)-1]
		if last.I > index {
			break
		}
		*unsortable = (*unsortable)[:len(*unsortable)-1]
		*vs = append(*vs, last.VS...)
		index++
	}
	return index
}

func layerGraphBorders(node *NodeData) (string, string) {
	if node == nil {
		return "", ""
	}
	left := firstBorderEntry(node.BorderLeft)
	right := firstBorderEntry(node.BorderRight)
	if left == "" || right == "" {
		return "", ""
	}
	return left, right
}

func firstBorderEntry(values map[int]string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
