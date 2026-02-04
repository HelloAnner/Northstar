export type UndoField =
  | 'salesCurrentMonth'
  | 'salesLastYearMonth'
  | 'salesCurrentCumulative'
  | 'salesLastYearCumulative'
  | 'retailCurrentMonth'
  | 'retailLastYearMonth'
  | 'retailCurrentCumulative'
  | 'retailLastYearCumulative'
  | 'revenueCurrentMonth'
  | 'revenueLastYearMonth'
  | 'revenueCurrentCumulative'
  | 'revenueLastYearCumulative'
  | 'roomCurrentMonth'
  | 'roomCurrentCumulative'
  | 'foodCurrentMonth'
  | 'foodCurrentCumulative'
  | 'goodsCurrentMonth'
  | 'goodsCurrentCumulative'

export type CompanySnapshotRow = {
  id: string
  kind: 'wr' | 'ac'
} & Partial<Record<UndoField, number>>

export type CompanySnapshot = Record<string, CompanySnapshotRow>

export type UndoFieldDelta = {
  before: number
  after: number
}

export type UndoChange = {
  companyId: string
  fields: Partial<Record<UndoField, UndoFieldDelta>>
}

const WR_FIELDS: UndoField[] = [
  'salesCurrentMonth',
  'salesLastYearMonth',
  'salesCurrentCumulative',
  'salesLastYearCumulative',
  'retailCurrentMonth',
  'retailLastYearMonth',
  'retailCurrentCumulative',
  'retailLastYearCumulative',
]

const AC_FIELDS: UndoField[] = [
  'revenueCurrentMonth',
  'revenueLastYearMonth',
  'revenueCurrentCumulative',
  'revenueLastYearCumulative',
  'roomCurrentMonth',
  'roomCurrentCumulative',
  'foodCurrentMonth',
  'foodCurrentCumulative',
  'goodsCurrentMonth',
  'goodsCurrentCumulative',
]

const isNumber = (v: unknown): v is number => typeof v === 'number' && Number.isFinite(v)

const numberChanged = (before: number, after: number) => Math.abs(before - after) > 1e-6

export type UndoUpdate = {
  id: string
  patch: Partial<Record<UndoField, number>>
}

export const buildCompanySnapshot = (items: CompanySnapshotRow[]): CompanySnapshot => {
  const out: CompanySnapshot = {}
  for (const item of items) {
    if (!item?.id || !item?.kind) continue
    const fields = item.kind === 'ac' ? AC_FIELDS : WR_FIELDS
    const row: CompanySnapshotRow = { id: item.id, kind: item.kind }
    for (const field of fields) {
      const value = item[field]
      if (!isNumber(value)) continue
      row[field] = value
    }
    out[item.id] = row
  }
  return out
}

export const buildUndoChanges = (before: CompanySnapshot, after: CompanySnapshot): UndoChange[] => {
  const out: UndoChange[] = []
  for (const [id, afterRow] of Object.entries(after)) {
    const beforeRow = before[id]
    if (!beforeRow || !afterRow) continue
    const fields = afterRow.kind === 'ac' ? AC_FIELDS : WR_FIELDS
    const changes: Partial<Record<UndoField, UndoFieldDelta>> = {}
    for (const field of fields) {
      const beforeValue = beforeRow[field]
      const afterValue = afterRow[field]
      if (!isNumber(beforeValue) || !isNumber(afterValue)) continue
      if (!numberChanged(beforeValue, afterValue)) continue
      changes[field] = { before: beforeValue, after: afterValue }
    }
    if (Object.keys(changes).length > 0) {
      out.push({ companyId: id, fields: changes })
    }
  }
  return out
}

export const buildUndoUpdates = (changes: UndoChange[]): UndoUpdate[] => {
  return changes.map((change) => {
    const patch: Partial<Record<UndoField, number>> = {}
    for (const [field, delta] of Object.entries(change.fields)) {
      if (!delta) continue
      patch[field as UndoField] = delta.before
    }
    return { id: change.companyId, patch }
  })
}
