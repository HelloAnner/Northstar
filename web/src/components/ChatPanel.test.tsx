/**
 * AI 对话面板测试
 *
 * @author Anner
 * Created on 2026/3/14
 */

import { describe, expect, it } from 'vitest'
import { formatAppliedRuleDescription } from './ChatPanel'

describe('formatAppliedRuleDescription', () => {
  it('formats clamp_target rule', () => {
    const result = formatAppliedRuleDescription({
      ruleId: 'c1',
      type: 'clamp_target',
      indicatorId: 'wholesale_month_rate',
      beforeValue: 15,
      afterValue: 10,
    })
    expect(result).toBe('目标值从 15 裁剪为 10')
  })

  it('formats filter_allocation rule', () => {
    const result = formatAppliedRuleDescription({
      ruleId: 'f1',
      type: 'filter_allocation',
      indicatorId: 'retail_month_rate',
      beforeCount: 8,
      afterCount: 3,
    })
    expect(result).toBe('参与企业从 8 过滤为 3 家')
  })

  it('formats compensate rule', () => {
    const result = formatAppliedRuleDescription({
      ruleId: 'cp1',
      type: 'compensate',
      ensureId: 'wholesale_month_rate',
      targetValue: 12,
    })
    expect(result).toBe('联动补偿 wholesale_month_rate → 12')
  })
})
