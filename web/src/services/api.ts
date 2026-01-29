import type {
  ApiResponse,
  Company,
  Config,
  FieldMapping,
  Indicators,
  ListResponse,
  OptimizeConstraints,
  OptimizeResult,
  CurrentProject,
  ProjectDetail,
  ProjectsIndex,
  ProjectSummary,
  SheetInfo,
  SheetRecognition,
  SheetType,
  ResolveResult,
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
      recognition: SheetRecognition[]
      months: number[]
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

  resolve: (fileId: string, month: number, overrides: Record<string, SheetType>) =>
    request<ResolveResult>(`/import/${fileId}/resolve`, {
      method: 'POST',
      body: JSON.stringify({ month, overrides }),
    }),
}

// 项目
export const projectsApi = {
  list: () => request<ProjectsIndex>('/projects'),

  create: (name: string) =>
    request<ProjectSummary>('/projects', {
      method: 'POST',
      body: JSON.stringify({ name }),
    }),

  select: (projectId: string) =>
    request<ProjectSummary>(`/projects/${encodeURIComponent(projectId)}/select`, {
      method: 'POST',
    }),

  current: () => request<CurrentProject>('/projects/current'),

  save: () =>
    request<{ saved: boolean }>('/projects/current/save', {
      method: 'POST',
    }),

  detail: (projectId: string) => request<ProjectDetail>(`/projects/${encodeURIComponent(projectId)}`),

  delete: (projectId: string) =>
    request<{ deleted: boolean }>(`/projects/${encodeURIComponent(projectId)}`, { method: 'DELETE' }),

  undoCurrent: () =>
    request<{ indicators: Indicators }>('/projects/current/undo', {
      method: 'POST',
    }),
}

// 企业数据
export const companiesApi = {
  list: (params?: {
    page?: number
    pageSize?: number
    search?: string
    industry?: string
    scale?: string
    sortBy?: string
    sortDir?: 'asc' | 'desc'
  }) => {
    const query = new URLSearchParams()
    if (params?.page) query.set('page', String(params.page))
    if (params?.pageSize) query.set('pageSize', String(params.pageSize))
    if (params?.search) query.set('search', params.search)
    if (params?.industry) query.set('industry', params.industry)
    if (params?.scale) query.set('scale', params.scale)
    if (params?.sortBy) query.set('sortBy', params.sortBy)
    if (params?.sortDir) query.set('sortDir', params.sortDir)

    return request<ListResponse<Company>>(`/companies?${query}`)
  },

  get: (id: string) => request<Company>(`/companies/${id}`),

  update: (
    id: string,
    patch: {
      name?: string
      retailCurrentMonth?: number
      retailLastYearMonth?: number
      salesCurrentMonth?: number
    }
  ) =>
    request<{ company: Partial<Company>; indicators: Indicators }>(`/companies/${id}`, {
      method: 'PATCH',
      body: JSON.stringify(patch),
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

  adjust: (key: string, value: number) =>
    request<Indicators>('/indicators/adjust', {
      method: 'POST',
      body: JSON.stringify({ key, value }),
    }),
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
