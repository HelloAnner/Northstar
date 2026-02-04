package dagcalc

import "sort"

// ImpactRange 计算影响范围（双向遍历）
func ImpactRange(g *Graph, anchor NodeID) []ImpactNode {
	if g == nil {
		return nil
	}
	visited := map[NodeID]bool{}
	queue := []NodeID{anchor}
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		if visited[node] {
			continue
		}
		visited[node] = true
		for _, next := range g.Edges[node] {
			if !visited[next] {
				queue = append(queue, next)
			}
		}
		for _, next := range g.ReverseEdges[node] {
			if !visited[next] {
				queue = append(queue, next)
			}
		}
	}

	out := make([]ImpactNode, 0, len(visited))
	for node := range visited {
		item := ImpactNode{NodeID: string(node)}
		if coord, ok := g.UICoords[node]; ok {
			item.UICoord = &coord
		}
		if id, ok := g.IndicatorIDs[node]; ok {
			item.IndicatorID = id
		}
		if coords, ok := g.ExcelCoords[node]; ok {
			item.ExcelCoords = coords
		}
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].NodeID < out[j].NodeID
	})
	return out
}
