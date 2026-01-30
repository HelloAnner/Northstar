# Excel 自动解析策略

## 设计目标

1. **全量解析**：Excel 所有 Sheet、所有字段都能入库
2. **自动识别**：无需用户手动映射字段，系统智能识别
3. **容错处理**：字段名差异、新增 Sheet 都能处理
4. **进度反馈**：实时显示解析进度和结果

---

## 解析流程

```
┌─────────────────────────────────────────────────────────────────┐
│                      Excel 文件上传                              │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  Step 1: 遍历所有 Sheet                                          │
│  - 读取 Sheet 名称列表                                           │
│  - 记录到 sheets_meta 表                                         │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  Step 2: 逐个 Sheet 识别类型                                     │
│  - 读取首行列名                                                  │
│  - 匹配关键字段集合                                              │
│  - 计算置信度，判定 Sheet 类型                                   │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  Step 3: 智能字段映射                                            │
│  - 根据 Sheet 类型选择映射规则                                   │
│  - 模糊匹配列名 → 数据库字段                                     │
│  - 动态识别年月信息                                              │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  Step 4: 数据入库                                                │
│  - 清空旧数据 (同类型)                                           │
│  - 逐行解析并写入数据库                                          │
│  - 自动计算衍生字段 (增速、分类标记)                             │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  Step 5: 结果汇总                                                │
│  - 更新 import_logs                                              │
│  - 返回解析报告                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## Sheet 类型识别

### 识别规则表

| Sheet 类型 | 目标表 | 关键字段 | 命中阈值 |
|-----------|--------|----------|----------|
| wholesale | wholesale_retail | 销售额 + 零售额 + 行业代码51开头 | ≥8/11 |
| retail | wholesale_retail | 销售额 + 零售额 + 行业代码52开头 | ≥8/11 |
| accommodation | accommodation_catering | 营业额 + 客房收入 + 餐费收入 | ≥8/12 |
| catering | accommodation_catering | 营业额 + 客房收入 + 餐费收入 | ≥8/12 |
| wr_snapshot | wr_snapshot | 本年-本月/上年-本月 口径 | ≥6/8 |
| ac_snapshot | ac_snapshot | 营业额;本年-本月 口径 | ≥6/8 |
| summary | (跳过) | 限上零售额/小微/吃穿用 | - |
| unknown | sheets_meta | 无法识别 | <0.5 |

### 关键字段匹配 (正则)

```go
// 批零主表识别字段
var wrMainFields = []string{
    `统一社会信用代码`,
    `单位(详细)?名称`,
    `行业代码`,
    `\d{4}年\d{1,2}月销售额`,       // 2025年12月销售额
    `\d{4}年.*商品销售额`,          // 2024年;12月;商品销售额
    `\d{4}年1-\d{1,2}月销售额`,     // 2025年1-12月销售额
    `\d{4}年\d{1,2}月零售额`,       // 2025年12月零售额
    `\d{4}年.*商品零售额`,          // 2024年;12月;商品零售额
    `单位规模`,
    `粮油食品类`,
    `小微企业`,
}

// 住餐主表识别字段
var acMainFields = []string{
    `统一社会信用代码`,
    `单位(详细)?名称`,
    `行业代码`,
    `\d{4}年\d{1,2}月营业额`,       // 2025年12月营业额
    `\d{4}年.*营业额总计`,          // 2024年12月;营业额总计
    `客房收入`,
    `餐费收入`,
    `\d{4}年\d{1,2}月销售额`,       // 商品销售额
}

// 批零快照识别字段
var wrSnapshotFields = []string{
    `统一社会信用代码`,
    `单位(详细)?名称`,
    `行业代码`,
    `商品销售额;本年-本月`,
    `商品销售额;本年-1—本月`,
    `零售额;本年-本月`,
    `零售额;本年-1—本月`,
    `单位规模`,
}
```

### Sheet 名称辅助判定

```go
// Sheet 名称关键词加权
var sheetNameBoost = map[string]float64{
    "批发":   0.2,
    "零售":   0.2,
    "住宿":   0.2,
    "餐饮":   0.2,
    "批零":   0.15,
    "住餐":   0.15,
    "小微":   0.1,
    "吃穿用": 0.1,
    "汇总":   0.1,
    "社零":   0.1,
}

// 识别年月
// "2024年12月批零" → year=2024, month=12, type=wr_snapshot
var sheetYearMonthRegex = regexp.MustCompile(`(\d{4})年(\d{1,2})月(批零|住餐|批发|零售|住宿|餐饮)?`)
```

---

## 智能字段映射

### 映射算法

```go
type FieldMapper struct {
    rules []MappingRule
}

type MappingRule struct {
    Pattern   *regexp.Regexp  // 列名匹配正则
    DBField   string          // 数据库字段名
    Priority  int             // 优先级 (数字越大优先)
    Transform func(string) interface{} // 值转换函数
}

// 批零主表映射规则
var wrMainMappingRules = []MappingRule{
    // 基础信息
    {Pattern: regexp.MustCompile(`统一社会信用代码`), DBField: "credit_code", Priority: 10},
    {Pattern: regexp.MustCompile(`单位(详细)?名称|企业名称`), DBField: "name", Priority: 10},
    {Pattern: regexp.MustCompile(`行业代码`), DBField: "industry_code", Priority: 10},
    {Pattern: regexp.MustCompile(`单位规模`), DBField: "company_scale", Priority: 10},

    // 销售额 - 本年本月 (优先匹配 2025年12月销售额)
    {Pattern: regexp.MustCompile(`^2025年12月销售额$`), DBField: "sales_current_month", Priority: 20},
    {Pattern: regexp.MustCompile(`^\d{4}年\d{1,2}月销售额$`), DBField: "sales_current_month", Priority: 15},

    // 销售额 - 上年同期 (匹配 2024年;12月;商品销售额)
    {Pattern: regexp.MustCompile(`2024年.*12月.*商品销售额`), DBField: "sales_last_year_month", Priority: 20},

    // 销售额 - 累计
    {Pattern: regexp.MustCompile(`^2025年1-12月销售额$`), DBField: "sales_current_cumulative", Priority: 20},
    {Pattern: regexp.MustCompile(`2024年.*1-12月.*商品销售额`), DBField: "sales_last_year_cumulative", Priority: 20},

    // 零售额 - 类似逻辑
    {Pattern: regexp.MustCompile(`^2025年12月零售额$`), DBField: "retail_current_month", Priority: 20},
    {Pattern: regexp.MustCompile(`2024年.*12月.*商品零售额`), DBField: "retail_last_year_month", Priority: 20},
    {Pattern: regexp.MustCompile(`^2025年1-12月零售额$`), DBField: "retail_current_cumulative", Priority: 20},
    {Pattern: regexp.MustCompile(`2024年.*1-12月.*商品零售额`), DBField: "retail_last_year_cumulative", Priority: 20},

    // 商品分类
    {Pattern: regexp.MustCompile(`粮油食品类`), DBField: "cat_grain_oil_food", Priority: 10},
    {Pattern: regexp.MustCompile(`饮料类`), DBField: "cat_beverage", Priority: 10},
    {Pattern: regexp.MustCompile(`烟酒类`), DBField: "cat_tobacco_liquor", Priority: 10},
    {Pattern: regexp.MustCompile(`服装鞋帽针纺`), DBField: "cat_clothing", Priority: 10},
    {Pattern: regexp.MustCompile(`日用品类`), DBField: "cat_daily_use", Priority: 10},
    {Pattern: regexp.MustCompile(`汽车类`), DBField: "cat_automobile", Priority: 10},

    // 标记
    {Pattern: regexp.MustCompile(`小微企业`), DBField: "is_small_micro", Priority: 10},
    {Pattern: regexp.MustCompile(`吃穿用`), DBField: "is_eat_wear_use", Priority: 10},
}
```

### 动态年月识别

Excel 中的列名和 Sheet 名都包含年月信息，解析器需要智能识别并正确映射。

```go
// 从列名中提取年月信息
func extractYearMonth(columnName string) (year, month int, found bool) {
    // 匹配模式: "2025年12月销售额" -> year=2025, month=12
    re := regexp.MustCompile(`(\d{4})年(\d{1,2})月`)
    matches := re.FindStringSubmatch(columnName)
    if len(matches) >= 3 {
        year, _ = strconv.Atoi(matches[1])
        month, _ = strconv.Atoi(matches[2])
        return year, month, true
    }
    return 0, 0, false
}

// 从 Sheet 名提取年月
func extractSheetYearMonth(sheetName string) (year, month int, found bool) {
    // 匹配: "2024年12月批零" / "2025年1月" 等
    re := regexp.MustCompile(`(\d{4})年(\d{1,2})月`)
    matches := re.FindStringSubmatch(sheetName)
    if len(matches) >= 3 {
        year, _ = strconv.Atoi(matches[1])
        month, _ = strconv.Atoi(matches[2])
        return year, month, true
    }
    return 0, 0, false
}

// 判断列名对应的字段类型
type FieldTimeType int

const (
    CurrentMonth     FieldTimeType = iota // 本月/当月
    PrevMonth                             // 上月
    LastYearMonth                         // 去年同期
    CurrentCumulative                     // 本年累计
    PrevCumulative                        // 本年累计到上月
    LastYearCumulative                    // 上年累计
)

// 智能推断字段时间类型
func inferFieldTimeType(columnName string, currentYear, currentMonth int) FieldTimeType {
    year, month, found := extractYearMonth(columnName)
    if !found {
        // 无法提取年月，通过关键词判断
        if strings.Contains(columnName, "上年") || strings.Contains(columnName, "去年") {
            if strings.Contains(columnName, "1-") || strings.Contains(columnName, "累计") {
                return LastYearCumulative
            }
            return LastYearMonth
        }
        return CurrentMonth
    }

    // 判断是累计还是单月
    isCumulative := strings.Contains(columnName, "1-") || strings.Contains(columnName, "累计")

    // 判断时间关系
    if year == currentYear {
        if isCumulative {
            if month == currentMonth {
                return CurrentCumulative
            } else if month == currentMonth-1 {
                return PrevCumulative
            }
        } else {
            if month == currentMonth {
                return CurrentMonth
            } else if month == currentMonth-1 {
                return PrevMonth
            }
        }
    } else if year == currentYear-1 {
        if isCumulative {
            return LastYearCumulative
        }
        return LastYearMonth
    }

    return CurrentMonth // 默认
}

// 字段映射示例
func mapColumnToDBField(columnName string, currentYear, currentMonth int, sheetType string) string {
    timeType := inferFieldTimeType(columnName, currentYear, currentMonth)

    if sheetType == "wholesale" || sheetType == "retail" {
        // 判断是销售额还是零售额
        isSales := strings.Contains(columnName, "销售额")
        isRetail := strings.Contains(columnName, "零售额")

        if isSales {
            switch timeType {
            case CurrentMonth:
                return "sales_current_month"
            case PrevMonth:
                return "sales_prev_month"
            case LastYearMonth:
                return "sales_last_year_month"
            case CurrentCumulative:
                return "sales_current_cumulative"
            case PrevCumulative:
                return "sales_prev_cumulative"
            case LastYearCumulative:
                return "sales_last_year_cumulative"
            }
        } else if isRetail {
            switch timeType {
            case CurrentMonth:
                return "retail_current_month"
            case PrevMonth:
                return "retail_prev_month"
            case LastYearMonth:
                return "retail_last_year_month"
            case CurrentCumulative:
                return "retail_current_cumulative"
            case PrevCumulative:
                return "retail_prev_cumulative"
            case LastYearCumulative:
                return "retail_last_year_cumulative"
            }
        }
    }

    // 住餐类似处理
    // ...

    return ""
}
```

### 月份自适应解析流程

```
1. 读取主表 Sheet (批发/零售/住宿/餐饮)
   ↓
2. 提取所有列名中的年月信息
   - 找出最大的 year+month 组合 (如 2026年1月)
   - 作为当前数据月份: data_year=2026, data_month=1
   ↓
3. 计算相对月份
   - current_year = 2026, current_month = 1
   - prev_month = 上一个月 (2025年12月)
   - last_year_month = 去年同期 (2025年1月)
   ↓
4. 遍历每一列
   - 提取列名中的年月
   - 判断与当前月份的关系
   - 映射到对应的数据库字段
   ↓
5. 设置企业记录的 data_year 和 data_month
   - 所有企业记录标记为 data_year=2026, data_month=1
```

### 示例：解析 1月数据

假设导入 "2026年1月月报.xlsx"，批发 Sheet 列名如下：

```
| 2025年12月销售额 | 2026年1月销售额 | 2025年1月销售额 | 2026年1月销售额 | 2025年1月销售额 |
| (上月)           | (本月)          | (去年同期)      | (本年累计)      | (上年累计)      |
```

解析逻辑：
1. 识别 data_year=2026, data_month=1
2. 列名映射：
   - "2025年12月销售额" → sales_prev_month (上月)
   - "2026年1月销售额" (无累计关键词) → sales_current_month (本月)
   - "2025年1月销售额" (无累计关键词) → sales_last_year_month (去年同期)
   - "2026年1月销售额" (含累计) → sales_current_cumulative (本年累计，1月时等于当月)
   - "2025年1月销售额" (含累计) → sales_last_year_cumulative (上年累计)

### 处理边界情况

1. **1月没有"本年累计到上月"**
   - sales_prev_cumulative 字段留空或设为 0
   - 解析器检测到 data_month=1 时自动跳过该字段

2. **列名格式不统一**
   - 支持多种格式: "2026年1月销售额" / "2026年01月销售额" / "销售额;2026年1月"
   - 正则表达式灵活匹配

3. **缺失字段**
   - 如果 Excel 中缺少某个时间维度的列，对应字段设为 NULL 或 0
   - 记录到日志中供用户确认

4. **多Sheet 自动识别当前月份**
   - 优先从主表 Sheet (批发/零售/住宿/餐饮) 识别当前月份
   - 历史快照 Sheet 自动识别各自的年月，导入到 snapshot 表


---

## 数据入库策略

### 清空策略

```go
// 导入前清空同类型数据
func (s *Store) ClearBeforeImport(sheetType string) error {
    switch sheetType {
    case "wholesale", "retail":
        return s.db.Exec("DELETE FROM wholesale_retail WHERE industry_type = ?", sheetType).Error
    case "accommodation", "catering":
        return s.db.Exec("DELETE FROM accommodation_catering WHERE industry_type = ?", sheetType).Error
    case "wr_snapshot":
        // 快照按年月清空
        return s.db.Exec("DELETE FROM wr_snapshot WHERE snapshot_year = ? AND snapshot_month = ?", year, month).Error
    }
    return nil
}
```

### 批量插入

```go
// 批量插入，每 100 条提交一次
const batchSize = 100

func (s *Store) BatchInsertWR(companies []*WholesaleRetail) error {
    tx := s.db.Begin()
    for i, c := range companies {
        if err := tx.Create(c).Error; err != nil {
            tx.Rollback()
            return err
        }
        if (i+1) % batchSize == 0 {
            tx.Commit()
            tx = s.db.Begin()
        }
    }
    return tx.Commit().Error
}
```

### 衍生字段计算

```go
// 入库后自动计算衍生字段
func (s *Store) CalculateDerivedFields() error {
    // 计算增速
    sql := `
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
    `
    return s.db.Exec(sql).Error
}
```

---

## 解析报告

### 报告结构

```go
type ImportReport struct {
    Filename      string         `json:"filename"`
    TotalSheets   int            `json:"total_sheets"`
    ImportedSheets int           `json:"imported_sheets"`
    SkippedSheets int            `json:"skipped_sheets"`
    TotalRows     int            `json:"total_rows"`
    ImportedRows  int            `json:"imported_rows"`
    ErrorRows     int            `json:"error_rows"`
    Duration      time.Duration  `json:"duration"`
    Sheets        []SheetReport  `json:"sheets"`
}

type SheetReport struct {
    Name       string   `json:"name"`
    Type       string   `json:"type"`       // wholesale/retail/...
    Confidence float64  `json:"confidence"` // 识别置信度
    Status     string   `json:"status"`     // imported/skipped/error
    Rows       int      `json:"rows"`
    Errors     []string `json:"errors,omitempty"`
}
```

### 前端展示

```
导入完成
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
文件: 12月月报（预估）.xlsx
耗时: 2.3s

Sheet 解析结果:
┌─────────────────┬──────────────┬────────┬────────┐
│ Sheet 名称       │ 类型         │ 状态   │ 行数   │
├─────────────────┼──────────────┼────────┼────────┤
│ 批发             │ wholesale    │ ✓ 导入 │ 58     │
│ 零售             │ retail       │ ✓ 导入 │ 142    │
│ 住宿             │ accommodation│ ✓ 导入 │ 12     │
│ 餐饮             │ catering     │ ✓ 导入 │ 28     │
│ 2024年12月批零   │ wr_snapshot  │ ✓ 导入 │ 200    │
│ 限上零售额       │ summary      │ ⊘ 跳过 │ -      │
│ 小微             │ summary      │ ⊘ 跳过 │ -      │
└─────────────────┴──────────────┴────────┴────────┘

汇总: 导入 240 家企业，4 个历史快照
```

---

## 错误处理

### 错误分级

| 级别 | 描述 | 处理方式 |
|------|------|----------|
| Error | 整个 Sheet 无法解析 | 跳过该 Sheet，记录日志 |
| Warn | 部分行数据异常 | 跳过异常行，继续导入 |
| Info | 字段缺失但有默认值 | 使用默认值，记录日志 |

### 常见错误

```go
var ErrSheetTypeUnknown = errors.New("无法识别 Sheet 类型")
var ErrMissingRequiredField = errors.New("缺少必填字段")
var ErrInvalidDataFormat = errors.New("数据格式错误")
var ErrDuplicateCreditCode = errors.New("重复的信用代码")
```
