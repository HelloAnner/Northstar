/**
 * 导入进度事件处理
 *
 * @author Anner
 * Created on 2026/2/5
 */

export type ImportTaskStatus = 'pending' | 'running' | 'imported' | 'skipped' | 'error'

export type ImportTask = {
  sheetName: string
  status: ImportTaskStatus
  progress: number
  message?: string
  sheetType?: string
}

export type ImportEvent = {
  type?: string
  message?: string
  data?: any
}

export function applyImportEvent(tasks: ImportTask[], event: ImportEvent): ImportTask[] {
  if (!event.type) return tasks
  if (event.type === 'sheet_start') {
    const sheetName = String(event.data?.sheet_name || '')
    if (!sheetName) return tasks
    if (tasks.some((t) => t.sheetName === sheetName)) {
      return tasks.map((t) =>
        t.sheetName === sheetName ? { ...t, status: 'running', progress: 40 } : t
      )
    }
    return [
      ...tasks,
      {
        sheetName,
        status: 'running',
        progress: 40,
        message: event.message,
        sheetType: event.data?.sheet_type,
      },
    ]
  }
  if (event.type === 'sheet_done') {
    const sheetName = String(event.data?.sheetName || '')
    if (!sheetName) return tasks
    const status = normalizeStatus(event.data?.status)
    return tasks.map((t) =>
      t.sheetName === sheetName
        ? {
            ...t,
            status,
            progress: 100,
            message: event.message,
            sheetType: event.data?.sheetType || t.sheetType,
          }
        : t
    )
  }
  return tasks
}

function normalizeStatus(raw: string | undefined): ImportTaskStatus {
  if (raw === 'imported') return 'imported'
  if (raw === 'skipped') return 'skipped'
  if (raw === 'error') return 'error'
  return 'imported'
}
