/**
 * 联动预览类型定义
 *
 * @author Anner
 * Created on 2026/2/3
 */

package linkage

import (
	"strconv"

	"northstar/internal/dagcalc"
)

// NodeID DAG 节点 ID
type NodeID = dagcalc.NodeID

// UICoord UI 坐标（企业表单元格）
type UICoord = dagcalc.UICoord

// ExcelCoord Excel 坐标
type ExcelCoord = dagcalc.ExcelCoord

// ImpactNode 影响范围节点
type ImpactNode = dagcalc.ImpactNode

// AnchorPreviewRequest 预览请求
type AnchorPreviewRequest struct {
	Anchor Anchor `json:"anchor"`
}

// Anchor 预览锚点
type Anchor struct {
	UI          *UICoord    `json:"ui,omitempty"`
	IndicatorID string      `json:"indicatorId,omitempty"`
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
