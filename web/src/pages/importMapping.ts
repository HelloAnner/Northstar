/**
 * 字段映射分组辅助
 *
 * @author Anner
 * Created on 2026/2/5
 */

export type ColumnMappingItem = {
  columnIndex: number
  columnName: string
  dbField: string
  timeType: number
  targetTable?: string
}

export type MappingTimeGroup = {
  timeType: number
  items: ColumnMappingItem[]
}

export type MappingTableGroup = {
  table: string
  timeGroups: MappingTimeGroup[]
}

export function groupMappings(
  items: ColumnMappingItem[],
  sheetType?: string
): MappingTableGroup[] {
  if (!items.length) return []

  const fallbackTable = getTargetTableBySheetType(sheetType) || 'unknown'
  const tableMap = new Map<string, Map<number, ColumnMappingItem[]>>()

  for (const item of items) {
    const table = item.targetTable || fallbackTable
    const timeType = typeof item.timeType === 'number' ? item.timeType : 999
    if (!tableMap.has(table)) {
      tableMap.set(table, new Map<number, ColumnMappingItem[]>())
    }
    const timeMap = tableMap.get(table)
    if (!timeMap) continue
    if (!timeMap.has(timeType)) {
      timeMap.set(timeType, [])
    }
    const list = timeMap.get(timeType)
    if (list) {
      list.push(item)
    }
  }

  const tableGroups = Array.from(tableMap.entries()).map(([table, timeMap]) => {
    const timeGroups = Array.from(timeMap.entries()).map(([timeType, list]) => {
      const sorted = [...list].sort((a, b) => {
        const nameCompare = (a.dbField || '').localeCompare(b.dbField || '')
        if (nameCompare !== 0) return nameCompare
        return (a.columnIndex ?? 0) - (b.columnIndex ?? 0)
      })
      return { timeType, items: sorted }
    })
    timeGroups.sort((a, b) => a.timeType - b.timeType)
    return { table, timeGroups }
  })

  tableGroups.sort((a, b) => a.table.localeCompare(b.table))
  return tableGroups
}

export function getTargetTableBySheetType(sheetType?: string): string {
  switch (sheetType) {
    case 'wholesale':
    case 'retail':
      return 'wholesale_retail'
    case 'accommodation':
    case 'catering':
      return 'accommodation_catering'
    case 'summary_limit_above_retail':
      return 'summary_limit_above_retail'
    case 'summary_micro_small':
      return 'summary_micro_small'
    case 'summary_eat_wear_use':
      return 'summary_eat_wear_use'
    default:
      return 'unknown'
  }
}
