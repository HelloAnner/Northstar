# 撤销能力 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 在会话内支持多步撤销（撤销上一步操作），替换原“重置”按钮。

**Architecture:** 前端维护撤销差异栈；每次用户动作生成字段级 diff；撤销时调用新的批量更新接口一次性回写并重算指标。

**Tech Stack:** Go + Gin + SQLite；React + Zustand + shadcn/ui。

---

### Task 1: 后端批量更新接口（含 AC 累计字段支持）

**Files:**
- Modify: `internal/api/v3/handler.go`
- Modify: `internal/api/v3/companies.go`
- Test: `internal/api/v3/companies_batch_update_test.go`

**Step 1: Write the failing test**

```go
func TestBatchUpdateCompanies_UpdatesWRAndAC(t *testing.T) {
    // 期望 /api/companies/batch 能批量更新 wr + ac，并在 RecalcAll 后反映到数据库
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/api/v3 -run TestBatchUpdateCompanies_UpdatesWRAndAC -v`
Expected: FAIL（404 或 handler 不存在）

**Step 3: Write minimal implementation**

- 新增 `PATCH /api/companies/batch` 路由
- 解析请求 `{ updates: [{ id, patch }] }`
- 按 id 类型调用 `pickWRUpdates/pickACUpdates` + `build*RateDrivenUpdates` + `apply*CumulativeUpdates`
- 逐条更新后仅 `RecalcAll` 一次，返回 `groups`
- `pickACUpdates` 允许 `room/food/goods` 的 `*_current_cumulative`

**Step 4: Run test to verify it passes**

Run: `go test ./internal/api/v3 -run TestBatchUpdateCompanies_UpdatesWRAndAC -v`
Expected: PASS

**Step 5: Commit**

跳过（项目要求禁止 git add/commit）。

---

### Task 2: 前端撤销 diff 工具 + 状态栈

**Files:**
- Create: `web/src/store/undoStore.ts`
- Create: `web/src/lib/undo.ts`
- Test: `web/src/lib/undo.test.ts`
- Modify: `web/package.json` (若需加入测试脚本)

**Step 1: Write the failing test**

```ts
import { buildUndoChanges } from '@/lib/undo'

test('buildUndoChanges only returns changed fields', () => {
  const before = { 'wr:1': { id: 'wr:1', kind: 'wr', salesCurrentMonth: 100 } }
  const after = { 'wr:1': { id: 'wr:1', kind: 'wr', salesCurrentMonth: 120 } }
  const changes = buildUndoChanges(before, after)
  expect(changes).toHaveLength(1)
  expect(changes[0].fields.salesCurrentMonth.before).toBe(100)
  expect(changes[0].fields.salesCurrentMonth.after).toBe(120)
})
```

**Step 2: Run test to verify it fails**

Run: `cd web && npm test -- --runTestsByPath src/lib/undo.test.ts`
Expected: FAIL（函数不存在或测试框架未配置）

**Step 3: Write minimal implementation**

- `buildUndoChanges` 根据 `kind` 选择字段集并生成 diff
- `undoStore` 维护 `stack`、`push`、`pop`、`clear`

**Step 4: Run test to verify it passes**

Run: `cd web && npm test -- --runTestsByPath src/lib/undo.test.ts`
Expected: PASS

**Step 5: Commit**

跳过（项目要求禁止 git add/commit）。

---

### Task 3: 前端集成撤销流程

**Files:**
- Modify: `web/src/pages/DashboardV3.tsx`
- Modify: `web/src/components/CompaniesTable.tsx`
- Modify: `web/src/services/api.ts`
- Modify: `web/src/pages/HelpDocument.tsx`

**Step 1: Write the failing test**

（若有前端测试框架）补充 UI 交互测试；否则跳过自动化测试，使用人工回归记录。

**Step 2: Run test to verify it fails**

略。

**Step 3: Write minimal implementation**

- 顶部按钮改为“撤销”，禁用条件为 `stack 为空`
- 单元格编辑成功后生成 diff 并入栈
- 智能调整/单项调整：前后拉取快照生成 diff 入栈
- 撤销调用 `/api/companies/batch` 反向回写并刷新数据
- 导入成功/月份切换时清空撤销栈
- 移除“重置”说明文案

**Step 4: Run test to verify it passes**

略。

**Step 5: Commit**

跳过（项目要求禁止 git add/commit）。

---

Plan complete and saved to `docs/plans/2026-02-04-undo-implementation-plan.md`. I will proceed in this session per user request (no additional prompts). Note: required sub-skill `superpowers:executing-plans` / `subagent-driven-development` is unavailable in this environment, so I will execute manually with the same step order.
