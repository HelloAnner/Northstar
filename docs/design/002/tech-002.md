# Northstar TECH-002：规则引擎统一 + AI 对话模块技术架构

> 作者：Anner
> 创建：2026-03-14
> 前置：TECH-001（Go/Gin + SQLite + DAG 引擎 + React/Vite 已完成）
> 对应：PRD-002

---

## 1. 总体架构变更

### 1.1 新增模块全景

```
internal/
  rules/              ← 新增：规则加载 + Constraint 类型 + LLM 转换
    loader.go
    constraint.go
    filter.go
    converter.go
  llm/
    system_prompt.go  ← 新增：内置提示词硬编码
    intent.go         ← 新增：AI 意图解析 → AdjustmentPlan
    chat.go           ← 已有，不变
  api/v3/
    rules.go          ← 新增：规则 CRUD + 转换触发 API
    settings.go       ← 新增：用户偏好提示词配置 API
    llm_chat.go       ← 已有，扩展 mode=adjust 分支
    optimize.go       ← 已有，注入 rules 参数
  dagcalc/
    adjust.go         ← 已有，扩展三个干预点
    engine.go         ← 已有，持有 RuleSet + ReloadRules
config/               ← 新增目录（与数据库文件同级）
  rules.md
  role.json
web/src/
  pages/Settings.tsx  ← 新增
  components/
    RuleList.tsx      ← 新增
    ChatPanel.tsx     ← 新增
  store/
    rulesStore.ts     ← 新增
```

### 1.2 数据流变化对比

| 环节 | 001 | 002 |
|------|-----|-----|
| 调整时约束来源 | 代码硬编码权重 | role.json 动态加载 |
| AI 调整路径 | 独立逻辑 | 解析为 AdjustmentPlan，复用同一引擎 |
| 系统提示词 | 单一固定 | 内置层（不可改）+ 用户层（可配置） |
| rules.md → role.json | 无 | LLM 多轮对话转换 + 校验重试 |

---

## 2. 后端包设计

### 2.1 internal/rules

本包是 002 版本核心新增包，三个职责：加载 role.json → 构建 RuleSet；定义三种 Constraint 类型及执行方法；将 rules.md 通过 LLM 异步转换为 role.json。

#### 2.1.1 loader.go

```go
// RuleSet 是内存中的规则集合，由 Engine 持有
type RuleSet struct {
    Clamps      []*ClampTargetConstraint
    Filters     []*FilterAllocationConstraint
    Compensates []*CompensateConstraint
}

// rawRule 对应 role.json 中每条规则的原始结构（所有类型字段共用一个结构体）
type rawRule struct {
    ID        string   `json:"id"`
    Name      string   `json:"name"`
    Type      string   `json:"type"`
    // clamp_target
    Indicator string   `json:"indicator,omitempty"`
    Min       *float64 `json:"min"`
    Max       *float64 `json:"max"`
    // filter_allocation
    Filter    string   `json:"filter,omitempty"`
    // compensate
    Trigger   string   `json:"trigger,omitempty"`
    Ensure    string   `json:"ensure,omitempty"`
    Relation  string   `json:"relation,omitempty"`
    Tolerance float64  `json:"tolerance,omitempty"`
}

type rawRoleJSON struct {
    Version string    `json:"version"`
    Rules   []rawRule `json:"rules"`
}

// Load 读取 role.json，按 type 字段分发构建 RuleSet
// role.json 不存在时返回空 RuleSet（不报错，降级运行）
func Load(path string) (*RuleSet, error)
```

分发规则：

| JSON type | Go 类型 |
|-----------|---------|
| `clamp_target` | `ClampTargetConstraint` |
| `filter_allocation` | `FilterAllocationConstraint` |
| `compensate` | `CompensateConstraint` |
| 其他 | 跳过，不影响加载 |

#### 2.1.2 constraint.go

三种 Constraint 类型及方法：

```go
// ClampTargetConstraint：分配前裁剪目标值
type ClampTargetConstraint struct {
    ID          string
    IndicatorID string
    Min         *float64 // nil = 无下限
    Max         *float64 // nil = 无上限
}

// Clamp 返回裁剪后目标值 + 是否发生了裁剪
func (c *ClampTargetConstraint) Clamp(indicatorID string, target float64) (float64, bool)

// FilterAllocationConstraint：分配中过滤企业集合
type FilterAllocationConstraint struct {
    ID          string
    IndicatorID string
    Filter      string // "positive_current"|"negative_current"|"large_scale_only"|"exclude_small_micro"
}

// Apply 过滤企业列表，返回过滤结果 + 是否有变化
func (c *FilterAllocationConstraint) Apply(indicatorID string, companies []CompanyRow) ([]CompanyRow, bool)

// CompensateConstraint：分配后自动补偿关联指标
type CompensateConstraint struct {
    ID        string
    TriggerID string
    EnsureID  string
    Relation  string  // "gte"|"lte"
    Tolerance float64
}

// Check 返回是否需要补偿 + 补偿目标值
func (c *CompensateConstraint) Check(triggerID string, indicators map[string]float64) (bool, float64)
```

`CompensateConstraint.Check` 逻辑：

```
relation="gte"：若 ensure < trigger - tolerance，需补偿，目标 = trigger - tolerance
relation="lte"：若 ensure > trigger + tolerance，需补偿，目标 = trigger + tolerance
```

#### 2.1.3 filter.go

```go
// filterByMode 根据枚举模式过滤企业列表
// 未知 mode 时不过滤（防御性降级）
func filterByMode(companies []CompanyRow, mode string) []CompanyRow
```

| mode | 过滤条件 | 所需字段 |
|------|---------|---------|
| `positive_current` | 对应指标当月增速 > 0 | 从 DB 预读取 |
| `negative_current` | 对应指标当月增速 < 0 | 从 DB 预读取 |
| `large_scale_only` | CompanyScale == 1 | company_scale |
| `exclude_small_micro` | IsSmallMicro == false | is_small_micro |

> 增速字段取最近一次 `RecalcAll` 后的 DB 存储值，不在分配中实时计算。

#### 2.1.4 converter.go

负责异步将 rules.md 通过 LLM 转换为 role.json，失败时携带具体错误反馈多轮重试。

```go
type Converter struct {
    llm      llm.Client
    rolePath string      // config/role.json
    mdPath   string      // config/rules.md
    store    store.Store // 写转换状态到 config 表
    engine   *Engine     // 转换成功后触发热重载
}

// ConvertAsync 启动异步 goroutine，立即返回
func (c *Converter) ConvertAsync(mdContent string)

// convert 同步执行，最多重试 3 轮
func (c *Converter) convert(mdContent string) (string, error)
```

**多轮重试流程（`convert` 内部）：**

```
messages = [{role:"user", content: buildUserMessage(mdContent)}]

for attempt = 1..3:
    resp = llm.Chat(system=buildConvertSystemPrompt(), messages)
    jsonStr, err = extractJSON(resp.Content)
    if err != nil:
        messages += [assistant: resp.Content,
                     user: "输出无法提取为 JSON，错误：{err}，请只输出纯 JSON"]
        continue
    errs = validateRoleJSON(jsonStr)
    if len(errs) == 0:
        return jsonStr   ← 成功

    messages += [assistant: resp.Content,
                 user: buildValidationErrorMessage(errs)]

return error("3次重试仍失败")
```

**辅助函数：**

```go
// extractJSON 从 LLM 输出提取 JSON 字符串
// 优先提取 ```json...``` 代码块，无则整体尝试解析
func extractJSON(content string) (string, error)

// validateRoleJSON 返回所有校验错误（结构见下节）
func validateRoleJSON(jsonStr string) []ValidationError

// buildValidationErrorMessage 将错误列表格式化为自然语言，供 LLM 重试上下文
func buildValidationErrorMessage(errs []ValidationError) string
```

**ValidationError：**

```go
type ValidationError struct {
    RuleID  string
    Field   string
    Message string
}
```

**validateRoleJSON 校验矩阵：**

| 规则类型 | 校验项 | 错误示例 |
|---------|--------|---------|
| `clamp_target` | indicator 在 16 项枚举中 | `"wholesale" 不在允许的指标 ID 列表中` |
| `clamp_target` | min/max 不能同时为 null | `min 和 max 不能同时为 null` |
| `filter_allocation` | indicator 在枚举中 | 同上 |
| `filter_allocation` | filter 在 4 个枚举值中 | `"big_company" 不在允许的 filter 枚举中` |
| `compensate` | trigger/ensure 均在枚举中 | 同上 |
| `compensate` | relation 为 gte 或 lte | `relation 必须为 gte 或 lte` |
| 任意 | type 未知 | `未知规则类型 "limit"` |

**状态写入 config 表时机：**

| 时机 | key | 值 |
|------|-----|----|
| ConvertAsync 开始 | `rules_convert_status` | `running` |
| 转换成功 | `rules_convert_status` | `ok` |
| 转换成功 | `rules_convert_at` | RFC3339 时间 |
| 转换失败 | `rules_convert_status` | `error` |
| 转换失败 | `rules_convert_error` | 最后一次错误详情 |

---

### 2.2 internal/llm 扩展

#### 2.2.1 system_prompt.go（新增）

内置规则提示词硬编码，不存数据库，不暴露编辑接口：

```go
type SystemPromptContext struct {
    Year             int
    Month            int
    RuleCount        int
    IndicatorSummary string // "限上增速:8.5%, 批发:6.2%, ..." 摘要
}

// BuildChatSystemPrompt 拼接内置层 + 用户偏好层
// userPrompt 为空时不拼接分隔符
func BuildChatSystemPrompt(ctx SystemPromptContext, userPrompt string) string
```

拼接结构：

```
{内置提示词（含动态填充占位符）}
---
{userPrompt}   ← 仅 userPrompt 非空时追加
```

内置提示词模板（`const`，代码内维护）：

```
你是 Northstar 月度经济数据统计系统的 AI 助手。

# 系统职责
- 帮助用户分析和调整批发、零售、住宿、餐饮四大行业的月度指标数据
- 调整建议必须基于已加载的规则（当前约束已注入上下文）
- 不允许建议删除企业数据或重置导入数据

# 当前数据上下文
数据期间：{year} 年 {month} 月
已加载规则数：{ruleCount} 条
当前核心指标：{indicatorSummary}

# 行为约束
- 涉及具体数值调整时，给出明确目标值，不使用"大约"等模糊表述
- 如用户意图超出约束范围，先说明约束，再给出约束内建议
- mode=adjust 时，输出结构化 AdjustmentPlan JSON，不含多余文字
```

转换专用 Prompt（`buildConvertSystemPrompt`）在 `converter.go` 中单独定义，与对话提示词代码路径隔离。

#### 2.2.2 intent.go（新增）

```go
type AdjustmentAction struct {
    Type        string  `json:"type"`        // 当前版本固定为 "set_indicator"
    IndicatorID string  `json:"indicatorId"`
    Value       float64 `json:"value"`
}

type AdjustmentPlan struct {
    Actions []AdjustmentAction `json:"actions"`
}

// ParseIntent 调用 LLM 将用户输入解析为 AdjustmentPlan
// 纯咨询意图时返回 actions 为空的 Plan（不报错）
func ParseIntent(
    client llm.Client,
    userMsg string,
    indicators map[string]float64,
) (*AdjustmentPlan, error)
```

意图解析 Prompt 要求（System Prompt 固定在代码中）：
- 注入当前 16 项指标值作为上下文
- 要求输出 `{"actions": [...]}` 纯 JSON
- 无调整意图时输出 `{"actions": []}`，不报错

---

### 2.3 dagcalc/engine.go 扩展

```go
type Engine struct {
    store    store.Store
    rulePath string
    rules    *rules.RuleSet
    mu       sync.RWMutex  // 保护并发读写
}

// ReloadRules 重新加载 role.json，原子替换 rules 指针
func (e *Engine) ReloadRules() error {
    rs, err := rules.Load(e.rulePath)
    if err != nil { return err }
    e.mu.Lock()
    e.rules = rs
    e.mu.Unlock()
    return nil
}

// getRules 并发安全地获取当前 RuleSet 快照
func (e *Engine) getRules() *rules.RuleSet {
    e.mu.RLock()
    defer e.mu.RUnlock()
    return e.rules
}

// Optimize 对上层 API 的统一调整入口，rules 由 Engine 持有并注入
func (e *Engine) Optimize(year, month int, indicatorID string, target float64) ([]AppliedRule, error) {
    return dagcalc.ApplyIndicatorTarget(e.store, year, month, indicatorID, target, e.getRules(), 0)
}
```

---

### 2.4 dagcalc/adjust.go 扩展

#### 2.4.1 新增类型

```go
// AppliedRule 记录某条规则在本次调整中的实际生效情况（用于 appliedRules 响应）
type AppliedRule struct {
    RuleID          string  `json:"ruleId"`
    Action          string  `json:"action"` // "clamp_target"|"filter_allocation"|"compensate"
    // clamp_target
    OriginalTarget  float64 `json:"originalTarget,omitempty"`
    EffectiveTarget float64 `json:"effectiveTarget,omitempty"`
    // filter_allocation
    CompaniesBefore int     `json:"companiesBefore,omitempty"`
    CompaniesAfter  int     `json:"companiesAfter,omitempty"`
    // compensate
    EnsureIndicator  string  `json:"ensureIndicator,omitempty"`
    CompensateTarget float64 `json:"compensateTarget,omitempty"`
}
```

#### 2.4.2 函数签名变更

```go
// ApplyIndicatorTarget 在原有分配逻辑基础上注入三个干预点
// depth > 0 时跳过 Compensate 干预点（防止 A↔B 循环补偿）
func ApplyIndicatorTarget(
    store store.Store,
    year, month int,
    indicatorID string,
    requestedTarget float64,
    rs *rules.RuleSet,
    depth int,
) ([]AppliedRule, error)
```

#### 2.4.3 三个干预点执行顺序

```
【干预点1：ClampTarget（分配前）】
effectiveTarget = requestedTarget
for _, c := range rs.Clamps:
    clamped, changed = c.Clamp(indicatorID, effectiveTarget)
    if changed:
        effectiveTarget = clamped
        append AppliedRule{action:"clamp_target", original:requestedTarget, effective:clamped}

delta = computeDelta(indicatorID, effectiveTarget, store, year, month)
  // 复用现有逻辑：增速类指标 → 折算到企业当月值变化量

【干预点2：FilterAllocation（分配中）】
candidates = loadCompaniesForIndicator(store, indicatorID, year, month)
for _, f := range rs.Filters:
    filtered, changed = f.Apply(indicatorID, candidates)
    if changed:
        candidates = filtered
        append AppliedRule{action:"filter_allocation", before:len(pre), after:len(post)}
if len(candidates) == 0:
    candidates = loadCompaniesForIndicator(...)  // 降级：回退全量

randomizeAllocations(delta, candidates)  // 现有权重随机分配算法，不变
writeBack(store)                          // 事务写 DB
RecalcAll(store, year, month)            // 派生字段 + 指标重算

【干预点3：Compensate（分配后，depth==0 时执行）】
if depth == 0:
    indicators = CalculateIndicators(store, year, month)
    for _, c := range rs.Compensates:
        ok, compTarget = c.Check(indicatorID, indicators)
        if ok:
            subApplied, _ = ApplyIndicatorTarget(..., c.EnsureID, compTarget, rs, depth+1)
            append AppliedRule{action:"compensate", ensure:c.EnsureID, target:compTarget}
            appliedRules = append(appliedRules, subApplied...)
```

---

### 2.5 api/v3/rules.go（新增）

**rules.md 文件读写**封装为私有 `rulesFileRepo`：

```go
type rulesFileRepo struct {
    path string
}

// ReadRules 解析 rules.md，返回编号列表
// 跳过空行和 # 开头的标题行，正则 ^\d+\.\s+(.+)$ 提取文本
func (r *rulesFileRepo) ReadRules() ([]RuleItem, error)

// WriteRules 将规则列表写回 rules.md
// 格式固定：# 调整规则\n\n1. ...\n2. ...\n
func (r *rulesFileRepo) WriteRules(items []RuleItem) error

type RuleItem struct {
    Index int
    Text  string
}
```

**Handler 实现：**

```
GET /api/rules
  → repo.ReadRules() → JSON

POST /api/rules
  Body: {text}
  → ReadRules → append → WriteRules → ConvertAsync(全文)
  → 201 Created

PUT /api/rules/:index
  Body: {text}
  → ReadRules → 替换 index 项 → WriteRules → ConvertAsync
  → 200 OK

DELETE /api/rules/:index
  → ReadRules → 删除 index 项 → 重新编号 → WriteRules → ConvertAsync
  → 200 OK

POST /api/rules/convert
  → 读取当前 rules.md 全文 → ConvertAsync(全文)
  → 200 {"status":"running"}

GET /api/rules/status
  → 读 config 表 rules_convert_status/at/error
  → {"status":"ok","updatedAt":"...","error":""}
```

---

### 2.6 api/v3/settings.go（新增）

```
GET /api/settings/user-prompt
  → store.GetConfig("llm_user_prompt")
  → {"content":"..."}

PUT /api/settings/user-prompt
  Body: {content}
  → 校验 len(content) ≤ 500
  → store.SetConfig("llm_user_prompt", content)
  → 200 OK
```

---

### 2.7 api/v3/llm_chat.go 扩展

请求新增 `mode` 字段：

```go
type ChatRequest struct {
    Message string `json:"message"`
    Year    int    `json:"year"`
    Month   int    `json:"month"`
    Mode    string `json:"mode"` // "" | "chat" | "adjust"
}
```

**mode=chat（原逻辑）：**

```
userPrompt = store.GetConfig("llm_user_prompt")
sysPrompt  = BuildChatSystemPrompt(ctx, userPrompt)
stream llm.Chat → 流式返回文本
```

**mode=adjust（新增）：**

```
1. indicators = CalculateIndicators(store, year, month)
2. plan, err = ParseIntent(client, message, indicators)
3. if len(plan.Actions) == 0 → 降级为 chat 模式
4. for each action in plan.Actions:
       applied, err = engine.Optimize(year, month, action.IndicatorID, action.Value)
       collect appliedRules
5. groups = store.GetGroups(year, month)
6. 将执行结果注入 LLM → 生成自然语言描述
7. Response: {reply, groups, appliedRules}
```

---

### 2.8 api/v3/optimize.go 变更

Handler 改调 `engine.Optimize`，响应新增 `appliedRules`：

```go
type OptimizeResponse struct {
    Groups       []IndicatorGroup `json:"groups"`
    AppliedRules []AppliedRule    `json:"appliedRules,omitempty"`
}
```

原 `ApplyIndicatorTarget` 直接调用改为 `engine.Optimize`，逻辑不变。

---

## 3. config 表新增键

| key | 类型 | 默认值 | 说明 |
|-----|------|--------|------|
| `llm_user_prompt` | string | `""` | 用户偏好提示词 |
| `rules_convert_status` | string | `idle` | `idle\|running\|ok\|error` |
| `rules_convert_at` | string | `""` | 最后成功时间 RFC3339 |
| `rules_convert_error` | string | `""` | 最后一次错误详情 |

无需新建数据库表，全部写入现有 `config` 表。

---

## 4. 文件系统

```
config/                  ← 新建目录，与 store.db 同级
  rules.md               # 用户规则（自然语言，可 git 追踪）
  role.json              # LLM 自动生成
```

**初始化逻辑（服务启动时）：**

```go
// server/server.go 启动时
os.MkdirAll("config", 0755)
ensureRulesMarkdown("config/rules.md") // 从 internal/server/defaults/rules.md 拷贝
ensureRoleJSON("config/role.json")     // 从 internal/server/defaults/role.json 拷贝
engine.ReloadRules()                   // 加载默认或现有 role.json
```

---

## 5. 前端设计

### 5.1 Settings 页面（/settings）

两个 Tab 组织内容：

```
[调整规则]  [AI 偏好]
```

**调整规则 Tab — RuleList.tsx：**

```
┌─────────────────────────────────────────────────┐
│ 调整规则                            [+ 新增规则] │
│ 状态：✅ 已生效  最后转换：03-14 10:30           │
├─────────────────────────────────────────────────┤
│ 1  限上社零额当月增速不超过 15%    [编辑] [删除]  │
│ 2  批发业当月增速在 -10% 到 20%   [编辑] [删除]  │
│ 3  调整零售增速时，仅调正增长企业  [编辑] [删除]  │
└─────────────────────────────────────────────────┘
```

状态徽标颜色：

| status | 展示 |
|--------|------|
| `idle` / `ok` | 绿色"已生效" |
| `running` | 蓝色"转换中…"，2s 轮询一次 |
| `error` | 红色"转换失败"，点击展开错误详情 |

新增/编辑使用 Dialog，输入框为 textarea（单行规则文本）。

**AI 偏好 Tab：**

```
┌──────────────────────────────────────────────────┐
│ AI 对话偏好（选填）                               │
│ 用于自定义 AI 的回复风格，不影响核心调整逻辑        │
│ ┌──────────────────────────────────────────────┐ │
│ │ 分析时优先关注零售业，使用简洁中文...          │ │
│ └──────────────────────────────────────────────┘ │
│                              [保存]  已输入 xx/500 │
└──────────────────────────────────────────────────┘
```

### 5.2 ChatPanel.tsx（新增）

位置：Dashboard 右侧抽屉（宽度 400px），通过工具栏按钮展开/收起。

**UI 结构：**

```
┌──────────────────────────────┐
│ AI 助手         [💬聊天][🔧调整] │
├──────────────────────────────┤
│ 对话历史区（滚动）             │
│                              │
│ [AI]: 当前限上增速 8.5%，     │
│       批发增速 6.2%...        │
│                              │
│ [用户]: 把限上增速调到 10%    │
│                              │
│ [AI]: 已为您调整：            │
│  ┌─────────────────────────┐ │
│  │ 限上增速：8.5% → 10.0%  │ │
│  │ 应用规则：clamp(上限15%) │ │
│  └─────────────────────────┘ │
├──────────────────────────────┤
│ [输入框...              ] [发送] │
└──────────────────────────────┘
```

**调整模式下 AI 回复展示 appliedRules 气泡：**
- `clamp_target`：显示"目标值从 X 裁剪为 Y（规则：规则名）"
- `filter_allocation`：显示"参与企业从 N 过滤为 M 家"
- `compensate`：显示"联动补偿 ensureIndicator → target"

调整执行成功后，调用 `dataStore.loadGroups()` 刷新 Dashboard 指标区。

### 5.3 rulesStore.ts（新增）

```typescript
interface RuleItem {
  index: number
  text: string
}

interface RulesStore {
  rules: RuleItem[]
  status: 'idle' | 'running' | 'ok' | 'error'
  statusError: string
  statusUpdatedAt: string
  pollingTimer: ReturnType<typeof setInterval> | null

  loadRules: () => Promise<void>
  addRule: (text: string) => Promise<void>
  updateRule: (index: number, text: string) => Promise<void>
  deleteRule: (index: number) => Promise<void>
  loadStatus: () => Promise<void>
  startPolling: () => void   // status=running 时启动，2s 间隔
  stopPolling: () => void    // status!=running 时停止
}
```

每次 CRUD 操作后调用 `startPolling()`，直到 status 变为 `ok` 或 `error` 后 `stopPolling()`。

---

## 6. 接口汇总

| 方法 | 路径 | 说明 | 是否新增 |
|------|------|------|---------|
| GET | `/api/rules` | 获取规则列表 | ✅ 新增 |
| POST | `/api/rules` | 新增规则 | ✅ 新增 |
| PUT | `/api/rules/:index` | 编辑规则 | ✅ 新增 |
| DELETE | `/api/rules/:index` | 删除规则 | ✅ 新增 |
| POST | `/api/rules/convert` | 手动触发转换 | ✅ 新增 |
| GET | `/api/rules/status` | 查询转换状态 | ✅ 新增 |
| GET | `/api/settings/user-prompt` | 获取用户偏好提示词 | ✅ 新增 |
| PUT | `/api/settings/user-prompt` | 更新用户偏好提示词 | ✅ 新增 |
| POST | `/api/llm/chat` | AI 对话（扩展 mode 字段） | 扩展 |
| POST | `/api/optimize` | 指标调整（响应含 appliedRules） | 扩展 |

---

## 7. 关键设计决策

### 7.1 rules.md 解析策略

不引入复杂解析库，按行处理：
- 跳过空行和 `#` 开头标题行
- 正则 `^\d+\.\s+(.+)$` 提取规则文本
- 写回时重新生成整个文件（不做行内 patch），保证格式稳定一致

### 7.2 RuleSet 并发安全

Engine 持有 `*rules.RuleSet`，用 `sync.RWMutex` 保护：
- `ReloadRules` 持写锁（转换完成时触发，低频）
- `Optimize` 通过 `getRules()` 持读锁，获取快照后调用不再持锁

转换 goroutine 写状态走 SQLite 事务，天然串行，无额外锁需求。

### 7.3 Compensate 循环防护

`depth` 参数控制递归深度：
- `depth=0`：执行三个干预点（含 Compensate）
- `depth=1`：只执行 ClampTarget + FilterAllocation，跳过 Compensate

设计上两个指标不会构成互补环路（A 补偿 B，B 补偿 A），但 depth 机制作为兜底保障。

### 7.4 FilterAllocation 降级

过滤后企业列表为空时，回退全量企业参与分配，记录降级日志但对用户透明，确保调整不因规则设置不当而失败。

### 7.5 提示词三类隔离

| 提示词 | 存储位置 | 可修改 |
|--------|---------|--------|
| 对话内置提示词 | `system_prompt.go` const | 不可（代码维护） |
| 转换内置提示词 | `converter.go` 函数内 | 不可（代码维护） |
| 用户偏好提示词 | config 表 `llm_user_prompt` | 可配置 |

三类代码路径完全分离，互不影响。

### 7.6 role.json 不存在的降级行为

| 场景 | 行为 |
|------|------|
| role.json 不存在 | `rules.Load` 返回空 RuleSet，无任何规则约束，等同 001 行为 |
| 转换失败 | 保留旧 role.json，使用上一次成功版本 |
| 旧 role.json 也不存在且转换失败 | 空 RuleSet，无约束运行 |

---

## 8. 测试覆盖规划

| 模块 | 测试文件 | 覆盖重点 |
|------|---------|---------|
| rules/loader | `loader_test.go` | 各 type 分发正确；不存在文件返回空 RuleSet |
| rules/constraint | `constraint_test.go` | ClampTarget 边界值；Filter 各枚举；Compensate gte/lte |
| rules/filter | `filter_test.go` | filterByMode 各模式正确过滤；未知 mode 不过滤 |
| rules/converter | `converter_test.go` | extractJSON 有/无代码块；validateRoleJSON 各错误场景；多轮重试流程 |
| dagcalc/adjust | `adjust_rules_test.go` | ClampTarget 生效、Filter 过滤减少企业数、Compensate 触发；depth 防循环 |
| api/v3/rules | `rules_api_test.go` | CRUD 正常路径；convert 触发状态流转；status 字段正确 |
| llm/intent | `intent_test.go` | 调整意图正确提取 indicator+value；纯咨询返回空 Actions |
