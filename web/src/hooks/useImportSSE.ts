/**
 * 导入 SSE 流处理 Hook（含超时降级和轮询兜底）
 *
 * @author Anner
 * Created on 2026/3/28
 */

import { useCallback, useRef, useState } from 'react'
import { applyImportEvent, type ImportEvent, type ImportTask } from '@/pages/importProgress'

export type ImportStatus = 'idle' | 'importing' | 'done' | 'error'

export interface ImportResult {
  totalSheets: number
  importedSheets: number
  skippedSheets: number
  totalRows: number
  importedRows: number
}

const WARN_TIMEOUT = 30_000
const POLL_TIMEOUT = 60_000
const POLL_INTERVAL = 5_000

export function useImportSSE() {
  const [status, setStatus] = useState<ImportStatus>('idle')
  const [progress, setProgress] = useState(0)
  const [events, setEvents] = useState<ImportEvent[]>([])
  const [sheetTasks, setSheetTasks] = useState<ImportTask[]>([])
  const [error, setError] = useState<string | null>(null)
  const [stale, setStale] = useState(false)
  const [result, setResult] = useState<ImportResult | null>(null)

  const totalSheetsRef = useRef(0)
  const doneSheetsRef = useRef(new Set<string>())
  const lastEventTimeRef = useRef(0)
  const warnTimerRef = useRef<ReturnType<typeof setTimeout>>(undefined)
  const pollTimerRef = useRef<ReturnType<typeof setInterval>>(undefined)
  const abortRef = useRef<AbortController>(undefined)

  const clearTimers = useCallback(() => {
    if (warnTimerRef.current) clearTimeout(warnTimerRef.current)
    if (pollTimerRef.current) clearInterval(pollTimerRef.current)
  }, [])

  const resetStaleTimer = useCallback(() => {
    lastEventTimeRef.current = Date.now()
    setStale(false)
    if (warnTimerRef.current) clearTimeout(warnTimerRef.current)
    warnTimerRef.current = setTimeout(() => setStale(true), WARN_TIMEOUT)
  }, [])

  // 轮询降级：SSE 超时后定期查询导入状态
  const startPolling = useCallback(() => {
    if (pollTimerRef.current) return
    pollTimerRef.current = setInterval(async () => {
      try {
        const res = await fetch('/api/import/preview')
        if (!res.ok) return
        const data = await res.json()
        if (data.importLog?.status === 'completed' || data.importLog?.status === 'failed') {
          clearTimers()
          const log = data.importLog
          if (log.status === 'completed') {
            setStatus('done')
            setProgress(100)
            setResult({
              totalSheets: log.totalSheets || 0,
              importedSheets: log.importedSheets || 0,
              skippedSheets: log.skippedSheets || 0,
              totalRows: log.totalRows || 0,
              importedRows: log.importedRows || 0,
            })
          } else {
            setStatus('error')
            setError(log.errorMessage || '导入失败')
          }
        }
      } catch {
        // 轮询失败不做额外处理
      }
    }, POLL_INTERVAL)
  }, [clearTimers])

  const appendEvent = useCallback((event: ImportEvent) => {
    setEvents((prev) => {
      const next = [...prev, event]
      return next.length > 20 ? next.slice(-20) : next
    })
  }, [])

  const handleEvent = useCallback((event: ImportEvent) => {
    resetStaleTimer()
    appendEvent(event)

    if (event.type === 'info' && event.data?.total_sheets) {
      totalSheetsRef.current = event.data.total_sheets
    }

    if (event.type === 'sheet_start' || event.type === 'sheet_done') {
      setSheetTasks((prev) => applyImportEvent(prev, event))
    }

    if (event.type === 'sheet_done' && event.data?.sheetName) {
      const name = String(event.data.sheetName)
      doneSheetsRef.current.add(name)
      const total = totalSheetsRef.current
      if (total > 0) {
        setProgress(Math.min(100, Math.round((doneSheetsRef.current.size / total) * 100)))
      }
    }

    if (event.type === 'done') {
      clearTimers()
      setStatus('done')
      setProgress(100)
      if (event.data) {
        setResult({
          totalSheets: event.data.totalSheets || 0,
          importedSheets: event.data.importedSheets || 0,
          skippedSheets: event.data.skippedSheets || 0,
          totalRows: event.data.totalRows || 0,
          importedRows: event.data.importedRows || 0,
        })
      }
    }

    if (event.type === 'error') {
      clearTimers()
      setStatus('error')
      setError(event.message || '导入失败')
    }
  }, [resetStaleTimer, appendEvent, clearTimers])

  const start = useCallback(async (file: File, clearExisting: boolean) => {
    // 重置状态
    setStatus('importing')
    setProgress(0)
    setEvents([])
    setSheetTasks([])
    setError(null)
    setStale(false)
    setResult(null)
    totalSheetsRef.current = 0
    doneSheetsRef.current = new Set()
    clearTimers()

    const abort = new AbortController()
    abortRef.current = abort
    resetStaleTimer()

    // 设置超时后启动轮询
    const pollTimeout = setTimeout(() => startPolling(), POLL_TIMEOUT)

    try {
      const formData = new FormData()
      formData.append('file', file)
      formData.append('clearExisting', clearExisting ? 'true' : 'false')
      formData.append('updateConfigYM', 'true')

      const response = await fetch('/api/import', {
        method: 'POST',
        body: formData,
        signal: abort.signal,
      })

      if (!response.ok) {
        throw new Error('导入请求失败')
      }

      const reader = response.body?.getReader()
      const decoder = new TextDecoder()
      if (!reader) {
        throw new Error('无法读取响应流')
      }

      let buffer = ''
      while (true) {
        const { done, value } = await reader.read()
        if (done) break
        buffer += decoder.decode(value, { stream: true })
        const lines = buffer.split('\n')
        buffer = lines.pop() || ''

        for (const line of lines) {
          if (!line.startsWith('data: ')) continue
          try {
            const event: ImportEvent = JSON.parse(line.slice(6))
            handleEvent(event)
          } catch {
            // 跳过格式错误的事件
          }
        }
      }
    } catch (err) {
      if (abort.signal.aborted) return
      setStatus('error')
      setError(err instanceof Error ? err.message : '导入失败')
    } finally {
      clearTimeout(pollTimeout)
    }
  }, [handleEvent, resetStaleTimer, clearTimers, startPolling])

  const reset = useCallback(() => {
    clearTimers()
    abortRef.current?.abort()
    setStatus('idle')
    setProgress(0)
    setEvents([])
    setSheetTasks([])
    setError(null)
    setStale(false)
    setResult(null)
  }, [clearTimers])

  return { start, status, progress, events, sheetTasks, error, stale, result, reset }
}
