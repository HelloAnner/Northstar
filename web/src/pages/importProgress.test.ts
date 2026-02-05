/**
 * 导入进度处理测试
 *
 * @author Anner
 * Created on 2026/2/5
 */

import { describe, expect, it } from 'vitest'
import { applyImportEvent, type ImportTask } from './importProgress'

describe('applyImportEvent', () => {
  it('adds task on sheet_start and marks done on sheet_done', () => {
    let tasks: ImportTask[] = []

    tasks = applyImportEvent(tasks, {
      type: 'sheet_start',
      message: '开始解析',
      data: { sheet_name: '批发' }
    })
    expect(tasks).toHaveLength(1)
    expect(tasks[0].sheetName).toBe('批发')
    expect(tasks[0].status).toBe('running')
    expect(tasks[0].progress).toBeGreaterThan(0)

    tasks = applyImportEvent(tasks, {
      type: 'sheet_done',
      message: '完成',
      data: { sheetName: '批发', status: 'imported' }
    })
    expect(tasks[0].status).toBe('imported')
    expect(tasks[0].progress).toBe(100)
  })

  it('keeps order when multiple sheets appear', () => {
    let tasks: ImportTask[] = []
    tasks = applyImportEvent(tasks, { type: 'sheet_start', data: { sheet_name: '零售' } })
    tasks = applyImportEvent(tasks, { type: 'sheet_start', data: { sheet_name: '住宿' } })
    expect(tasks.map((t) => t.sheetName)).toEqual(['零售', '住宿'])
  })
})
