# 指标计算联动逻辑设计

## 1 指标总览

系统共包含 **16 个核心指标**，分为 4 组：

| 组别 | 指标名称 | 指标ID | 计算依赖 |
|------|----------|--------|----------|
| 限上社零额 | 限上社零额(当月值) | `limitAbove_month_value` | 所有企业 |
| | 限上社零额增速(当月) | `limitAbove_month_rate` | 所有企业 |
| | 限上社零额(累计值) | `limitAbove_cumulative_value` | 所有企业 |
| | 限上社零额增速(累计) | `limitAbove_cumulative_rate` | 所有企业 |
| 专项增速 | 吃穿用增速(当月) | `eatWearUse_month_rate` | 特定行业企业 |
| | 小微企业增速(当月) | `microSmall_month_rate` | 规模3/4企业 |
| 四大行业增速 | 批发业销售额增速(当月) | `wholesale_month_rate` | 批发行业 |
| | 批发业销售额增速(累计) | `wholesale_cumulative_rate` | 批发行业 |
| | 零售业销售额增速(当月) | `retail_month_rate` | 零售行业 |
| | 零售业销售额增速(累计) | `retail_cumulative_rate` | 零售行业 |
| | 住宿业营业额增速(当月) | `accommodation_month_rate` | 住宿行业 |
| | 住宿业营业额增速(累计) | `accommodation_cumulative_rate` | 住宿行业 |
| | 餐饮业营业额增速(当月) | `catering_month_rate` | 餐饮行业 |
| | 餐饮业营业额增速(累计) | `catering_cumulative_rate` | 餐饮行业 |
| 社零总额 | 社零总额(累计值) | `totalSocial_cumulative_value` | 限上累计 + 限下估算 |
| | 社零总额增速(累计) | `totalSocial_cumulative_rate` | 限上 + 限下 |

---

## 2 数据字段定义

### 2.1 企业基础字段

```typescript
interface Company {
  id: string                        // 企业唯一标识
  name: string                      // 企业名称
  industryCode: string              // 行业代码
  industryType: IndustryType        // 行业类型 (批发/零售/住宿/餐饮)
  companyScale: number              // 单位规模 (1/2/3/4)
  isEatWearUse: boolean             // 是否属于吃穿用类

  // 零售额相关
  retailLastYearMonth: number       // 上年同期零售额
  retailCurrentMonth: number        // 本期零售额 [可编辑]
  retailLastYearCumulative: number  // 上年累计零售额
  retailCurrentCumulative: number   // 本年累计零售额 [计算值]

  // 销售额/营业额相关
  salesLastYearMonth: number        // 上年同期销售额/营业额
  salesCurrentMonth: number         // 本期销售额/营业额
  salesLastYearCumulative: number   // 上年累计销售额/营业额
  salesCurrentCumulative: number    // 本年累计销售额/营业额
}

enum IndustryType {
  WHOLESALE = 'wholesale'      // 批发
  RETAIL = 'retail'            // 零售
  ACCOMMODATION = 'accommodation'  // 住宿
  CATERING = 'catering'        // 餐饮
}
```

### 2.2 配置字段

```typescript
interface Config {
  lastYearLimitBelowCumulative: number  // 上年累计限下社零额 (手动输入)
  currentMonth: number                   // 当前操作月份 (1-12)
}
```

---

## 3 计算公式详解

### 3.1 指标组一：限上社零额 (4个指标)

```
指标1: 限上社零额(当月值)
  = SUM(所有企业.retailCurrentMonth)

指标2: 限上社零额增速(当月)
  = (SUM(retailCurrentMonth) - SUM(retailLastYearMonth)) / SUM(retailLastYearMonth)
  = (指标1 - SUM(retailLastYearMonth)) / SUM(retailLastYearMonth)

指标3: 限上社零额(累计值)
  = SUM(所有企业.retailCurrentCumulative)

指标4: 限上社零额增速(累计)
  = (SUM(retailCurrentCumulative) - SUM(retailLastYearCumulative)) / SUM(retailLastYearCumulative)
```

### 3.2 指标组二：专项增速 (2个指标)

```
指标5: 吃穿用增速(当月)
  企业筛选: isEatWearUse = true
  = (SUM(筛选企业.retailCurrentMonth) - SUM(筛选企业.retailLastYearMonth))
    / SUM(筛选企业.retailLastYearMonth)

指标6: 小微企业增速(当月)
  企业筛选: companyScale IN (3, 4)
  = (SUM(筛选企业.retailCurrentMonth) - SUM(筛选企业.retailLastYearMonth))
    / SUM(筛选企业.retailLastYearMonth)
```

### 3.3 指标组三：四大行业增速 (8个指标)

以**零售业**为例：
```
指标9: 零售业销售额增速(当月)
  企业筛选: industryType = RETAIL
  = (SUM(筛选企业.salesCurrentMonth) - SUM(筛选企业.salesLastYearMonth))
    / SUM(筛选企业.salesLastYearMonth)

指标10: 零售业销售额增速(累计)
  企业筛选: industryType = RETAIL
  = (SUM(筛选企业.salesCurrentCumulative) - SUM(筛选企业.salesLastYearCumulative))
    / SUM(筛选企业.salesLastYearCumulative)
```

其他行业 (批发/住宿/餐饮) 同理计算。

### 3.4 指标组四：社零总额 (2个指标)

```
指标15: 社零总额(累计值)
  第一步: 估算本年累计限下社零额
    estimatedLimitBelowCumulative = config.lastYearLimitBelowCumulative × (1 + 指标6)

  第二步: 计算社零总额
    = 指标3 + estimatedLimitBelowCumulative

指标16: 社零总额增速(累计)
  第一步: 计算上年社零总额(累计)
    lastYearTotalCumulative = SUM(retailLastYearCumulative) + config.lastYearLimitBelowCumulative

  第二步: 计算累计增速
    = (指标15 - lastYearTotalCumulative) / lastYearTotalCumulative
```

---

## 4 联动依赖图

```
                    ┌─────────────────────────────────────────────────────────┐
                    │              企业数据变更                                │
                    │       retailCurrentMonth 修改                           │
                    └───────────────────────┬─────────────────────────────────┘
                                            │
                    ┌───────────────────────┼───────────────────────┐
                    │                       │                       │
                    ▼                       ▼                       ▼
            ┌───────────────┐       ┌───────────────┐       ┌───────────────┐
            │ 更新该企业的   │       │ 筛选条件判断   │       │ 重新计算       │
            │ cumulative    │       │               │       │ 所有SUM值     │
            └───────┬───────┘       └───────┬───────┘       └───────┬───────┘
                    │                       │                       │
                    │              ┌────────┴────────┐              │
                    │              │                 │              │
                    │              ▼                 ▼              │
                    │      ┌─────────────┐   ┌─────────────┐       │
                    │      │ 是否小微企业 │   │ 是否吃穿用  │       │
                    │      │ (scale 3/4) │   │             │       │
                    │      └──────┬──────┘   └──────┬──────┘       │
                    │             │                 │              │
                    ▼             ▼                 ▼              ▼
            ┌───────────────────────────────────────────────────────────────┐
            │                    指标计算引擎                               │
            └───────────────────────────┬───────────────────────────────────┘
                                        │
        ┌───────────┬───────────┬───────┴───────┬───────────┬───────────┐
        │           │           │               │           │           │
        ▼           ▼           ▼               ▼           ▼           ▼
   ┌─────────┐ ┌─────────┐ ┌─────────┐   ┌─────────┐ ┌─────────┐ ┌─────────┐
   │ 指标1-4 │ │ 指标5   │ │ 指标6   │   │ 指标7-14│ │ 指标15  │ │ 指标16  │
   │ 限上    │ │ 吃穿用  │ │ 小微    │   │ 四大行业│ │ 总额值  │ │ 总额率  │
   └────┬────┘ └─────────┘ └────┬────┘   └─────────┘ └────┬────┘ └────┬────┘
        │                       │                         │           │
        │                       └────────────┬────────────┘           │
        │                                    │                        │
        │                                    ▼                        │
        │                        ┌───────────────────────┐            │
        │                        │ 指标6 影响 指标15/16 │            │
        │                        │ (限下估算依赖小微增速)│            │
        │                        └───────────────────────┘            │
        │                                                             │
        └─────────────────────────────────────────────────────────────┘
```

---

## 5 联动规则矩阵

当某个企业的 `retailCurrentMonth` 发生变化时，需要重新计算的指标：

| 企业类型 | 指标1-4 | 指标5 | 指标6 | 指标7-8 | 指标9-10 | 指标11-12 | 指标13-14 | 指标15-16 |
|----------|---------|-------|-------|---------|----------|-----------|-----------|-----------|
| 批发 + 小微 | ✓ | - | ✓ | ✓ | - | - | - | ✓ |
| 批发 + 非小微 | ✓ | - | - | ✓ | - | - | - | ✓ |
| 零售 + 小微 + 吃穿用 | ✓ | ✓ | ✓ | - | ✓ | - | - | ✓ |
| 零售 + 小微 | ✓ | - | ✓ | - | ✓ | - | - | ✓ |
| 零售 + 吃穿用 | ✓ | ✓ | - | - | ✓ | - | - | ✓ |
| 零售 | ✓ | - | - | - | ✓ | - | - | ✓ |
| 住宿 + 小微 | ✓ | - | ✓ | - | - | ✓ | - | ✓ |
| 住宿 | ✓ | - | - | - | - | ✓ | - | ✓ |
| 餐饮 + 小微 | ✓ | - | ✓ | - | - | - | ✓ | ✓ |
| 餐饮 | ✓ | - | - | - | - | - | ✓ | ✓ |

---

## 6 累计值联动规则

### 6.1 本年累计零售额自动更新

当 `retailCurrentMonth` 变化时，需要同步更新 `retailCurrentCumulative`：

```
retailCurrentCumulative(新) = retailCurrentCumulative(原) + (retailCurrentMonth(新) - retailCurrentMonth(原))
```

**简化实现**：
```
retailCurrentCumulative = retailCurrentCumulative(导入时) + Δ(retailCurrentMonth)
```

其中 `Δ(retailCurrentMonth) = retailCurrentMonth(当前) - retailCurrentMonth(导入时原值)`

### 6.2 销售额累计同理

当 `salesCurrentMonth` 变化时：
```
salesCurrentCumulative = salesCurrentCumulative(导入时) + Δ(salesCurrentMonth)
```

---

## 7 计算精度要求

1. **中间计算**: 使用 `decimal.Decimal` 或 `float64`，保留 8 位小数
2. **存储精度**: 金额保留 2 位小数（单位：万元）
3. **增速展示**: 百分比保留 2 位小数 (如 7.50%)
4. **避免精度丢失**: 除法运算在最后一步执行

---

## 8 实现伪代码

### 8.1 指标计算引擎

```go
type IndicatorEngine struct {
    companies []Company
    config    Config
}

func (e *IndicatorEngine) Calculate() Indicators {
    // 预计算各分组的汇总值
    sums := e.calculateSums()

    result := Indicators{}

    // 指标组一
    result.LimitAboveMonthValue = sums.allRetailCurrent
    result.LimitAboveMonthRate = e.calcRate(sums.allRetailCurrent, sums.allRetailLastYear)
    result.LimitAboveCumulativeValue = sums.allRetailCurrentCumulative
    result.LimitAboveCumulativeRate = e.calcRate(sums.allRetailCurrentCumulative, sums.allRetailLastYearCumulative)

    // 指标组二
    result.EatWearUseMonthRate = e.calcRate(sums.eatWearUseRetailCurrent, sums.eatWearUseRetailLastYear)
    result.MicroSmallMonthRate = e.calcRate(sums.microSmallRetailCurrent, sums.microSmallRetailLastYear)

    // 指标组三 (四大行业)
    for _, industry := range []string{"wholesale", "retail", "accommodation", "catering"} {
        result.IndustryRates[industry] = IndustryRate{
            MonthRate:      e.calcRate(sums.industries[industry].salesCurrent, sums.industries[industry].salesLastYear),
            CumulativeRate: e.calcRate(sums.industries[industry].salesCurrentCumulative, sums.industries[industry].salesLastYearCumulative),
        }
    }

    // 指标组四
    estimatedLimitBelow := e.config.LastYearLimitBelowCumulative * (1 + result.MicroSmallMonthRate)
    result.TotalSocialCumulativeValue = result.LimitAboveCumulativeValue + estimatedLimitBelow

    lastYearTotal := sums.allRetailLastYearCumulative + e.config.LastYearLimitBelowCumulative
    result.TotalSocialCumulativeRate = e.calcRate(result.TotalSocialCumulativeValue, lastYearTotal)

    return result
}

func (e *IndicatorEngine) calcRate(current, lastYear float64) float64 {
    if lastYear == 0 {
        return 0
    }
    return (current - lastYear) / lastYear
}
```

### 8.2 前端响应式计算

```typescript
// 使用 Zustand 的 middleware 实现自动重算
const useDataStore = create<DataStore>((set, get) => ({
  companies: [],
  indicators: defaultIndicators,

  updateCompanyRetail: (id: string, newValue: number) => {
    const companies = get().companies.map(c => {
      if (c.id !== id) return c

      const delta = newValue - c.retailCurrentMonth
      return {
        ...c,
        retailCurrentMonth: newValue,
        retailCurrentCumulative: c.retailCurrentCumulative + delta,
      }
    })

    set({ companies })

    // 触发指标重算
    const indicators = calculateIndicators(companies, get().config)
    set({ indicators })
  },
}))
```

---

## 9 校验规则

### 9.1 业务规则校验

| 规则ID | 描述 | 校验逻辑 |
|--------|------|----------|
| V001 | 零售额不能超过销售额 | `retailCurrentMonth <= salesCurrentMonth` |
| V002 | 数值不能为负数 | `retailCurrentMonth >= 0` |
| V003 | 增速合理范围 | `-100% <= rate <= 500%` (警告级别) |

### 9.2 校验时机

- 用户输入时实时校验
- 导入数据时批量校验
- 智能调整前校验约束条件

---

## 10 性能优化策略

1. **增量计算**: 只重算受影响的汇总值，避免全量遍历
2. **分组缓存**: 缓存各分组的企业列表，避免重复筛选
3. **批量更新**: 多个字段变更时合并为一次计算
4. **防抖处理**: 输入框 300ms 防抖，减少计算频率
