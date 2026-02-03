/**
 * 联动预览构图与影响范围计算
 *
 * @author Anner
 * Created on 2026/2/3
 */

package linkage

import (
	"errors"
	"sort"
	"strings"

	"northstar/internal/model"
)

// BuildGraphOptions 构图参数
type BuildGraphOptions struct {
	WRRecords     []*model.WholesaleRetail
	ACRecords     []*model.AccommodationCatering
	TemplateIndex *TemplateIndex
}

// Graph 联动 DAG 图
type Graph struct {
	Edges        map[NodeID][]NodeID
	ReverseEdges map[NodeID][]NodeID
	UICoords     map[NodeID]UICoord
	ExcelCoords  map[NodeID][]ExcelCoord
	IndicatorIDs map[NodeID]string
	UIIndex      map[string]NodeID
	ExcelIndex   map[string]NodeID
}

// BuildGraph 构建联动 DAG
func BuildGraph(opts BuildGraphOptions) (*Graph, error) {
	index := opts.TemplateIndex
	if index == nil {
		loaded, err := LoadTemplateIndex()
		if err != nil {
			return nil, err
		}
		index = loaded
	}

	graph := newGraph()
	attachIndicatorNodes(graph)

	if err := attachCompanyCoords(graph, index, opts.WRRecords, opts.ACRecords); err != nil {
		return nil, err
	}
	attachCompanyEdges(graph, opts.WRRecords, opts.ACRecords)
	attachAggregateEdges(graph, opts.WRRecords, opts.ACRecords)
	addIndicatorEdges(graph)
	attachIndicatorCoords(graph, index)
	attachAggregateCoords(graph, index)
	attachReverseEdges(graph, opts.WRRecords, opts.ACRecords)
	buildIndexes(graph)

	return graph, nil
}

// ComputeImpact 计算影响范围
func ComputeImpact(graph *Graph, anchor NodeID) []ImpactNode {
	if graph == nil {
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
		for _, next := range graph.Edges[node] {
			if !visited[next] {
				queue = append(queue, next)
			}
		}
		for _, next := range graph.ReverseEdges[node] {
			if !visited[next] {
				queue = append(queue, next)
			}
		}
	}

	nodes := make([]ImpactNode, 0, len(visited))
	for node := range visited {
		item := ImpactNode{
			NodeID: string(node),
		}
		if ui, ok := graph.UICoords[node]; ok {
			item.UICoord = &ui
		}
		if id, ok := graph.IndicatorIDs[node]; ok {
			item.IndicatorID = id
		}
		if coords, ok := graph.ExcelCoords[node]; ok {
			item.ExcelCoords = coords
		}
		nodes = append(nodes, item)
	}
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].NodeID < nodes[j].NodeID
	})
	return nodes
}

// ResolveAnchorNode 解析锚点节点
func ResolveAnchorNode(graph *Graph, anchor Anchor) (NodeID, error) {
	if anchor.UI != nil {
		return parseUIAnchor(anchor.UI), nil
	}
	if strings.TrimSpace(anchor.IndicatorID) != "" {
		return indicatorNode(strings.TrimSpace(anchor.IndicatorID)), nil
	}
	if anchor.Excel != nil {
		key := excelKey(anchor.Excel.Sheet, anchor.Excel.Cell)
		if node, ok := graph.ExcelIndex[key]; ok {
			return node, nil
		}
		return "", errors.New("excel anchor not found")
	}
	return "", errors.New("missing anchor")
}

func parseUIAnchor(ui *UICoord) NodeID {
	parts := strings.Split(ui.RowID, ":")
	if len(parts) != 2 {
		return NodeID(ui.RowID + ":" + ui.ColumnKey)
	}
	return NodeID(parts[0] + ":" + parts[1] + ":" + ui.ColumnKey)
}

func newGraph() *Graph {
	return &Graph{
		Edges:        map[NodeID][]NodeID{},
		ReverseEdges: map[NodeID][]NodeID{},
		UICoords:     map[NodeID]UICoord{},
		ExcelCoords:  map[NodeID][]ExcelCoord{},
		IndicatorIDs: map[NodeID]string{},
		UIIndex:      map[string]NodeID{},
		ExcelIndex:   map[string]NodeID{},
	}
}

func (g *Graph) addEdge(from NodeID, to NodeID) {
	g.Edges[from] = append(g.Edges[from], to)
}

func (g *Graph) addReverseEdge(from NodeID, to NodeID) {
	g.ReverseEdges[from] = append(g.ReverseEdges[from], to)
}

func (g *Graph) addUICoord(node NodeID, coord UICoord) {
	g.UICoords[node] = coord
}

func (g *Graph) addExcelCoord(node NodeID, coord ExcelCoord) {
	g.ExcelCoords[node] = append(g.ExcelCoords[node], coord)
}

func buildIndexes(g *Graph) {
	for node, coord := range g.UICoords {
		key := coord.RowID + "|" + coord.ColumnKey
		g.UIIndex[key] = node
	}
	for node, coords := range g.ExcelCoords {
		for _, c := range coords {
			g.ExcelIndex[excelKey(c.Sheet, c.Cell)] = node
		}
	}
}

func excelKey(sheet, cell string) string {
	return strings.TrimSpace(sheet) + "!" + strings.ToUpper(strings.TrimSpace(cell))
}
