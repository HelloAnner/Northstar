# 后端 API 设计

## 设计目标

1. **简洁高效**：API 数量精简，只提供必要的接口
2. **实时计算**：数据修改后立即触发 DAG 计算，返回最新指标
3. **月份灵活**：自动识别和处理任意月份的数据

---

## 技术栈

- **语言**: Go 1.21+
- **框架**: Gin (轻量 HTTP 框架)
- **数据库**: SQLite + database/sql
- **Excel**: excelize (Excel 解析库)

---

## API 列表

### 1. 系统状态

#### GET /api/status

获取系统状态和当前数据月份

**响应**

```json
{
  "has_data": true,
  "current_year": 2026,
  "current_month": 1,
  "company_count": {
    "wholesale": 58,
    "retail": 142,
    "accommodation": 12,
    "catering": 28
  },
  "last_import": "2026-01-15T10:30:00Z"
}
```

---

### 2. 数据导入

#### POST /api/import

上传并解析 Excel 文件

**请求**

```
Content-Type: multipart/form-data

file: <Excel 文件>
```

**响应 (SSE 流式)**

```
event: progress
data: {"percent": 10, "message": "正在读取 Sheet 列表..."}

event: progress
data: {"percent": 30, "message": "解析批发 Sheet，58 行"}

event: progress
data: {"percent": 60, "message": "解析零售 Sheet，142 行"}

event: progress
data: {"percent": 90, "message": "计算衍生指标..."}

event: complete
data: {"success": true, "report": {...}}
```

**完成报告结构**

```json
{
  "filename": "2026年1月月报.xlsx",
  "data_year": 2026,
  "data_month": 1,
  "total_sheets": 15,
  "imported_sheets": 6,
  "skipped_sheets": 9,
  "total_rows": 240,
  "imported_rows": 240,
  "duration_ms": 2300,
  "sheets": [
    {
      "name": "批发",
      "type": "wholesale",
      "confidence": 0.95,
      "status": "imported",
      "rows": 58
    },
    {
      "name": "零售",
      "type": "retail",
      "confidence": 0.98,
      "status": "imported",
      "rows": 142
    },
    {
      "name": "2024年12月批零",
      "type": "wr_snapshot",
      "confidence": 0.92,
      "status": "imported",
      "rows": 200,
      "snapshot_year": 2024,
      "snapshot_month": 12
    }
  ]
}
```

---

### 3. 指标查询

#### GET /api/indicators

获取当前月份的所有指标

**响应**

```json
{
  "data_year": 2026,
  "data_month": 1,
  "indicators": {
    // 限上社零额
    "limit_above_retail_month": 123456.78,
    "limit_above_retail_month_rate": 5.2,
    "limit_above_retail_cumulative": 123456.78,
    "limit_above_retail_cumulative_rate": 5.2,

    // 四大行业增速
    "wholesale_sales_month_rate": 3.8,
    "wholesale_sales_cumulative_rate": 4.1,
    "retail_sales_month_rate": 6.5,
    "retail_sales_cumulative_rate": 5.9,
    "accommodation_revenue_month_rate": 2.3,
    "accommodation_revenue_cumulative_rate": 3.1,
    "catering_revenue_month_rate": 8.1,
    "catering_revenue_cumulative_rate": 7.5,

    // 专项增速
    "small_micro_month_rate": 4.5,
    "eat_wear_use_month_rate": 5.8,

    // 社零总额
    "total_retail_cumulative": 234567.89,
    "total_retail_cumulative_rate": 5.5
  }
}
```

---

### 4. 企业数据查询

#### GET /api/companies

获取企业列表（批零或住餐）

**请求参数**

```
type: wholesale_retail | accommodation_catering
industry_type: wholesale | retail | accommodation | catering | all
is_small_micro: 0 | 1 | all
is_eat_wear_use: 0 | 1 | all
search: <企业名称关键词>
page: 1
page_size: 50
```

**响应**

```json
{
  "total": 200,
  "page": 1,
  "page_size": 50,
  "data_year": 2026,
  "data_month": 1,
  "items": [
    {
      "id": 1,
      "credit_code": "91110000XXXX",
      "name": "北京XX商贸有限公司",
      "industry_code": "5111",
      "industry_type": "wholesale",
      "company_scale": 3,
      "sales_current_month": 1234.56,
      "sales_last_year_month": 1150.00,
      "sales_month_rate": 7.35,
      "retail_current_month": 1000.00,
      "retail_last_year_month": 950.00,
      "retail_month_rate": 5.26,
      "is_small_micro": 1,
      "is_eat_wear_use": 0,
      "row_no": 1
    }
  ]
}
```

---

### 5. 企业数据更新

#### PATCH /api/companies/:id

修改企业的可调整字段

**请求体**

```json
{
  "sales_current_month": 1300.00,
  "retail_current_month": 1100.00
}
```

**响应**

```json
{
  "success": true,
  "company": { ... },
  "updated_indicators": {
    "limit_above_retail_month": 125678.90,
    "limit_above_retail_month_rate": 5.5,
    "wholesale_sales_month_rate": 4.2
  }
}
```

**说明**
- 更新后自动触发 DAG 增量计算
- 返回受影响的指标列表
- 前端接收后更新指标卡片

---

### 6. 配置管理

#### GET /api/config

获取系统配置

**响应**

```json
{
  "current_year": 2026,
  "current_month": 1,
  "small_micro_rate_month": 4.5,
  "eat_wear_use_rate_month": 5.8,
  "weight_small_micro": 0.3,
  "weight_eat_wear_use": 0.3,
  "weight_sample": 0.4,
  "last_year_limit_below_cumulative": 100000.00
}
```

#### PUT /api/config

批量更新配置

**请求体**

```json
{
  "small_micro_rate_month": 5.0,
  "eat_wear_use_rate_month": 6.0
}
```

**响应**

```json
{
  "success": true,
  "updated_indicators": {
    "total_retail_cumulative": 240000.00,
    "total_retail_cumulative_rate": 5.8
  }
}
```

---

### 7. 数据导出

#### GET /api/export

导出定稿 Excel

**请求参数**

```
format: final | current
```

**响应**

```
Content-Type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
Content-Disposition: attachment; filename="2026年1月月报(定).xlsx"

<Excel 文件二进制流>
```

---

## 后端模块设计

### 目录结构

```
internal/
├── server/
│   ├── handlers/
│   │   ├── status.go       # 系统状态
│   │   ├── import.go       # 数据导入
│   │   ├── indicators.go   # 指标查询
│   │   ├── companies.go    # 企业数据
│   │   └── config.go       # 配置管理
│   └── server.go           # Gin 路由配置
├── store/
│   ├── sqlite.go           # SQLite 封装
│   ├── wholesale_retail.go # 批零数据操作
│   ├── accommodation.go    # 住餐数据操作
│   ├── snapshot.go         # 快照数据操作
│   └── config.go           # 配置操作
├── service/
│   ├── excel/
│   │   ├── parser.go       # Excel 解析器
│   │   ├── recognizer.go   # Sheet 类型识别
│   │   ├── mapper.go       # 字段映射
│   │   └── exporter.go     # 导出生成器
│   ├── calculator/
│   │   ├── engine.go       # DAG 计算引擎
│   │   ├── indicators.go   # 指标计算
│   │   └── aggregator.go   # 汇总计算
│   └── importer.go         # 导入协调器
└── model/
    ├── company.go          # 企业数据模型
    ├── indicator.go        # 指标模型
    └── config.go           # 配置模型
```

---

## 核心逻辑实现

### 导入流程 (internal/service/importer.go)

```go
type Importer struct {
    store      *store.Store
    parser     *excel.Parser
    recognizer *excel.Recognizer
    calculator *calculator.Engine
}

func (imp *Importer) Import(file io.Reader, progress chan<- ImportProgress) (*ImportReport, error) {
    // 1. 打开 Excel 文件
    f, err := excelize.OpenReader(file)
    if err != nil {
        return nil, err
    }
    defer f.Close()

    report := &ImportReport{}

    // 2. 遍历所有 Sheet
    sheets := f.GetSheetList()
    report.TotalSheets = len(sheets)

    var dataYear, dataMonth int
    mainSheets := []string{}

    for _, sheetName := range sheets {
        progress <- ImportProgress{Percent: 10, Message: fmt.Sprintf("识别 Sheet: %s", sheetName)}

        // 3. 识别 Sheet 类型
        sheetType, confidence := imp.recognizer.Recognize(f, sheetName)

        sheetReport := SheetReport{
            Name:       sheetName,
            Type:       sheetType,
            Confidence: confidence,
        }

        // 4. 根据类型处理
        switch sheetType {
        case "wholesale", "retail":
            // 解析主表
            companies, year, month, err := imp.parser.ParseWRMain(f, sheetName, sheetType)
            if err != nil {
                sheetReport.Status = "error"
                sheetReport.Errors = []string{err.Error()}
            } else {
                // 记录数据月份
                if dataYear == 0 {
                    dataYear, dataMonth = year, month
                }
                mainSheets = append(mainSheets, sheetName)
                sheetReport.Status = "imported"
                sheetReport.Rows = len(companies)
                report.ImportedRows += len(companies)
            }

        case "wr_snapshot":
            // 解析历史快照
            year, month, _ := excel.ExtractSheetYearMonth(sheetName)
            snapshots, err := imp.parser.ParseWRSnapshot(f, sheetName, year, month)
            if err != nil {
                sheetReport.Status = "error"
            } else {
                imp.store.SaveWRSnapshots(snapshots)
                sheetReport.Status = "imported"
                sheetReport.Rows = len(snapshots)
            }

        default:
            sheetReport.Status = "skipped"
            report.SkippedSheets++
        }

        report.Sheets = append(report.Sheets, sheetReport)
    }

    // 5. 更新配置中的当前月份
    imp.store.UpdateConfig("current_year", dataYear)
    imp.store.UpdateConfig("current_month", dataMonth)

    // 6. 计算衍生指标
    progress <- ImportProgress{Percent: 90, Message: "计算衍生指标..."}
    imp.calculator.CalculateAll(dataYear, dataMonth)

    progress <- ImportProgress{Percent: 100, Message: "导入完成"}

    return report, nil
}
```

---

### DAG 计算引擎 (internal/service/calculator/engine.go)

```go
type Engine struct {
    store *store.Store
}

// 全量计算（导入后）
func (e *Engine) CalculateAll(dataYear, dataMonth int) error {
    // 1. 计算企业级衍生字段（增速等）
    e.calculateCompanyMetrics(dataYear, dataMonth)

    // 2. 计算行业汇总
    e.calculateIndustryAggregates(dataYear, dataMonth)

    // 3. 计算全局指标
    e.calculateGlobalIndicators(dataYear, dataMonth)

    return nil
}

// 增量计算（单个企业修改后）
func (e *Engine) RecalculateFromCompany(companyID int, companyType string) (map[string]float64, error) {
    // 1. 重算该企业的衍生字段
    e.recalculateCompanyMetrics(companyID, companyType)

    // 2. 重算受影响的行业汇总
    affectedIndustries := e.getAffectedIndustries(companyID, companyType)
    for _, industry := range affectedIndustries {
        e.recalculateIndustryAggregate(industry)
    }

    // 3. 重算全局指标
    updatedIndicators := e.recalculateGlobalIndicators()

    return updatedIndicators, nil
}

func (e *Engine) calculateCompanyMetrics(dataYear, dataMonth int) error {
    // 批零企业
    sql := `
    UPDATE wholesale_retail SET
        sales_month_rate = CASE
            WHEN sales_last_year_month = 0 THEN NULL
            ELSE (sales_current_month - sales_last_year_month) / sales_last_year_month * 100
        END,
        retail_month_rate = CASE
            WHEN retail_last_year_month = 0 THEN NULL
            ELSE (retail_current_month - retail_last_year_month) / retail_last_year_month * 100
        END
    WHERE data_year = ? AND data_month = ?
    `
    return e.store.Exec(sql, dataYear, dataMonth)
}

func (e *Engine) calculateIndustryAggregates(dataYear, dataMonth int) error {
    // 计算四大行业汇总
    // 存储到临时表或缓存
    return nil
}

func (e *Engine) calculateGlobalIndicators(dataYear, dataMonth int) error {
    // 计算限上社零额、吃穿用、小微等全局指标
    return nil
}
```

---

### Excel 解析器 (internal/service/excel/parser.go)

```go
type Parser struct{}

// 解析批零主表
func (p *Parser) ParseWRMain(f *excelize.File, sheetName, sheetType string) ([]*model.WholesaleRetail, int, int, error) {
    rows, err := f.GetRows(sheetName)
    if err != nil {
        return nil, 0, 0, err
    }

    if len(rows) < 2 {
        return nil, 0, 0, errors.New("sheet 数据不足")
    }

    // 1. 解析表头，提取年月信息
    header := rows[0]
    dataYear, dataMonth := extractDataYearMonth(header)

    // 2. 建立字段映射
    mapper := NewFieldMapper(header, dataYear, dataMonth, sheetType)

    // 3. 逐行解析数据
    companies := []*model.WholesaleRetail{}
    for i, row := range rows[1:] {
        company := &model.WholesaleRetail{
            DataYear:     dataYear,
            DataMonth:    dataMonth,
            IndustryType: sheetType,
            RowNo:        i + 1,
        }

        // 遍历每列，根据映射填充字段
        for colIdx, cellValue := range row {
            dbField := mapper.GetDBField(colIdx)
            if dbField != "" {
                setValue(company, dbField, cellValue)
            }
        }

        companies = append(companies, company)
    }

    return companies, dataYear, dataMonth, nil
}

// 从表头提取数据年月
func extractDataYearMonth(header []string) (int, int) {
    maxYear, maxMonth := 0, 0

    for _, colName := range header {
        year, month, found := extractYearMonth(colName)
        if found {
            // 找出最大的年月组合（认为是当前数据月份）
            if year > maxYear || (year == maxYear && month > maxMonth) {
                maxYear, maxMonth = year, month
            }
        }
    }

    return maxYear, maxMonth
}
```

---

## 错误处理

### 错误码定义

```go
const (
    ErrCodeInvalidFile        = 1001 // 无效的 Excel 文件
    ErrCodeSheetNotFound      = 1002 // Sheet 不存在
    ErrCodeFieldMissing       = 1003 // 缺少必填字段
    ErrCodeDuplicateCreditCode= 1004 // 重复的信用代码
    ErrCodeInvalidDataFormat  = 1005 // 数据格式错误
)
```

### 错误响应

```json
{
  "success": false,
  "error": {
    "code": 1003,
    "message": "批发 Sheet 缺少必填字段: 统一社会信用代码"
  }
}
```

---

## 性能优化

### 1. 批量插入

```go
// 每 100 条提交一次事务
const batchSize = 100

func (s *Store) BatchInsert(companies []*WholesaleRetail) error {
    tx, _ := s.db.Begin()
    stmt, _ := tx.Prepare("INSERT INTO wholesale_retail (...) VALUES (...)")

    for i, c := range companies {
        stmt.Exec(...)
        if (i+1)%batchSize == 0 {
            tx.Commit()
            tx, _ = s.db.Begin()
            stmt, _ = tx.Prepare(...)
        }
    }

    tx.Commit()
    return nil
}
```

### 2. 增量计算缓存

```go
type IndicatorCache struct {
    indicators map[string]float64
    lastUpdate time.Time
    mu         sync.RWMutex
}

func (c *IndicatorCache) Get(key string) (float64, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    val, ok := c.indicators[key]
    return val, ok
}

func (c *IndicatorCache) Set(indicators map[string]float64) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.indicators = indicators
    c.lastUpdate = time.Now()
}
```

### 3. 数据库索引

参考 `01_database.md` 中的索引设计，确保高频查询字段都有索引。

---

## 日志记录

```go
// 使用结构化日志
log.Info("导入开始",
    "filename", filename,
    "size", fileSize,
)

log.Error("解析失败",
    "sheet", sheetName,
    "error", err,
)

log.Info("导入完成",
    "filename", filename,
    "total_rows", report.ImportedRows,
    "duration", report.Duration,
)
```
