package dagcalc

import "strings"

// BuildIndexes 构建坐标索引
func BuildIndexes(g *Graph) {
	buildIndexes(g)
}

func buildIndexes(g *Graph) {
	if g == nil {
		return
	}
	if g.UIIndex == nil {
		g.UIIndex = map[string]NodeID{}
	}
	if g.ExcelIndex == nil {
		g.ExcelIndex = map[string]NodeID{}
	}
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
