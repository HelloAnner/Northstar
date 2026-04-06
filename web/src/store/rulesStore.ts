/**
 * 规则管理状态
 *
 * @author Anner
 * Created on 2026/3/14
 */

import { createStore } from 'zustand/vanilla'
import { useStore } from 'zustand'
import { toast } from 'sonner'
import {
  constraintsApi,
  naturalRulesApi,
  type AdjustmentConstraint,
  type NaturalRule,
} from '@/services/api'

export interface RulesStoreState {
  // 硬约束
  constraints: AdjustmentConstraint[]
  constraintsLoading: boolean

  // 自然语言规则
  naturalRules: NaturalRule[]
  naturalRulesLoading: boolean

  // 通用
  submitting: boolean

  // 硬约束操作
  loadConstraints: () => Promise<void>
  addConstraint: (data: Omit<AdjustmentConstraint, 'id'>) => Promise<void>
  updateConstraint: (id: number, data: Omit<AdjustmentConstraint, 'id'>) => Promise<void>
  deleteConstraint: (id: number) => Promise<void>

  // 自然语言规则操作
  loadNaturalRules: () => Promise<void>
  addNaturalRule: (text: string) => Promise<void>
  updateNaturalRule: (id: number, text: string) => Promise<void>
  deleteNaturalRule: (id: number) => Promise<void>
}

export function createRulesStore() {
  return createStore<RulesStoreState>((set, get) => ({
    constraints: [],
    constraintsLoading: false,
    naturalRules: [],
    naturalRulesLoading: false,
    submitting: false,

    loadConstraints: async () => {
      set({ constraintsLoading: true })
      try {
        const constraints = await constraintsApi.list()
        set({ constraints })
      } finally {
        set({ constraintsLoading: false })
      }
    },

    addConstraint: async (data) => {
      set({ submitting: true })
      try {
        await constraintsApi.create(data)
        await get().loadConstraints()
        toast.success('���束已添加')
      } finally {
        set({ submitting: false })
      }
    },

    updateConstraint: async (id, data) => {
      set({ submitting: true })
      try {
        await constraintsApi.update(id, data)
        await get().loadConstraints()
        toast.success('约束已更新')
      } finally {
        set({ submitting: false })
      }
    },

    deleteConstraint: async (id) => {
      set({ submitting: true })
      try {
        await constraintsApi.remove(id)
        await get().loadConstraints()
        toast.success('约束已删除')
      } finally {
        set({ submitting: false })
      }
    },

    loadNaturalRules: async () => {
      set({ naturalRulesLoading: true })
      try {
        const naturalRules = await naturalRulesApi.list()
        set({ naturalRules })
      } finally {
        set({ naturalRulesLoading: false })
      }
    },

    addNaturalRule: async (text) => {
      set({ submitting: true })
      try {
        await naturalRulesApi.create(text)
        await get().loadNaturalRules()
        toast.success('规则已添加')
      } finally {
        set({ submitting: false })
      }
    },

    updateNaturalRule: async (id, text) => {
      set({ submitting: true })
      try {
        await naturalRulesApi.update(id, text)
        await get().loadNaturalRules()
        toast.success('规则已更新')
      } finally {
        set({ submitting: false })
      }
    },

    deleteNaturalRule: async (id) => {
      set({ submitting: true })
      try {
        await naturalRulesApi.remove(id)
        await get().loadNaturalRules()
        toast.success('规则已删除')
      } finally {
        set({ submitting: false })
      }
    },
  }))
}

const rulesStore = createRulesStore()

export function useRulesStore<T>(selector: (state: RulesStoreState) => T) {
  return useStore(rulesStore, selector)
}
