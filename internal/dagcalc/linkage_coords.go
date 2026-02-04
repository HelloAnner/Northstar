/**
 * 联动坐标映射（统一入口）
 *
 * @author Anner
 * Created on 2026/2/4
 */

package dagcalc

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
	"northstar/internal/model"
	"northstar/internal/reporttpl"
)

// TemplateIndex 模板索引（行业码 → 行号）
type TemplateIndex struct {
	codeRows map[string]map[string][]int
	maxRows  map[string]int
}

// LoadTemplateIndex 加载模板索引
func LoadTemplateIndex() (*TemplateIndex, error) {
	return loadTemplateIndex()
}

func loadTemplateIndex() (*TemplateIndex, error) {
	f, err := reporttpl.OpenEmbeddedMonthReportTemplate()
	if err != nil {
		return nil, err
	}
	defer f.Close()

	sheets := []string{"批零总表", "批发", "零售", "住餐总表", "住宿", "餐饮"}
	index := &TemplateIndex{
		codeRows: map[string]map[string][]int{},
		maxRows:  map[string]int{},
	}
	for _, sheet := range sheets {
		rows, maxRow, err := readSheetCodeRows(f, sheet)
		if err != nil {
			return nil, err
		}
		index.codeRows[sheet] = rows
		index.maxRows[sheet] = maxRow
	}
	return index, nil
}

func readSheetCodeRows(f *excelize.File, sheet string) (map[string][]int, int, error) {
	rows := map[string][]int{}
	maxRow := 0
	for r := 2; r <= 50000; r++ {
		cell := fmt.Sprintf("C%d", r)
		raw, err := f.GetCellValue(sheet, cell)
		if err != nil {
			return nil, 0, err
		}
		raw = strings.TrimSpace(raw)
		if raw == "" {
			if r == 2 {
				return nil, 0, fmt.Errorf("%s 没有数据行", sheet)
			}
			maxRow = r - 1
			break
		}
		code := normalizeCodeText(raw)
		rows[code] = append(rows[code], r)
	}
	return rows, maxRow, nil
}

func normalizeCodeText(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	s = strings.ReplaceAll(s, ",", "")
	if v, err := strconv.ParseFloat(s, 64); err == nil {
		if math.Abs(v-math.Round(v)) <= 1e-9 {
			return strconv.FormatInt(int64(math.Round(v)), 10)
		}
	}
	return s
}

func pickFirstCode(index *TemplateIndex, sheet string) string {
	rows := index.codeRows[sheet]
	for code := range rows {
		if code != "" {
			return code
		}
	}
	return ""
}

// FirstCode 获取模板内首个行业码（用于测试）
func (t *TemplateIndex) FirstCode(sheet string) string {
	return pickFirstCode(t, sheet)
}

func buildWRRowMap(index *TemplateIndex, sheet string, records []*model.WholesaleRetail) (map[int64]int, error) {
	codeRows := index.codeRows[sheet]
	grouped := map[string][]*model.WholesaleRetail{}
	for _, r := range records {
		code := normalizeCodeText(r.IndustryCode)
		grouped[code] = append(grouped[code], r)
	}
	for code := range grouped {
		rs := grouped[code]
		sort.Slice(rs, func(i, j int) bool {
			if rs[i].RowNo != rs[j].RowNo {
				return rs[i].RowNo < rs[j].RowNo
			}
			return rs[i].ID < rs[j].ID
		})
		grouped[code] = rs
	}
	out := map[int64]int{}
	for code, rs := range grouped {
		rows := codeRows[code]
		if len(rows) < len(rs) {
			return nil, fmt.Errorf("%s 行数不足（行业=%s）", sheet, code)
		}
		for i, r := range rs {
			out[r.ID] = rows[i]
		}
	}
	return out, nil
}

func buildACRowMap(index *TemplateIndex, sheet string, records []*model.AccommodationCatering) (map[int64]int, error) {
	codeRows := index.codeRows[sheet]
	grouped := map[string][]*model.AccommodationCatering{}
	for _, r := range records {
		code := normalizeCodeText(r.IndustryCode)
		grouped[code] = append(grouped[code], r)
	}
	for code := range grouped {
		rs := grouped[code]
		sort.Slice(rs, func(i, j int) bool {
			if rs[i].RowNo != rs[j].RowNo {
				return rs[i].RowNo < rs[j].RowNo
			}
			return rs[i].ID < rs[j].ID
		})
		grouped[code] = rs
	}
	out := map[int64]int{}
	for code, rs := range grouped {
		rows := codeRows[code]
		if len(rows) < len(rs) {
			return nil, fmt.Errorf("%s 行数不足（行业=%s）", sheet, code)
		}
		for i, r := range rs {
			out[r.ID] = rows[i]
		}
	}
	return out, nil
}
