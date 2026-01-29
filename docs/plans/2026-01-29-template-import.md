# Template Import + Fixed Output Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.
>
> **Note:** 本仓库规则禁止 git add / git commit，计划中不包含提交步骤。

**Goal:** 让输入 Excel 容错识别并人工选择操作月份，归纳后展示到仪表盘，支持联动调整，并按定稿模板 1:1 导出。

**Architecture:** 输入端使用“结构相似度识别 + 人工确认”，生成统一口径 Canonical 数据；DAG 引擎负责增量联动计算；输出端以定稿模板填充值并保留样式/公式。

**Tech Stack:** Go (excelize), React + Vite + shadcn/ui, Zustand, Tailwind

---

### Task 1: 识别规则落地文档（算法+伪代码+正则）

**Files:**
- Create: `prd/07_识别规则_算法伪代码与正则.md`

**Step 1: 写文档初稿（算法+伪代码+正则）**
- 包含：字段标准化规则、相似度计算公式、sheet 冲突处理、月份识别正则。

**Step 2: 自检文档是否覆盖所有 sheet 类型**
- 确认：批零主表、住餐主表、批零快照、住餐快照、吃穿用、小微、剔除、汇总/社零额模板。

---

### Task 2: 新增 Sheet 识别模块（Go）

**Files:**
- Create: `internal/service/excel/recognizer.go`
- Create: `internal/service/excel/recognizer_test.go`
- Modify: `internal/model/types.go`（或新增 `internal/model/sheet.go`）

**Step 1: 写失败测试（识别分类）**
```go
func TestRecognizeSheetTypes(t *testing.T) {
    wb := buildWorkbookWithHeaders(t, map[string][]string{
        "批发": {...},
        "零售": {...},
        "2025年11月批零": {...},
        "2025年11月住餐": {...},
        "吃穿用": {...},
        "小微": {...},
    })

    rec := excel.NewRecognizer()
    result := rec.RecognizeWorkbook(wb)

    assert.Equal(t, model.SheetTypeWholesaleMain, result["批发"].Type)
    assert.Equal(t, model.SheetTypeRetailMain, result["零售"].Type)
    assert.Equal(t, model.SheetTypeWholesaleRetailSnapshot, result["2025年11月批零"].Type)
}
```

**Step 2: 运行测试并确认失败**
Run: `go test ./internal/service/excel -run TestRecognizeSheetTypes -v`
Expected: FAIL（缺少 Recognizer 实现）

**Step 3: 写最小实现**
- 实现列名标准化、命中率计算、sheet 类型判定。
- 正则：`(\d{4})年(\d{1,2})月`，同时兼容“;12月;”风格。

**Step 4: 运行测试通过**
Run: `go test ./internal/service/excel -run TestRecognizeSheetTypes -v`
Expected: PASS

---

### Task 3: 上传接口返回识别结果 + 可选月份列表

**Files:**
- Modify: `internal/server/handlers/handlers.go`
- Modify: `internal/server/handlers/types.go`（若存在）
- Modify: `internal/model/sheet.go`
- Test: `tests/e2e/test_api.py`（新增接口字段断言）

**Step 1: 写失败测试（API 返回识别结果）**
```python
# tests/e2e/test_api.py

def test_upload_returns_sheet_recognition(api_session, test_excel_file):
    res = api_session.post(...)
    data = res.json()["data"]
    assert "recognition" in data
    assert "months" in data
```

**Step 2: 运行测试确认失败**
Run: `pytest tests/e2e/test_api.py::test_upload_returns_sheet_recognition -v`
Expected: FAIL（字段不存在）

**Step 3: 写最小实现**
- UploadFile：调用 Recognizer，返回 `recognition` + `months`。
- recognition 包含：sheetName、type、score、missingFields。

**Step 4: 测试通过**
Run: `pytest tests/e2e/test_api.py::test_upload_returns_sheet_recognition -v`
Expected: PASS

---

### Task 4: 解析阶段支持“手动选择操作月份 + 人工确认 sheet 类型”

**Files:**
- Modify: `internal/server/handlers/handlers.go`
- Create: `internal/service/excel/resolve.go`
- Create: `internal/service/excel/resolve_test.go`
- Modify: `internal/model/import.go`（新增 ResolveRequest/ResolveResult）

**Step 1: 写失败测试（Resolve 逻辑）**
```go
func TestResolveWorkbookForMonth(t *testing.T) {
    wb := buildWorkbookWithHeaders(...)
    result := excel.ResolveWorkbook(wb, excel.ResolveOptions{Month: 12, Overrides: map[string]SheetType{...}})
    require.NotNil(t, result.MainSheets[model.SheetTypeWholesaleMain])
}
```

**Step 2: 运行测试确认失败**
Run: `go test ./internal/service/excel -run TestResolveWorkbookForMonth -v`
Expected: FAIL

**Step 3: 写最小实现**
- ResolveWorkbook 根据“识别结果 + 用户 override + 月份”选择主表与快照。
- 无法识别的 sheet 进入 unknown 列表。

**Step 4: 测试通过**
Run: `go test ./internal/service/excel -run TestResolveWorkbookForMonth -v`
Expected: PASS

---

### Task 5: Canonical 数据模型 + DAG 计算引擎

**Files:**
- Modify: `internal/model/company.go`
- Create: `internal/model/canonical.go`
- Create: `internal/service/calculator/dag.go`
- Create: `internal/service/calculator/dag_test.go`
- Modify: `internal/service/calculator/engine.go`

**Step 1: 写失败测试（DAG 更新）**
```go
func TestDagRecomputeOnRetailChange(t *testing.T) {
    dag := calculator.NewDag()
    dag.SetRetailCurrent("c1", 100)
    dag.RecomputeFrom("c1.retailCurrent")
    assert.Equal(t, 100.0, dag.GetTotalRetailCurrent())
}
```

**Step 2: 运行测试确认失败**
Run: `go test ./internal/service/calculator -run TestDagRecomputeOnRetailChange -v`
Expected: FAIL

**Step 3: 写最小实现**
- DAG 节点：企业 → 行业 → 全局指标。
- 支持增量更新（delta 累加）。

**Step 4: 测试通过**
Run: `go test ./internal/service/calculator -run TestDagRecomputeOnRetailChange -v`
Expected: PASS

---

### Task 6: 解析输入 → Canonical 数据

**Files:**
- Modify: `internal/service/excel/parser.go`
- Create: `internal/service/excel/parser_test.go`

**Step 1: 写失败测试（解析主表与快照）**
```go
func TestParseCanonicalFromWorkbook(t *testing.T) {
    wb := buildWorkbookWithHeaders(...)
    result, err := excel.ParseCanonical(wb, excel.ParseOptions{Month: 12})
    require.NoError(t, err)
    require.NotEmpty(t, result.Companies)
}
```

**Step 2: 运行测试确认失败**
Run: `go test ./internal/service/excel -run TestParseCanonicalFromWorkbook -v`
Expected: FAIL

**Step 3: 写最小实现**
- 将“批发/零售/住宿/餐饮”主表映射为统一 CanonicalCompany。
- 处理单位规模、小微、吃穿用标记。

**Step 4: 测试通过**
Run: `go test ./internal/service/excel -run TestParseCanonicalFromWorkbook -v`
Expected: PASS

---

### Task 7: 模板导出（固定 1:1 输出）

**Files:**
- Create: `internal/service/excel/template_exporter.go`
- Create: `internal/service/excel/template_exporter_test.go`
- Modify: `internal/server/handlers/handlers.go`

**Step 1: 写失败测试（写入模板指定单元格）**
```go
func TestTemplateExporterWritesValues(t *testing.T) {
    tmpl := buildTemplateWorkbook(t)
    exporter := excel.NewTemplateExporter(tmpl)
    err := exporter.WriteSummary(...)
    require.NoError(t, err)
    assert.Equal(t, "63041.5", tmpl.GetCellValue("汇总表（定）", "G4"))
}
```

**Step 2: 运行测试确认失败**
Run: `go test ./internal/service/excel -run TestTemplateExporterWritesValues -v`
Expected: FAIL

**Step 3: 写最小实现**
- 加载模板（路径从 config 读取）
- 按固定坐标写入数据区与汇总区

**Step 4: 测试通过**
Run: `go test ./internal/service/excel -run TestTemplateExporterWritesValues -v`
Expected: PASS

---

### Task 8: UI 导入页容错识别 + 手工确认 + 月份选择

**Files:**
- Modify: `web/src/pages/ImportWizard.tsx`
- Modify: `web/src/store/importStore.ts`
- Modify: `web/src/types/index.ts`
- Modify: `web/src/services/api.ts`

**Step 1: 写失败测试（ImportWizard 显示识别结果）**
```tsx
it('renders recognition table', () => {
  render(<ImportWizard />)
  expect(screen.getByText('识别结果')).toBeInTheDocument()
})
```

**Step 2: 运行测试确认失败**
Run: `npm run test -- ImportWizard` (若新增 Vitest)
Expected: FAIL

**Step 3: 写最小实现**
- 增加识别结果表格、疑似 sheet 标记、人工确认下拉框
- 增加“操作月份”选择器

**Step 4: 测试通过**
Run: `npm run test -- ImportWizard`
Expected: PASS

---

### Task 9: 仪表盘归纳数据展示 + 明细表扩展

**Files:**
- Modify: `web/src/pages/Dashboard.tsx`
- Modify: `web/src/store/dataStore.ts`

**Step 1: 写失败测试（仪表盘显示归纳字段）**
```tsx
it('shows canonical fields in table', () => {
  render(<Dashboard />)
  expect(screen.getByText('本期零售额')).toBeInTheDocument()
})
```

**Step 2: 运行测试确认失败**
Run: `npm run test -- Dashboard`
Expected: FAIL

**Step 3: 写最小实现**
- 新增“归纳数据”表格（批零/住餐/吃穿用/小微视图）
- 指标区显示联动结果

**Step 4: 测试通过**
Run: `npm run test -- Dashboard`
Expected: PASS

---

### Task 10: 联动调整 + 规则校验

**Files:**
- Modify: `internal/service/calculator/adjuster.go`
- Create: `internal/service/calculator/rules.go`
- Create: `internal/service/calculator/rules_test.go`

**Step 1: 写失败测试（规则校验）**
```go
func TestRuleRetailNotExceedSales(t *testing.T) {
    errs := rules.ValidateCompany(c)
    assert.Contains(t, errs, "零售额不能超过销售额")
}
```

**Step 2: 运行测试确认失败**
Run: `go test ./internal/service/calculator -run TestRuleRetailNotExceedSales -v`
Expected: FAIL

**Step 3: 写最小实现**
- 实现规则引擎，返回 error/warn

**Step 4: 测试通过**
Run: `go test ./internal/service/calculator -run TestRuleRetailNotExceedSales -v`
Expected: PASS

---

### Task 11: 端到端导出校验（模板一致性）

**Files:**
- Modify: `tests/e2e/test_full_workflow.py`

**Step 1: 写失败测试（导出结构一致）**
```python
# Compare sheets/columns count with template
```

**Step 2: 运行测试确认失败**
Run: `pytest tests/e2e/test_full_workflow.py::test_export_matches_template -v`
Expected: FAIL

**Step 3: 写最小实现**
- 调整导出路径与模板写入

**Step 4: 测试通过**
Run: `pytest tests/e2e/test_full_workflow.py::test_export_matches_template -v`
Expected: PASS

---

## Testing Notes
- 当前 `go test ./...` 在缺少 `dist` 时失败；执行计划前应先 `make build` 或跳过该包测试。
- 若新增 Vitest，请在 `web/package.json` 添加 `test` 脚本。

