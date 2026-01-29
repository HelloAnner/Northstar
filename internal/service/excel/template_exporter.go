package excel

import (
	"errors"
	"fmt"
	"os"

	"github.com/xuri/excelize/v2"
)

var fixedTemplateSheets = []string{
	"批零总表",
	"住餐总表",
	"批发",
	"零售",
	"住宿",
	"餐饮",
	"吃穿用",
	"小微",
	"吃穿用（剔除）",
	"社零额（定）",
	"汇总表（定）",
}

// TemplateExporter 定稿模板导出器：在模板上填充数据，保留样式/公式/合并
type TemplateExporter struct {
	tmpl *excelize.File
}

// NewTemplateExporter 创建导出器
func NewTemplateExporter(tmpl *excelize.File) *TemplateExporter {
	return &TemplateExporter{tmpl: tmpl}
}

// OpenTemplate 从路径打开模板
func OpenTemplate(path string) (*excelize.File, error) {
	if path == "" {
		return nil, errors.New("template path is empty")
	}
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("template not found: %w", err)
	}
	return excelize.OpenFile(path)
}

// NewFixedTemplateWorkbook 创建一个“固定结构”的定稿模板骨架（无样式，供缺省导出使用）
func NewFixedTemplateWorkbook() *excelize.File {
	wb := excelize.NewFile()
	wb.SetSheetName("Sheet1", fixedTemplateSheets[0])
	for _, name := range fixedTemplateSheets[1:] {
		wb.NewSheet(name)
	}
	wb.SetActiveSheet(0)
	return wb
}

// SummaryValues 汇总表（定）写入值（最小集合）
type SummaryValues struct {
	G4 float64
}

// WriteSummary 向“汇总表（定）”写入固定单元格
func (e *TemplateExporter) WriteSummary(v SummaryValues) error {
	if e == nil || e.tmpl == nil {
		return errors.New("template workbook is nil")
	}
	ensureSheet(e.tmpl, "汇总表（定）")
	if err := e.tmpl.SetCellValue("汇总表（定）", "G4", v.G4); err != nil {
		return err
	}
	return nil
}

func ensureSheet(wb *excelize.File, sheetName string) {
	if _, err := wb.GetSheetIndex(sheetName); err == nil {
		return
	}
	wb.NewSheet(sheetName)
}
