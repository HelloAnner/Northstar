import { create } from 'zustand'
import type { UndoChange } from '@/lib/undo'

export type UndoStep = {
  type: 'cell' | 'indicator' | 'optimize' | 'undo'
  summary: string
  changes: UndoChange[]
  createdAt: number
}

interface UndoStore {
  stack: UndoStep[]
  push: (step: UndoStep) => void
  pop: () => UndoStep | undefined
  clear: () => void
}

export const useUndoStore = create<UndoStore>((set, get) => ({
  stack: [],
  push: (step) => set((state) => ({ stack: [...state.stack, step] })),
  pop: () => {
    const stack = get().stack
    if (stack.length === 0) return undefined
    const next = stack.slice(0, -1)
    const last = stack[stack.length - 1]
    set({ stack: next })
    return last
  },
  clear: () => set({ stack: [] }),
}))
