// 行业类型
export type IndustryType = 'wholesale' | 'retail' | 'accommodation' | 'catering'

// 企业数据
export interface Company {
  id: string
  name: string
  creditCode: string
  industryCode: string
  industryType: IndustryType
  companyScale: number
  isEatWearUse: boolean

  retailLastYearMonth: number
  retailCurrentMonth: number
  retailLastYearCumulative: number
  retailCurrentCumulative: number

  salesLastYearMonth: number
  salesCurrentMonth: number
  salesLastYearCumulative: number
  salesCurrentCumulative: number

  // 计算字段
  monthGrowthRate?: number
  cumulativeGrowthRate?: number

  // 校验
  validation?: {
    hasError: boolean
    errors: ValidationError[]
  }
}

export interface ValidationError {
  field: string
  message: string
  severity: 'error' | 'warning'
}

// 行业增速
export interface IndustryRate {
  monthRate: number
  cumulativeRate: number
}

// 指标数据
export interface Indicators {
  limitAboveMonthValue: number
  limitAboveMonthRate: number
  limitAboveCumulativeValue: number
  limitAboveCumulativeRate: number

  eatWearUseMonthRate: number
  microSmallMonthRate: number

  industryRates: Record<IndustryType, IndustryRate>

  totalSocialCumulativeValue: number
  totalSocialCumulativeRate: number
}

// 配置
export interface Config {
  currentMonth: number
  lastYearLimitBelowCumulative: number
}

// 优化约束
export interface OptimizeConstraints {
  targetGrowthRate: number
  maxIndividualRate: number
  minIndividualRate: number
  priorityIndustries: string[]
}

// 优化结果
export interface OptimizeResult {
  success: boolean
  achievedValue: number
  adjustments: CompanyAdjustment[]
  summary: {
    adjustedCount: number
    totalAdjustment: number
    averageChangePercent: number
  }
  indicators: Indicators
}

export interface CompanyAdjustment {
  companyId: string
  companyName: string
  originalValue: number
  adjustedValue: number
  changePercent: number
}

// 字段映射
export interface FieldMapping {
  companyName: string
  creditCode: string
  industryCode: string
  companyScale: string
  retailCurrentMonth: string
  retailLastYearMonth: string
  retailCurrentCumulative: string
  retailLastYearCumulative: string
  salesCurrentMonth: string
  salesLastYearMonth: string
  salesCurrentCumulative: string
  salesLastYearCumulative: string
}

// 工作表信息
export interface SheetInfo {
  name: string
  rowCount: number
}

export type SheetType =
  | 'unknown'
  | 'wholesale_main'
  | 'retail_main'
  | 'accommodation_main'
  | 'catering_main'
  | 'wholesale_retail_snapshot'
  | 'accommodation_catering_snapshot'
  | 'eat_wear_use'
  | 'micro_small'
  | 'eat_wear_use_excluded'
  | 'fixed_social_retail'
  | 'fixed_summary'

export interface SheetRecognition {
  sheetName: string
  type: SheetType
  score: number
  missingFields: string[]
}

export interface ResolveResult {
  month: number
  mainSheets: Record<string, string>
  snapshotSheets: Record<string, string>
  unknownSheets: string[]
  unusedSheets: string[]
}

// API 响应
export interface ApiResponse<T> {
  code: number
  message: string
  data: T
}

// 列表响应
export interface ListResponse<T> {
  total: number
  page: number
  pageSize: number
  items: T[]
}
