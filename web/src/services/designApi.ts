import type {
  IndicatorDefinition,
  RuleDetail,
  RuleEvaluation,
  UpsertIndicatorDefinitionPayload,
  UpsertRulePayload,
} from '@/types/design'

async function jsonRequest<T>(url: string, init?: RequestInit): Promise<T> {
  const response = await fetch(url, {
    headers: {
      'Content-Type': 'application/json',
    },
    ...init,
  })

  const data = await response.json()
  if (!response.ok) {
    throw new Error(data?.error || '请求失败')
  }
  return data as T
}

export const designApi = {
  listIndicatorDefinitions: async (enabledOnly = true) => {
    const query = new URLSearchParams()
    if (!enabledOnly) {
      query.set('enabledOnly', 'false')
    }
    const suffix = query.toString() ? `?${query.toString()}` : ''
    const payload = await jsonRequest<{ items: IndicatorDefinition[] }>(`/api/indicator-definitions${suffix}`)
    return Array.isArray(payload.items) ? payload.items : []
  },

  upsertIndicatorDefinition: async (code: string, body: UpsertIndicatorDefinitionPayload) => {
    return jsonRequest<{ message: string }>(`/api/indicator-definitions/${encodeURIComponent(code)}`, {
      method: 'PATCH',
      body: JSON.stringify(body),
    })
  },

  listRules: async (enabledOnly = true) => {
    const query = new URLSearchParams()
    if (!enabledOnly) {
      query.set('enabledOnly', 'false')
    }
    const suffix = query.toString() ? `?${query.toString()}` : ''
    const payload = await jsonRequest<{ items: RuleDetail[] }>(`/api/rules${suffix}`)
    if (!Array.isArray(payload.items)) {
      return []
    }

    return payload.items.map((item) => ({
      ...item,
      links: Array.isArray(item.links) ? item.links : [],
    }))
  },



  listRuleEvaluations: async (enabledOnly = true) => {
    const query = new URLSearchParams()
    if (!enabledOnly) {
      query.set('enabledOnly', 'false')
    }
    const suffix = query.toString() ? `?${query.toString()}` : ''
    const payload = await jsonRequest<{ items: RuleEvaluation[] }>(`/api/rules/evaluate${suffix}`)
    return Array.isArray(payload.items) ? payload.items : []
  },
  upsertRule: async (code: string, body: UpsertRulePayload) => {
    return jsonRequest<{ message: string }>(`/api/rules/${encodeURIComponent(code)}`, {
      method: 'PATCH',
      body: JSON.stringify(body),
    })
  },
}
