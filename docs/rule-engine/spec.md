# 规则引擎模块

> 模块：rule-engine
> 状态：已完成
> 最后更新：2026-04-06

---

## 一、模块职责

1. **定义硬约束类型**：两种约束（clamp_target / filter_allocation）+ 一种联动（compensate）
2. **从数据库加载约束 → RuleSet**：`internal/rules/` 包（loader + constraint + filter）
3. **注入 dagcalc 分配流程**：在 `ApplyIndicatorTarget` 中实现三个干预点

不负责：规则 CRUD（→ rule-management）、AI 对话（→ ai-chat）。

---

## 二、架构设计

### 三层规则体系

| 层级 | 类型 | 存储 | 执行方式 | 特点 |
|------|------|------|----------|------|
| 第一层 | 硬约束 | SQLite `adjustment_constraints` 表 | 确定性代码逻辑 | 每次调整自动生效，不依赖 LLM |
| 第二层 | 自然语言规则 | SQLite `natural_rules` 表 | 注入 LLM 系统提示词作为上下文 | 表达力强，由 LLM 理解和遵守 |
| 第三层 | Function Call | LLM 工具定义 | LLM 主动调用 | 直接操作引擎接口 |

### 与旧架构的区别

- **删除**：rules.md → LLM 转换 → role.json 的管道（converter.go）
- **删除**：文件读写（rulesFileRepo）、转换状态轮询
- **保留**：constraint.go（执行逻辑）、filter.go（过滤逻辑）、adjust_rules.go（三注入点）
- **变更**：loader.go 从数据库加载，不再读 JSON 文件

---

## 三、核心文件

| 文件 | 职�� |
|------|------|
| `internal/rules/loader.go` | 从数据库读取约束，构建 RuleSet |
| `internal/rules/constraint.go` | 三种 Constraint 类型定义及执行方法 |
| `internal/rules/filter.go` | `filterByMode` 四种过滤模式实现 |
| `internal/dagcalc/engine.go` | Engine 持有 RuleSet，提供 ReloadRules + Optimize |
| `internal/dagcalc/adjust.go` | 16 个指标调整函数 + 6 个底层分配函数 |
| `internal/dagcalc/adjust_rules.go` | adjustRuleScope 桥接规则与引擎 |

---

## 四、数据库表

### adjustment_constraints

```sql
CREATE TABLE IF NOT EXISTS adjustment_constraints (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    type TEXT NOT NULL,              -- clamp_target / filter_allocation / compensate
    indicator_id TEXT,               -- 目标指标 ID（clamp/filter 用）
    min_value REAL,                  -- 最小值（clamp 用）
    max_value REAL,                  -- 最大值（clamp 用）
    filter_mode TEXT,                -- 过滤模式（filter 用）
    trigger_id TEXT,                 -- 触发指标（compensate 用）
    ensure_id TEXT,                  -- 保障指标（compensate 用）
    relation TEXT,                   -- gte / lte（compensate 用）
    tolerance REAL DEFAULT 0,        -- 容差（compensate 用）
    enabled INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

---

## 五、三个注入点

### 1. ClampTarget（分配前）

遍历 `RuleSet.Clamps`，将目标值裁剪到 [min, max] 区间。记录原始值和裁剪后值。

### 2. FilterAllocation（分配中）

遍历 `RuleSet.Filters`，缩小参与分配的企业集合。6 个底层分配函数全覆盖。

### 3. Compensate（分配后）

��历 `RuleSet.Compensates`，检查关联指标关系。depth=0 时递归补偿一次，depth=1 时跳过（防循环）。

---

## 六、降级策略

| 场景 | 行为 |
|------|------|
| 约束表为空 | 返回空 RuleSet，无约束运行 |
| 未知规则类型 | 跳过，不阻塞加载 |
| filter 结果为空 | 回退全量企业集合 |
| compensate depth > 0 | 停止递归 |
