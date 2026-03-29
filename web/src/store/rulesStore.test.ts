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

  it('loads rules and stops polling after status becomes ok', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(jsonResponse([{ index: 1, text: '零售增速不超过 15%' }]))
      .mockResolvedValueOnce(jsonResponse({ status: 'running', updatedAt: '', error: '' }))
      .mockResolvedValueOnce(jsonResponse({ status: 'ok', updatedAt: '2026-03-14T10:30:00Z', error: '' }))
    vi.stubGlobal('fetch', fetchMock)

    const store = createRulesStore()
    await store.getState().loadRules()
    expect(store.getState().rules).toEqual([{ index: 1, text: '零售增速不超过 15%' }])

    await store.getState().loadStatus()
    expect(store.getState().status).toBe('running')

    store.getState().startPolling()
    await vi.advanceTimersByTimeAsync(1500)

    expect(store.getState().status).toBe('ok')
    expect(store.getState().statusUpdatedAt).toBe('2026-03-14T10:30:00Z')
    expect(store.getState().pollingTimer).toBeNull()
  })

  it('starts polling after manual convert', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(jsonResponse({ status: 'running' }))
      .mockResolvedValueOnce(jsonResponse({ status: 'ok', updatedAt: '2026-03-14T11:00:00Z', error: '' }))
    vi.stubGlobal('fetch', fetchMock)

    const store = createRulesStore()
    await store.getState().convertRules()

    expect(store.getState().status).toBe('running')
    expect(store.getState().pollingTimer).not.toBeNull()

    await vi.advanceTimersByTimeAsync(1500)

    expect(store.getState().status).toBe('ok')
    expect(store.getState().statusUpdatedAt).toBe('2026-03-14T11:00:00Z')
    expect(store.getState().pollingTimer).toBeNull()
  })
})

function jsonResponse(data: unknown) {
  return new Response(JSON.stringify(data), {
    status: 200,
    headers: { 'Content-Type': 'application/json' },
  })
}
