# AI Chat Phase 3 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 实现 Phase 3 AI 对话闭环，覆盖双层提示词、意图解析、调整模式执行、Settings AI 偏好页和 Dashboard ChatPanel。

**Architecture:** 后端沿用现有 `llm.Client` 与 SSE 输出层，在 `internal/llm` 新增 prompt 与 intent 编排，在 `api/v3/llm_chat.go` 中按 `mode=chat|adjust` 分流；前端复用现有对话入口，升级为独立 `ChatPanel` 组件，并在 `/settings` 中开放 AI 偏好配置。整个过程按 TDD 推进，先补后端行为测试，再接前端视图与 Phase 3 e2e。

**Tech Stack:** Go, Gin, SQLite, React, Vite, Zustand, shadcn/ui

---

### Task 1: 补齐 ai-chat 模块文档

**Files:**
- Modify: `docs/ai-chat/002/prd.md`
- Create: `docs/ai-chat/002/e2e.md`
- Create: `docs/ai-chat/002/plans/2026-03-14-phase-3-ai-chat.md`

**Step 1: 写详细 PRD**

- 明确双层提示词、模式分流、意图解析、调整执行、前端交互边界

**Step 2: 写 E2E 验收**

- 覆盖 AI 偏好保存、chat 模式、adjust 模式、adjust 降级 chat、前端规则气泡刷新

**Step 3: 写实现计划**

- 拆成 prompt/intent、settings API、chat handler、前端 ChatPanel、Phase 3 e2e 五部分

### Task 2: 先写 llm prompt 与 intent 失败测试

**Files:**
- Create: `internal/llm/system_prompt_test.go`
- Create: `internal/llm/intent_test.go`
- Modify: `internal/llm/client.go`

**Step 1: Write the failing test**

- `BuildChatSystemPrompt` 覆盖内置层、用户层、年月、规则数量、指标摘要
- `ParseIntent` 覆盖合法 JSON、空 actions、非法 JSON、非法 indicatorId

**Step 2: Run test to verify it fails**

Run: `go test ./internal/llm -run 'Test(BuildChatSystemPrompt|ParseIntent)'`
Expected: FAIL，因为新文件和能力尚未实现

**Step 3: Write minimal implementation**

- 新建 `internal/llm/system_prompt.go`
- 新建 `internal/llm/intent.go`
- 让 `llm.Client` 支持不带工具的普通结构化调用

**Step 4: Run test to verify it passes**

Run: `go test ./internal/llm -run 'Test(BuildChatSystemPrompt|ParseIntent)'`
Expected: PASS

### Task 3: 为 settings API 与 llm chat 分流写失败测试

**Files:**
- Create: `internal/api/v3/settings_test.go`
- Modify: `internal/api/v3/llm_chat.go`
- Create: `internal/api/v3/llm_chat_phase3_test.go`
- Modify: `internal/api/v3/handler.go`

**Step 1: Write the failing test**

- `GET/PUT /api/settings/user-prompt` 正常读写与 500 字校验
- `mode=chat` 使用双层 prompt
- `mode=adjust` 能调用 `ParseIntent` 与 `engine.Optimize()`
- `actions=[]` 时自动降级为 chat

**Step 2: Run test to verify it fails**

Run: `go test ./internal/api/v3 -run 'Test(UserPrompt|LLMChatPhase3)'`
Expected: FAIL，因为新路由和新编排尚未实现

**Step 3: Write minimal implementation**

- 新建 `internal/api/v3/settings.go`
- 扩展 `RegisterRoutes`
- 重构 `internal/api/v3/llm_chat.go` 为 chat/adjust 双分支

**Step 4: Run test to verify it passes**

Run: `go test ./internal/api/v3 -run 'Test(UserPrompt|LLMChatPhase3)'`
Expected: PASS

### Task 4: 实现前端 ChatPanel 与 Settings AI 偏好页

**Files:**
- Create: `web/src/components/ChatPanel.tsx`
- Create: `web/src/components/ChatPanel.test.tsx`
- Modify: `web/src/pages/DashboardV3.tsx`
- Modify: `web/src/pages/Settings.tsx`
- Modify: `web/src/services/api.ts`
- Create: `web/src/store/chatStore.ts`（如需要）

**Step 1: Write the failing test**

- ChatPanel 展示消息、模式切换、规则气泡
- Settings AI 偏好页加载与保存字数限制

**Step 2: Run test to verify it fails**

Run: `cd web && npm test -- ChatPanel.test.tsx --run`
Expected: FAIL，因为新组件与 API 尚未接通

**Step 3: Write minimal implementation**

- 用 shadcn 组件完成右侧抽屉对话面板
- 替换旧 `LlmChatDialog`
- 打通 AI 偏好接口

**Step 4: Run test to verify it passes**

Run: `cd web && npm test -- ChatPanel.test.tsx --run`
Expected: PASS

### Task 5: 补齐 Phase 3 e2e 与总计划状态

**Files:**
- Create: `tests/e2e/ai_chat_phase3_test.go`
- Modify: `docs/design/002/plan-002.md`

**Step 1: Write the failing test**

- 覆盖 chat 模式、adjust 模式、降级路径、Settings AI 偏好保存

**Step 2: Run test to verify it fails**

Run: `go test -tags=e2e ./tests/e2e -run TestAIChatPhase3E2E`
Expected: FAIL，因为 Phase 3 闭环尚未全部打通

**Step 3: Write minimal implementation**

- 接通后端与前端直至 e2e 通过
- 完成后将 `docs/design/002/plan-002.md` 的 Phase 3 标记为完成

**Step 4: Run full verification**

Run: `go test ./internal/llm ./internal/api/v3 ./internal/server`
Expected: PASS

Run: `cd web && npm test -- --run`
Expected: PASS

Run: `go test -tags=e2e ./tests/e2e -run TestAIChatPhase3E2E`
Expected: PASS
