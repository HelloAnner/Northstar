# rule-engine 模块 PRD-002

> 模块：规则引擎核心
> 版本：002
> 创建：2026-03-14
> 上级设计：docs/design/002/prd-002.md § 三、role.json 规则设计 + docs/design/002/tech-002.md § 2.1 / 2.3 / 2.4

---

## 一、模块职责

负责以下三件事：

1. **定义 role.json Schema**：三种规则类型（clamp_target / filter_allocation / compensate）的 JSON 结构
2. **加载 role.json → RuleSet**：`internal/rules` 包 loader + constraint + filter
3. **注入 dagcalc 分配流程**：在 `ApplyIndicatorTarget` 中实现三个干预点，使规则直接驱动随机分配

不负责：规则的 CRUD 管理（→ rule-management 模块）、LLM 转换（→ rule-convert 模块）。

---

## 二、详细设计

### 2.1 输入输出

**输入：**

- `config/role.json`
- `api/v3/optimize` 传入的目标指标和值
- 当前月份企业数据（批零 / 住餐）

**输出：**

- 内存态 `RuleSet`
- 调整完成后的企业数据
- `engine.Optimize()` / `/api/optimize` 返回的 `appliedRules`

### 2.2 规则类型

#### clamp_target

- 作用点：分配前
- 目标：裁剪非法或过激目标值
- 典型场景：`retail_month_rate` 不能高于 15

执行结果需要记录：

- 规则 ID
- 指标 ID
- 原目标值
- 裁剪后目标值

#### filter_allocation

- 作用点：分配中
- 目标：缩小参与分配的企业集合
- 典型场景：仅允许当前正增长企业参与零售指标调整

执行结果需要记录：

- 规则 ID
- 指标 ID
- 过滤前企业数
- 过滤后企业数
- 若过滤结果为空，自动降级为全量集合，同时不报错

#### compensate

- 作用点：分配后
- 目标：维护指标间关系
- 典型场景：`wholesale_month_rate >= retail_month_rate`

执行结果需要记录：

- 规则 ID
- 触发指标
- 被补偿指标
- 补偿目标值

### 2.3 运行流程

`engine.Optimize()` 对每个目标指标按固定顺序执行：

1. 加载当前 `RuleSet`
2. 执行 `ClampTargetConstraint`
3. 执行对应指标的实际调整
4. 在调整数据集时执行 `FilterAllocationConstraint`
5. 调整完成后重算指标快照
6. 若命中 `CompensateConstraint`，递归补偿一次
7. 返回 `groups + appliedRules`

### 2.4 降级策略

- `role.json` 不存在：返回空 `RuleSet`，引擎按 001 逻辑继续运行
- 未知规则类型：跳过，不阻塞加载
- 未知过滤枚举：不过滤
- 过滤后无企业：回退到原始企业集合
- 补偿递归深度超过 1：停止继续补偿，避免循环

### 2.5 非目标范围

- 不实现 `rules.md → role.json` 转换
- 不实现规则管理 UI
- 不实现 AI 对话模式
