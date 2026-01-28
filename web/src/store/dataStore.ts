import { create } from 'zustand'
import type { Company, Config, Indicators } from '@/types'
import { companiesApi, configApi, indicatorsApi } from '@/services/api'

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
  companies: Company[]
  totalCompanies: number
  currentPage: number
  pageSize: number
  searchKeyword: string
  industryFilter: string
  scaleFilter: string
  sortBy: string
  sortDir: 'asc' | 'desc'

  indicators: Indicators
  config: Config

  loading: boolean

  setCompanies: (companies: Company[], total: number) => void
  setIndicators: (indicators: Indicators) => void
  setConfig: (config: Config) => void
  setLoading: (loading: boolean) => void
  setSearchKeyword: (keyword: string) => void
  setIndustryFilter: (industry: string) => void
  setScaleFilter: (scale: string) => void
  setSort: (sortBy: string, sortDir: 'asc' | 'desc') => void
  setPage: (page: number) => void

  fetchCompanies: () => Promise<void>
  fetchIndicators: () => Promise<void>
  fetchConfig: () => Promise<void>
  updateCompany: (id: string, patch: Partial<Pick<Company, 'name' | 'retailCurrentMonth' | 'retailLastYearMonth' | 'salesCurrentMonth'>>) => Promise<void>
  resetCompanies: (ids?: string[]) => Promise<void>
}

export const useDataStore = create<DataStore>((set, get) => ({
  companies: [],
  totalCompanies: 0,
  currentPage: 1,
  pageSize: 10,
  searchKeyword: '',
  industryFilter: '',
  scaleFilter: '',
  sortBy: 'name',
  sortDir: 'asc',
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
  setSort: (sortBy, sortDir) => set({ sortBy, sortDir, currentPage: 1 }),
  setPage: (page) => set({ currentPage: page }),

  fetchCompanies: async () => {
    const { currentPage, pageSize, searchKeyword, industryFilter, scaleFilter, sortBy, sortDir } = get()
    set({ loading: true })
    try {
      const response = await companiesApi.list({
        page: currentPage,
        pageSize,
        search: searchKeyword,
        industry: industryFilter,
        scale: scaleFilter,
        sortBy,
        sortDir,
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

  updateCompany: async (id, patch) => {
    const response = await companiesApi.update(id, patch)
    set({ indicators: response.indicators })

    const { companies } = get()
    const updated = companies.map((c) =>
      c.id === id
        ? {
            ...c,
            name: response.company.name ?? c.name,
            salesCurrentMonth: response.company.salesCurrentMonth ?? c.salesCurrentMonth,
            retailCurrentMonth: response.company.retailCurrentMonth ?? c.retailCurrentMonth,
            retailLastYearMonth: response.company.retailLastYearMonth ?? c.retailLastYearMonth,
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
