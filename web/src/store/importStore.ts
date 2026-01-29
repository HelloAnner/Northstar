import { create } from 'zustand'
import type { FieldMapping, ResolveResult, SheetInfo, SheetRecognition, SheetType } from '@/types'

interface ImportStore {
  step: number
  fileId: string | null
  fileName: string | null
  sheets: SheetInfo[]
  recognition: SheetRecognition[]
  months: number[]
  overrides: Record<string, SheetType>
  resolveResult: ResolveResult | null
  selectedSheet: string | null
  columns: string[]
  previewRows: string[][]
  mapping: FieldMapping
  generateHistory: boolean
  currentMonth: number

  setStep: (step: number) => void
  setFileInfo: (
    fileId: string,
    fileName: string,
    sheets: SheetInfo[],
    recognition: SheetRecognition[],
    months: number[]
  ) => void
  setOverride: (sheetName: string, sheetType: SheetType) => void
  setResolveResult: (result: ResolveResult) => void
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
  recognition: [],
  months: [],
  overrides: {},
  resolveResult: null,
  selectedSheet: null,
  columns: [],
  previewRows: [],
  mapping: { ...defaultMapping },
  generateHistory: true,
  currentMonth: 6,

  setStep: (step) => set({ step }),
  setFileInfo: (fileId, fileName, sheets, recognition, months) =>
    set({
      fileId,
      fileName,
      sheets,
      recognition,
      months,
      overrides: {},
      resolveResult: null,
      selectedSheet: sheets[0]?.name || null,
    }),
  setOverride: (sheetName, sheetType) =>
    set((state) => ({
      overrides: {
        ...state.overrides,
        [sheetName]: sheetType,
      },
    })),
  setResolveResult: (result) => set({ resolveResult: result }),
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
      recognition: [],
      months: [],
      overrides: {},
      resolveResult: null,
      selectedSheet: null,
      columns: [],
      previewRows: [],
      mapping: { ...defaultMapping },
      generateHistory: true,
      currentMonth: 6,
    }),
}))
