/**
 * 导出对比状态处理测试
 *
 * @author Anner
 * Created on 2026/2/5
 */

import { describe, expect, it } from 'vitest'
import { normalizeCompareItems } from './exportCompare'

describe('normalizeCompareItems', () => {
  it('sorts compare items by raw/business/export order', () => {
    const items = normalizeCompareItems([
      { key: 'export', label: '导出结果', status: 'pass', message: 'ok' },
      { key: 'raw', label: '原始数据', status: 'pass', message: 'ok' },
      { key: 'business', label: '业务表', status: 'pass', message: 'ok' }
    ])
    expect(items.map((item) => item.key)).toEqual(['raw', 'business', 'export'])
  })
})
