# rule-management 模块 PRD-002

> 模块：规则管理（前端 + API）
> 版本：002
> 创建：2026-03-14
> 上级设计：docs/design/002/prd-002.md § 四、规则管理模块 + docs/design/002/tech-002.md § 2.5 / 5.1 / 5.3

---

## 一、模块职责

负责用户对自然语言规则的增删改查，以及规则文件的持久化：

1. **后端 API**：`/api/rules` CRUD + `/api/rules/convert` 触发转换 + `/api/rules/status` 状态查询（`api/v3/rules.go`）
2. **rules.md 文件读写**：`rulesFileRepo`，按行解析编号列表，写回时重新生成整个文件
3. **前端规则列表**：`RuleList.tsx` + `rulesStore.ts`，含转换状态 polling

不负责：role.json 的 LLM 转换逻辑（→ rule-convert 模块）、规则如何驱动调整（→ rule-engine 模块）。

---

## 二、详细设计

> 待补充
