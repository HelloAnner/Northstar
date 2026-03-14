# rule-management 模块 PRD-002

> 模块：规则管理（前端 + API + LLM 转换）
> 版本：002
> 创建：2026-03-14
> 上级设计：docs/design/002/prd-002.md § 四 + docs/design/002/tech-002.md § 2.1.4 / 2.5 / 5.1 / 5.3

---

## 一、模块职责

负责规则完整生命周期，从用户界面到文件持久化到 LLM 转换：

1. **后端 API**：`/api/rules` CRUD + `/api/rules/convert` 触发转换 + `/api/rules/status` 状态查询（`api/v3/rules.go`）
2. **rules.md 文件读写**：`rulesFileRepo`，按行解析编号列表，写回时重新生成整个文件
3. **LLM 异步转换**：`converter.go`，单次非流式调用 Claude API，最多 3 轮多轮校验重试
4. **JSON 提取与校验**：`extractJSON` + `validateRoleJSON`，校验 indicator ID / filter 枚举 / relation 合法性
5. **转换状态管理**：写 config 表 `rules_convert_status / at / error`
6. **成功后热重载**：调用 `engine.ReloadRules()`
7. **前端规则列表**：`RuleList.tsx` + `rulesStore.ts`，含转换状态 polling

不负责：规则如何驱动调整（→ rule-engine 模块）、AI 对话界面（→ ai-chat 模块）。

> 说明：原 rule-convert 模块因与 rule-management 耦合极高（每次 CRUD 均触发转换）合并至本模块统一实现。

---

## 二、详细设计

> 见 docs/design/002/tech-002.md § 2.1.4 / 2.5 / 5.1 / 5.3
