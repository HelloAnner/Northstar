/**
 * 规则列表组件测试
 *
 * @author Anner
 * Created on 2026/3/14
 */

import { describe, expect, it, vi } from 'vitest'

describe('RuleList', () => {
  it('module exports default component', async () => {
    const mod = await import('./RuleList')
    expect(mod.default).toBeDefined()
    expect(typeof mod.default).toBe('function')
  })
})
