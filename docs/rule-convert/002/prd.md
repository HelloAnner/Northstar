# rule-convert 模块 PRD-002

> 模块：规则转换（已合并至 rule-management）
> 版本：002
> 创建：2026-03-14

---

## 说明

本模块（rule-convert）的全部职责已合并至 **rule-management** 模块统一实现。

原因：`converter.go` 由 `rulesFileRepo` 的每次 CRUD 操作触发，二者紧密耦合，拆分会造成不必要的模块边界。

请参考：`docs/rule-management/002/prd.md`
