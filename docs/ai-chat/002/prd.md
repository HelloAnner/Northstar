# ai-chat 模块 PRD-002

> 模块：AI 对话
> 版本：002
> 创建：2026-03-14
> 上级设计：docs/design/002/prd-002.md § 五、六 + docs/design/002/tech-002.md § 2.2 / 2.6 / 2.7 / 5.2

---

## 一、模块职责

负责 AI 对话能力的完整实现，包含提示词体系、意图解析、调整执行：

1. **提示词分层**：内置规则提示词（`system_prompt.go`，不可修改）+ 用户偏好提示词（config 表，可配置）
2. **用户偏好设置**：`/api/settings/user-prompt` CRUD + 前端 AI 偏好 Tab（`settings.go`）
3. **AI 对话接口扩展**：`/api/llm/chat` 新增 `mode` 字段，`mode=chat` 纯对话 / `mode=adjust` 触发调整
4. **意图解析**：`intent.go`，将用户输入解析为 `AdjustmentPlan`（结构化 JSON），再复用 `engine.Optimize` 执行
5. **前端 ChatPanel**：对话历史 + 模式切换 + appliedRules 气泡展示 + 执行后刷新指标

不负责：规则引擎的 Constraint 执行（→ rule-engine 模块）、规则的 CRUD 管理（→ rule-management 模块）。

---

## 二、详细设计

> 待补充
