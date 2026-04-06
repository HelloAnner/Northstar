# 规则管理模���

> 模块：rule-management
> 状态：��完成
> 最后更���：2026-04-06

---

## 一、模块职责

管理两类规则的完整生命周期：

1. **硬约束 CRUD**：`/api/constraints` 增删改查，存入 `adjustment_constraints` 表
2. **自然语言规则 CRUD**：`/api/natural-rules` 增删改查，存入 `natural_rules` 表
3. **约束热重载**：硬约束变更后 `engine.ReloadRules()`
4. **前端规则页面**：硬约束表单 + 自然语言规则列表

不负责：规则如何驱动调整（→ rule-engine）、AI 对话界面���→ ai-chat）。

---

## 二、核心文件

| 文件 | 职责 |
|------|------|
| `internal/store/adjustment_constraints.go` | 硬约束表 CRUD |
| `internal/store/natural_rules.go` | 自然语言规则表 CRUD |
| `internal/api/v3/rules.go` | 两类规则的 REST API |
| `web/src/store/rulesStore.ts` | 前端状态管理 |
| `web/src/components/RuleList.tsx` | 规则管理 UI |

---

## 三、API

### 硬约束

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/constraints` | 列出所有硬约束 |
| POST | `/api/constraints` | 新增硬约束 |
| PUT | `/api/constraints/:id` | 更新硬约束 |
| DELETE | `/api/constraints/:id` | 删除硬约束 |

### 自然语言规则

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/natural-rules` | 列出所有自然语言规则 |
| POST | `/api/natural-rules` | 新增规则 |
| PUT | `/api/natural-rules/:id` | 更新规则 |
| DELETE | `/api/natural-rules/:id` | 删除规则 |

---

## 四、硬约束数据结构

```json
{
  "id": 1,
  "type": "clamp_target",
  "indicatorId": "wholesale_month_rate",
  "minValue": -30,
  "maxValue": 50,
  "filterMode": null,
  "triggerId": null,
  "ensureId": null,
  "relation": null,
  "tolerance": 0,
  "enabled": true
}
```

### 三种类型

| type | 必填字段 | 说明 |
|------|---------|------|
| `clamp_target` | indicatorId + (minValue 或 maxValue) | 目标值裁剪到 [min, max] |
| `filter_allocation` | indicatorId + filterMode | 过滤参与分配的企业 |
| `compensate` | triggerId + ensureId + relation | 指标间联动补偿 |

---

## 五、自然语言规则数据结构

```json
{
  "id": 1,
  "text": "如果零售连续两个月下降，优先调整大企业",
  "enabled": true
}
```

自然语言规则不做结构化转换，直接注入 AI 对话的系统提示词中，由 LLM 理解和遵守。

---

## 六、前端交互

### Settings 页 → 调整规则 Tab

两个区域：

1. **硬约束区**：表单式 CRUD
   - 选择约束类型（clamp / filter / compensate）
   - 选择指标、填写参数
   - 即时生效，无需转换

2. **自然语言规则区**：文本列表
   - 新增/编辑 Dialog（textarea）
   - 规则在 AI 对话时作为上下文生效
