/**
 * 规则管理 store 测试
 *
 * @author Anner
 * Created on 2026/3/14
 */

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createRulesStore } from './rulesStore'

describe('rulesStore', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.restoreAllMocks()
  })

  it('loads constraints', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(jsonResponse([
        { id: 1, type: 'clamp_target', indicatorId: 'retail_month_rate', minValue: -5, maxValue: 10, enabled: true, tolerance: 0 },
      ]))
    vi.stubGlobal('fetch', fetchMock)

    const store = createRulesStore()
    await store.getState().loadConstraints()
    expect(store.getState().constraints).toHaveLength(1)
    expect(store.getState().constraints[0].type).toBe('clamp_target')
  })

  it('loads natural rules', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(jsonResponse([
        { id: 1, text: '零售增速不超过 15%', enabled: true },
      ]))
    vi.stubGlobal('fetch', fetchMock)

    const store = createRulesStore()
    await store.getState().loadNaturalRules()
    expect(store.getState().naturalRules).toHaveLength(1)
    expect(store.getState().naturalRules[0].text).toBe('零售增速不超过 15%')
  })
})

function jsonResponse(data: unknown) {
  return new Response(JSON.stringify(data), {
    status: 200,
    headers: { 'Content-Type': 'application/json' },
  })
}
