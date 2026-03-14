# Northstar TECH-001：技术实现汇总文档

> 作者：Anner
> 创建：2026-03-14
> 版本：v1.0（全量汇总）

---

## 1. 技术栈

| 层 | 技术 |
|----|------|
| 后端语言 | Go（Gin 框架） |
| 数据库 | SQLite（单文件，`store.Store` 封装） |
| Excel 处理 | excelize |
| 前端 | React + Vite + TypeScript |
| UI 组件 | shadcn/ui + Tailwind CSS |
| 状态管理 | Zustand |
| 测试 | Go test（单元 + 集成） / pytest（e2e） |

---

## 2. 项目包结构

```
internal/
  api/v3/          # HTTP Handler 层
    handler.go             # 路由注册
    companies.go           # 企业 CRUD + 派生字段重算
    linkage_preview.go     # 联动预览接口
    import.go              # Excel 导入接口
    import_preview.go      # 导入预览接口
    export_stream.go       # 导出流式接口
    optimize.go            # 智能调整接口
    llm_chat.go            # LLM 对话接口
    months.go              # 月份管理接口
    stubs.go               # 测试桩
  dagcalc/         # 核心 DAG 计算引擎
    types.go               # 基础类型（NodeID/UICoord/ExcelCoord/ImpactNode）
    graph.go               # DAG 图结构（正/反向边 + 坐标索引）
    linkage_build.go       # 联动图构建（统一入口 BuildLinkageGraph）
    linkage_coords.go      # 坐标映射（行号索引 TemplateIndex）
    impact.go              # 影响范围 BFS（ImpactRange）
    derived.go             # 派生字段重算（SQL CASE WHEN）
    indicator.go           # 16 指标计算（IndicatorCalculator）
    adjust.go              # 反向调整（ApplyIndicatorTarget + 随机分配）
    recalc.go              # 统一重算入口（RecalcAll）
    engine.go              # 引擎封装（Engine/ForwardRecalc/ReverseAdjust）
    exports.go             # 导出辅助
    indexes.go             # 图索引构建
  linkage/         # 早期联动模块（现为 dagcalc 的薄壳）
    types.go / impact.go / coords.go
  calculator/      # 早期指标计算（已被 dagcalc 替代）
  importer/        # Excel 导入逻辑
    raw_collector.go       # raw 层数据采集
  parser/          # Sheet 解析
    sheet_recognizer.go    # Sheet 类型识别
    wr_parser.go           # 批零数据解析
    ac_parser.go           # 住餐数据解析
    summary_parser.go      # 汇总 Sheet 解析
  exporter/        # Excel 导出
    exporter.go            # 主导出逻辑
    extra_sheets.go        # 汇总/指标 Sheet 写入
    compare_summary.go     # 导出前数据对账
  store/           # 数据库访问层
    store.go               # Store 封装
    wholesale_retail.go    # WR 表 CRUD
    accommodation_catering.go # AC 表 CRUD
    sheet_raw.go           # raw 层 CRUD
    summary.go             # summary 层 CRUD
    months.go              # 月份管理
  model/           # 数据模型定义
    company.go / canonical.go / sheet.go / indicator.go
  reporttpl/       # 内嵌模板
    embedded_template_data.go  # gzip+base64 编码的 xlsx
  llm/             # LLM 接口封装（Claude API）
  server/          # HTTP 服务启动
  config/          # 配置加载
web/               # 前端
  src/
    pages/         # DashboardV3.tsx / ImportV3.tsx 等
    components/    # CompaniesTable.tsx 等
    services/      # api.ts
    store/         # undoStore / dataStore 等
```

---

## 3. DAG 引擎核心设计

### 3.1 节点命名规则

| 类型 | 命名格式 | 说明 |
|------|---------|------|
| 企业字段 | `wr:{id}:{field}` / `ac:{id}:{field}` | 与 UI ColumnKey 对齐 |
| 行业聚合 | `industry:{type}:{metric}` | 按行业类型汇总 |
| 全局汇总 | `aggregate:{metric}` | 限上/吃穿用/小微汇总 |
| 指标 | `indicator:{id}` | 16 项固定指标 |
| summary | `summary:{kind}:{field}` | 汇总 Sheet 节点 |

### 3.2 Graph 数据结构

```go
type Graph struct {
    Edges        map[NodeID][]NodeID      // 正向边（父→子）
    ReverseEdges map[NodeID][]NodeID      // 反向边（子→父），用于预览 BFS
    UICoords     map[NodeID]UICoord       // 节点→UI 坐标
    ExcelCoords  map[NodeID][]ExcelCoord  // 节点→Excel 坐标（可多个）
    IndicatorIDs map[NodeID]string        // 指标节点→指标 ID
    UIIndex      map[string]NodeID        // "rowId|columnKey"→节点
    ExcelIndex   map[string]NodeID        // "sheet|cell"→节点
}
```

### 3.3 BuildLinkageGraph 构图顺序

```
1. attachIndicatorNodes      // 注册 16 指标节点
2. attachCompanyCoords       // WR/AC 企业节点绑定 UI + Excel 坐标
3. attachCompanyEdges        // 企业字段内部正向边
4. attachAggregateEdges      // 企业 → 行业聚合/全局汇总正向边
5. attachSummaryEdges        // summary 节点 → 指标节点边
6. addIndicatorEdges         // 汇总/行业 → 16 指标正向边
7. attachIndicatorCoords     // 指标节点绑定 Excel 坐标
8. attachAggregateCoords     // 行业/汇总节点绑定 Excel 坐标
9. attachReverseEdges        // 补充反向边（派生字段 + 指标反查企业）
10. buildIndexes             // 构建 UIIndex + ExcelIndex
```

### 3.4 正向边规则（WR）

```
salesCurrentMonth  → salesMonthRate, salesYoYDiff, salesMoMDiff, salesMoMRate
salesLastYearMonth → salesMonthRate, salesYoYDiff
salesCurrentMonth  → salesCurrentCumulative
salesLastYearMonth → salesLastYearCumulative
salesCurrentCumulative + salesLastYearCumulative → salesCumulativeRate, salesCumulativeYoYDiff
retailCurrentMonth + salesCurrentMonth → retailRatio
// 企业字段 → 行业聚合 → 全局汇总 → 指标
```

### 3.5 正向边规则（AC 特有）

```
foodCurrentMonth + goodsCurrentMonth → retailCurrentMonth
foodLastYearMonth + goodsLastYearMonth → retailLastYearMonth
foodCurrentCumulative + goodsCurrentCumulative → retailCurrentCumulative
```

### 3.6 反向边（用于预览高亮）

```
// 增速/派生字段反查输入字段
salesMonthRate ← salesCurrentMonth, salesLastYearMonth
salesCumulativeRate ← salesCurrentCumulative, salesLastYearCumulative
retailCurrentMonth（AC）← foodCurrentMonth, goodsCurrentMonth

// 指标反查企业字段
limitAbove_month_value ← WR/AC retailCurrentMonth（全量企业）
wholesale_month_rate ← WR salesCurrentMonth（批发企业）
accommodation_month_rate ← AC salesCurrentMonth（住宿企业）
totalSocial_cumulative_value ← limitAbove_cumulative_value, limitAbove_cumulative_rate
```

---

## 4. 指标计算实现

### 4.1 派生字段重算（SQL 批量更新）

```sql
-- WR 派生字段
UPDATE wholesale_retail SET
    sales_month_rate = (sales_current_month - sales_last_year_month) / sales_last_year_month * 100,
    retail_ratio = retail_current_month / sales_current_month * 100,
    ...
WHERE data_year = ? AND data_month = ?

-- AC 派生字段（含住餐零售额联动）
UPDATE accommodation_catering SET
    revenue_month_rate = ...,
    retail_current_month = food_current_month + goods_current_month,
    retail_last_year_month = food_last_year_month + goods_last_year_month
WHERE data_year = ? AND data_month = ?
```

### 4.2 指标计算调用链

```
RecalcAll(store, year, month)
  └─ RecalcDerivedFields(store, year, month)   // 派生字段重算（SQL）
  └─ CalculateIndicators(store, year, month)   // 16 指标计算（Go）
       ├─ calculateLimitAbove()                // 限上社零额 4 项
       ├─ calculateSpecialRates()              // 吃穿用 + 小微 2 项
       ├─ calculateIndustryRates()             // 四大行业 8 项
       └─ calculateTotalSocial()              // 社零总额 2 项
```

### 4.3 社零总额计算

```go
estimatedLimitBelow = lastYearLimitBelow * (1 + microSmallRate/100)
totalSocial = limitAboveCumulative + estimatedLimitBelow
totalSocialRate = (totalSocial - lastYearTotal) / lastYearTotal * 100
```

---

## 5. 反向调整算法

### 5.1 入口

```go
ApplyIndicatorTarget(store, year, month, indicatorID, target float64)
```

16 个指标各有独立处理函数，统一策略：

1. 根据指标类型计算目标值（当月值 or 累计值）
2. 从 DB 加载相关企业行
3. 调用 `randomizeAllocations(target, bases, scales)` 生成分配方案
4. 事务写回 DB

### 5.2 随机分配算法

```
weights[i] = (0.5 * industryShare[i] + 0.5 * scaleShare[i]) * jitter
// jitter ∈ [0.7, 1.3]，防止完全按比例分配

values[i] = round(weight[i] / sum(weights) * target)
// 最后修正舍入差 adjustAllocationDiff()
```

### 5.3 累计类指标调整路径

```
目标累计 → 计算 desiredCurrentSum = targetCum - prevSum
→ 随机分配到各企业当月值
→ 同步更新累计值（prevCum + newCurrent）
```

### 5.4 社零总额 50/50 分摊

```go
delta = target - currentTotal
limitAboveTarget = currentLimitAbove + delta/2
limitBelowTarget = currentLimitBelow + delta/2
// 各自截断为非负，分别调整
```

---

## 6. Excel 坐标映射

### 6.1 模板行号索引（TemplateIndex）

- 读取模板 C 列行业码，扫描至空行结束
- 按 `RowNo` → `ID` 排序，依序绑定行号
- 支持 6 张表：批零总表 / 批发 / 零售 / 住餐总表 / 住宿 / 餐饮
- `index.maxRows[sheet]` = 最大数据行号，用于计算汇总行/增速行

### 6.2 动态行计算

```
sumRow = maxRow + 1      // 行业合计行
growthRow = maxRow + 2   // 增速行

// 批发特有：限上汇总行
whGrowthRow = whMaxRow + 2
totalRow = whGrowthRow + 3
totalGrowthRow = whGrowthRow + 4
```

### 6.3 Excel 写入策略

- 企业字段：按行业码匹配行，按字段匹配列，`SetCellValue`
- 汇总/指标：固定单元格地址，直接写入
- 公式单元格：**不写入数值，保留模板公式**
- 导出时检查 `summary` 数据与实时计算是否一致，差异仅提示不阻断

---

## 7. API 接口概览

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/months` | 获取所有月份 |
| POST | `/api/import` | 上传 Excel，执行导入 |
| GET | `/api/import/preview` | 导入预览（raw + 映射） |
| GET | `/api/companies` | 获取企业列表 + 指标 |
| PATCH | `/api/companies/:id` | 更新单企业字段 |
| PATCH | `/api/companies/batch` | 批量更新（撤销用） |
| POST | `/api/optimize` | 智能调整（反向调整指标） |
| POST | `/api/linkage/preview` | 联动影响范围预览 |
| GET | `/api/export` | 流式导出 Excel |
| POST | `/api/llm/chat` | LLM 对话 |

---

## 8. 数据库主要表

| 表 | 说明 |
|----|------|
| `wholesale_retail` | WR 企业数据（批发/零售） |
| `accommodation_catering` | AC 企业数据（住宿/餐饮） |
| `months` | 月份记录 |
| `config` | 全局配置（如 `last_year_limit_below_cumulative`） |
| `import_log` | 导入记录 |
| `sheet_cells` | raw 层：单元格原始数据 |
| `sheet_columns` | raw 层：列信息 |
| `sheet_rows` | raw 层：行信息 |
| `sheets_meta` | Sheet 元数据（识别结果/列映射） |
| `summary_*` | 业务汇总表（限上/小微/吃穿用等） |

---

## 9. 前端关键设计

### 9.1 撤销栈（undoStore）

```typescript
interface UndoStep {
  type: 'cell' | 'indicator' | 'optimize' | 'undo'
  summary: string
  changes: Array<{
    companyId: number
    fields: Record<string, { before: unknown; after: unknown }>
  }>
  createdAt: number
}
```

流程：编辑前读旧值 → 操作成功后计算 diff → push 入栈 → 撤销时反向 batch PATCH

### 9.2 联动高亮（highlightNodes）

Dashboard 维护 `highlightNodes: Set<string>`（nodeId 集合），企业表 TableCell 根据当前行 `rowId + columnKey` 拼装 nodeId，命中则渲染黄色边框。

点击任意单元格：

```
click → POST /api/linkage/preview → 返回 ImpactNode[] → 更新 highlightNodes
document.click（空白处）→ 清空 highlightNodes
```

---

## 10. 关键设计决策与演进

| 决策 | 当前实现 | 历史对比 |
|------|----------|----------|
| DAG 计算引擎 | `internal/dagcalc` 统一包 | 早期 `linkage` / `calculator` 分散 |
| 派生字段重算 | SQL CASE WHEN 批量 UPDATE | 早期 Go 逐行计算 |
| 反向调整分配 | 加权随机（行业占比 + 企业规模 50/50） | 早期均等分配 |
| 联动预览 | dagcalc.ImpactRange BFS | 早期 linkage.ComputeImpact 独立实现 |
| 导入数据层 | raw 层（sheet_cells）+ 业务表双轨 | 早期仅写业务表 |
| 模板嵌入 | gzip+base64 内嵌 Go 文件 | 早期外部文件路径 |
| 撤销 | 前端差异栈 + 后端 batch PATCH | 无撤销（仅重置） |

---

## 11. 测试覆盖

| 模块 | 测试文件 | 覆盖重点 |
|------|---------|---------|
| dagcalc | `impact_test.go` / `adjust_test.go` / `indicator_test.go` | BFS 影响范围 / 反向调整 / 指标计算 |
| linkage | `impact_test.go` / `coords_test.go` | 坐标映射 / 影响范围 |
| api/v3 | `companies_dag_linkage_test.go` / `optimize_test.go` | 联动 API / 智能调整 API |
| store | `wholesale_retail_upsert_test.go` / `sheet_raw_test.go` | 数据读写 |
| parser | `wr_parser_test.go` / `recognizer_test.go` | Sheet 解析与识别 |
| exporter | `export_compare_test.go` | 导出一致性 |

---

## 12. 运行环境

```bash
# 启动服务
go run ./cmd/server       # 端口 20261

# 构建
make build

# 后端测试
go test ./...

# e2e 测试（端口 20260）
pytest tests/
```
