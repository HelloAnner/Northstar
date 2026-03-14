# Northstar PLAN-002：规则引擎统一 + AI 对话模块实施计划

> 对应：PRD-002 + TECH-002
> 创建：2026-03-14

---

## 模块划分

| 模块 | 职责范围 | 文档 |
|------|---------|------|
| rule-engine | role.json 加载 + Constraint 类型 + dagcalc 三干预点 | docs/rule-engine/002/ |
| rule-management | 规则 CRUD API + rules.md 文件管理 + LLM 转换 + 前端 RuleList | docs/rule-management/002/ |
| ai-chat | 双层提示词 + 意图解析 + 前端 ChatPanel + mode=adjust | docs/ai-chat/002/ |

> `rule-convert` 与 `rule-management` 耦合极高（每次 CRUD 均触发转换），合并为 rule-management 统一实现。

---

## Phase 1：rule-engine

**目标**：建立规则引擎核心后端，使 role.json 能驱动 DAG 调整算法。

### 实现任务

**后端 internal/rules/**

- [x] `loader.go`：定义 `RuleSet`、`rawRule`、`rawRoleJSON` 结构体；实现 `Load(path string) (*RuleSet, error)`，role.json 不存在时返回空 RuleSet 不报错
- [x] `constraint.go`：实现三种 Constraint 类型及方法
  - `ClampTargetConstraint`：`Clamp(indicatorID, target)` → `(float64, bool)`
  - `FilterAllocationConstraint`：`Apply(indicatorID, companies)` → `([]CompanyRow, bool)`
  - `CompensateConstraint`：`Check(triggerID, indicators)` → `(bool, float64)`
- [x] `filter.go`：`filterByMode(companies, mode)` 支持 4 个枚举值，未知 mode 不过滤

**后端 dagcalc/**

- [x] `engine.go`：`Engine` 新增 `rules *rules.RuleSet` + `mu sync.RWMutex` + `rulePath string`；实现 `ReloadRules()`、`getRules()`、`Optimize()`
- [x] `adjust.go`：扩展 `ApplyIndicatorTarget` 函数签名（新增 `rs *rules.RuleSet`、`depth int` 参数）；实现三个干预点（ClampTarget → FilterAllocation → Compensate）；定义 `AppliedRule` 结构体

**后端 api/v3/optimize.go**

- [x] Handler 改调 `engine.Optimize()`（不直接调用 `ApplyIndicatorTarget`）
- [x] 响应新增 `appliedRules []AppliedRule`

### 单元测试

- [x] `internal/rules/loader_test.go`：各 type 分发正确；不存在文件返回空 RuleSet
- [x] `internal/rules/constraint_test.go`：ClampTarget 边界值；Filter 各枚举；Compensate gte/lte 两向逻辑
- [x] `internal/rules/filter_test.go`：filterByMode 各模式；未知 mode 不过滤
- [x] `dagcalc/adjust_rules_test.go`：ClampTarget 生效；Filter 过滤减少企业数；Compensate 触发；depth=1 时跳过 Compensate

### 完成标准

- role.json 正确加载并按 type 分发三种 Constraint
- `engine.Optimize()` 执行后返回含 `appliedRules` 的结果
- 过滤后企业列表为空时降级回全量，不报错
- `depth` 参数防止 Compensate 循环递归
- 所有单元测试通过

---

## Phase 2：rule-management

**目标**：实现规则完整生命周期，包含 CRUD、LLM 自动转换、热重载和前端管理界面。

### 实现任务

**后端 internal/rules/converter.go**

- [x] 定义 `Converter` 结构体（llm.Client + rolePath + mdPath + store + engine）
- [x] `ConvertAsync(mdContent string)`：启动 goroutine，立即返回；写入 `rules_convert_status=running`
- [x] `convert(mdContent string)`：同步执行，最多 3 轮重试，每轮将具体 `ValidationError` 列表反馈给 LLM
- [x] `extractJSON(content string)`：优先提取 \`\`\`json...\`\`\` 块，无则整体解析
- [x] `validateRoleJSON(jsonStr string) []ValidationError`：校验矩阵见 TECH-002 §2.1.4
- [x] `buildValidationErrorMessage(errs []ValidationError) string`：格式化错误为自然语言
- [x] 转换成功后：写 `rules_convert_status=ok`、`rules_convert_at`，调用 `engine.ReloadRules()`
- [x] 转换失败后：写 `rules_convert_status=error`、`rules_convert_error`

**后端 api/v3/rules.go**

- [x] `rulesFileRepo`：`ReadRules()` 按行解析编号列表（正则 `^\d+\.\s+(.+)$`）；`WriteRules()` 重新生成整个文件
- [x] `GET /api/rules`：返回规则列表 JSON
- [x] `POST /api/rules`：追加规则，触发 `ConvertAsync`
- [x] `PUT /api/rules/:index`：替换指定项，触发 `ConvertAsync`
- [x] `DELETE /api/rules/:index`：删除并重新编号，触发 `ConvertAsync`
- [x] `POST /api/rules/convert`：读取当前 rules.md，手动触发 `ConvertAsync`，返回 `{"status":"running"}`
- [x] `GET /api/rules/status`：读 config 表三个键，返回状态 + 时间 + 错误详情

**后端 server/server.go**

- [x] 启动时检测 `config/rules.md` 是否存在，不存在则创建默认空文件
- [x] 启动时调用 `engine.ReloadRules()`（role.json 不存在时正常运行）
- [x] config 表新增四个键的默认值注入

**前端 web/src/store/rulesStore.ts**

- [x] 定义 `RuleItem`、`RulesStore` 接口
- [x] 实现 `loadRules`、`addRule`、`updateRule`、`deleteRule`、`loadStatus`
- [x] 实现 `startPolling`（2s 间隔）/ `stopPolling`：CRUD 后启动，状态变为 ok/error 后停止

**前端 web/src/components/RuleList.tsx**

- [x] 规则列表展示（编号 + 文本 + 编辑/删除按钮）
- [x] 状态徽标：idle/ok=绿色已生效，running=蓝色转换中（轮询），error=红色转换失败（点击展开详情）
- [x] 新增/编辑使用 Dialog + textarea

**前端 web/src/pages/Settings.tsx**（第一个 Tab）

- [x] 路由 `/settings`，包含「调整规则」Tab（嵌入 RuleList）

### 完成标准

- CRUD API 全路径正常，每次操作后 status 正确流转
- rules.md 文件格式稳定（重写不丢内容）
- LLM 转换完成后 role.json 合法，`engine.ReloadRules()` 成功
- 3 轮重试后仍失败时 status=error，错误详情可读
- 前端 RuleList 正确展示规则和状态，轮询在 running 期间有效，非 running 时停止

---

## Phase 3：ai-chat

**目标**：完成 AI 对话模块，实现双层提示词、意图解析、调整执行和前端对话界面。

### 实现任务

**后端 internal/llm/system_prompt.go**

- [ ] 定义 `SystemPromptContext`（Year、Month、RuleCount、IndicatorSummary）
- [ ] 实现 `BuildChatSystemPrompt(ctx SystemPromptContext, userPrompt string) string`
- [ ] 内置提示词模板（const，代码内维护）：含角色定义、系统职责、数据上下文占位符、行为约束
- [ ] userPrompt 非空时以 `---` 分隔追加

**后端 internal/llm/intent.go**

- [ ] 定义 `AdjustmentAction`（type/indicatorId/value）和 `AdjustmentPlan`（actions 数组）
- [ ] 实现 `ParseIntent(client, userMsg, indicators) (*AdjustmentPlan, error)`
- [ ] 意图解析 Prompt：注入当前 16 项指标值，要求输出纯 JSON；无调整意图时返回 `{"actions":[]}`

**后端 api/v3/settings.go**

- [ ] `GET /api/settings/user-prompt`：读 config 表 `llm_user_prompt`
- [ ] `PUT /api/settings/user-prompt`：校验 `len(content) ≤ 500`，写 config 表

**后端 api/v3/llm_chat.go**

- [ ] `ChatRequest` 新增 `Mode string` 字段（`""` / `"chat"` / `"adjust"`）
- [ ] `mode=chat` 分支：读用户偏好，拼接双层 Prompt，流式返回
- [ ] `mode=adjust` 分支：
  1. `CalculateIndicators` 获取当前指标
  2. `ParseIntent` 解析意图 → `AdjustmentPlan`
  3. actions 为空时降级为 chat 模式
  4. 逐 action 调用 `engine.Optimize()`，收集 `appliedRules`
  5. 将执行结果注入 LLM 生成自然语言摘要
  6. 返回 `{reply, groups, appliedRules}`

**前端 web/src/components/ChatPanel.tsx**

- [ ] 右侧抽屉（400px），工具栏按钮展开/收起
- [ ] 顶部模式切换：💬聊天 / 🔧调整
- [ ] 对话历史区（滚动），含用户消息和 AI 回复
- [ ] 调整模式 appliedRules 气泡展示：
  - `clamp_target`：「目标值从 X 裁剪为 Y」
  - `filter_allocation`：「参与企业从 N 过滤为 M 家」
  - `compensate`：「联动补偿 {ensureIndicator} → {target}」
- [ ] 调整成功后调用 `dataStore.loadGroups()` 刷新 Dashboard

**前端 web/src/pages/Settings.tsx**（第二个 Tab）

- [ ] 「AI 偏好」Tab：textarea 输入，字数统计（xx/500），保存按钮

### 完成标准

- `mode=chat` 使用双层 Prompt（内置 + 用户偏好），内置层不可修改
- `mode=adjust` 正确解析指标 ID 和目标值，执行 `engine.Optimize()`
- 纯咨询意图（无调整动作）自动降级为 chat 模式
- appliedRules 气泡正确展示三种规则类型的生效详情
- 调整后 Dashboard 指标区自动刷新
- Settings AI 偏好 Tab 500 字符限制有效，保存持久化

---

## 实施顺序与依赖

```
Phase 1 (rule-engine)
  ↓
Phase 2 (rule-management) — 依赖 Phase 1 的 engine.ReloadRules()
  ↓
Phase 3 (ai-chat)         — 依赖 Phase 1 的 engine.Optimize()
                            依赖 Phase 2 的 /api/settings/user-prompt 路由注册
```

---

## 完成状态

| Phase | 模块 | 状态 |
|-------|------|------|
| 1 | rule-engine | ✅ 已完成（2026-03-14） |
| 2 | rule-management | ✅ 已完成（2026-03-14） |
| 3 | ai-chat | ⬜ 未开始 |
