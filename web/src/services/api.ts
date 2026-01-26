import type {
  ApiResponse,
  Company,
  Config,
  FieldMapping,
  Indicators,
  ListResponse,
  OptimizeConstraints,
  OptimizeResult,
  SheetInfo,
} from '@/types'

const BASE_URL = '/api/v1'

async function request<T>(url: string, options?: RequestInit): Promise<T> {
  const response = await fetch(`${BASE_URL}${url}`, {
    headers: {
      'Content-Type': 'application/json',
    },
    ...options,
  })

  const data: ApiResponse<T> = await response.json()

  if (data.code !== 0) {
    throw new Error(data.message)
  }

  return data.data
}

// 导入相关
export const importApi = {
  upload: async (file: File) => {
    const formData = new FormData()
    formData.append('file', file)

    const response = await fetch(`${BASE_URL}/import/upload`, {
      method: 'POST',
      body: formData,
    })

    const data = await response.json()
    if (data.code !== 0) {
      throw new Error(data.message)
    }

    return data.data as {
      fileId: string
      fileName: string
      fileSize: number
      sheets: SheetInfo[]
    }
  },

  getColumns: (fileId: string, sheet: string) =>
    request<{ columns: string[]; previewRows: string[][] }>(
      `/import/${fileId}/columns?sheet=${encodeURIComponent(sheet)}`
    ),

  setMapping: (fileId: string, sheet: string, mapping: FieldMapping) =>
    request<{ validRows: number; invalidRows: number; warnings: string[] }>(
      `/import/${fileId}/mapping`,
      {
        method: 'POST',
        body: JSON.stringify({ sheet, mapping }),
      }
    ),

  execute: (fileId: string, sheet: string, generateHistory: boolean, currentMonth: number) =>
    request<{ importedCount: number; generatedHistoryCount: number; indicators: Indicators }>(
      `/import/${fileId}/execute`,
      {
        method: 'POST',
        body: JSON.stringify({ sheet, generateHistory, currentMonth }),
      }
    ),
}

// 企业数据
export const companiesApi = {
  list: (params?: { page?: number; pageSize?: number; search?: string; industry?: string; scale?: string }) => {
    const query = new URLSearchParams()
    if (params?.page) query.set('page', String(params.page))
    if (params?.pageSize) query.set('pageSize', String(params.pageSize))
    if (params?.search) query.set('search', params.search)
    if (params?.industry) query.set('industry', params.industry)
    if (params?.scale) query.set('scale', params.scale)

    return request<ListResponse<Company>>(`/companies?${query}`)
  },

  get: (id: string) => request<Company>(`/companies/${id}`),

  update: (id: string, retailCurrentMonth: number) =>
    request<{ company: Partial<Company>; indicators: Indicators }>(`/companies/${id}`, {
      method: 'PATCH',
      body: JSON.stringify({ retailCurrentMonth }),
    }),

  batchUpdate: (updates: { id: string; retailCurrentMonth: number }[]) =>
    request<{ updatedCount: number; indicators: Indicators }>('/companies/batch', {
      method: 'PATCH',
      body: JSON.stringify({ updates }),
    }),

  reset: (companyIds?: string[]) =>
    request<{ indicators: Indicators }>('/companies/reset', {
      method: 'POST',
      body: JSON.stringify({ companyIds }),
    }),
}

// 指标
export const indicatorsApi = {
  get: () => request<Indicators>('/indicators'),
}

// 智能调整
export const optimizeApi = {
  run: (targetValue: number, constraints?: Partial<OptimizeConstraints>) =>
    request<OptimizeResult>('/optimize', {
      method: 'POST',
      body: JSON.stringify({
        targetIndicator: 'limitAboveCumulativeRate',
        targetValue,
        constraints,
      }),
    }),

  preview: (targetValue: number, constraints?: Partial<OptimizeConstraints>) =>
    request<OptimizeResult>('/optimize/preview', {
      method: 'POST',
      body: JSON.stringify({
        targetIndicator: 'limitAboveCumulativeRate',
        targetValue,
        constraints,
      }),
    }),
}

// 配置
export const configApi = {
  get: () => request<Config>('/config'),

  update: (updates: Partial<Config>) =>
    request<{ config: Config; indicators: Indicators }>('/config', {
      method: 'PATCH',
      body: JSON.stringify(updates),
    }),
}

// 导出
export const exportApi = {
  export: (options: { format?: string; includeIndicators?: boolean; includeChanges?: boolean }) =>
    request<{ downloadUrl: string; expiresAt: string }>('/export', {
      method: 'POST',
      body: JSON.stringify(options),
    }),
}
