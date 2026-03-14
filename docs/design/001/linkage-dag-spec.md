# 联动预览 DAG 规格

> 目标：明细表全部字段、16 指标、汇总/社零额坐标全部纳入 DAG，用于“点击位置 → 返回全量受影响坐标 → 黄色边框标记”。

## 1. 节点命名规则

- 企业节点：`wr:{id}:{field}` / `ac:{id}:{field}`
- 行业聚合：`industry:{type}:{metric}`
- 全局聚合：`aggregate:{metric}`
- 指标节点：`indicator:{id}`

## 2. 明细表字段（UI 全覆盖）

> 以下 ColumnKey 全部是 DAG 节点（含可编辑、衍生列、标记列）。

- companyScale / flags / sourceSheet
- salesPrevMonth / salesCurrentMonth / salesLastYearMonth
- salesYoYDiff / salesMoMDiff / salesMoMRate / salesMonthRate
- salesCurrentCumulative / salesLastYearCumulative
- salesCumulativeYoYDiff / salesCumulativeRate
- retailPrevMonth / retailCurrentMonth / retailLastYearMonth
- retailYoYDiff / retailMoMDiff / retailMoMRate / retailMonthRate
- retailCurrentCumulative / retailLastYearCumulative
- retailCumulativeYoYDiff / retailCumulativeRate
- retailRatio

## 3. 住餐扩展字段（非 UI 列）

> 仅用于 Excel 坐标映射，不展示在前端表格。

- roomPrevMonth / roomCurrentMonth / roomLastYearMonth
- roomCurrentCumulative / roomLastYearCumulative
- foodPrevMonth / foodCurrentMonth / foodLastYearMonth
- foodCurrentCumulative / foodLastYearCumulative
- goodsPrevMonth / goodsCurrentMonth / goodsLastYearMonth
- goodsCurrentCumulative / goodsLastYearCumulative

## 4. 行业/汇总节点定义

### 4.1 行业聚合节点

`industry:{type}:{metric}` 中 metric 列表如下：

- salesCurSum / salesLastSum / salesCurCumSum / salesLastCumSum
- retailCurSum / retailLastSum / retailCurCumSum / retailLastCumSum
- salesMonthRate / salesCumulativeRate

### 4.2 全局汇总节点

`aggregate:{metric}` 中 metric 列表如下：

- limitAboveRetailCurSum / limitAboveRetailLastSum
- limitAboveRetailCurCumSum / limitAboveRetailLastCumSum
- limitAboveRetailMonthRate / limitAboveRetailCumulativeRate
- eatWearUseRetailCurSum / eatWearUseRetailLastSum
- microSmallRetailCurSum / microSmallRetailLastSum

## 5. 指标 ID（16 项）

- limitAbove_month_value / limitAbove_month_rate
- limitAbove_cumulative_value / limitAbove_cumulative_rate
- eatWearUse_month_rate / microSmall_month_rate
- wholesale_month_rate / wholesale_cumulative_rate
- retail_month_rate / retail_cumulative_rate
- accommodation_month_rate / accommodation_cumulative_rate
- catering_month_rate / catering_cumulative_rate
- totalSocial_cumulative_value / totalSocial_cumulative_rate

## 6. Excel 坐标映射

### 6.1 行号定位规则

- 模板表读取列 C 的行业码作为行索引，依次扫描至空行结束。
- 相同行业码按 `RowNo` 再按 `ID` 排序，依序绑定模板行。
- 支持表：批零总表 / 批发 / 零售 / 住餐总表 / 住宿 / 餐饮。

### 6.2 批零模板列（WR）

| 字段 | 列 |
| --- | --- |
| salesCurrentMonth | D |
| salesLastYearMonth | E |
| salesMonthRate | F |
| salesCurrentCumulative | G |
| salesLastYearCumulative | H |
| salesCumulativeRate | I |
| retailCurrentMonth | J |
| retailLastYearMonth | K |
| retailMonthRate | L |
| retailCurrentCumulative | M |
| retailLastYearCumulative | N |
| retailCumulativeRate | O |

### 6.3 住餐模板列（AC）

| 字段 | 列 |
| --- | --- |
| salesCurrentMonth | D |
| salesLastYearMonth | E |
| salesMonthRate | F |
| salesCurrentCumulative | G |
| salesLastYearCumulative | H |
| salesCumulativeRate | I |
| roomCurrentMonth | J |
| roomLastYearMonth | K |
| roomCurrentCumulative | L |
| roomLastYearCumulative | M |
| foodCurrentMonth | N |
| foodLastYearMonth | O |
| foodCurrentCumulative | P |
| foodLastYearCumulative | Q |
| goodsCurrentMonth | R |
| goodsLastYearMonth | S |
| goodsCurrentCumulative | T |
| goodsLastYearCumulative | U |
| retailCurrentMonth | V |
| retailLastYearMonth | W |
| retailCurrentCumulative | X |
| retailLastYearCumulative | Y |

### 6.4 指标坐标

- 汇总表（定）
  - limitAbove_month_value: G4
  - limitAbove_month_rate: S4
  - limitAbove_cumulative_value: I4
  - limitAbove_cumulative_rate: T4
  - wholesale_month_rate: K4
  - wholesale_cumulative_rate: L4
  - retail_month_rate: M4
  - retail_cumulative_rate: N4
  - accommodation_month_rate: O4
  - accommodation_cumulative_rate: P4
  - catering_month_rate: Q4
  - catering_cumulative_rate: R4
  - eatWearUse_month_rate: U4
  - microSmall_month_rate: V4
  - totalSocial_cumulative_value: N10
  - totalSocial_cumulative_rate: S10
- 社零额（定）
  - eatWearUse_month_rate: C4
  - microSmall_month_rate: B4
- 批发/零售/住宿/餐饮
  - 指标增速行：`maxRow + 2`，E 列（当月）/ H 列（累计）
  - 批发全量指标行：`maxRow + 3`/`maxRow + 4`（批发总计与增速）

### 6.5 行业/汇总坐标

- 行业合计行：`sumRow = maxRow + 1`，增速行：`growthRow = maxRow + 2`
- 批发汇总行：`totalRow = whGrowthRow + 3`，增速行：`totalGrowthRow = whGrowthRow + 4`
- 行业合计（批零样式）
  - 销售额：D/E/G/H，零售额：J/K/M/N
  - 增速：E（当月）/ H（累计）
- 行业合计（住餐样式）
  - 销售额：D/E/G/H，零售额：V/W/X/Y
  - 增速：E（当月）/ H（累计）
- 批发汇总（限上零售额）
  - limitAboveRetailCurSum: J
  - limitAboveRetailLastSum: K
  - limitAboveRetailCurCumSum: M
  - limitAboveRetailLastCumSum: N
  - limitAboveRetailMonthRate: K（增速行）
  - limitAboveRetailCumulativeRate: N（增速行）
