# 规则引擎模块

> 模块：rule-engine
> 状态：已完成
> 最后更新：2026-03-28

---

## 一、模块职责

1. **定义 role.json Schema**：三种规则类型（clamp_target / filter_allocation / compensate）
2. **加载 role.json → RuleSet**：`internal/rules/` 包（loader + constraint + filter）
3. **注入 dagcalc 分配流程**：在 `ApplyIndicatorTarget` 中实现三个干预点

不负责：规则 CRUD（→ rule-management）、LLM 转换（→ rule-management）、AI 对话（→ ai-chat）。

---

## 二、核心文件

| 文件 | 职责 |
|------|------|
| `internal/rules/loader.go` | 读取 role.json，按 type 分发构建 RuleSet |
| `internal/rules/constraint.go` | 三种 Constraint 类型定义及执行方法 |
| `internal/rules/filter.go` | `filterByMode` 四种过滤模式实现 |
| `internal/dagcalc/engine.go` | Engine 持有 RuleSet，提供 ReloadRules + Optimize |
| `internal/dagcalc/adjust.go` | 16 个指标调整函数 + 6 个底层分配函数 |
| `internal/dagcalc/adjust_rules.go` | adjustRuleScope 桥接规则与引擎 |

---

## 三、三个注入点

### 1. ClampTarget（分配前）

遍历 `RuleSet.Clamps`，将目标值裁剪到 [min, max] 区间。记录原始值和裁剪后值。

### 2. FilterAllocation（分配中）

遍历 `RuleSet.Filters`，缩小参与分配的企业集合。6 个底层分配函数全覆盖：
- `scaleAcrossWRAndACDerivedRetail` → `filterMixedRows`
- `scaleAcrossWRAndACDerivedRetailByCumulative` → `filterMixedRows`
- `scaleWRField` → `filterWRRows`
- `scaleWRFieldByCumulative` → `filterWRRows`
- `scaleACField` → `filterACRows`
- `scaleACFieldByCumulative` → `filterACRows`

### 3. Compensate（分配后）

遍历 `RuleSet.Compensates`，检查关联指标关系。depth=0 时递归补偿一次，depth=1 时跳过（防循环）。

---

## 四、降级策略

| 场景 | 行为 |
|------|------|
| role.json 不存在 | 返回空 RuleSet，无约束运行 |
| 未知规则类型 | 跳过，不阻塞加载 |
| filter 结果为空 | 回退全量企业集合 |
| compensate depth > 0 | 停止递归 |

---

## 五、测试覆盖

| 测试文件 | 覆盖 |
|---------|------|
| `rules/loader_test.go` | type 分发、空文件降级 |
| `rules/constraint_test.go` | ClampTarget 边界、Filter 枚举、Compensate gte/lte |
| `rules/filter_test.go` | 四种模式 + 未知 mode |
| `dagcalc/adjust_rules_test.go` | 三注入点生效、depth 防循环 |
| `dagcalc/engine_test.go` | ReloadRules、Optimize |
