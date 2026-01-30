package exporter

import (
	"fmt"
	"strconv"

	"github.com/xuri/excelize/v2"
	"northstar/internal/store"
)

// Exporter 导出器
type Exporter struct {
	store *store.Store
}

// NewExporter 创建导出器
func NewExporter(store *store.Store) *Exporter {
	return &Exporter{
		store: store,
	}
}

// ExportOptions 导出选项
type ExportOptions struct {
	Year  int
	Month int
}

// Export 导出 Excel
func (e *Exporter) Export(opts ExportOptions) (*excelize.File, error) {
	f := excelize.NewFile()

	// 删除默认的 Sheet1
	f.DeleteSheet("Sheet1")

	// 1. 批零总表
	if err := e.exportWholesaleRetailSheet(f, opts); err != nil {
		return nil, fmt.Errorf("导出批零总表失败: %w", err)
	}

	// 2. 住餐总表
	if err := e.exportAccommodationCateringSheet(f, opts); err != nil {
		return nil, fmt.Errorf("导出住餐总表失败: %w", err)
	}

	// 3. 吃穿用
	if err := e.exportEatWearUseSheet(f, opts); err != nil {
		return nil, fmt.Errorf("导出吃穿用失败: %w", err)
	}

	// 4. 小微
	if err := e.exportSmallMicroSheet(f, opts); err != nil {
		return nil, fmt.Errorf("导出小微失败: %w", err)
	}

	// 设置第一个 Sheet 为活动 Sheet
	f.SetActiveSheet(0)

	return f, nil
}

// exportWholesaleRetailSheet 导出批零总表
func (e *Exporter) exportWholesaleRetailSheet(f *excelize.File, opts ExportOptions) error {
	sheetName := "批零总表"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return err
	}

	// 获取数据
	wrOpts := store.WRQueryOptions{
		DataYear:  &opts.Year,
		DataMonth: &opts.Month,
	}
	records, err := e.store.GetWRByYearMonth(wrOpts)
	if err != nil {
		return err
	}

	// 定义表头
	headers := []string{
		"序号",
		"统一社会信用代码",
		"单位详细名称",
		"行业代码",
		"单位规模",
		fmt.Sprintf("%d年%d月销售额", opts.Year, opts.Month),
		fmt.Sprintf("%d年%d月零售额", opts.Year, opts.Month),
		fmt.Sprintf("%d年1-%d月销售额", opts.Year, opts.Month),
		fmt.Sprintf("%d年1-%d月零售额", opts.Year, opts.Month),
		"粮油食品类",
		"饮料类",
		"烟酒类",
		"服装鞋帽针纺织品类",
		"日用品类",
		"汽车类",
		"网络销售额",
		"第一次上报的IP",
		"填报IP",
		"开业年份",
		"开业月份",
	}

	// 写入表头
	for colIdx, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(colIdx+1, 1)
		f.SetCellValue(sheetName, cell, header)
	}

	// 写入数据
	for rowIdx, record := range records {
		row := rowIdx + 2 // 从第2行开始（第1行是表头）

		data := []interface{}{
			rowIdx + 1,
			record.CreditCode,
			record.Name,
			record.IndustryCode,
			record.CompanyScale,
			record.SalesCurrentMonth,
			record.RetailCurrentMonth,
			record.SalesCurrentCumulative,
			record.RetailCurrentCumulative,
			record.CatGrainOilFood,
			record.CatBeverage,
			record.CatTobaccoLiquor,
			record.CatClothing,
			record.CatDailyUse,
			record.CatAutomobile,
			record.NetworkSales,
			record.FirstReportIP,
			record.FillIP,
			nilIntToStr(record.OpeningYear),
			nilIntToStr(record.OpeningMonth),
		}

		for colIdx, value := range data {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, row)
			f.SetCellValue(sheetName, cell, value)
		}
	}

	// 设置列宽
	f.SetColWidth(sheetName, "A", "A", 8)  // 序号
	f.SetColWidth(sheetName, "B", "B", 20) // 信用代码
	f.SetColWidth(sheetName, "C", "C", 30) // 名称
	f.SetColWidth(sheetName, "D", "T", 12) // 其他列

	// 设置表头样式
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold: true,
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Pattern: 1,
			Color:   []string{"#E0E0E0"},
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
	})

	f.SetCellStyle(sheetName, "A1", getLastColumn(len(headers))+"1", headerStyle)

	f.SetActiveSheet(index)

	return nil
}

// exportAccommodationCateringSheet 导出住餐总表
func (e *Exporter) exportAccommodationCateringSheet(f *excelize.File, opts ExportOptions) error {
	sheetName := "住餐总表"
	_, err := f.NewSheet(sheetName)
	if err != nil {
		return err
	}

	// 获取数据
	acOpts := store.ACQueryOptions{
		DataYear:  &opts.Year,
		DataMonth: &opts.Month,
	}
	records, err := e.store.GetACByYearMonth(acOpts)
	if err != nil {
		return err
	}

	// 定义表头
	headers := []string{
		"序号",
		"统一社会信用代码",
		"单位详细名称",
		"行业代码",
		"单位规模",
		fmt.Sprintf("%d年%d月营业额", opts.Year, opts.Month),
		fmt.Sprintf("%d年1-%d月营业额", opts.Year, opts.Month),
		fmt.Sprintf("%d年%d月客房收入", opts.Year, opts.Month),
		fmt.Sprintf("%d年1-%d月客房收入", opts.Year, opts.Month),
		fmt.Sprintf("%d年%d月餐费收入", opts.Year, opts.Month),
		fmt.Sprintf("%d年1-%d月餐费收入", opts.Year, opts.Month),
		fmt.Sprintf("%d年%d月商品销售额", opts.Year, opts.Month),
		fmt.Sprintf("%d年1-%d月商品销售额", opts.Year, opts.Month),
		fmt.Sprintf("%d年%d月零售额", opts.Year, opts.Month),
		"网络销售额",
		"第一次上报的IP",
		"填报IP",
		"开业年份",
		"开业月份",
	}

	// 写入表头
	for colIdx, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(colIdx+1, 1)
		f.SetCellValue(sheetName, cell, header)
	}

	// 写入数据
	for rowIdx, record := range records {
		row := rowIdx + 2

		data := []interface{}{
			rowIdx + 1,
			record.CreditCode,
			record.Name,
			record.IndustryCode,
			record.CompanyScale,
			record.RevenueCurrentMonth,
			record.RevenueCurrentCumulative,
			record.RoomCurrentMonth,
			record.RoomCurrentCumulative,
			record.FoodCurrentMonth,
			record.FoodCurrentCumulative,
			record.GoodsCurrentMonth,
			record.GoodsCurrentCumulative,
			record.RetailCurrentMonth,
			record.NetworkSales,
			record.FirstReportIP,
			record.FillIP,
			nilIntToStr(record.OpeningYear),
			nilIntToStr(record.OpeningMonth),
		}

		for colIdx, value := range data {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, row)
			f.SetCellValue(sheetName, cell, value)
		}
	}

	// 设置列宽
	f.SetColWidth(sheetName, "A", "A", 8)
	f.SetColWidth(sheetName, "B", "B", 20)
	f.SetColWidth(sheetName, "C", "C", 30)
	f.SetColWidth(sheetName, "D", "S", 12)

	// 设置表头样式
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{
			Type:    "pattern",
			Pattern: 1,
			Color:   []string{"#E0E0E0"},
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
	})

	f.SetCellStyle(sheetName, "A1", getLastColumn(len(headers))+"1", headerStyle)

	return nil
}

// exportEatWearUseSheet 导出吃穿用
func (e *Exporter) exportEatWearUseSheet(f *excelize.File, opts ExportOptions) error {
	sheetName := "吃穿用"
	_, err := f.NewSheet(sheetName)
	if err != nil {
		return err
	}

	// 获取吃穿用企业
	isEatWearUse := 1
	wrOpts := store.WRQueryOptions{
		DataYear:     &opts.Year,
		DataMonth:    &opts.Month,
		IsEatWearUse: &isEatWearUse,
	}
	records, err := e.store.GetWRByYearMonth(wrOpts)
	if err != nil {
		return err
	}

	// 定义表头（吃穿用包含更多字段）
	headers := []string{
		"序号",
		"统一社会信用代码",
		"单位详细名称",
		"行业代码",
		"单位规模",
		fmt.Sprintf("%d年%d月销售额", opts.Year, opts.Month),
		fmt.Sprintf("%d年%d月零售额", opts.Year, opts.Month),
		fmt.Sprintf("%d年1-%d月销售额", opts.Year, opts.Month),
		fmt.Sprintf("%d年1-%d月零售额", opts.Year, opts.Month),
		"粮油食品类",
		"饮料类",
		"烟酒类",
		"服装鞋帽针纺织品类",
		"日用品类",
		"汽车类",
		"网络销售额",
		"开业年份",
		"开业月份",
	}

	// 写入表头
	for colIdx, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(colIdx+1, 1)
		f.SetCellValue(sheetName, cell, header)
	}

	// 写入数据
	for rowIdx, record := range records {
		row := rowIdx + 2

		data := []interface{}{
			rowIdx + 1,
			record.CreditCode,
			record.Name,
			record.IndustryCode,
			record.CompanyScale,
			record.SalesCurrentMonth,
			record.RetailCurrentMonth,
			record.SalesCurrentCumulative,
			record.RetailCurrentCumulative,
			record.CatGrainOilFood,
			record.CatBeverage,
			record.CatTobaccoLiquor,
			record.CatClothing,
			record.CatDailyUse,
			record.CatAutomobile,
			record.NetworkSales,
			nilIntToStr(record.OpeningYear),
			nilIntToStr(record.OpeningMonth),
		}

		for colIdx, value := range data {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, row)
			f.SetCellValue(sheetName, cell, value)
		}
	}

	// 设置样式
	f.SetColWidth(sheetName, "A", "A", 8)
	f.SetColWidth(sheetName, "B", "B", 20)
	f.SetColWidth(sheetName, "C", "C", 30)
	f.SetColWidth(sheetName, "D", "R", 12)

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{
			Type:    "pattern",
			Pattern: 1,
			Color:   []string{"#E0E0E0"},
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
	})

	f.SetCellStyle(sheetName, "A1", getLastColumn(len(headers))+"1", headerStyle)

	return nil
}

// exportSmallMicroSheet 导出小微
func (e *Exporter) exportSmallMicroSheet(f *excelize.File, opts ExportOptions) error {
	sheetName := "小微"
	_, err := f.NewSheet(sheetName)
	if err != nil {
		return err
	}

	// 获取小微企业
	isSmallMicro := 1
	wrOpts := store.WRQueryOptions{
		DataYear:     &opts.Year,
		DataMonth:    &opts.Month,
		IsSmallMicro: &isSmallMicro,
	}
	records, err := e.store.GetWRByYearMonth(wrOpts)
	if err != nil {
		return err
	}

	// 定义表头（小微只需要基础字段）
	headers := []string{
		"序号",
		"统一社会信用代码",
		"单位详细名称",
		"行业代码",
		"单位规模",
		fmt.Sprintf("%d年%d月零售额", opts.Year, opts.Month),
	}

	// 写入表头
	for colIdx, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(colIdx+1, 1)
		f.SetCellValue(sheetName, cell, header)
	}

	// 写入数据
	for rowIdx, record := range records {
		row := rowIdx + 2

		data := []interface{}{
			rowIdx + 1,
			record.CreditCode,
			record.Name,
			record.IndustryCode,
			record.CompanyScale,
			record.RetailCurrentMonth,
		}

		for colIdx, value := range data {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, row)
			f.SetCellValue(sheetName, cell, value)
		}
	}

	// 设置样式
	f.SetColWidth(sheetName, "A", "A", 8)
	f.SetColWidth(sheetName, "B", "B", 20)
	f.SetColWidth(sheetName, "C", "C", 30)
	f.SetColWidth(sheetName, "D", "F", 12)

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{
			Type:    "pattern",
			Pattern: 1,
			Color:   []string{"#E0E0E0"},
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
	})

	f.SetCellStyle(sheetName, "A1", getLastColumn(len(headers))+"1", headerStyle)

	return nil
}

// 辅助函数

// getLastColumn 根据列数获取最后一列的字母表示
func getLastColumn(colCount int) string {
	col, _ := excelize.ColumnNumberToName(colCount)
	return col
}

// nilIntToStr 将可能为 nil 的整数指针转为字符串
func nilIntToStr(val *int) string {
	if val == nil {
		return ""
	}
	return strconv.Itoa(*val)
}
