/**
 * 联动预览类型定义
 *
 * @author Anner
 * Created on 2026/2/3
 */

package linkage

import "strconv"

// NodeID DAG 节点 ID
type NodeID string

// UICoord UI 坐标（企业表单元格）
type UICoord struct {
	RowID     string `json:"rowId"`
	ColumnKey string `json:"columnKey"`
}

// ExcelCoord Excel 坐标
type ExcelCoord struct {
	Sheet string `json:"sheet"`
	Cell  string `json:"cell"`
}

// ImpactNode 影响范围节点
type ImpactNode struct {
	NodeID      string       `json:"nodeId"`
	UICoord     *UICoord     `json:"ui,omitempty"`
	IndicatorID string       `json:"indicatorId,omitempty"`
	ExcelCoords []ExcelCoord `json:"excel,omitempty"`
	Reason      string       `json:"reason,omitempty"`
}

// AnchorPreviewRequest 预览请求
type AnchorPreviewRequest struct {
	Anchor Anchor `json:"anchor"`
}

// Anchor 预览锚点
type Anchor struct {
	UI          *UICoord   `json:"ui,omitempty"`
	IndicatorID string     `json:"indicatorId,omitempty"`
	Excel       *ExcelCoord `json:"excel,omitempty"`
}

// BuildRowID 构建行 ID
func BuildRowID(kind string, id int64) string {
	return kind + ":" + formatInt(id)
}

// BuildNodeID 构建节点 ID
func BuildNodeID(kind string, id int64, field string) NodeID {
	return NodeID(kind + ":" + formatInt(id) + ":" + field)
}

func formatInt(v int64) string {
	return strconv.FormatInt(v, 10)
}
