/**
 * 导出三方对比构建
 *
 * @author Anner
 * Created on 2026/2/5
 */

package v3

import (
	"fmt"

	"github.com/xuri/excelize/v2"
	"northstar/internal/exporter"
	"northstar/internal/store"
)

func buildExportCompareSummary(st *store.Store, f *excelize.File, year, month int) (exporter.CompareSummary, error) {
	if st == nil || f == nil {
		return exporter.CompareSummary{}, fmt.Errorf("invalid compare input")
	}

	rawCells, rawErr := countRawCells(st)
	businessRows, businessErr := countBusinessRows(st, year, month)
	missingSheets := findMissingSheets(f, expectedExportSheets())

	summary := exporter.BuildCompareSummary(rawCells, businessRows, missingSheets)
	if rawErr != nil {
		updateCompareItem(&summary, "raw", "warn", "读取原始数据失败")
	}
	if businessErr != nil {
		updateCompareItem(&summary, "business", "warn", "读取业务数据失败")
	}
	return summary, nil
}

func expectedExportSheets() []string {
	return []string{
		"批发",
		"零售",
		"批零总表",
		"住宿",
		"餐饮",
		"住餐总表",
		"小微",
		"吃穿用",
		"吃穿用（剔除）",
		"社零额（定）",
		"汇总表（定）",
	}
}

func countRawCells(st *store.Store) (int, error) {
	log, err := st.GetLatestImportLog()
	if err != nil {
		return 0, err
	}
	if log == nil {
		return 0, nil
	}
	return st.CountSheetCellsByImportLog(log.ID)
}

func countBusinessRows(st *store.Store, year, month int) (int, error) {
	wrCount, err := st.CountWR(store.WRQueryOptions{
		DataYear:  &year,
		DataMonth: &month,
	})
	if err != nil {
		return 0, err
	}
	acCount, err := st.CountAC(store.ACQueryOptions{
		DataYear:  &year,
		DataMonth: &month,
	})
	if err != nil {
		return 0, err
	}
	return wrCount + acCount, nil
}

func findMissingSheets(f *excelize.File, expected []string) []string {
	exist := map[string]bool{}
	for _, name := range f.GetSheetList() {
		exist[name] = true
	}
	missing := make([]string, 0)
	for _, name := range expected {
		if !exist[name] {
			missing = append(missing, name)
		}
	}
	return missing
}

func updateCompareItem(summary *exporter.CompareSummary, key, status, message string) {
	for i := range summary.Items {
		if summary.Items[i].Key == key {
			summary.Items[i].Status = status
			summary.Items[i].Message = message
			return
		}
	}
}
