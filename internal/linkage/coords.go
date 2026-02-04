/**
 * 联动坐标映射（兼容入口）
 *
 * @author Anner
 * Created on 2026/2/4
 */

package linkage

import "northstar/internal/dagcalc"

// TemplateIndex 模板索引（行业码 → 行号）
type TemplateIndex = dagcalc.TemplateIndex

// LoadTemplateIndex 加载模板索引
func LoadTemplateIndex() (*TemplateIndex, error) {
	return dagcalc.LoadTemplateIndex()
}

func loadTemplateIndex() (*TemplateIndex, error) {
	return dagcalc.LoadTemplateIndex()
}

func pickFirstCode(index *TemplateIndex, sheet string) string {
	if index == nil {
		return ""
	}
	return index.FirstCode(sheet)
}
