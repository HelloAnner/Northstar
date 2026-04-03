/**
 * 规则管理状态
 *
 * @author Anner
 * Created on 2026/3/14
 */

import { createStore } from 'zustand/vanilla'
import { useStore } from 'zustand'
import { toast } from 'sonner'
import { rulesApi, type RuleItem, type RuleStatus } from '@/services/api'

export type RulesConvertStatus = 'idle' | 'pending' | 'running' | 'ok' | 'error'

export interface RulesStoreState {
  rules: RuleItem[]
  status: RulesConvertStatus
  statusError: string
  statusUpdatedAt: string
  statusStep: string
  statusAttempt: string
  loading: boolean
  submitting: boolean
  pollingTimer: number | null
  loadRules: () => Promise<void>
  addRule: (text: string) => Promise<void>
  updateRule: (index: number, text: string) => Promise<void>
  deleteRule: (index: number) => Promise<void>
  convertRules: () => Promise<void>
  loadStatus: () => Promise<void>
  startPolling: () => void
  stopPolling: () => void
}

const pollingIntervalMs = 3000

export function createRulesStore() {
  return createStore<RulesStoreState>((set, get) => ({
    rules: [],
    status: 'idle',
    statusError: '',
    statusUpdatedAt: '',
    statusStep: '',
    statusAttempt: '',
    loading: false,
    submitting: false,
    pollingTimer: null,

    loadRules: async () => {
      set({ loading: true })
      try {
        const rules = await rulesApi.list()
        set({ rules })
      } finally {
        set({ loading: false })
      }
    },

    addRule: async (text) => {
      await mutateRules(set, get, () => rulesApi.create(text))
    },

    updateRule: async (index, text) => {
      await mutateRules(set, get, () => rulesApi.update(index, text))
    },

    deleteRule: async (index) => {
      await mutateRules(set, get, () => rulesApi.remove(index))
    },

    convertRules: async () => {
      set({ submitting: true })
      try {
        await rulesApi.convert()
        set({ status: 'running', statusError: '', statusStep: '', statusAttempt: '' })
        get().startPolling()
      } catch (err) {
        const msg = err instanceof Error ? err.message : '触发规则转换失败'
        set({ status: 'error', statusError: msg })
        toast.error(msg)
      } finally {
        set({ submitting: false })
      }
    },

    loadStatus: async () => {
      const status = await rulesApi.status()
      applyRuleStatus(set, get, status)
    },

    startPolling: () => {
      if (get().pollingTimer != null) {
        return
      }
      const timer = window.setInterval(async () => {
        await get().loadStatus()
      }, pollingIntervalMs)
      set({ pollingTimer: timer })
    },

    stopPolling: () => {
      const timer = get().pollingTimer
      if (timer != null) {
        window.clearInterval(timer)
      }
      set({ pollingTimer: null })
    },
  }))
}

async function mutateRules(
  set: (partial: Partial<RulesStoreState>) => void,
  get: () => RulesStoreState,
  action: () => Promise<unknown>
) {
  set({ submitting: true })
  try {
    await action()
    await get().loadRules()
    set({ status: 'pending' })
  } finally {
    set({ submitting: false })
  }
}

function applyRuleStatus(
  set: (partial: Partial<RulesStoreState>) => void,
  get: () => RulesStoreState,
  status: RuleStatus
) {
  const prevStatus = get().status
  // pending 是客户端状态（规则已修改未转换），只有 running 到来时才覆盖
  const nextStatus = prevStatus === 'pending' && status.status !== 'running'
    ? 'pending'
    : status.status
  set({
    status: nextStatus,
    statusError: status.error,
    statusUpdatedAt: status.updatedAt,
    statusStep: status.step ?? '',
    statusAttempt: status.attempt ?? '',
  })

  // 状态变化时发送 Toast 通知
  if (prevStatus === 'running' && status.status === 'ok') {
    toast.success('规则转换成功，已生效')
  } else if (prevStatus === 'running' && status.status === 'error') {
    toast.error('规则转换失败：' + (status.error || '未知错误'))
  }

  if (status.status === 'running') {
    get().startPolling()
    return
  }
  get().stopPolling()
}

const rulesStore = createRulesStore()

export function useRulesStore<T>(selector: (state: RulesStoreState) => T) {
  return useStore(rulesStore, selector)
}
