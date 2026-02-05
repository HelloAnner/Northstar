/**
 * 字段映射分组测试
 *
 * @author Anner
 * Created on 2026/2/5
 */

import { describe, expect, it } from 'vitest'
import { groupMappings } from './importMapping'

describe('groupMappings', () => {
  it('groups mapping items by table and time type, sorted by field name', () => {
    const groups = groupMappings(
      [
        { columnIndex: 2, columnName: 'B列', dbField: 'b_field', timeType: 3 },
        { columnIndex: 1, columnName: 'A列', dbField: 'a_field', timeType: 0 },
        { columnIndex: 3, columnName: 'C列', dbField: 'c_field', timeType: 0 }
      ],
      'wholesale'
    )

    expect(groups).toHaveLength(1)
    expect(groups[0].table).toBe('wholesale_retail')
    expect(groups[0].timeGroups.map((group) => group.timeType)).toEqual([0, 3])
    expect(groups[0].timeGroups[0].items.map((item) => item.dbField)).toEqual([
      'a_field',
      'c_field'
    ])
  })
})
