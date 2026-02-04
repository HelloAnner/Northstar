package dagcalc

// Graph DAG 图
type Graph struct {
	Edges        map[NodeID][]NodeID
	ReverseEdges map[NodeID][]NodeID
	UICoords     map[NodeID]UICoord
	ExcelCoords  map[NodeID][]ExcelCoord
	IndicatorIDs map[NodeID]string
	UIIndex      map[string]NodeID
	ExcelIndex   map[string]NodeID
}

// NewGraph 创建 DAG 图
func NewGraph() *Graph {
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

// AddEdge 添加正向边
func (g *Graph) AddEdge(from NodeID, to NodeID) {
	g.Edges[from] = append(g.Edges[from], to)
}

// AddReverseEdge 添加反向边
func (g *Graph) AddReverseEdge(from NodeID, to NodeID) {
	g.ReverseEdges[from] = append(g.ReverseEdges[from], to)
}

// AddUICoord 绑定 UI 坐标
func (g *Graph) AddUICoord(node NodeID, coord UICoord) {
	g.UICoords[node] = coord
}

// AddExcelCoord 绑定 Excel 坐标
func (g *Graph) AddExcelCoord(node NodeID, coord ExcelCoord) {
	g.ExcelCoords[node] = append(g.ExcelCoords[node], coord)
}

// AddIndicatorID 绑定指标 ID
func (g *Graph) AddIndicatorID(node NodeID, id string) {
	g.IndicatorIDs[node] = id
}
