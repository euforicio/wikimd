package dagre

import "container/list"

type fasResultEdge struct {
	v string
	w string
}

type fasNodeEntry struct {
	v   string
	in  int
	out int

	bucket *fasBucket
	elem   *list.Element
}

type fasBucket struct {
	entries list.List
}

func (b *fasBucket) enqueue(entry *fasNodeEntry) {
	if entry == nil {
		return
	}
	if entry.bucket != nil {
		entry.bucket.unlink(entry)
	}
	entry.elem = b.entries.PushFront(entry)
	entry.bucket = b
}

func (b *fasBucket) dequeue() *fasNodeEntry {
	elem := b.entries.Back()
	if elem == nil {
		return nil
	}
	entry, _ := elem.Value.(*fasNodeEntry)
	if entry == nil {
		return nil
	}
	b.unlink(entry)
	return entry
}

func (b *fasBucket) unlink(entry *fasNodeEntry) {
	if entry == nil || entry.elem == nil {
		return
	}
	b.entries.Remove(entry.elem)
	entry.elem = nil
	entry.bucket = nil
}

type fasState struct {
	fasGraph *Graph
	entries  map[string]*fasNodeEntry
	buckets  []*fasBucket
	zeroIdx  int
}

func (s *fasState) sources() *fasBucket {
	return s.buckets[len(s.buckets)-1]
}

func (s *fasState) sinks() *fasBucket {
	return s.buckets[0]
}

func (s *fasState) assignBucket(entry *fasNodeEntry) {
	if entry == nil || s.fasGraph == nil || !s.fasGraph.HasNode(entry.v) {
		return
	}

	if entry.out == 0 {
		s.sinks().enqueue(entry)
		return
	}
	if entry.in == 0 {
		s.sources().enqueue(entry)
		return
	}

	idx := entry.out - entry.in + s.zeroIdx
	if idx >= 0 && idx < len(s.buckets) {
		s.buckets[idx].enqueue(entry)
	}
}

func (s *fasState) removeNode(entry *fasNodeEntry, collectPredecessors bool, results *[]fasResultEdge) {
	if s.fasGraph == nil || entry == nil || !s.fasGraph.HasNode(entry.v) {
		return
	}

	v := entry.v

	for _, edge := range s.fasGraph.InEdges(v) {
		weight := 1
		if label := s.fasGraph.EdgeWithName(edge.V, edge.W, edge.Name); label != nil {
			weight = int(label.Weight)
		}

		if collectPredecessors {
			*results = append(*results, fasResultEdge{v: edge.V, w: edge.W})
		}

		uEntry := s.entries[edge.V]
		if uEntry != nil {
			uEntry.out -= weight
			s.assignBucket(uEntry)
		}
	}

	for _, edge := range s.fasGraph.OutEdges(v) {
		weight := 1
		if label := s.fasGraph.EdgeWithName(edge.V, edge.W, edge.Name); label != nil {
			weight = int(label.Weight)
		}

		wEntry := s.entries[edge.W]
		if wEntry != nil {
			wEntry.in -= weight
			s.assignBucket(wEntry)
		}
	}

	s.fasGraph.RemoveNode(v)
}

func buildFASState(g *Graph) *fasState {
	state := &fasState{
		fasGraph: NewGraphWithOptions(GraphOptions{Directed: true, Multigraph: false, Compound: false}),
		entries:  make(map[string]*fasNodeEntry, g.NodeCount()),
	}

	maxIn := 0
	maxOut := 0

	nodeOrder := make([]string, 0, g.NodeCount())
	seen := make(map[string]bool, g.NodeCount())

	for _, edge := range g.Edges() {
		if !seen[edge.V] {
			seen[edge.V] = true
			nodeOrder = append(nodeOrder, edge.V)
		}
		if !seen[edge.W] {
			seen[edge.W] = true
			nodeOrder = append(nodeOrder, edge.W)
		}
	}

	for _, v := range g.Nodes() {
		if seen[v] {
			continue
		}
		seen[v] = true
		nodeOrder = append(nodeOrder, v)
	}

	for _, v := range nodeOrder {
		state.fasGraph.SetNode(v, &NodeData{})
		state.entries[v] = &fasNodeEntry{v: v}
	}

	for _, edge := range g.Edges() {
		weight := 1
		if label := g.EdgeWithName(edge.V, edge.W, edge.Name); label != nil {
			weight = int(label.Weight)
		}

		if existing := state.fasGraph.Edge(edge.V, edge.W); existing != nil {
			existing.Weight += float64(weight)
		} else {
			state.fasGraph.SetEdge(edge.V, edge.W, &EdgeData{Weight: float64(weight), MinLen: 1})
		}

		vEntry := state.entries[edge.V]
		wEntry := state.entries[edge.W]
		if vEntry != nil {
			vEntry.out += weight
			if vEntry.out > maxOut {
				maxOut = vEntry.out
			}
		}
		if wEntry != nil {
			wEntry.in += weight
			if wEntry.in > maxIn {
				maxIn = wEntry.in
			}
		}
	}

	bucketCount := maxOut + maxIn + 3
	if bucketCount < 3 {
		bucketCount = 3
	}

	state.zeroIdx = maxIn + 1
	state.buckets = make([]*fasBucket, bucketCount)
	for i := range state.buckets {
		state.buckets[i] = &fasBucket{}
	}

	for _, v := range nodeOrder {
		state.assignBucket(state.entries[v])
	}

	return state
}

func doGreedyFAS(state *fasState) []fasResultEdge {
	results := make([]fasResultEdge, 0)
	if state == nil || state.fasGraph == nil {
		return results
	}

	for state.fasGraph.NodeCount() > 0 {
		for {
			entry := state.sinks().dequeue()
			if entry == nil {
				break
			}
			state.removeNode(entry, false, &results)
		}

		for {
			entry := state.sources().dequeue()
			if entry == nil {
				break
			}
			state.removeNode(entry, false, &results)
		}

		if state.fasGraph.NodeCount() == 0 {
			continue
		}

		removed := false
		for i := len(state.buckets) - 2; i >= 1; i-- {
			entry := state.buckets[i].dequeue()
			if entry == nil {
				continue
			}
			state.removeNode(entry, true, &results)
			removed = true
			break
		}
		if !removed {
			break
		}
	}

	return results
}

// findFeedbackArcSetGreedy finds a feedback arc set with the greedy heuristic.
func findFeedbackArcSetGreedy(g *Graph) []EdgeObj {
	if g.NodeCount() <= 1 {
		return nil
	}

	state := buildFASState(g)
	resultEdges := doGreedyFAS(state)

	fas := make([]EdgeObj, 0)
	for _, result := range resultEdges {
		for _, edge := range g.OutEdges(result.v) {
			if edge.W == result.w {
				fas = append(fas, edge)
			}
		}
	}

	return fas
}
