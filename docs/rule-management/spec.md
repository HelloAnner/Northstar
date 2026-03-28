# 规则管理模块

> 模块：rule-management
> 状态：已完成
> 最后更新：2026-03-28

---

## 一、模块职责

负责规则完整生命周期：

1. **后端 API**：`/api/rules` CRUD + `/api/rules/convert` 触发 + `/api/rules/status` 状态查询
2. **rules.md 文件读写**：`rulesFileRepo` 按行解析编号列表，写回时重新生成整个文件
3. **LLM 异步转换**：`converter.go` 最多 3 轮多轮校验重试
4. **JSON 提取与校验**：`extractJSON` + `validateRoleJSON`
5. **转换状态管理**：config 表 `rules_convert_status / at / error`
6. **成功后热重载**：`engine.ReloadRules()`
7. **前端规则列表**：RuleList + rulesStore + Toast 通知链
8. **聊天添加规则**：`addRuleFromChat()` 与页面添加共用底层逻辑

> 原 rule-convert 模块因耦合已合并至本模块。

不负责：规则如何驱动调整（→ rule-engine）、AI 对话界面（→ ai-chat）。

---

## 二、核心文件

| 文件 | 职责 |
|------|------|
| `internal/rules/converter.go` | LLM 转换、3 轮重试、extractJSON、validateRoleJSON |
| `internal/api/v3/rules.go` | CRUD API + rulesFileRepo + triggerRuleConvert |
| `web/src/store/rulesStore.ts` | Zustand store，CRUD + 轮询 + Toast 通知 |
| `web/src/components/RuleList.tsx` | 规则列表 UI + Dialog 编辑 + 状态徽标 |

---

## 三、API

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/rules` | 返回规则列表（从 rules.md 解析） |
| POST | `/api/rules` | 新增规则，触发异步转换 |
| PUT | `/api/rules/:index` | 编辑第 index 条，触发异步转换 |
| DELETE | `/api/rules/:index` | 删除并重新编号，触发异步转换 |
| POST | `/api/rules/convert` | 手动触发 rules.md → role.json 转换 |
| GET | `/api/rules/status` | 查询转换状态（status/updatedAt/error） |

---

## 四、转换流程

```
CRUD 操作 → 写入 rules.md → ConvertAsync(全文)
  → goroutine 启动，status=running
  → LLM 调用（system prompt + rules.md 内容）
  → extractJSON 提取 JSON
  → validateRoleJSON 校验
    ├─ 校验通过 → 写 role.json → engine.ReloadRules() → status=ok
    └─ 校验失败 → 追加错误反馈，重试（最多 3 轮）
      └─ 3 轮失败 → status=error，保留旧 role.json
```

### 校验矩阵

| 规则类型 | 校验项 |
|---------|--------|
| `clamp_target` | indicator 在 16 项枚举中；min/max 不能同时为 null |
| `filter_allocation` | indicator 在枚举中；filter 在 4 个枚举值中 |
| `compensate` | trigger/ensure 在枚举中；relation 为 gte/lte |

---

## 五、前端交互

### RuleList 组件

- 列表展示所有规则（编号 + 文本 + 编辑/删除按钮）
- 状态徽标：待转换(outline) / 转换中(secondary) / 已生效(default) / 转换失败(destructive)
- 新增/编辑使用 Dialog + Textarea（有 placeholder 灰色提示）
- 手动"重新转换"和"刷新状态"按钮

### Toast 通知链

| 时机 | 内容 |
|------|------|
| CRUD 完成 | `规则已新增` / `规则已更新` / `规则已删除` |
| 轮询开始 | `正在转换规则为 JSON…` |
| status → ok | `JSON 规则转换成功，规则已生效` |
| status → error | `规则转换失败：{error}` |

### rulesStore

Zustand vanilla store，CRUD 后自动设置 status=running 并启动 2s 间隔轮询，status 变为 ok/error 时自动停止。

---

## 六、聊天添加规则

`addRuleFromChat(ruleText)` 在 `llm_chat.go` 中实现：
1. 读取当前 rules.md
2. 追加新规则
3. 写回 rules.md
4. 触发异步转换
5. 返回 `ruleAddedPayload{text, status:"converting"}`

与页面 CRUD 共用同一套 `rulesFileRepo` + `triggerRuleConvert`。

---

## 七、测试覆盖

| 测试文件 | 覆盖 |
|---------|------|
| `rules/converter_test.go` | extractJSON、validateRoleJSON、多轮重试 |
| `api/v3/rules_test.go` | CRUD 正常路径、convert 触发、status 流转 |
| `web/src/store/rulesStore.test.ts` | 加载规则、轮询停止 |
| `web/src/components/RuleList.test.tsx` | 渲染规则、状态展示 |
