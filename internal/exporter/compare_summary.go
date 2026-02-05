/**
 * 三方数据检测对比汇总
 *
 * @author Anner
 * Created on 2026/2/5
 */

package exporter

import "strings"

// CompareItem 单项对比结果
type CompareItem struct {
	Key     string `json:"key"`
	Label   string `json:"label"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// CompareSummary 三方对比汇总
type CompareSummary struct {
	Items []CompareItem `json:"items"`
}

// BuildCompareSummary 生成三方对比汇总
func BuildCompareSummary(rawCells int, businessRows int, missingSheets []string) CompareSummary {
	items := []CompareItem{
		buildCompareItem("raw", "原始数据", rawCells > 0, "原始单元格: "+itoa(rawCells)),
		buildCompareItem("business", "业务表", businessRows > 0, "业务记录: "+itoa(businessRows)),
		buildExportItem(missingSheets),
	}
	return CompareSummary{Items: items}
}

func buildCompareItem(key, label string, ok bool, message string) CompareItem {
	status := "pass"
	if !ok {
		status = "warn"
	}
	return CompareItem{
		Key:     key,
		Label:   label,
		Status:  status,
		Message: message,
	}
}

func buildExportItem(missingSheets []string) CompareItem {
	if len(missingSheets) == 0 {
		return CompareItem{
			Key:     "export",
			Label:   "导出结果",
			Status:  "pass",
			Message: "模板结构完整",
		}
	}
	return CompareItem{
		Key:     "export",
		Label:   "导出结果",
		Status:  "warn",
		Message: "缺少 Sheet: " + strings.Join(missingSheets, ", "),
	}
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	sign := ""
	if v < 0 {
		sign = "-"
		v = -v
	}
	buf := make([]byte, 0, 12)
	for v > 0 {
		buf = append(buf, byte('0'+v%10))
		v /= 10
	}
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return sign + string(buf)
}
