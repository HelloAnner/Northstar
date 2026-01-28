import { create } from 'zustand'
import type { FieldMapping, SheetInfo } from '@/types'

interface ImportStore {
  step: number
  fileId: string | null
  fileName: string | null
  sheets: SheetInfo[]
  selectedSheet: string | null
  columns: string[]
  previewRows: string[][]
  mapping: FieldMapping
  generateHistory: boolean
  currentMonth: number

  setStep: (step: number) => void
  setFileInfo: (fileId: string, fileName: string, sheets: SheetInfo[]) => void
  setSelectedSheet: (sheet: string) => void
  setColumns: (columns: string[], previewRows: string[][]) => void
  setMapping: (field: keyof FieldMapping, value: string) => void
  setGenerateHistory: (generate: boolean) => void
  setCurrentMonth: (month: number) => void
  reset: () => void
}

const defaultMapping: FieldMapping = {
  companyName: '',
  creditCode: '',
  industryCode: '',
  companyScale: '',
  retailCurrentMonth: '',
  retailLastYearMonth: '',
  retailCurrentCumulative: '',
  retailLastYearCumulative: '',
  salesCurrentMonth: '',
  salesLastYearMonth: '',
  salesCurrentCumulative: '',
  salesLastYearCumulative: '',
}

export const useImportStore = create<ImportStore>((set) => ({
  step: 1,
  fileId: null,
  fileName: null,
  sheets: [],
  selectedSheet: null,
  columns: [],
  previewRows: [],
  mapping: { ...defaultMapping },
  generateHistory: true,
  currentMonth: 6,

  setStep: (step) => set({ step }),
  setFileInfo: (fileId, fileName, sheets) =>
    set({ fileId, fileName, sheets, selectedSheet: sheets[0]?.name || null }),
  setSelectedSheet: (sheet) => set({ selectedSheet: sheet }),
  setColumns: (columns, previewRows) => set({ columns, previewRows }),
  setMapping: (field, value) =>
    set((state) => ({ mapping: { ...state.mapping, [field]: value } })),
  setGenerateHistory: (generate) => set({ generateHistory: generate }),
  setCurrentMonth: (month) => set({ currentMonth: month }),
  reset: () =>
    set({
      step: 1,
      fileId: null,
      fileName: null,
      sheets: [],
      selectedSheet: null,
      columns: [],
      previewRows: [],
      mapping: { ...defaultMapping },
      generateHistory: true,
      currentMonth: 6,
    }),
}))

