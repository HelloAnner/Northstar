package importer

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/xuri/excelize/v2"
	"northstar/internal/parser"
	"northstar/internal/store"
)

// Coordinator 导入协调器
type Coordinator struct {
	store      *store.Store
	recognizer *parser.SheetRecognizer
}

// NewCoordinator 创建导入协调器
func NewCoordinator(store *store.Store) *Coordinator {
	return &Coordinator{
		store:      store,
		recognizer: parser.NewSheetRecognizer(),
	}
}

// ImportOptions 导入选项
type ImportOptions struct {
	FilePath        string
	ClearExisting   bool // 是否清空现有数据
	UpdateConfigYM  bool // 是否更新配置中的当前年月
	CalculateFields bool // 是否计算衍生字段
}

// ProgressEvent 进度事件
type ProgressEvent struct {
	Type      string      `json:"type"`      // start/sheet_start/sheet_done/done/error
	Message   string      `json:"message"`   // 事件消息
	Data      interface{} `json:"data"`      // 附加数据
	Timestamp time.Time   `json:"timestamp"` // 时间戳
}

// ImportContext 导入上下文
type ImportContext struct {
	FilePath       string
	File           *excelize.File
	StartTime      time.Time
	Report         *parser.ImportReport
	ProgressChan   chan ProgressEvent
	CurrentYear    int // 从主表识别出的当前年份
	CurrentMonth   int // 从主表识别出的当前月份
}

// Import 执行导入，返回进度通道
func (c *Coordinator) Import(opts ImportOptions) <-chan ProgressEvent {
	progressChan := make(chan ProgressEvent, 100)

	go func() {
		defer close(progressChan)
		c.doImport(opts, progressChan)
	}()

	return progressChan
}

// doImport 执行导入逻辑
func (c *Coordinator) doImport(opts ImportOptions, progressChan chan ProgressEvent) {
	startTime := time.Now()

	// 发送开始事件
	c.sendProgress(progressChan, ProgressEvent{
		Type:    "start",
		Message: "开始导入 Excel 文件",
		Data: map[string]string{
			"filename": filepath.Base(opts.FilePath),
		},
		Timestamp: time.Now(),
	})

	// 打开 Excel 文件
	file, err := excelize.OpenFile(opts.FilePath)
	if err != nil {
		c.sendProgress(progressChan, ProgressEvent{
			Type:      "error",
			Message:   fmt.Sprintf("打开文件失败: %v", err),
			Timestamp: time.Now(),
		})
		return
	}
	defer file.Close()

	// 创建导入上下文
	ctx := &ImportContext{
		FilePath:     opts.FilePath,
		File:         file,
		StartTime:    startTime,
		ProgressChan: progressChan,
		Report: &parser.ImportReport{
			Filename: filepath.Base(opts.FilePath),
			Sheets:   []parser.ParseResult{},
		},
	}

	// 获取所有 Sheet
	sheetList := file.GetSheetList()
	ctx.Report.TotalSheets = len(sheetList)

	c.sendProgress(progressChan, ProgressEvent{
		Type:    "info",
		Message: fmt.Sprintf("发现 %d 个 Sheet", len(sheetList)),
		Data: map[string]interface{}{
			"total_sheets": len(sheetList),
		},
		Timestamp: time.Now(),
	})

	// 遍历所有 Sheet
	for _, sheetName := range sheetList {
		c.processSheet(ctx, sheetName, opts)
	}

	// 计算衍生字段
	if opts.CalculateFields && ctx.CurrentYear > 0 && ctx.CurrentMonth > 0 {
		c.calculateDerivedFields(ctx)
	}

	// 更新配置中的当前年月
	if opts.UpdateConfigYM && ctx.CurrentYear > 0 && ctx.CurrentMonth > 0 {
		c.updateCurrentYearMonth(ctx)
	}

	// 汇总统计
	ctx.Report.Duration = time.Since(startTime)

	// 发送完成事件
	c.sendProgress(progressChan, ProgressEvent{
		Type:    "done",
		Message: "导入完成",
		Data:    ctx.Report,
		Timestamp: time.Now(),
	})
}

// processSheet 处理单个 Sheet
func (c *Coordinator) processSheet(ctx *ImportContext, sheetName string, opts ImportOptions) {
	sheetStartTime := time.Now()

	c.sendProgress(ctx.ProgressChan, ProgressEvent{
		Type:    "sheet_start",
		Message: fmt.Sprintf("正在解析 Sheet: %s", sheetName),
		Data: map[string]string{
			"sheet_name": sheetName,
		},
		Timestamp: time.Now(),
	})

	// 读取表头
	rows, err := ctx.File.GetRows(sheetName)
	if err != nil || len(rows) < 1 {
		c.recordSheetResult(ctx, parser.ParseResult{
			SheetName: sheetName,
			SheetType: parser.SheetTypeUnknown,
			Status:    "error",
			Errors:    []string{fmt.Sprintf("读取 Sheet 失败: %v", err)},
			Duration:  time.Since(sheetStartTime),
		})
		return
	}

	headers := rows[0]

	// 识别 Sheet 类型
	recognition := c.recognizer.Recognize(sheetName, headers)

	c.sendProgress(ctx.ProgressChan, ProgressEvent{
		Type:    "info",
		Message: fmt.Sprintf("Sheet \"%s\" 识别为: %s (置信度: %.2f)", sheetName, recognition.SheetType, recognition.Confidence),
		Data: map[string]interface{}{
			"sheet_name": sheetName,
			"sheet_type": recognition.SheetType,
			"confidence": recognition.Confidence,
		},
		Timestamp: time.Now(),
	})

	// 根据类型处理
	switch recognition.SheetType {
	case parser.SheetTypeWholesale, parser.SheetTypeRetail:
		c.processWholesaleRetail(ctx, sheetName, opts)
	case parser.SheetTypeAccommodation, parser.SheetTypeCatering:
		c.processAccommodationCatering(ctx, sheetName, opts)
	case parser.SheetTypeSummary:
		c.recordSheetResult(ctx, parser.ParseResult{
			SheetName: sheetName,
			SheetType: parser.SheetTypeSummary,
			Status:    "skipped",
			Duration:  time.Since(sheetStartTime),
		})
		c.sendProgress(ctx.ProgressChan, ProgressEvent{
			Type:    "info",
			Message: fmt.Sprintf("跳过汇总表: %s", sheetName),
			Timestamp: time.Now(),
		})
	case parser.SheetTypeUnknown:
		c.recordSheetResult(ctx, parser.ParseResult{
			SheetName: sheetName,
			SheetType: parser.SheetTypeUnknown,
			Status:    "skipped",
			Errors:    []string{"无法识别 Sheet 类型"},
			Duration:  time.Since(sheetStartTime),
		})
		c.sendProgress(ctx.ProgressChan, ProgressEvent{
			Type:    "warning",
			Message: fmt.Sprintf("无法识别 Sheet: %s (置信度过低)", sheetName),
			Timestamp: time.Now(),
		})
	default:
		c.recordSheetResult(ctx, parser.ParseResult{
			SheetName: sheetName,
			SheetType: recognition.SheetType,
			Status:    "skipped",
			Errors:    []string{"暂不支持此类型"},
			Duration:  time.Since(sheetStartTime),
		})
	}
}

// processWholesaleRetail 处理批零主表
func (c *Coordinator) processWholesaleRetail(ctx *ImportContext, sheetName string, opts ImportOptions) {
	sheetStartTime := time.Now()

	// 解析 Sheet
	wrParser := parser.NewWRParser(ctx.File)
	records, err := wrParser.ParseSheet(sheetName)
	if err != nil {
		c.recordSheetResult(ctx, parser.ParseResult{
			SheetName: sheetName,
			SheetType: parser.SheetTypeWholesale,
			Status:    "error",
			Errors:    []string{err.Error()},
			Duration:  time.Since(sheetStartTime),
		})
		return
	}

	// 识别当前年月（从第一条记录）
	if len(records) > 0 {
		if ctx.CurrentYear == 0 || ctx.CurrentMonth == 0 {
			ctx.CurrentYear = records[0].DataYear
			ctx.CurrentMonth = records[0].DataMonth
			c.sendProgress(ctx.ProgressChan, ProgressEvent{
				Type:    "info",
				Message: fmt.Sprintf("识别数据月份: %d年%d月", ctx.CurrentYear, ctx.CurrentMonth),
				Data: map[string]int{
					"year":  ctx.CurrentYear,
					"month": ctx.CurrentMonth,
				},
				Timestamp: time.Now(),
			})
		}
	}

	// 清空现有数据（可选）
	if opts.ClearExisting && len(records) > 0 {
		year := records[0].DataYear
		month := records[0].DataMonth
		if err := c.store.DeleteWRByYearMonth(year, month); err != nil {
			c.sendProgress(ctx.ProgressChan, ProgressEvent{
				Type:    "warning",
				Message: fmt.Sprintf("清空旧数据失败: %v", err),
				Timestamp: time.Now(),
			})
		}
	}

	// 批量插入
	if err := c.store.BatchInsertWR(records); err != nil {
		c.recordSheetResult(ctx, parser.ParseResult{
			SheetName:    sheetName,
			SheetType:    parser.SheetTypeWholesale,
			Status:       "error",
			ImportedRows: 0,
			ErrorRows:    len(records),
			Errors:       []string{fmt.Sprintf("批量插入失败: %v", err)},
			Duration:     time.Since(sheetStartTime),
		})
		return
	}

	// 记录成功
	c.recordSheetResult(ctx, parser.ParseResult{
		SheetName:    sheetName,
		SheetType:    parser.SheetTypeWholesale,
		Status:       "imported",
		ImportedRows: len(records),
		Duration:     time.Since(sheetStartTime),
	})

	c.sendProgress(ctx.ProgressChan, ProgressEvent{
		Type:    "sheet_done",
		Message: fmt.Sprintf("Sheet \"%s\" 导入成功: %d 行", sheetName, len(records)),
		Data: map[string]interface{}{
			"sheet_name":    sheetName,
			"imported_rows": len(records),
		},
		Timestamp: time.Now(),
	})
}

// processAccommodationCatering 处理住餐主表
func (c *Coordinator) processAccommodationCatering(ctx *ImportContext, sheetName string, opts ImportOptions) {
	sheetStartTime := time.Now()

	// 解析 Sheet
	acParser := parser.NewACParser(ctx.File)
	records, err := acParser.ParseSheet(sheetName)
	if err != nil {
		c.recordSheetResult(ctx, parser.ParseResult{
			SheetName: sheetName,
			SheetType: parser.SheetTypeAccommodation,
			Status:    "error",
			Errors:    []string{err.Error()},
			Duration:  time.Since(sheetStartTime),
		})
		return
	}

	// 识别当前年月
	if len(records) > 0 {
		if ctx.CurrentYear == 0 || ctx.CurrentMonth == 0 {
			ctx.CurrentYear = records[0].DataYear
			ctx.CurrentMonth = records[0].DataMonth
			c.sendProgress(ctx.ProgressChan, ProgressEvent{
				Type:    "info",
				Message: fmt.Sprintf("识别数据月份: %d年%d月", ctx.CurrentYear, ctx.CurrentMonth),
				Data: map[string]int{
					"year":  ctx.CurrentYear,
					"month": ctx.CurrentMonth,
				},
				Timestamp: time.Now(),
			})
		}
	}

	// 清空现有数据（可选）
	if opts.ClearExisting && len(records) > 0 {
		year := records[0].DataYear
		month := records[0].DataMonth
		if err := c.store.DeleteACByYearMonth(year, month); err != nil {
			c.sendProgress(ctx.ProgressChan, ProgressEvent{
				Type:    "warning",
				Message: fmt.Sprintf("清空旧数据失败: %v", err),
				Timestamp: time.Now(),
			})
		}
	}

	// 批量插入
	if err := c.store.BatchInsertAC(records); err != nil {
		c.recordSheetResult(ctx, parser.ParseResult{
			SheetName:    sheetName,
			SheetType:    parser.SheetTypeAccommodation,
			Status:       "error",
			ImportedRows: 0,
			ErrorRows:    len(records),
			Errors:       []string{fmt.Sprintf("批量插入失败: %v", err)},
			Duration:     time.Since(sheetStartTime),
		})
		return
	}

	// 记录成功
	c.recordSheetResult(ctx, parser.ParseResult{
		SheetName:    sheetName,
		SheetType:    parser.SheetTypeAccommodation,
		Status:       "imported",
		ImportedRows: len(records),
		Duration:     time.Since(sheetStartTime),
	})

	c.sendProgress(ctx.ProgressChan, ProgressEvent{
		Type:    "sheet_done",
		Message: fmt.Sprintf("Sheet \"%s\" 导入成功: %d 行", sheetName, len(records)),
		Data: map[string]interface{}{
			"sheet_name":    sheetName,
			"imported_rows": len(records),
		},
		Timestamp: time.Now(),
	})
}

// calculateDerivedFields 计算衍生字段
func (c *Coordinator) calculateDerivedFields(ctx *ImportContext) {
	c.sendProgress(ctx.ProgressChan, ProgressEvent{
		Type:    "info",
		Message: "正在计算衍生字段...",
		Timestamp: time.Now(),
	})

	// 批零增速计算
	err := c.store.Exec(`
		UPDATE wholesale_retail SET
			sales_month_rate = CASE
				WHEN sales_last_year_month = 0 THEN NULL
				ELSE (sales_current_month - sales_last_year_month) / sales_last_year_month * 100
			END,
			sales_cumulative_rate = CASE
				WHEN sales_last_year_cumulative = 0 THEN NULL
				ELSE (sales_current_cumulative - sales_last_year_cumulative) / sales_last_year_cumulative * 100
			END,
			retail_month_rate = CASE
				WHEN retail_last_year_month = 0 THEN NULL
				ELSE (retail_current_month - retail_last_year_month) / retail_last_year_month * 100
			END,
			retail_cumulative_rate = CASE
				WHEN retail_last_year_cumulative = 0 THEN NULL
				ELSE (retail_current_cumulative - retail_last_year_cumulative) / retail_last_year_cumulative * 100
			END,
			retail_ratio = CASE
				WHEN sales_current_month = 0 THEN NULL
				ELSE retail_current_month / sales_current_month * 100
			END
		WHERE data_year = ? AND data_month = ?
	`, ctx.CurrentYear, ctx.CurrentMonth)

	if err != nil {
		c.sendProgress(ctx.ProgressChan, ProgressEvent{
			Type:    "warning",
			Message: fmt.Sprintf("批零增速计算失败: %v", err),
			Timestamp: time.Now(),
		})
	}

	// 住餐增速计算
	err = c.store.Exec(`
		UPDATE accommodation_catering SET
			revenue_month_rate = CASE
				WHEN revenue_last_year_month = 0 THEN NULL
				ELSE (revenue_current_month - revenue_last_year_month) / revenue_last_year_month * 100
			END,
			revenue_cumulative_rate = CASE
				WHEN revenue_last_year_cumulative = 0 THEN NULL
				ELSE (revenue_current_cumulative - revenue_last_year_cumulative) / revenue_last_year_cumulative * 100
			END
		WHERE data_year = ? AND data_month = ?
	`, ctx.CurrentYear, ctx.CurrentMonth)

	if err != nil {
		c.sendProgress(ctx.ProgressChan, ProgressEvent{
			Type:    "warning",
			Message: fmt.Sprintf("住餐增速计算失败: %v", err),
			Timestamp: time.Now(),
		})
	}

	c.sendProgress(ctx.ProgressChan, ProgressEvent{
		Type:    "info",
		Message: "衍生字段计算完成",
		Timestamp: time.Now(),
	})
}

// updateCurrentYearMonth 更新配置中的当前年月
func (c *Coordinator) updateCurrentYearMonth(ctx *ImportContext) {
	if err := c.store.SetCurrentYearMonth(ctx.CurrentYear, ctx.CurrentMonth); err != nil {
		c.sendProgress(ctx.ProgressChan, ProgressEvent{
			Type:    "warning",
			Message: fmt.Sprintf("更新当前年月失败: %v", err),
			Timestamp: time.Now(),
		})
	} else {
		c.sendProgress(ctx.ProgressChan, ProgressEvent{
			Type:    "info",
			Message: fmt.Sprintf("当前操作月份已更新为: %d年%d月", ctx.CurrentYear, ctx.CurrentMonth),
			Timestamp: time.Now(),
		})
	}
}

// recordSheetResult 记录 Sheet 处理结果
func (c *Coordinator) recordSheetResult(ctx *ImportContext, result parser.ParseResult) {
	ctx.Report.Sheets = append(ctx.Report.Sheets, result)

	if result.Status == "imported" {
		ctx.Report.ImportedSheets++
		ctx.Report.ImportedRows += result.ImportedRows
	} else if result.Status == "skipped" {
		ctx.Report.SkippedSheets++
	}

	if result.ErrorRows > 0 {
		ctx.Report.ErrorRows += result.ErrorRows
	}

	ctx.Report.TotalRows += result.ImportedRows + result.ErrorRows
}

// sendProgress 发送进度事件
func (c *Coordinator) sendProgress(ch chan ProgressEvent, event ProgressEvent) {
	select {
	case ch <- event:
	default:
		// 通道已满，丢弃事件
	}
}
