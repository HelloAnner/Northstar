# ai-chat 模块 E2E-002

> 模块：AI 对话
> 版本：002
> 创建：2026-03-14

---

## 一、验收目标

验证 AI 对话模块已经打通 `双层提示词 -> 意图解析 -> 智能调整 -> 自然语言总结 -> 前端抽屉交互` 的完整闭环。

## 二、验收场景

### 场景 1：Settings 页面可读写 AI 偏好提示词

前置条件：

- 服务已启动
- config 表已存在 `llm_user_prompt`

步骤：

1. 打开 `/settings`
2. 切换到「AI 偏好」Tab
3. 输入一段不超过 500 字的偏好内容并保存
4. 刷新页面
5. 调用 `GET /api/settings/user-prompt`

期望：

- 页面展示当前提示词内容
- 保存成功后刷新仍能回显
- 接口返回内容与页面一致
- 超过 500 字时前后端都拒绝保存

### 场景 2：chat 模式使用双层提示词返回普通问答

前置条件：

- LLM 配置完整
- `llm_user_prompt` 已写入示例偏好

步骤：

1. 调用 `/api/llm/chat/stream`，传入 `mode=chat`
2. 发送一条咨询型问题，例如“解释当前批发增速含义”
3. 读取 SSE 返回内容

期望：

- 系统正常流式返回回复
- 构造给 LLM 的系统提示词同时包含内置层和用户偏好层
- 响应不触发 `engine.Optimize()`
- 返回仅包含文本回复，不包含调整结果

### 场景 3：adjust 模式识别到调整意图并执行优化

前置条件：

- 当前月份已有可计算指标数据
- role.json 已存在有效规则
- LLM 意图解析返回合法 `AdjustmentPlan`

步骤：

1. 调用 `/api/llm/chat/stream`，传入 `mode=adjust`
2. 发送“把批发当月增速调到 15%”
3. 读取 SSE 结果
4. 再查询 `/api/indicators`

期望：

- `ParseIntent` 返回 `wholesale_month_rate=15`
- 后端调用 `engine.Optimize()`
- SSE 最终事件包含 `reply`、`groups`、`appliedRules`
- 指标结果已更新到目标值或规则裁剪后的目标值

### 场景 4：adjust 模式遇到纯咨询自动降级为 chat

前置条件：

- `mode=adjust`
- LLM 意图解析返回 `{"actions":[]}`

步骤：

1. 发送“当前零售增速为什么偏低？”
2. 读取 SSE 返回
3. 再查询 `/api/indicators`

期望：

- 后端不调用 `engine.Optimize()`
- 返回普通文本回复
- 指标数据不发生变化

### 场景 5：前端 ChatPanel 展示规则生效气泡并刷新 Dashboard

前置条件：

- Dashboard 已加载
- ChatPanel 可以正常打开
- 调整请求会返回至少一条 `appliedRules`

步骤：

1. 打开右侧 ChatPanel
2. 切换到「调整」模式
3. 发送一条带规则干预的请求
4. 观察回复区的规则气泡
5. 观察指标卡片和明细表

期望：

- `clamp_target`、`filter_allocation`、`compensate` 三类规则均能展示可读文案
- 调整完成后 Dashboard 指标区自动刷新
- 对话历史保留本轮用户消息和 AI 回复

## 三、通过标准

- 上述 5 个场景全部通过
- 新增后端单元测试、前端测试通过
- `go test -tags=e2e` 的 Phase 3 用例通过
- 不破坏 Phase 1/2 已完成的规则引擎与规则管理闭环

## 四、自动化执行

- 后端闭环验收：`go test -tags=e2e -v ./tests/e2e -run TestAIChatPhase3E2E`
- 后端单测：`go test ./internal/llm ./internal/api/v3 ./internal/server`
- 前端测试：`cd web && npm test -- --run`
