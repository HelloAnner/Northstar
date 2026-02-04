/**
 * 联动预览构图与影响范围计算
 *
 * @author Anner
 * Created on 2026/2/4
 */

package linkage

import (
	"errors"
	"strings"

	"northstar/internal/dagcalc"
	"northstar/internal/model"
)

// BuildGraphOptions 构图参数
type BuildGraphOptions struct {
	WRRecords     []*model.WholesaleRetail
	ACRecords     []*model.AccommodationCatering
	TemplateIndex *TemplateIndex
}

// Graph 联动 DAG 图
type Graph = dagcalc.Graph

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

	return dagcalc.BuildLinkageGraph(index, opts.WRRecords, opts.ACRecords)
}

// ComputeImpact 计算影响范围
func ComputeImpact(graph *Graph, anchor NodeID) []ImpactNode {
	if graph == nil {
		return nil
	}
	return dagcalc.ImpactRange(graph, anchor)
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

func indicatorNode(id string) NodeID {
	return NodeID("indicator:" + id)
}

func excelKey(sheet, cell string) string {
	return strings.TrimSpace(sheet) + "!" + strings.ToUpper(strings.TrimSpace(cell))
}
