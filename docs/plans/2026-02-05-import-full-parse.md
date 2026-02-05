# 全量导入与导出适配 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 预估 Excel 全量无损入库（raw + 业务表）并让导出基于新结构准确计算，导入页面全屏左右对比。

**Architecture:** 以 `sheet_cells/sheet_columns/sheet_rows` 作为原始真相层，新增 `summary_*` 业务汇总表；导入流程先落 raw，再做业务解析与汇总，导出固定模板但数据来源改为新结构，并做对账提示。

**Tech Stack:** Go (excelize/sqlite/gin), React + shadcn/ui, sqlite schema.sql

---

### Task 1: 新增 raw/summary 表结构

**Files:**
- Modify: `internal/store/schema.sql`
- Test: `internal/store/schema_raw_test.go`

**Step 1: Write the failing test**
```go
func TestSchema_IncludesRawAndSummaryTables(t *testing.T) {
    // 初始化临时 DB，检查 sheet_cells/sheet_columns/sheet_rows/summary_* 是否存在
}
```

**Step 2: Run test to verify it fails**
Run: `go test ./internal/store -run TestSchema_IncludesRawAndSummaryTables`
Expected: FAIL (missing tables)

**Step 3: Write minimal implementation**
- 在 `schema.sql` 添加 raw 表与 summary 表

**Step 4: Run test to verify it passes**
Run: `go test ./internal/store -run TestSchema_IncludesRawAndSummaryTables`
Expected: PASS

**Step 5: 跳过提交（仓库规定禁止 git add/commit）**

---

### Task 2: raw 表插入与结构持久化

**Files:**
- Create: `internal/store/sheet_raw.go`
- Modify: `internal/store/store.go`
- Test: `internal/store/sheet_raw_test.go`

**Step 1: Write the failing test**
```go
func TestInsertSheetRaw_SavesColumnsRowsCells(t *testing.T) {
    // 插入一张 2x2 的 sheet，校验 columns/rows/cells 均可查询
}
```

**Step 2: Run test to verify it fails**
Run: `go test ./internal/store -run TestInsertSheetRaw_SavesColumnsRowsCells`
Expected: FAIL

**Step 3: Write minimal implementation**
- 实现 `InsertSheetColumns/InsertSheetRows/BatchInsertSheetCells/InsertSheetMerges`

**Step 4: Run test to verify it passes**
Run: `go test ./internal/store -run TestInsertSheetRaw_SavesColumnsRowsCells`
Expected: PASS

**Step 5: 跳过提交（仓库规定禁止 git add/commit）**

---

### Task 3: raw 采集器（单元格、公式、格式、合并）

**Files:**
- Create: `internal/importer/raw_collector.go`
- Test: `internal/importer/raw_collector_test.go`

**Step 1: Write the failing test**
```go
func TestCollectSheetRaw_FormulaAndFormat(t *testing.T) {
    // 构造含公式/合并/格式的 sheet，验证采集结果
}
```

**Step 2: Run test to verify it fails**
Run: `go test ./internal/importer -run TestCollectSheetRaw_FormulaAndFormat`
Expected: FAIL

**Step 3: Write minimal implementation**
- 读取单元格 raw_value/formula/calc_value/num_format/style_id/merge_range

**Step 4: Run test to verify it passes**
Run: `go test ./internal/importer -run TestCollectSheetRaw_FormulaAndFormat`
Expected: PASS

**Step 5: 跳过提交（仓库规定禁止 git add/commit）**

---

### Task 4: summary 业务表解析

**Files:**
- Create: `internal/parser/summary_parser.go`
- Modify: `internal/parser/sheet_recognizer.go`
- Test: `internal/parser/summary_parser_test.go`

**Step 1: Write the failing test**
```go
func TestParseSummarySheet_LimitAboveRetail(t *testing.T) {
    // 构造“限上零售额”样例，验证 row_key/row_no/value/rate
}
```

**Step 2: Run test to verify it fails**
Run: `go test ./internal/parser -run TestParseSummarySheet_LimitAboveRetail`
Expected: FAIL

**Step 3: Write minimal implementation**
- summary 识别与解析（限上/小微/吃穿用）

**Step 4: Run test to verify it passes**
Run: `go test ./internal/parser -run TestParseSummarySheet_LimitAboveRetail`
Expected: PASS

**Step 5: 跳过提交（仓库规定禁止 git add/commit）**

---

### Task 5: 导入流程接入 raw + summary

**Files:**
- Modify: `internal/importer/coordinator.go`
- Modify: `internal/store/sheets_meta.go`
- Test: `internal/importer/import_summary_test.go`

**Step 1: Write the failing test**
```go
func TestImport_SavesRawAndSummary(t *testing.T) {
    // 导入含 summary 的文件，验证 raw 与 summary 表都有数据
}
```

**Step 2: Run test to verify it fails**
Run: `go test ./internal/importer -run TestImport_SavesRawAndSummary`
Expected: FAIL

**Step 3: Write minimal implementation**
- 导入先落 raw，再按类型落业务表
- 写入 column_mapping_json

**Step 4: Run test to verify it passes**
Run: `go test ./internal/importer -run TestImport_SavesRawAndSummary`
Expected: PASS

**Step 5: 跳过提交（仓库规定禁止 git add/commit）**

---

### Task 6: 导出取数迁移到新结构 + 对账提示

**Files:**
- Modify: `internal/exporter/exporter.go`
- Modify: `internal/exporter/extra_sheets.go`
- Test: `internal/exporter/exporter_summary_test.go`

**Step 1: Write the failing test**
```go
func TestExport_UsesSummaryTables(t *testing.T) {
    // 构造 summary 表数据，导出后校验模板对应单元格
}
```

**Step 2: Run test to verify it fails**
Run: `go test ./internal/exporter -run TestExport_UsesSummaryTables`
Expected: FAIL

**Step 3: Write minimal implementation**
- 导出时从 summary 表取值
- 对账差异仅提示不阻断

**Step 4: Run test to verify it passes**
Run: `go test ./internal/exporter -run TestExport_UsesSummaryTables`
Expected: PASS

**Step 5: 跳过提交（仓库规定禁止 git add/commit）**

---

### Task 7: DAG 扩展 summary 节点

**Files:**
- Modify: `internal/dagcalc/linkage_build.go`
- Modify: `docs/linkage-dag-spec.md`
- Test: `internal/dagcalc/linkage_summary_test.go`

**Step 1: Write the failing test**
```go
func TestDAG_IncludesSummaryNodes(t *testing.T) {
    // 校验 summary 节点存在与坐标绑定
}
```

**Step 2: Run test to verify it fails**
Run: `go test ./internal/dagcalc -run TestDAG_IncludesSummaryNodes`
Expected: FAIL

**Step 3: Write minimal implementation**
- 新增 summary 节点与 Excel 坐标映射

**Step 4: Run test to verify it passes**
Run: `go test ./internal/dagcalc -run TestDAG_IncludesSummaryNodes`
Expected: PASS

**Step 5: 跳过提交（仓库规定禁止 git add/commit）**

---

### Task 8: 导入全屏页面 + 左右对比

**Files:**
- Create: `web/src/pages/ImportV3.tsx`
- Modify: `web/src/App.tsx`
- Modify: `web/src/services/api.ts`
- Test: `web/src/pages/ImportV3.test.tsx`

**Step 1: Write the failing test**
```tsx
it('renders import layout with left/right panes', () => {
  // 渲染页面，断言左右面板存在
})
```

**Step 2: Run test to verify it fails**
Run: `cd web && npm test -- ImportV3.test.tsx`
Expected: FAIL

**Step 3: Write minimal implementation**
- 全屏布局、左右滚动面板
- 左侧多 sheet 预览，右侧业务映射

**Step 4: Run test to verify it passes**
Run: `cd web && npm test -- ImportV3.test.tsx`
Expected: PASS

**Step 5: 跳过提交（仓库规定禁止 git add/commit）**

---

### Task 9: API 输出 raw 预览与映射

**Files:**
- Modify: `internal/api/v3/handler.go`
- Create: `internal/api/v3/import_preview.go`
- Test: `internal/api/v3/import_preview_test.go`

**Step 1: Write the failing test**
```go
func TestImportPreview_ReturnsRawAndMapping(t *testing.T) {
    // 调用 API，检查 raw + mapping 数据结构
}
```

**Step 2: Run test to verify it fails**
Run: `go test ./internal/api/v3 -run TestImportPreview_ReturnsRawAndMapping`
Expected: FAIL

**Step 3: Write minimal implementation**
- 新增接口：返回 sheet 列表、raw preview、映射信息

**Step 4: Run test to verify it passes**
Run: `go test ./internal/api/v3 -run TestImportPreview_ReturnsRawAndMapping`
Expected: PASS

**Step 5: 跳过提交（仓库规定禁止 git add/commit）**

---

**执行说明**
- 由于仓库指令禁止 git add/commit，本计划中的提交步骤均跳过。
- 原计划建议在独立 worktree 执行；当前按你的要求直接在现有目录开始实施。
