# Rule Engine Phase 1 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 实现 role.json 规则加载与执行闭环，使 `/api/optimize` 可返回规则生效明细。

**Architecture:** 新增 `internal/rules` 负责规则结构和过滤逻辑，`dagcalc.Engine` 持有可热重载的 `RuleSet`，`ApplyIndicatorTarget` 扩展为支持 clamp/filter/compensate 的统一入口。`api/v3/optimize` 不再直接调裸函数，而是通过引擎统一执行并输出 `appliedRules`。

**Tech Stack:** Go, Gin, SQLite, 现有 dagcalc 引擎

---

### Task 1: 补齐规则包测试骨架

**Files:**
- Create: `internal/rules/loader_test.go`
- Create: `internal/rules/constraint_test.go`
- Create: `internal/rules/filter_test.go`

**Step 1: Write the failing test**

- 覆盖 `role.json` 三类规则分发
- 覆盖 `Clamp` / `Apply` / `Check`
- 覆盖未知 filter 降级

**Step 2: Run test to verify it fails**

Run: `go test ./internal/rules`
Expected: FAIL，因为 `internal/rules` 尚不存在

**Step 3: Write minimal implementation**

- 新建 `loader.go`
- 新建 `constraint.go`
- 新建 `filter.go`

**Step 4: Run test to verify it passes**

Run: `go test ./internal/rules`
Expected: PASS

### Task 2: 为 dagcalc 规则干预点补测试

**Files:**
- Create: `internal/dagcalc/adjust_rules_test.go`
- Modify: `internal/dagcalc/engine_test.go`

**Step 1: Write the failing test**

- 验证 clamp 会裁剪目标
- 验证 filter 会减少参与企业数
- 验证 compensate 会触发一次补偿
- 验证 `depth=1` 时不再继续补偿

**Step 2: Run test to verify it fails**

Run: `go test ./internal/dagcalc -run 'TestApplyIndicatorTargetWithRules|TestEngineOptimize'`
Expected: FAIL，因为新接口和行为尚未实现

**Step 3: Write minimal implementation**

- 扩展 `ApplyIndicatorTarget`
- 扩展 `Engine`
- 增加 `AppliedRule`

**Step 4: Run test to verify it passes**

Run: `go test ./internal/dagcalc -run 'TestApplyIndicatorTargetWithRules|TestEngineOptimize'`
Expected: PASS

### Task 3: 接入 optimize API

**Files:**
- Modify: `internal/api/v3/handler.go`
- Modify: `internal/api/v3/optimize.go`
- Modify: `internal/api/v3/optimize_test.go`
- Modify: `internal/server/server.go`

**Step 1: Write the failing test**

- 验证 `/api/optimize` 通过 `engine.Optimize()`
- 验证响应包含 `appliedRules`

**Step 2: Run test to verify it fails**

Run: `go test ./internal/api/v3 -run TestOptimize`
Expected: FAIL，因为响应结构和调用链尚未更新

**Step 3: Write minimal implementation**

- `Handler` 持有 `dagcalc.Engine`
- `Optimize` 使用引擎
- 返回 `appliedRules`

**Step 4: Run test to verify it passes**

Run: `go test ./internal/api/v3 -run TestOptimize`
Expected: PASS

### Task 4: 回归验证

**Files:**
- Modify: `docs/design/002/plan-002.md`

**Step 1: Run focused tests**

Run: `go test ./internal/rules ./internal/dagcalc ./internal/api/v3`
Expected: PASS

**Step 2: Run broader regression**

Run: `go test ./...`
Expected: PASS，若有与现有数据样例相关的非本次问题，记录实际阻塞点

**Step 3: Update plan status**

- 将 `docs/design/002/plan-002.md` 的 Phase 1 标记为完成
