# Rule Management Phase 2 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 实现规则管理完整生命周期，覆盖 `rules.md` CRUD、LLM 转换、热重载、Settings 页面规则管理与 Phase 2 e2e。

**Architecture:** 后端以 `internal/rules/converter.go` 和 `api/v3/rules.go` 打通文件、状态、转换和热重载闭环；前端在 `/settings` 中新增规则管理页，使用独立 store 维护规则列表和状态轮询。实现过程严格按 TDD 分阶段推进，先后端闭环，再接前端。

**Tech Stack:** Go, Gin, SQLite, React, Vite, Zustand, shadcn/ui

---

### Task 1: 补齐模块文档

**Files:**
- Create: `docs/rule-management/002/e2e.md`
- Create: `docs/rule-management/002/plans/2026-03-14-phase-2-rule-management.md`

**Step 1: 写验收文档**

- 明确 Phase 2 的 API、文件系统、转换状态和前端交互验收场景

**Step 2: 写实现计划**

- 拆为后端转换、API、前端页面、e2e 四个任务块

### Task 2: 先写 converter 与 rules API 的失败测试

**Files:**
- Create: `internal/rules/converter_test.go`
- Create: `internal/api/v3/rules_test.go`

**Step 1: Write the failing test**

- `extractJSON` 代码块提取、裸 JSON 提取、非法输出报错
- `validateRoleJSON` 对未知 type、非法 indicator/filter/relation、空 min/max 返回结构化错误
- `/api/rules` CRUD、`/api/rules/status`、`/api/rules/convert` 正常路径

**Step 2: Run test to verify it fails**

Run: `go test ./internal/rules ./internal/api/v3 -run 'Test(ExtractJSON|ValidateRoleJSON|Rules)'`
Expected: FAIL，因为实现尚不存在

**Step 3: Write minimal implementation**

- 新建 `internal/rules/converter.go`
- 新建 `internal/api/v3/rules.go`
- 扩展 `internal/api/v3/handler.go`

**Step 4: Run test to verify it passes**

Run: `go test ./internal/rules ./internal/api/v3 -run 'Test(ExtractJSON|ValidateRoleJSON|Rules)'`
Expected: PASS

### Task 3: 接入服务初始化与转换热重载

**Files:**
- Modify: `internal/server/server.go`
- Modify: `internal/store/config.go`
- Test: `internal/server/server_test.go`

**Step 1: Write the failing test**

- 启动时自动创建 `config/rules.md`
- 注入 `rules_convert_status / at / error / llm_user_prompt` 默认值

**Step 2: Run test to verify it fails**

Run: `go test ./internal/server -run TestNewServer`
Expected: FAIL，因为初始化逻辑未实现

**Step 3: Write minimal implementation**

- 启动时初始化规则文件和配置默认值
- 保持 `role.json` 缺失时降级运行

**Step 4: Run test to verify it passes**

Run: `go test ./internal/server -run TestNewServer`
Expected: PASS

### Task 4: 实现前端规则管理页面

**Files:**
- Create: `web/src/pages/Settings.tsx`
- Create: `web/src/components/RuleList.tsx`
- Create: `web/src/store/rulesStore.ts`
- Modify: `web/src/services/api.ts`
- Modify: `web/src/App.tsx`
- Test: `web/src/components/RuleList.test.tsx`

**Step 1: Write the failing test**

- `RuleList` 能加载规则、展示状态、执行新增/编辑/删除
- `running` 时开启轮询，`ok/error` 后停止

**Step 2: Run test to verify it fails**

Run: `cd web && npm test -- RuleList.test.tsx --run`
Expected: FAIL，因为组件和 store 尚不存在

**Step 3: Write minimal implementation**

- 扩展 API client
- 实现 Zustand store
- 用 shadcn 组件完成 Settings 页面与 RuleList

**Step 4: Run test to verify it passes**

Run: `cd web && npm test -- RuleList.test.tsx --run`
Expected: PASS

### Task 5: 补齐 Phase 2 e2e 与总计划状态

**Files:**
- Create: `tests/e2e/rule_management_phase2_test.go`
- Modify: `docs/design/002/plan-002.md`

**Step 1: Write the failing test**

- 覆盖 rules 文件初始化、CRUD、状态流转、失败保留旧 role.json

**Step 2: Run test to verify it fails**

Run: `go test -tags=e2e ./tests/e2e -run TestRuleManagementPhase2E2E`
Expected: FAIL，因为用例与实现尚未补齐

**Step 3: Write minimal implementation**

- 接通后端闭环直到 e2e 通过
- 完成后将 `docs/design/002/plan-002.md` 的 Phase 2 标为完成

**Step 4: Run full verification**

Run: `go test ./internal/rules ./internal/api/v3 ./internal/server`
Expected: PASS

Run: `cd web && npm test -- --run`
Expected: PASS

Run: `go test -tags=e2e ./tests/e2e -run TestRuleManagementPhase2E2E`
Expected: PASS
