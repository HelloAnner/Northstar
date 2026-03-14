# rule-convert 模块 PRD-002

> 模块：规则转换（rules.md → role.json）
> 版本：002
> 创建：2026-03-14
> 上级设计：docs/design/002/prd-002.md § 七.1 + docs/design/002/tech-002.md § 2.1.4

---

## 一、模块职责

负责将用户自然语言规则文件（`config/rules.md`）通过大模型转换为结构化规则文件（`config/role.json`）：

1. **LLM 异步调用**：`converter.go`，单次非流式调用 Claude API
2. **多轮校验重试**：最多 3 轮，每轮将具体错误反馈给 LLM 让其修正
3. **JSON 提取与校验**：`extractJSON` + `validateRoleJSON`，校验 indicator ID / filter 枚举 / relation 合法性
4. **转换状态管理**：写 config 表 `rules_convert_status / at / error`
5. **成功后热重载**：调用 `engine.ReloadRules()`

不负责：规则的前端 CRUD（→ rule-management 模块）、规则如何驱动调整（→ rule-engine 模块）。

---

## 二、详细设计

> 待补充
