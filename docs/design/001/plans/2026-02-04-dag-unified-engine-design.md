# 统一 DAG 计算引擎设计

日期：2026-02-04

## 目标
- 底层计算、联动修改、影响范围高亮使用同一套 DAG 规则。
- 行业/指标修改可反向调整父节点（随机分配），再正向重算子节点。
- 明细字段修改只做正向重算，同时高亮全部受影响父/子节点。

## 范围与约束（来自确认偏好）
- 企业字段修改：只正向重算后代。
- 行业/汇总/指标修改：反向随机分配父节点 + 正向重算子节点。
- 反向随机只作用于 L0/L1 原始字段，不锁定任何字段或企业。
- 参与企业范围：目标所涉及行业的全部企业；跨行业指标覆盖全部相关行业。
- 约束：仅保证非负；无单企业调整上限；随机可每次不同。
- 高亮范围：父节点 + 子节点全量纳入；单企业修改时也高亮全部受影响汇总/指标节点。
- 增速类目标（当月）：只调整当月值，上年同期不动。
- 增速类目标（累计）：只调整本年累计（通过调整当月值），历史月份不动。
- 派生差值/比值字段被编辑：只调整本年本月值。
- 累计值类指标被编辑：直接调整当月值，再由“累计=上月累计+当月”推导。
- 多父节点指标：同时调整父链路；按“改动量最小化”自动分配。
- 最小化目标：加权最小化（行业占比 + 企业规模 50/50）。

## 总体架构
新增独立包 `internal/dagcalc`，统一管理：
1) DAG 图构建
2) 正向计算
3) 反向调整
4) 影响范围
5) 坐标映射

现有模块改造为“调用壳”：
- `internal/linkage`: 预览高亮改为调用 `dagcalc.ImpactRange`。
- `internal/calculator`: 指标计算改为调用 `dagcalc.ForwardRecalc` 结果。
- `internal/api/v3/companies.go`: 替换 `recalcDerivedFields`，使用 `dagcalc` 执行正向重算；新增反向入口用于行业/指标修改。
- `internal/exporter`: 统一使用 `dagcalc` 计算结果填表。

## DAG 节点模型
- L0/L1：企业字段（WR/AC + 派生字段），与 UI ColumnKey 对齐。
- L2：行业聚合节点（行业 × 指标口径）。
- L3：全局汇总与 16 指标节点。
- L4：输出节点（Excel/汇总表/社零额输入区）。

节点属性：
- `NodeID`、`NodeType`、`Level`、`Formula`、`Parents/Children`、`CoordMapping`。

## 规则系统
### 正向规则（ForwardRule）
- 累计：`currentCumulative = prevCumulative + currentMonth`。
- 住餐零售：`retail = food + goods`。
- 增速：`(current - lastYear) / lastYear * 100`。
- 行业汇总：按行业聚合字段求和。
- 全局汇总与 16 指标：沿用现有指标口径。

### 反向规则（ReverseRule）
- 目标为增速类（当月）：只调整当月值。
- 目标为累计增速类：只调整本年累计（通过当月值实现）。
- 目标为值/汇总类：按权重随机分配调整量。
- 目标为派生差值/比值类：只调整当月值。
- 多父节点指标：同时调整，自动最小化改动量。

## 反向随机分配算法
1) 目标分解：解析目标节点依赖链，得到可调父节点集合。
2) 计算调整总量：目标值 - 当前值。
3) 计算权重：行业占比 + 企业规模（50/50）。
4) 随机分配：
   - 生成随机向量 r_i；
   - Δ_i = totalDelta * normalize(r_i * weight_i)；
   - 仅约束非负（若出现负值，截断并重新归一）。
5) 写回父节点计划，正向重算子节点。

## 引擎接口（拟）
- `BuildGraph(ctx) *Graph`
- `ForwardRecalc(anchor NodeID) Plan`
- `ReverseAdjust(target NodeID, newValue float64) Plan`
- `ImpactRange(anchor NodeID) []ImpactNode`
- `ApplyPlan(plan Plan) error`

## 分阶段落地
1) 建 `internal/dagcalc` 基础设施，覆盖 L0/L1→L3；与现有计算做对照测试。
2) 迁移 `linkage` 预览与 `calculator` 指标计算。
3) 接入 `api/v3` 更新流程与导出逻辑。
4) 补齐 L4 输出节点与社零额输入区。

