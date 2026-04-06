# AI 对话模块

> 模块：ai-chat
> ��态：已完成
> ���后更新：2026-04-06

---

## 一、模块职责

1. **提示词分层**：内置提示词 + 用户偏好提示词 + 自然语言规则上下文
2. **用户偏好设置**：`/api/settings/user-prompt` + 前端 AI 偏好 Tab
3. **AI 对话接口**：`/api/llm/chat/stream`（SSE），支持 mode=chat / mode=adjust
4. **意图解析**：`intent.go`，支持 set_target / adjust_percent / add_rule 三种意图
5. **前端 ChatPanel**：对话历史 + 模式切换 + appliedRules 气泡 + 规则添加状态

不负责：规则引擎执行（→ rule-engine）、规则 CRUD 管理（→ rule-management）。

---

## 二、核心文件

| 文件 | 职责 |
|------|------|
| `internal/llm/system_prompt.go` | 三层提示词拼接（内置 + 自然语言规则 + 用户偏好） |
| `internal/llm/intent.go` | ParseIntent，三种 action type |
| `internal/llm/tools.go` | LLM 工具定义 |
| `internal/api/v3/llm_chat.go` | SSE 流式对话，mode 分流，adjust 执行 |
| `internal/api/v3/settings.go` | 用户偏好提示词 API |
| `web/src/components/ChatPanel.tsx` | 对话 UI |
| `web/src/components/AIPreferenceForm.tsx` | AI 偏好设置 |

---

## 三、三层提示词

### 内置层（不可修改）

代码常量，动态填充 `{year}`、`{month}`、`{constraintCount}`、`{indicatorSummary}`：

- 系统角色定义
- 四大行业指标上下文
- 行为约束（不模糊、不编造）

### 自然语言规则层（动态注入）

从 `natural_rules` 表加载 enabled 的规则，拼接到系统提示词中：

```
---
用户定义的调整规则（请在调整时遵守）：
1. 如果零售连续两个月下降，优先调整大企业
2. 批发增速尽量保持稳定
```

LLM 在做 Function Call 调整时参考这些规则，但不做强制校验（由 LLM 自主判断）。

### 用户层（可配置）

config 表 `llm_user_prompt`，最大 500 字符。

拼接规则：内置层 + 自然语言规则层 + 用户层。

---

## 四、意图解析

### 三种 action type

| type | 字段 | 说明 |
|------|------|------|
| `set_target` | indicatorId + value | 设定绝对目标值 |
| `adjust_percent` | indicatorId + percent | 基于当前值的相对调整 |
| `add_rule` | ruleText | 添加自然语言规则（直接存 DB，无需转换） |

### add_rule 变更

- **旧流程**：写入 rules.md → 触发 LLM 异步转换 → 轮询状态
- **新流程**：直接写入 `natural_rules` 表 → 即时生效（下次对话自动读取）

---

## 五、mode=adjust 执行流程

```
1. 提取最后一条用户消息
2. ParseIntent → AdjustmentPlan
3. 若 actions 为空 → 降级 runChatMode
4. splitActions：分离 targets 和 ruleActions
5. 处理 add_rule → 写入 natural_rules 表（即时生效）
6. 处理 targets → engine.Optimize()
7. LLM 生成自然语言总结
8. SSE 返回 result 事件
```

---

## 六、SSE 事件类型

| type | 说明 |
|------|------|
| `message_delta` | 流式文本增量 |
| `result` | 最终结果（mode/reply/groups/appliedRules/ruleAdded） |
| `error` | 错误信息 |
| `final` | 流结束标记 |

---

## 七、API

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/llm/chat/stream` | SSE 对话（body: {mode, messages}） |
| GET | `/api/settings/user-prompt` | 获取偏好提��词 |
| PUT | `/api/settings/user-prompt` | 更新偏好提示词（500 字限制） |
