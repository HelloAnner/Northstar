import { create } from 'zustand'
import type { Company, Config, Indicators, SheetInfo, FieldMapping } from '@/types'
import { companiesApi, indicatorsApi, configApi } from '@/services/api'

// 默认指标
const defaultIndicators: Indicators = {
  limitAboveMonthValue: 0,
  limitAboveMonthRate: 0,
  limitAboveCumulativeValue: 0,
  limitAboveCumulativeRate: 0,
  eatWearUseMonthRate: 0,
  microSmallMonthRate: 0,
  industryRates: {
    wholesale: { monthRate: 0, cumulativeRate: 0 },
    retail: { monthRate: 0, cumulativeRate: 0 },
    accommodation: { monthRate: 0, cumulativeRate: 0 },
    catering: { monthRate: 0, cumulativeRate: 0 },
  },
  totalSocialCumulativeValue: 0,
  totalSocialCumulativeRate: 0,
}

// 默认配置
const defaultConfig: Config = {
  currentMonth: 6,
  lastYearLimitBelowCumulative: 50000,
}

interface DataStore {
  // 企业数据
  companies: Company[]
  totalCompanies: number
  currentPage: number
  pageSize: number
  searchKeyword: string
  industryFilter: string
  scaleFilter: string

  // 指标
  indicators: Indicators

  // 配置
  config: Config

  // 加载状态
  loading: boolean

  // Actions
  setCompanies: (companies: Company[], total: number) => void
  setIndicators: (indicators: Indicators) => void
  setConfig: (config: Config) => void
  setLoading: (loading: boolean) => void
  setSearchKeyword: (keyword: string) => void
  setIndustryFilter: (industry: string) => void
  setScaleFilter: (scale: string) => void
  setPage: (page: number) => void

  // API Actions
  fetchCompanies: () => Promise<void>
  fetchIndicators: () => Promise<void>
  fetchConfig: () => Promise<void>
  updateCompanyRetail: (id: string, value: number) => Promise<void>
  resetCompanies: (ids?: string[]) => Promise<void>
}

export const useDataStore = create<DataStore>((set, get) => ({
  companies: [],
  totalCompanies: 0,
  currentPage: 1,
  pageSize: 50,
  searchKeyword: '',
  industryFilter: '',
  scaleFilter: '',
  indicators: defaultIndicators,
  config: defaultConfig,
  loading: false,

  setCompanies: (companies, total) => set({ companies, totalCompanies: total }),
  setIndicators: (indicators) => set({ indicators }),
  setConfig: (config) => set({ config }),
  setLoading: (loading) => set({ loading }),
  setSearchKeyword: (keyword) => set({ searchKeyword: keyword, currentPage: 1 }),
  setIndustryFilter: (industry) => set({ industryFilter: industry, currentPage: 1 }),
  setScaleFilter: (scale) => set({ scaleFilter: scale, currentPage: 1 }),
  setPage: (page) => set({ currentPage: page }),

  fetchCompanies: async () => {
    const { currentPage, pageSize, searchKeyword, industryFilter, scaleFilter } = get()
    set({ loading: true })
    try {
      const response = await companiesApi.list({
        page: currentPage,
        pageSize,
        search: searchKeyword,
        industry: industryFilter,
        scale: scaleFilter,
      })
      set({ companies: response.items, totalCompanies: response.total })
    } finally {
      set({ loading: false })
    }
  },

  fetchIndicators: async () => {
    const indicators = await indicatorsApi.get()
    set({ indicators })
  },

  fetchConfig: async () => {
    const config = await configApi.get()
    set({ config })
  },

  updateCompanyRetail: async (id, value) => {
    const response = await companiesApi.update(id, value)
    set({ indicators: response.indicators })

    // 更新本地企业数据
    const { companies } = get()
    const updated = companies.map((c) =>
      c.id === id
        ? {
            ...c,
            retailCurrentMonth: response.company.retailCurrentMonth ?? c.retailCurrentMonth,
            retailCurrentCumulative: response.company.retailCurrentCumulative ?? c.retailCurrentCumulative,
            monthGrowthRate: response.company.monthGrowthRate,
            validation: response.company.validation,
          }
        : c
    )
    set({ companies: updated })
  },

  resetCompanies: async (ids) => {
    const response = await companiesApi.reset(ids)
    set({ indicators: response.indicators })
    await get().fetchCompanies()
  },
}))

// 导入状态
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
  importing: boolean
  importResult: { importedCount: number; generatedHistoryCount: number } | null

  setStep: (step: number) => void
  setFileInfo: (fileId: string, fileName: string, sheets: SheetInfo[]) => void
  setSelectedSheet: (sheet: string) => void
  setColumns: (columns: string[], previewRows: string[][]) => void
  setMapping: (field: keyof FieldMapping, value: string) => void
  setGenerateHistory: (generate: boolean) => void
  setCurrentMonth: (month: number) => void
  setImporting: (importing: boolean) => void
  setImportResult: (result: { importedCount: number; generatedHistoryCount: number } | null) => void
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
  importing: false,
  importResult: null,

  setStep: (step) => set({ step }),
  setFileInfo: (fileId, fileName, sheets) =>
    set({ fileId, fileName, sheets, selectedSheet: sheets[0]?.name || null }),
  setSelectedSheet: (sheet) => set({ selectedSheet: sheet }),
  setColumns: (columns, previewRows) => set({ columns, previewRows }),
  setMapping: (field, value) =>
    set((state) => ({ mapping: { ...state.mapping, [field]: value } })),
  setGenerateHistory: (generate) => set({ generateHistory: generate }),
  setCurrentMonth: (month) => set({ currentMonth: month }),
  setImporting: (importing) => set({ importing }),
  setImportResult: (result) => set({ importResult: result }),
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
      importing: false,
      importResult: null,
    }),
}))
