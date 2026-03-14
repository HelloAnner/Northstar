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

### 2.1 双层提示词

系统提示词由两层组成：

1. **内置层**：代码常量维护，定义角色、能力边界、指标上下文、输出约束，不允许在前端修改
2. **用户层**：保存在 config 表 `llm_user_prompt`，由 `/settings` 页面「AI 偏好」Tab 维护

拼接规则：

- 先输出内置层
- `userPrompt` 非空时，以 `---` 分隔后再追加用户层
- 内置层必须始终生效，用户层只能补充风格偏好和回答习惯，不能覆盖系统职责

### 2.2 模式分流

`/api/llm/chat/stream` 新增 `mode` 字段：

- `mode=chat`：普通咨询，AI 仅基于当前指标做解释、分析、建议
- `mode=adjust`：先解析用户意图，识别为结构化 `AdjustmentPlan` 后执行调整，再生成自然语言总结
- `mode=""`：兼容旧调用，按 `chat` 处理

纯咨询语句即使落在 `adjust` 模式，也允许自动降级回 `chat`，避免用户因为模式切换错误而得到空结果。

### 2.3 意图解析

`intent.go` 负责把自然语言解析为：

```json
{
  "actions": [
    { "type": "set_target", "indicatorId": "wholesale_month_rate", "value": 15 }
  ]
}
```

约束：

- `indicatorId` 必须是 16 项指标中的合法 ID
- `value` 必须是数字
- 暂不支持企业级 patch；Phase 3 只支持指标目标调整
- 无调整意图时返回 `{"actions":[]}`，不视为错误

### 2.4 调整执行

`mode=adjust` 执行流程：

1. 查询当前年月与指标数据
2. 调用 `ParseIntent`
3. 若 `actions` 为空，退回普通 `chat`
4. 将 action 列表转成 `targets map[string]float64`
5. 调用 `engine.Optimize()`
6. 收集 `groups` 与 `appliedRules`
7. 再调用一次 LLM，把“用户原请求 + 实际执行结果 + 规则生效情况”整理成用户可读回复

同一轮请求内多个 action 允许一次性传给 `engine.Optimize()`，由引擎按既定顺序执行。

### 2.5 前端交互

Dashboard 右下角入口升级为 `ChatPanel`：

- 右侧抽屉宽度固定约 400px
- 顶部可切换 `聊天` / `调整`
- 消息区显示用户消息、AI 回复、加载中状态
- 调整模式额外展示 `appliedRules` 气泡
- 调整完成后自动刷新指标卡片和表格

Settings 页新增「AI 偏好」Tab：

- textarea 输入
- 实时字数统计
- 最大 500 字
- 保存后提示成功并持久化到 config 表

### 2.6 非目标范围

本阶段明确不做：

- 企业字段级别的 LLM 工具调用编辑
- 多轮 function call 链式执行
- AI 自动改写 `rules.md`
- 历史会话持久化到数据库
