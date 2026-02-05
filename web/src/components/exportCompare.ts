/**
 * 导出三方对比状态辅助
 *
 * @author Anner
 * Created on 2026/2/5
 */

export type CompareItem = {
  key: string
  label: string
  status: string
  message: string
}

const compareOrder: Record<string, number> = {
  raw: 0,
  business: 1,
  export: 2
}

export function normalizeCompareItems(items?: CompareItem[]): CompareItem[] {
  if (!items || items.length === 0) return []
  const sorted = [...items].sort((a, b) => {
    const orderA = compareOrder[a.key] ?? 99
    const orderB = compareOrder[b.key] ?? 99
    if (orderA !== orderB) return orderA - orderB
    return (a.key || '').localeCompare(b.key || '')
  })
  return sorted.map((item) => ({
    ...item,
    label: item.label || fallbackLabel(item.key)
  }))
}

export function fallbackLabel(key: string): string {
  if (key === 'raw') return '原始数据'
  if (key === 'business') return '业务表'
  if (key === 'export') return '导出结果'
  return key || '未知'
}
