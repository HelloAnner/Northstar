# Indicator DAG + Export Params + Parse Normalize Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 修复指标联动随机化、导出数字参数化、输入解析标准化三类差距。

**Architecture:** 调整指标优化逻辑为“随机权重分配+总量校准”，对导出做“数值参数化→回填”，对导入做“列名规范化+识别打分+值标准化”。

**Tech Stack:** Go、Gin、SQLite、excelize

---

### Task 1: 指标联动随机化（DAG 下游明细随机）

**Files:**
- Modify: `internal/api/v3/optimize_test.go`
- Modify: `internal/api/v3/optimize.go`
- Modify: `internal/api/v3/companies.go`

**Step 1: 写失败测试（随机分配）**

```go
func TestOptimize_RandomizeLimitAboveMonthValue(t *testing.T) {
  // 2 行 WR，目标 300，期望非等比分配且总和=300
}
```

**Step 2: 运行测试验证失败**

Run: `go test ./internal/api/v3 -run TestOptimize_RandomizeLimitAboveMonthValue -v`
Expected: 编译失败或断言失败（因仍为等比例缩放）

**Step 3: 实现随机分配**

```go
// 随机权重 -> 目标分配 -> diff 修正
values := randomizeAllocations(target, bases)
```

**Step 4: 更新衍生字段回算**

```sql
UPDATE accommodation_catering SET
  retail_current_month = food_current_month + goods_current_month,
  retail_last_year_month = food_last_year_month + goods_last_year_month
```

**Step 5: 运行测试验证通过**

Run: `go test ./internal/api/v3 -run TestOptimize_RandomizeLimitAboveMonthValue -v`
Expected: PASS

---

### Task 2: 导出数字参数化与回填

**Files:**
- Modify: `internal/exporter/formula_preserve_test.go`
- Modify: `internal/exporter/exporter.go`

**Step 1: 写失败测试（参数化/回填）**

```go
func TestParameterizeNumbersThenMaterialize(t *testing.T) {
  // A1=123, B1=包含数字文本, C1=公式
  // 参数化后含占位符，回填后还原
}
```

**Step 2: 运行测试验证失败**

Run: `go test ./internal/exporter -run TestParameterizeNumbersThenMaterialize -v`
Expected: 编译失败（缺少函数）

**Step 3: 实现参数化与回填**

```go
params, _ := parameterizeWorkbookNumbers(f)
_ = materializeWorkbookNumbers(f, params)
```

**Step 4: 运行测试验证通过**

Run: `go test ./internal/exporter -run TestParameterizeNumbersThenMaterialize -v`
Expected: PASS

---

### Task 3: 输入解析标准化与识别增强

**Files:**
- Modify: `internal/parser/utils_test.go`
- Modify: `internal/parser/recognizer_test.go`
- Modify: `internal/parser/utils.go`
- Modify: `internal/parser/sheet_recognizer.go`
- Modify: `internal/parser/field_mapper.go`
- Modify: `internal/parser/wr_parser.go`
- Modify: `internal/parser/ac_parser.go`

**Step 1: 写失败测试（列名规范/汇总识别）**

```go
func TestNormalizeColumnName_FullWidthSeparators(t *testing.T) {}
func TestSheetRecognizer_SummaryByHeaders(t *testing.T) {}
```

**Step 2: 运行测试验证失败**

Run: `go test ./internal/parser -run TestNormalizeColumnName_FullWidthSeparators -v`
Expected: FAIL（全角分隔符未归一）

**Step 3: 实现列名规范化与打分识别**

```go
normalized := normalizeHeaderForRecognition(col)
confidence := scoreByRequiredHeaders(normalized, required)
```

**Step 4: 标准化字段值存储**

```go
record.IndustryCode = normalizeIndustryCode(value)
record.CreditCode = normalizeCodeValue(value)
```

**Step 5: 运行识别回归测试**

Run: `go test ./internal/parser -run TestSheetRecognizer_DecemberMonthlyReport_20260129 -v`
Expected: PASS

---

### Task 4: 基础回归

**Files:**
- N/A

**Step 1: 运行核心测试**

Run:
- `go test ./internal/api/v3 -run TestOptimize_AdjustLimitAboveCumulativeRate -v`
- `go test ./internal/exporter -run TestExport_PreserveTemplateFormulas -v`
- `go test ./internal/parser -run TestSheetRecognizer_DecemberMonthlyReport_20260129 -v`

Expected: PASS
