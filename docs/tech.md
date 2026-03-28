# Northstar 技术架构

> 作者：Anner
> 创建：2026-03-14
> 最后更新：2026-03-28
> 前置：Go/Gin + SQLite + DAG 引擎 + React/Vite

---

## 1. 模块全景

```
internal/
  rules/              ← 规则加载 + Constraint 类型 + LLM 转换
    loader.go
    constraint.go
    filter.go
    converter.go
  llm/
    system_prompt.go  ← 内置提示词
    intent.go         ← 意图解析（set_target / adjust_percent / add_rule）
    tools.go          ← LLM 工具定义（set_indicator_targets / update_companies / add_rule）
    client.go         ← LLM 客户端封装
  api/v3/
    rules.go          ← 规则 CRUD + 转换触发
    settings.go       ← 用户偏好提示词配置
    llm_chat.go       ← AI 对话 SSE 流式接口
    optimize.go       ← 指标调整入口
    handler.go        ← 路由注册
  dagcalc/
    engine.go         ← DAG 引擎，持有 RuleSet + ReloadRules
    adjust.go         ← 反向调整算法（16 指标 + 三注入点）
    adjust_rules.go   ← 规则注入桥接层
config/
  rules.md            ← 用户规则（自然语言）
  role.json           ← LLM 自动生成（结构化 JSON）
web/src/
  pages/Settings.tsx
  pages/DashboardV3.tsx
  components/
    RuleList.tsx
    ChatPanel.tsx
    AIPreferenceForm.tsx
  store/
    rulesStore.ts
    dataStore.ts
```

---

## 2. 后端包设计

### 2.1 internal/rules

#### loader.go

```go
type RuleSet struct {
    Clamps      []*ClampTargetConstraint
    Filters     []*FilterAllocationConstraint
    Compensates []*CompensateConstraint
}

// Load 读取 role.json，按 type 分发构建 RuleSet
// 文件不存在时返回空 RuleSet（降级运行）
func Load(path string) (*RuleSet, error)
```

#### constraint.go

三种 Constraint 类型：

- `ClampTargetConstraint.Clamp(indicatorID, target) → (float64, bool)`
- `FilterAllocationConstraint.Apply(indicatorID, companies) → ([]CompanyRow, bool)`
- `CompensateConstraint.Check(triggerID, indicators) → (bool, float64)`

#### filter.go

`filterByMode(companies, mode)` 支持 4 个枚举值，未知 mode 不过滤。

#### converter.go

```go
type Converter struct {
    llm      llm.Client
    rolePath string
    mdPath   string
    store    store.Store
    engine   *Engine
}

// ConvertAsync 启动异步 goroutine
// 最多 3 轮重试，每轮将 ValidationError 反馈给 LLM
// 成功后写 role.json + engine.ReloadRules()
func (c *Converter) ConvertAsync(mdContent string)
```

校验矩阵：

| 规则类型 | 校验项 |
|---------|--------|
| `clamp_target` | indicator 在 16 项枚举中；min/max 不能同时为 null |
| `filter_allocation` | indicator 在枚举中；filter 在 4 个枚举值中 |
| `compensate` | trigger/ensure 均在枚举中；relation 为 gte 或 lte |

状态写入 config 表：`rules_convert_status`（running/ok/error）、`rules_convert_at`、`rules_convert_error`

### 2.2 internal/llm

#### system_prompt.go

```go
type SystemPromptContext struct {
    Year, Month, RuleCount int
    IndicatorSummary       string
}

// BuildChatSystemPrompt 拼接内置层 + 用户偏好层
func BuildChatSystemPrompt(ctx SystemPromptContext, userPrompt string) string
```

#### intent.go

```go
type AdjustmentAction struct {
    Type        string  `json:"type"`         // set_target / adjust_percent / add_rule
    IndicatorID string  `json:"indicatorId,omitempty"`
    Value       float64 `json:"value,omitempty"`
    Percent     float64 `json:"percent,omitempty"`
    RuleText    string  `json:"ruleText,omitempty"`
}

type AdjustmentPlan struct {
    Actions []AdjustmentAction `json:"actions"`
}

// ParseIntent 将用户输入解析为结构化调整计划
func ParseIntent(client IntentClient, userMsg string, indicators map[string]float64) (*AdjustmentPlan, error)
```

#### tools.go

三个 LLM 工具：
- `set_indicator_targets` — 设置指标目标值
- `update_companies` — 批量修改企业字段
- `add_rule` — 添加持久规则

### 2.3 dagcalc/engine.go

```go
type Engine struct {
    store    store.Store
    rulePath string
    rules    *rules.RuleSet
    mu       sync.RWMutex
}

func (e *Engine) ReloadRules() error    // 原子替换 rules
func (e *Engine) Optimize(targets) OptimizeResult  // 统一调整入口
func (e *Engine) RuleCount() int
```

### 2.4 dagcalc/adjust.go

`ApplyIndicatorTarget(store, year, month, indicatorID, target, ruleSet, depth)` 三个干预点：

1. **ClampTarget（分配前）** — 裁剪目标值到 min/max
2. **FilterAllocation（分配中）** — 过滤参与企业，6 个底层分配函数全覆盖
3. **Compensate（分配后）** — depth=0 时检查关联指标，递归补偿一次

### 2.5 api/v3/llm_chat.go

`POST /api/llm/chat/stream`（SSE）：

- `mode=chat` — 纯对话
- `mode=adjust` — 意图解析 → 分离三类动作（set_target / adjust_percent / add_rule）
  - `set_target` + `adjust_percent` → 转为 targets → `runOptimize`
  - `add_rule` → `addRuleFromChat()` 写入 rules.md + 触发转换
  - 混合操作同时处理

SSE 事件类型：`message_delta`、`result`、`error`、`final`

---

## 3. config 表键

| key | 类型 | 默认值 | 说明 |
|-----|------|--------|------|
| `llm_user_prompt` | string | `""` | 用户偏好提示词 |
| `rules_convert_status` | string | `idle` | idle/running/ok/error |
| `rules_convert_at` | string | `""` | 最后成功时间 RFC3339 |
| `rules_convert_error` | string | `""` | 最后错误详情 |

---

## 4. 前端设计

### Settings 页（/settings）

两个 Tab：「调整规则」（RuleList）+ 「AI 偏好」（AIPreferenceForm）

### RuleList

- 规则列表：编号 + 文本 + 编辑/删除
- 状态徽标：待转换/转换中/已生效/转换失败
- 新增/编辑 Dialog（textarea + placeholder 提示）
- CRUD 后自动轮询转换状态（2s 间隔）+ Toast 通知链

### ChatPanel

- Dashboard 右侧抽屉（420px）
- 模式切换：聊天/调整
- SSE 流式消息 + appliedRules 气泡 + 规则添加状态卡片
- 默认快捷问题（含"添加规则"、"随机调整"）
- 调整完成后刷新 Dashboard

---

## 5. 关键设计决策

| 决策 | 说明 |
|------|------|
| rules.md 按行解析 | 不引入复杂解析库，正则提取 |
| RWMutex 并发安全 | ReloadRules 写锁，Optimize 读锁 |
| depth 防循环 | compensate 递归最多 1 层 |
| filter 降级 | 过滤后为空时回退全量 |
| 提示词三类隔离 | 对话/转换/意图各自独立 prompt |
| 两条路径共用底层 | 页面添加和聊天添加共用 rulesFileRepo |
