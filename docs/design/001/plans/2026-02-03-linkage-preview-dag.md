# 联动预览 + DAG 全覆盖 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 实现“点击位置→后端计算联动范围→前端黄色边框标记→修改后数据联动更新”，并让明细表全部字段、16 指标、汇总/社零额映射都纳入 DAG。

**Architecture:** 后端新增 DAG/坐标映射模块，提供联动影响范围计算与 UI/Excel 双坐标回传；前端为企业表与指标面板增加点击高亮；同时补齐“月值→累计值”的联动规则以保证口径一致。

**Tech Stack:** Go (Gin), SQLite, React + shadcn/ui, TypeScript

---

### Task 1: 梳理字段与坐标映射清单（UI + Excel）

**Files:**
- Create: `docs/linkage-dag-spec.md`

**Step 1: 写出 UI 字段全覆盖清单**
- 列出企业表 `ColumnKey` 全量字段、是否可编辑、公式来源。
- 列出 16 指标 ID 与含义。

**Step 2: 写出 Excel 坐标映射清单**
- 逐 sheet 说明企业字段对应列（参考 exporter 写入列）。
- 明确 UI-only 字段（如环比/同比增量）在 Excel 中无对应坐标。

**Step 3: 写出 DAG 依赖边**
- 按 “企业原始字段 → 企业衍生字段 → 行业汇总 → 指标 → 汇总/社零额” 列表化。

---

### Task 2: 新增后端 DAG + 坐标映射模块

**Files:**
- Create: `internal/linkage/types.go`
- Create: `internal/linkage/edges.go`
- Create: `internal/linkage/coords.go`

**Step 1: 写 failing 测试（节点/坐标映射）**
- Create: `internal/linkage/coords_test.go`
- 覆盖：WR/AC 行坐标可生成；汇总/指标节点返回 Excel/指标准确坐标。

**Step 2: 实现最小类型**
- `NodeID`、`UICoord`、`ExcelCoord`、`ImpactNode`。
- 映射 ColumnKey/indicatorId 到 NodeID 的解析器。

**Step 3: 实现 Excel 行序映射**
- 读取模板 C 列行业码 → 生成行业码→行序数组。
- 按 `row_no` 排序分配行号。
- 支持批发/零售/住宿/餐饮四张表。

**Step 4: 复跑测试**
- Run: `go test ./internal/linkage -v`
- Expected: PASS

---

### Task 3: 实现 DAG 影响范围计算（全量字段）

**Files:**
- Create: `internal/linkage/impact.go`
- Test: `internal/linkage/impact_test.go`

**Step 1: 写 failing 测试（影响范围闭包）**
- 点击 `wr:{id}:retailCurrentMonth` → 必须包含：本行同比/环比/零销比/当月增速、吃穿用/小微相关聚合、16 指标相关。
- 点击 `indicator:limitAbove_month_rate` → 必须包含：指标自身 + 其下游（社零总额）。

**Step 2: 实现 DAG 规则**
- 企业衍生字段：同比/环比/增速/零销比/累计增速等。
- AC 口径：`retail_current_month = food + goods`。
- 行业汇总与 16 指标：与 `calculator` 同口径。

**Step 3: 复跑测试**
- Run: `go test ./internal/linkage -v`
- Expected: PASS

---

### Task 4: 新增联动预览 API

**Files:**
- Modify: `internal/api/v3/handler.go`
- Create: `internal/api/v3/linkage_preview.go`
- Test: `internal/api/v3/linkage_preview_test.go`

**Step 1: 写 failing 测试（API）**
- POST `/api/linkage/preview` with UI anchor → 返回 nodes 列表。
- 返回 nodes 必须含 UI 坐标与 Excel 坐标。

**Step 2: 实现 Handler**
- 解析 `anchor`（支持 UI / indicator / Excel）。
- 调用 linkage 模块计算影响范围。

**Step 3: 复跑测试**
- Run: `go test ./internal/api/v3 -run LinkagePreview -v`
- Expected: PASS

---

### Task 5: 补齐“月值→累计值”联动规则

**Files:**
- Modify: `internal/api/v3/companies.go`
- Test: `internal/api/v3/companies_linkage_cumulative_test.go`

**Step 1: 写 failing 测试**
- 修改 `salesCurrentMonth` → `salesCurrentCumulative` 应基于 `sales_prev_cumulative + salesCurrentMonth` 更新（当累计未显式修改时）。
- 同理覆盖 `retailCurrentMonth`、AC `revenue/room/food/goods`。

**Step 2: 实现最小更新逻辑**
- 若 patch 未显式设置累计，则由 `prev_cumulative + current_month` 推导。
- 保留手动输入累计的优先级。

**Step 3: 复跑测试**
- Run: `go test ./internal/api/v3 -run LinkageCumulative -v`
- Expected: PASS

---

### Task 6: 前端高亮联动范围

**Files:**
- Modify: `web/src/components/CompaniesTable.tsx`
- Modify: `web/src/pages/DashboardV3.tsx`
- Modify: `web/src/services/api.ts`

**Step 1: 写 failing 前端测试（如果已有框架）**
- 若无现成测试基础，可跳过并在手动验证步骤说明。

**Step 2: 实现 API 调用**
- 新增 `linkageApi.preview()`。
- Dashboard 维护 `highlightNodes` 状态。

**Step 3: 实现 UI 高亮**
- 企业表 `TableCell` 绑定点击事件 → 请求 preview → 设置黄色边框。
- 指标面板输入框同样支持点击高亮。
- 点击空白处清除（document click 监听）。

---

### Task 7: 补齐算法文档

**Files:**
- Create: `docs/linkage-dag-algorithm.md`

**Step 1: 写出 DAG/坐标/影响范围算法**
- 说明节点命名规则、依赖边方向、BFS 传播。
- 说明 Excel 坐标映射与 UI-only 字段处理。

---

### Task 8: 验证

**Step 1: 后端测试**
- Run: `go test ./internal/linkage -v`
- Run: `go test ./internal/api/v3 -run Linkage -v`

**Step 2: 前端手动验证**
- 点击表格任意字段：黄色边框出现。
- 点击指标输入框：企业表相关字段高亮。

---

**执行说明（重要）：** 当前仓库禁止 git add/commit，本计划不包含提交步骤。
