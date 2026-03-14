# rule-engine 模块 E2E-002

> 模块：规则引擎核心
> 版本：002
> 创建：2026-03-14

---

## 一、验收目标

验证 `role.json` 已能驱动 `/api/optimize` 的实际调整过程，而不是仅做事后校验。

## 二、验收场景

### 场景 1：ClampTarget 生效

前置条件：

- `config/role.json` 中存在 `limitAbove_month_rate` 的 `clamp_target`
- 最大值设置为 `15`

步骤：

1. 调用 `/api/optimize`，传入 `limitAbove_month_rate=20`
2. 查看返回 `appliedRules`
3. 重新查询指标

期望：

- `appliedRules` 中存在 `clamp_target` 记录
- 记录中原值为 `20`，生效值为 `15`
- 最终指标结果接近 `15`，而不是 `20`

### 场景 2：FilterAllocation 生效

前置条件：

- `config/role.json` 中存在 `retail_month_rate` 的 `filter_allocation`
- `filter=positive_current`
- 当前月份存在正增长和负增长企业

步骤：

1. 调用 `/api/optimize`，传入 `retail_month_rate`
2. 查看返回 `appliedRules`
3. 核对被调整企业集合

期望：

- `appliedRules` 中存在 `filter_allocation` 记录
- 记录中 `beforeCount > afterCount`
- 仅正增长企业被调整

### 场景 3：Compensate 生效

前置条件：

- `config/role.json` 中存在 `retail_month_rate -> wholesale_month_rate` 的 `compensate`
- 关系为 `gte`

步骤：

1. 调用 `/api/optimize`，拉高 `retail_month_rate`
2. 查看返回 `appliedRules`
3. 重新查询批发、零售指标

期望：

- `appliedRules` 中存在 `compensate` 记录
- 调整后 `wholesale_month_rate >= retail_month_rate`

### 场景 4：无规则文件降级运行

前置条件：

- `config/role.json` 不存在

步骤：

1. 启动服务
2. 调用 `/api/optimize`

期望：

- 接口正常返回 200
- 不因缺少规则文件报错
- `appliedRules` 为空数组或缺省

## 三、通过标准

- 四个场景全部通过
- 所有新增单元测试通过
- 不影响 001 既有智能调整能力

## 四、自动化执行

- 后端闭环验收：`go test -tags=e2e -v ./tests/e2e -run TestRuleEnginePhase1E2E`
- 该用例覆盖 `role.json -> engine.ReloadRules() -> /api/optimize -> /api/indicators` 的完整链路
- `make test-e2e` 会先执行这组 Phase 1 后端 E2E，再执行既有的 agent-browser UI 主流程
