# Northstar 技术架构

> 作者：Anner
> 创建：2026-03-14
> 最后更新：2026-04-06
> 前置：Go/Gin + SQLite + DAG 引擎 + React/Vite

---

## 1. 模块全景

```
internal/
  rules/              ← 约束加载 + Constraint 类型
    loader.go            从 DB 加载约束 → RuleSet
    constraint.go        三种约束类型定义
    filter.go            四种过滤模式
  llm/
    system_prompt.go  ← 三层提示词（内置 + 自然语言规则 + 用户偏好）
    intent.go         ← 意图解析（set_target / adjust_percent / add_rule）
    tools.go          ← LLM 工具定义（set_indicator_targets / update_companies / add_rule）
    client.go         ← LLM 客户端封装
  api/v3/
    rules.go          ← 硬约束 + 自然语言规则 CRUD
    settings.go       ← 用户偏好提示词配置
    llm_chat.go       ← AI 对话 SSE 流式接口
    optimize.go       ← ���标调整入口
    handler.go        ← 路由注册
  dagcalc/
    engine.go         ← DAG 引擎，持有 RuleSet + ReloadRules
    adjust.go         ← 反向调整算法（16 指标 + 三注入点）
    adjust_rules.go   ← 规则注入桥接层
  store/
    adjustment_constraints.go  ← 硬约束表 CRUD
    natural_rules.go           ← 自然语言规则表 CRUD
web/src/
  pages/Settings.tsx
  pages/DashboardV3.tsx
  components/
    RuleList.tsx          ← 硬约束 + 自然语言规则管理
    ChatPanel.tsx
    AIPreferenceForm.tsx
  store/
    rulesStore.ts         ← 约束 + 规则状态管理
    dataStore.ts
```

---

## 2. 三层规则体系

### 2.1 硬约束（确定性，每次调整自动执行）

存储在 `adjustment_constraints` 表，用户通过前端表单直接管理。

三种约束类型：
- `ClampTargetConstraint.Clamp(indicatorID, target) → (float64, bool)` — 目标值裁剪
- `FilterAllocationConstraint.Apply(indicatorID, companies) → ([]CompanyRow, bool)` — 企业过滤
- `CompensateConstraint.Check(triggerID, indicators) → (bool, float64)` — 联动补偿

加载方式：`rules.LoadFromStore(store) → *RuleSet`

### 2.2 自然语言规则（注入 LLM 上下文）

存储在 `natural_rules` 表，以文本形式注入 AI 对话的系统提示词。
LLM 在做 Function Call 调整时参考这些规则。

### 2.3 Function Call（LLM 主动调用）

三个 LLM 工具：
- `set_indicator_targets` — 设置指标目标值
- `update_companies` — 批量修改企业字段
- `add_rule` — 添加自然语言规则

---

## 3. 后端包设计

### 3.1 internal/rules

#### loader.go

```go
type RuleSet struct {
    Clamps      []*ClampTargetConstraint
    Filters     []*FilterAllocationConstraint
    Compensates []*CompensateConstraint
}

// LoadFromStore 从数据库加载约束，构建 RuleSet
func LoadFromStore(st *store.Store) (*RuleSet, error)
```

#### constraint.go

三种 Constraint 类型（不变）。

#### filter.go

`filterByMode(companies, mode)` 支持 4 个枚举值（不变）。

### 3.2 internal/llm

#### system_prompt.go

```go
type SystemPromptContext struct {
    Year, Month, ConstraintCount int
    NaturalRules                 []string
    IndicatorSummary             string
}

// BuildChatSystemPrompt 拼接内置层 + 自然语言规则层 + 用户偏好层
func BuildChatSystemPrompt(ctx SystemPromptContext, userPrompt string) string
```

#### intent.go

三种 action type（不变）。add_rule 的处理变更为直接写入 `natural_rules` 表。

### 3.3 dagcalc/engine.go

```go
type Engine struct {
    graph *Graph
    store *store.Store
    year  int
    month int
    rules *rules.RuleSet
    mu    sync.RWMutex
}

func (e *Engine) ReloadRules() error    // 从 DB 重新加载约束
func (e *Engine) Optimize(targets) OptimizeResult
func (e *Engine) ConstraintCount() int
```

### 3.4 api/v3/rules.go

硬约束 + 自然语言规则的 CRUD API。

---

## 4. 数据库表

### adjustment_constraints — 硬约束

```sql
CREATE TABLE IF NOT EXISTS adjustment_constraints (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    type TEXT NOT NULL,
    indicator_id TEXT,
    min_value REAL,
    max_value REAL,
    filter_mode TEXT,
    trigger_id TEXT,
    ensure_id TEXT,
    relation TEXT,
    tolerance REAL DEFAULT 0,
    enabled INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### natural_rules — 自然语言规则

```sql
CREATE TABLE IF NOT EXISTS natural_rules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    text TEXT NOT NULL,
    enabled INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

---

## 5. config 表键

| key | 类型 | 默认值 | 说明 |
|-----|------|--------|------|
| `llm_user_prompt` | string | `""` | 用户偏好提示词 |

已删除的 config 键：`rules_convert_status`、`rules_convert_at`、`rules_convert_error`、`rules_convert_step`、`rules_convert_attempt`

---

## 6. 前端设计

### Settings 页（/settings）

两个 Tab：「调整规则」（RuleList）+ 「AI 偏好」（AIPreferenceForm）

### RuleList

两个区域：
- **硬约束**：表单式 CRUD（选择类型 + 指标 + 参数），变更即时生效
- **自然语言规则**：文本列表 + 新增/编辑 Dialog，作为 LLM 上下文生效

### ChatPanel

- Dashboard 右侧抽屉（420px）
- 模式切换：聊天/调整
- SSE 流式消息 + appliedRules 气泡
- 规则添加即时反馈（无需等待转换）
- 调整完成后刷新 Dashboard

---

## 7. 关键设计决策

| 决策 | 说明 |
|------|------|
| 硬约束存 DB | 用户表单直接管理，无需 LLM 转换，确定性执行 |
| 自然语言规则不转 JSON | 直接注入 LLM 上下文，表达力不受限 |
| 删除转换管道 | 砍掉 converter.go、rules.md、role.json、状态轮询 |
| RWMutex 并发安全 | ReloadRules 写锁，Optimize 读锁 |
| depth 防循环 | compensate 递归最多 1 层 |
| filter 降级 | 过滤后为空时回退全量 |
