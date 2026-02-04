import { describe, expect, it } from 'vitest'
import { buildCompanySnapshot, buildUndoChanges, buildUndoUpdates, type CompanySnapshot } from '@/lib/undo'

describe('buildUndoChanges', () => {
  it('only returns changed fields', () => {
    const before: CompanySnapshot = {
      'wr:1': { id: 'wr:1', kind: 'wr', salesCurrentMonth: 100 },
    }
    const after: CompanySnapshot = {
      'wr:1': { id: 'wr:1', kind: 'wr', salesCurrentMonth: 120 },
    }
    const changes = buildUndoChanges(before, after)
    expect(changes).toHaveLength(1)
    expect(changes[0].fields.salesCurrentMonth?.before).toBe(100)
    expect(changes[0].fields.salesCurrentMonth?.after).toBe(120)
  })

  it('skips rows without differences', () => {
    const before: CompanySnapshot = {
      'wr:1': { id: 'wr:1', kind: 'wr', salesCurrentMonth: 100 },
    }
    const after: CompanySnapshot = {
      'wr:1': { id: 'wr:1', kind: 'wr', salesCurrentMonth: 100 },
    }
    const changes = buildUndoChanges(before, after)
    expect(changes).toHaveLength(0)
  })

  it('builds snapshots with only undo fields', () => {
    const items = [
      { id: 'wr:1', kind: 'wr', salesCurrentMonth: 10, retailCurrentMonth: 20, name: 'A' },
      { id: 'ac:2', kind: 'ac', foodCurrentMonth: 5, goodsCurrentMonth: 6, unknown: 99 },
    ]
    const snapshot = buildCompanySnapshot(items as any)
    expect(snapshot['wr:1'].salesCurrentMonth).toBe(10)
    expect(snapshot['wr:1'].retailCurrentMonth).toBe(20)
    expect((snapshot['wr:1'] as any).name).toBeUndefined()
    expect(snapshot['ac:2'].foodCurrentMonth).toBe(5)
    expect(snapshot['ac:2'].goodsCurrentMonth).toBe(6)
    expect((snapshot['ac:2'] as any).unknown).toBeUndefined()
  })

  it('builds undo updates from changes', () => {
    const updates = buildUndoUpdates([
      {
        companyId: 'wr:1',
        fields: {
          salesCurrentMonth: { before: 100, after: 120 },
        },
      },
    ])
    expect(updates).toHaveLength(1)
    expect(updates[0].id).toBe('wr:1')
    expect(updates[0].patch.salesCurrentMonth).toBe(100)
  })
})
