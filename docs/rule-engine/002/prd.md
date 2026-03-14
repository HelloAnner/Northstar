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

> 待补充
