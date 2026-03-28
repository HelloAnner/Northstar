# AI 对话模块

> 模块：ai-chat
> 状态：已完成
> 最后更新：2026-03-28

---

## 一、模块职责

1. **提示词分层**：内置规则提示词（代码常量）+ 用户偏好提示词（config 表可配置）
2. **用户偏好设置**：`/api/settings/user-prompt` + 前端 AI 偏好 Tab
3. **AI 对话接口**：`/api/llm/chat/stream`（SSE），支持 mode=chat / mode=adjust
4. **意图解析**：`intent.go`，支持 set_target / adjust_percent / add_rule 三种意图
5. **前端 ChatPanel**：对话历史 + 模式切换 + appliedRules 气泡 + 规则添加状态

不负责：规则引擎执行（→ rule-engine）、规则 CRUD 管理（→ rule-management）。

---

## 二、核心文件

| 文件 | 职责 |
|------|------|
| `internal/llm/system_prompt.go` | BuildChatSystemPrompt，双层拼接 |
| `internal/llm/intent.go` | ParseIntent，三种 action type |
| `internal/llm/tools.go` | LLM 工具定义（3 个 tool） |
| `internal/api/v3/llm_chat.go` | SSE 流式对话，mode 分流，adjust 执行 |
| `internal/api/v3/settings.go` | 用户偏好提示词 API |
| `web/src/components/ChatPanel.tsx` | 对话 UI |
| `web/src/components/AIPreferenceForm.tsx` | AI 偏好设置 |

---

## 三、双层提示词

### 内置层（不可修改）

代码常量，动态填充 `{year}`、`{month}`、`{ruleCount}`、`{indicatorSummary}`：

- 系统角色定义
- 四大行业指标上下文
- 行为约束（不模糊、不编造）

### 用户层（可配置）

config 表 `llm_user_prompt`，最大 500 字符。

拼接规则：内置层 + `---` + 用户层（为空时不拼接）。

---

## 四、意图解析

### 三种 action type

| type | 字段 | 说明 |
|------|------|------|
| `set_target` | indicatorId + value | 设定绝对目标值 |
| `adjust_percent` | indicatorId + percent | 基于当前值的相对调整 |
| `add_rule` | ruleText | 添加持久约束规则 |

### 判断规则

- "调到 X"、"设为 X" → `set_target`
- "调整 X%"、"增加/减少 X%" → `adjust_percent`
- "不能超过"、"限制"、"帮我加规则" → `add_rule`
- 永久约束 vs 一次性调整的区分

### 校验

- `set_target` / `adjust_percent`：indicatorId 必须在 16 项枚举中
- `add_rule`：ruleText 不能为空
- 无调整意图 → 返回空 actions → 降级为 chat 模式

---

## 五、mode=adjust 执行流程

```
1. 提取最后一条用户消息
2. ParseIntent → AdjustmentPlan
3. 若 actions 为空 → 降级 runChatMode
4. splitActions：分离 targets（set_target + adjust_percent→绝对值）和 ruleActions（add_rule）
5. 处理 add_rule → addRuleFromChat()
6. 处理 targets → engine.Optimize()
7. 仅有规则操作 → buildRuleOnlyResult()
8. LLM 生成自然语言总结
9. SSE 返回 result 事件（含 reply + groups + appliedRules + ruleAdded）
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

## 七、前端交互

### ChatPanel

- Dashboard 右侧抽屉（420px），浮动按钮触发
- 模式切换 Tab：聊天/调整
- 默认快捷问题：
  - 聊天：解释批发增速、分析零售走势
  - 调整：调整批发增速、随机调整零售、添加规则
- SSE 流式消息渲染 + 光标动画
- appliedRules 气泡：clamp/filter/compensate 三种格式
- ruleAdded 状态卡片（BookPlus 图标 + 转换状态）
- 调整完成后自动刷新指标和表格
- 规则添加 Toast 通知

### AIPreferenceForm

- textarea + 字数统计（xx/500）
- 超出 500 禁用保存
- 保存成功 Toast

---

## 八、API

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/llm/chat/stream` | SSE 对话（body: {mode, messages}） |
| GET | `/api/settings/user-prompt` | 获取偏好提示词 |
| PUT | `/api/settings/user-prompt` | 更新偏好提示词（500 字限制） |

---

## 九、测试覆盖

| 测试文件 | 覆盖 |
|---------|------|
| `llm/intent_test.go` | 意图提取、空 actions、非法 indicator、alias 兼容 |
| `llm/system_prompt_test.go` | 双层拼接、空用户层 |
| `llm/tools_test.go` | 3 个工具定义完整性 |
| `api/v3/llm_chat_test.go` | chat/adjust 模式、降级 |
| `web/src/components/ChatPanel.test.tsx` | 渲染、appliedRules 气泡 |
| `web/src/components/AIPreferenceForm.test.tsx` | 字数统计、保存 |
