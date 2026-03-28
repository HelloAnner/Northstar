# Northstar 产品设计

> 作者：Anner
> 创建：2026-03-14
> 最后更新：2026-03-28

---

## 一、产品定位

Northstar 是月度经济数据统计系统，管理批发、零售、住宿、餐饮四大行业的月度指标数据。核心能力是**规则驱动的联动指标调整**：用户用自然语言定义约束规则，系统自动将规则转换为结构化 JSON，引擎在分配过程中实时执行规则。

---

## 二、核心架构

```
用户自然语言规则 (rules.md)
       ↓  LLM 转换（3轮重试 + 校验）
结构化规则 (role.json)
       ↓  加载为可执行对象
引擎规则集 (RuleSet)
       ↓  注入 dagcalc 分配算法
调整结果（三个注入点：clamp → filter → compensate）
       ↑              ↑              ↑
   手动调整       AI 调整       聊天添加规则
```

**关键点：** 规则不是事后校验，而是在 `ApplyIndicatorTarget` 的分配过程中实时生效。

---

## 三、role.json 规则设计

### 3.1 三种规则类型

| 类型 | 干预时机 | 说明 | 示例 |
|------|----------|------|------|
| `clamp_target` | 分配**前** | 裁剪目标值到合理区间 | 限上增速不超过 15% |
| `filter_allocation` | 分配**中** | 过滤参与分配的企业集合 | 只调正增长企业 |
| `compensate` | 分配**后** | 自动触发关联指标补偿 | 批发增速不低于零售 |

### 3.2 role.json Schema

```json
{
  "version": "1.0",
  "updatedAt": "2026-03-14T10:00:00Z",
  "sourceFile": "config/rules.md",
  "rules": [
    {
      "id": "rule_001",
      "name": "限上增速上限 15%",
      "type": "clamp_target",
      "indicator": "limitAbove_month_rate",
      "min": null,
      "max": 15.0
    },
    {
      "id": "rule_002",
      "name": "零售正增长过滤",
      "type": "filter_allocation",
      "indicator": "retail_month_rate",
      "filter": "positive_current"
    },
    {
      "id": "rule_003",
      "name": "批发补偿零售",
      "type": "compensate",
      "trigger": "retail_month_rate",
      "ensure": "wholesale_month_rate",
      "relation": "gte",
      "tolerance": 0.0
    }
  ]
}
```

### 3.3 指标 ID 枚举（16 项）

```
limitAbove_month_value / limitAbove_month_rate
limitAbove_cumulative_value / limitAbove_cumulative_rate
eatWearUse_month_rate / microSmall_month_rate
wholesale_month_rate / wholesale_cumulative_rate
retail_month_rate / retail_cumulative_rate
accommodation_month_rate / accommodation_cumulative_rate
catering_month_rate / catering_cumulative_rate
totalSocial_cumulative_value / totalSocial_cumulative_rate
```

### 3.4 filter 枚举

| 值 | 含义 |
|----|------|
| `positive_current` | 仅当前值 > 0 的企业 |
| `negative_current` | 仅当前值 < 0 的企业 |
| `large_scale_only` | 仅大型企业（company_scale=1） |
| `exclude_small_micro` | 排除小微企业 |

---

## 四、规则管理

### 4.1 rules.md 格式

```markdown
# 调整规则

1. 限上社零额当月增速不超过 15%
2. 批发业当月增速不低于 -10%，不高于 20%
3. 调整零售增速时，仅分配给当前已是正增长的企业
```

### 4.2 两条添加路径

| 路径 | 入口 | 流程 |
|------|------|------|
| 页面手动添加 | Settings 页 RuleList → "新增规则" | 输入文本 → 写入 rules.md → 异步 LLM 转换 → Toast 通知链 |
| 聊天添加 | ChatPanel 对话 | 用户说"加一条规则" → 意图识别为 add_rule → 写入 rules.md → SSE 推送状态 |

两条路径底层共用同一套 `rulesFileRepo` + `triggerRuleConvert`，数据完全共通。

### 4.3 转换流程

```
用户操作（页面/聊天）→ 写入 rules.md → 异步 LLM 转换（最多 3 轮重试）
     → 校验通过 → 写入 role.json → engine.ReloadRules() 热重载
     → 校验失败 → 保留旧 role.json，status=error
```

### 4.4 Toast 通知链

添加操作后前端依次展示：
1. `规则已新增` — 立即
2. `正在转换规则为 JSON…` — CRUD 后
3. `JSON 规则转换成功，规则已生效` — 轮询 status=ok 时
4. `规则转换失败：{error}` — 轮询 status=error 时

---

## 五、系统提示词

### 5.1 双层设计

| 层级 | 存储 | 可修改 | 作用 |
|------|------|--------|------|
| 内置层 | `system_prompt.go` const | 否 | 系统角色、指标上下文、行为约束 |
| 用户层 | config 表 `llm_user_prompt` | 是（500 字限制） | 回复风格、分析偏好 |

拼接方式：`内置层 + --- + 用户层`（用户层为空时不拼接）。

---

## 六、AI 调整

### 6.1 三种意图类型

| type | 含义 | 示例 |
|------|------|------|
| `set_target` | 设定绝对目标值 | "把批发增速调到 15%" |
| `adjust_percent` | 相对百分比调整 | "帮我将零售增速随机调整 5%" |
| `add_rule` | 添加持久规则 | "批发增速不能超过 20%" |

### 6.2 执行流程

```
用户输入 → ParseIntent（识别意图）
  ├─ set_target / adjust_percent → 转为 targets → engine.Optimize() → 返回结果 + appliedRules
  ├─ add_rule → addRuleFromChat() → 写入 rules.md + 触发转换 → 返回规则添加状态
  └─ 无调整意图 → 降级为普通 chat 模式
```

---

## 七、降级策略

| 场景 | 行为 |
|------|------|
| role.json 不存在 | 空 RuleSet，无约束运行（等同无规则） |
| 转换失败 | 保留旧 role.json |
| 未知规则类型 | 跳过，不阻塞加载 |
| filter 结果为空 | 回退全量企业 |
| compensate 递归 | depth=1 时停止 |

---

## 八、接口汇总

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/rules` | 获取规则列表 |
| POST | `/api/rules` | 新增规则 |
| PUT | `/api/rules/:index` | 编辑规则 |
| DELETE | `/api/rules/:index` | 删除规则 |
| POST | `/api/rules/convert` | 手动触发转换 |
| GET | `/api/rules/status` | 查询转换状态 |
| GET | `/api/settings/user-prompt` | 获取用户偏好提示词 |
| PUT | `/api/settings/user-prompt` | 更新用户偏好提示词 |
| POST | `/api/llm/chat/stream` | AI 对话（SSE，支持 mode=chat/adjust） |
| POST | `/api/optimize` | 指标调整（响应含 appliedRules） |
