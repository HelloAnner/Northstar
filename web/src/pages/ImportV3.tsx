/**
 * 导入全屏页面（进度看板 + 日志追踪）
 *
 * @author Anner
 * Created on 2026/2/5
 */

import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { RefreshCw, Upload, X } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Progress } from '@/components/ui/progress'
import { ScrollArea } from '@/components/ui/scroll-area'
import { type ColumnMappingItem } from './importMapping'
import { applyImportEvent, type ImportEvent, type ImportTask } from './importProgress'

type PreviewSheet = {
  sheetName: string
  sheetType: string
  confidence: number
  status: string
  totalRows: number
  totalColumns: number
  columns: string[]
  columnMapping: ColumnMappingItem[]
  errors?: string
  importedRows: number
}

type ImportPreview = {
  importLog?: {
    filename: string
    totalSheets: number
    importedSheets: number
    skippedSheets: number
  }
  sheets: PreviewSheet[]
}

type ImportLogLevel = 'info' | 'success' | 'warn' | 'error'

type ImportLogLine = {
  id: string
  time: string
  level: ImportLogLevel
  message: string
}

type ProgressItem = {
  sheetName: string
  status?: string
  progress?: number
  message?: string
  sheetType?: string
}

const statusLabel = (status?: string) => {
  switch (status) {
    case 'pending':
      return '等待'
    case 'running':
      return '进行中'
    case 'imported':
      return '已导入'
    case 'skipped':
      return '已跳过'
    case 'error':
      return '错误'
    case 'warn':
      return '警告'
    default:
      return status || '未知'
  }
}

const statusBadgeClass = (status?: string) => {
  switch (status) {
    case 'imported':
      return 'border-[#BBF7D0] bg-[#F0FDF4] text-[#15803D]'
    case 'running':
      return 'border-[#BFDBFE] bg-[#EFF6FF] text-[#1D4ED8]'
    case 'warn':
    case 'skipped':
      return 'border-[#FDE68A] bg-[#FFFBEB] text-[#B45309]'
    case 'error':
      return 'border-[#FECACA] bg-[#FEF2F2] text-[#B91C1C]'
    default:
      return 'border-[#E2E8F0] bg-[#F8FAFC] text-[#64748B]'
  }
}

const logBadgeClass = (level: ImportLogLevel) => {
  switch (level) {
    case 'success':
      return 'border-[#BBF7D0] bg-[#F0FDF4] text-[#15803D]'
    case 'warn':
      return 'border-[#FDE68A] bg-[#FFFBEB] text-[#B45309]'
    case 'error':
      return 'border-[#FECACA] bg-[#FEF2F2] text-[#B91C1C]'
    default:
      return 'border-[#BFDBFE] bg-[#EFF6FF] text-[#1D4ED8]'
  }
}

const sheetTypeLabel = (sheetType?: string) => {
  switch (sheetType) {
    case 'wholesale':
      return '批发'
    case 'retail':
      return '零售'
    case 'accommodation':
      return '住宿'
    case 'catering':
      return '餐饮'
    case 'summary_limit_above_retail':
      return '限上汇总'
    case 'summary_micro_small':
      return '小微汇总'
    case 'summary_eat_wear_use':
      return '吃穿用汇总'
    default:
      return sheetType || '未知'
  }
}

interface ImportV3Props {
  onClose?: () => void
  onImported?: () => void
  modal?: boolean
}

export default function ImportV3(props: ImportV3Props = {}) {
  const navigate = useNavigate()
  const { onClose, onImported, modal = false } = props

  const [preview, setPreview] = useState<ImportPreview | null>(null)
  const [selectedFile, setSelectedFile] = useState<File | null>(null)
  const [importing, setImporting] = useState(false)
  const [importError, setImportError] = useState<string | null>(null)
  const [importStage, setImportStage] = useState('等待选择文件')
  const [importTotalSheets, setImportTotalSheets] = useState<number | null>(null)
  const [doneSheets, setDoneSheets] = useState(0)
  const [importTasks, setImportTasks] = useState<ImportTask[]>([])
  const [importCompleted, setImportCompleted] = useState(false)
  const [importLogs, setImportLogs] = useState<ImportLogLine[]>([])

  const fileInputRef = useRef<HTMLInputElement>(null)
  const logEndRef = useRef<HTMLDivElement>(null)

  const sheets = useMemo(() => preview?.sheets ?? [], [preview])
  const importLog = preview?.importLog

  const totalSheets = importLog?.totalSheets ?? sheets.length
  const importedSheets = importLog?.importedSheets ?? sheets.filter((sheet) => sheet.status === 'imported').length
  const warnSheets = Math.max(totalSheets - importedSheets, 0)
  const progressValue = totalSheets > 0 ? Math.round((importedSheets / totalSheets) * 100) : 0

  const importPercent = useMemo(() => {
    if (!importTotalSheets || importTotalSheets <= 0) {
      return 0
    }
    const pct = Math.round((doneSheets / importTotalSheets) * 100)
    return Math.min(100, Math.max(0, pct))
  }, [doneSheets, importTotalSheets])

  const displayProgress = importing ? importPercent : progressValue

  const loadPreview = useCallback(async () => {
    try {
      const response = await fetch('/api/import/preview')
      if (!response.ok) {
        return
      }
      const data = (await response.json()) as ImportPreview
      setPreview(data)
    } catch {
      // ignore
    }
  }, [])

  useEffect(() => {
    void loadPreview()
  }, [loadPreview])

  const appendLog = useCallback((line: ImportLogLine) => {
    setImportLogs((previous) => {
      const next = [...previous, line]
      if (next.length > 600) {
        return next.slice(next.length - 600)
      }
      return next
    })
  }, [])

  useEffect(() => {
    if (!logEndRef.current) {
      return
    }
    logEndRef.current.scrollIntoView({ behavior: 'smooth', block: 'end' })
  }, [importLogs])

  const handleFileChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0] || null
    setSelectedFile(file)
    setImportError(null)
    setImportCompleted(false)
    setImportTasks([])
    if (file) {
      setImportStage('已选择文件，准备导入')
    } else {
      setImportStage('等待选择文件')
    }
  }

  const handleImport = async () => {
    if (!selectedFile) {
      setImportError('请选择文件')
      return
    }

    setImporting(true)
    setImportStage('开始导入...')
    setImportError(null)
    setImportTotalSheets(null)
    setDoneSheets(0)
    setImportTasks([])
    setImportCompleted(false)
    setImportLogs([])

    appendLog({
      id: String(Date.now()),
      time: new Date().toLocaleTimeString('zh-CN', { hour12: false }),
      level: 'info',
      message: `开始导入：${selectedFile.name}`,
    })

    try {
      const formData = new FormData()
      formData.append('file', selectedFile)
      formData.append('clearExisting', 'true')
      formData.append('updateConfigYM', 'true')

      const response = await fetch('/api/import', {
        method: 'POST',
        body: formData,
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
      const doneSet = new Set<string>()

      while (true) {
        const { done, value } = await reader.read()
        if (done) {
          break
        }

        buffer += decoder.decode(value, { stream: true })
        const lines = buffer.split('\n')
        buffer = lines.pop() || ''

        for (const line of lines) {
          if (!line.startsWith('data: ')) {
            continue
          }

          const jsonStr = line.slice(6)
          try {
            const event = JSON.parse(jsonStr) as ImportEvent
            const now = new Date().toLocaleTimeString('zh-CN', { hour12: false })

            const sheetName = String(event.data?.sheetName || event.data?.sheet_name || '')

            if (event.type === 'sheet_start') {
              appendLog({
                id: `${now}-${Math.random()}`,
                time: now,
                level: 'info',
                message: sheetName ? `开始解析 Sheet：${sheetName}` : (event.message || '开始解析 Sheet'),
              })
            } else if (event.type === 'sheet_done') {
              const status = String(event.data?.status || '')
              const level: ImportLogLevel = status === 'error' ? 'error' : status === 'skipped' ? 'warn' : 'success'
              appendLog({
                id: `${now}-${Math.random()}`,
                time: now,
                level,
                message: sheetName ? `完成 Sheet：${sheetName}（${status || 'imported'}）` : (event.message || 'Sheet 完成'),
              })
            } else if (event.type === 'error') {
              appendLog({
                id: `${now}-${Math.random()}`,
                time: now,
                level: 'error',
                message: event.message || '导入失败',
              })
            } else if (event.type === 'done') {
              appendLog({
                id: `${now}-${Math.random()}`,
                time: now,
                level: 'success',
                message: event.message || '导入完成',
              })
            } else if (event.message) {
              appendLog({
                id: `${now}-${Math.random()}`,
                time: now,
                level: 'info',
                message: event.message,
              })
            }

            if (event.message) {
              setImportStage(event.message)
            }

            if (event.type === 'info' && event.data && typeof event.data === 'object') {
              const total = event.data.total_sheets as number | undefined
              if (typeof total === 'number' && total > 0) {
                setImportTotalSheets(total)
              }
            }

            if (event.type === 'sheet_done' && event.data && typeof event.data === 'object') {
              if (sheetName && !doneSet.has(sheetName)) {
                doneSet.add(sheetName)
                setDoneSheets(doneSet.size)
              }
            }

            if (event.type === 'sheet_start' || event.type === 'sheet_done') {
              setImportTasks((previous) => applyImportEvent(previous, event))
            }

            if (event.type === 'error') {
              setImportError(event.message || '导入失败')
              setImporting(false)
            }

            if (event.type === 'done') {
              setImportStage('导入完成')
              setImporting(false)
              setImportCompleted(true)
              setSelectedFile(null)
              if (fileInputRef.current) {
                fileInputRef.current.value = ''
              }
              await loadPreview()
            }
          } catch {
            const now = new Date().toLocaleTimeString('zh-CN', { hour12: false })
            appendLog({
              id: `${now}-${Math.random()}`,
              time: now,
              level: 'warn',
              message: '解析导入日志失败，已跳过一条消息',
            })
          }
        }
      }
    } catch (error) {
      setImportError(error instanceof Error ? error.message : '导入失败')
      setImporting(false)
      const now = new Date().toLocaleTimeString('zh-CN', { hour12: false })
      appendLog({
        id: `${now}-${Math.random()}`,
        time: now,
        level: 'error',
        message: error instanceof Error ? error.message : '导入失败',
      })
    }
  }

  const progressItems = useMemo<ProgressItem[]>(() => {
    if (importTasks.length > 0) {
      return importTasks.map((task) => ({
        sheetName: task.sheetName,
        status: task.status,
        progress: task.progress,
        message: task.message,
        sheetType: task.sheetType,
      }))
    }
    if (sheets.length > 0) {
      return sheets.map((sheet) => ({
        sheetName: sheet.sheetName,
        status: sheet.status,
        progress: sheet.status === 'imported' || sheet.status === 'skipped' ? 100 : sheet.status === 'warn' ? 70 : 30,
        message: sheet.errors,
        sheetType: sheet.sheetType,
      }))
    }
    return []
  }, [importTasks, sheets])

  const importStatusText = useMemo(() => {
    if (importing) {
      return '导入中'
    }
    if (importCompleted) {
      return '已完成'
    }
    if (selectedFile) {
      return '待导入'
    }
    return '未开始'
  }, [importing, importCompleted, selectedFile])

  const liveSummary = useMemo(() => {
    if (importing || importTotalSheets !== null || importTasks.length > 0) {
      const total = importTotalSheets ?? Math.max(importTasks.length, 0)
      const imported = importTasks.filter((task) => task.status === 'imported').length
      const warn = importTasks.filter((task) => task.status === 'skipped' || task.status === 'error').length
      const errors = importTasks.filter((task) => task.status === 'error').length
      return { total, imported, warn, errors, live: true }
    }
    const errors = sheets.filter((sheet) => sheet.status === 'error').length
    return { total: totalSheets, imported: importedSheets, warn: warnSheets, errors, live: false }
  }, [importing, importTotalSheets, importTasks, totalSheets, importedSheets, warnSheets, sheets])

  const closeImport = () => {
    if (onClose) {
      onClose()
      return
    }
    navigate('/')
  }

  const finishImport = () => {
    if (onImported) {
      onImported()
    }
    closeImport()
  }

  const containerClass = modal ? "h-full min-h-0" : "min-h-full bg-[#F8FAFC] px-4 py-4"
  const panelClass = modal
    ? "h-full w-full overflow-hidden rounded-none border-0 bg-white shadow-none"
    : "mx-auto h-[calc(100vh-32px)] max-w-[1220px] overflow-hidden rounded-2xl border border-[#E2E8F0] bg-white shadow-[0_12px_36px_rgba(15,23,42,0.08)]"
  const contentClass = modal ? "flex h-[calc(100%-84px)] min-h-0" : "flex h-[calc(100%-92px)] min-h-0"

  return (
    <div className={containerClass}>
      <div className={panelClass}>
        <header className="flex items-center justify-between border-b border-[#E2E8F0] px-6 py-4">
          <div>
            <h1 className="text-[28px] font-semibold leading-[36px] text-[#0F172A]">导入数据</h1>
            <p className="mt-1 text-xs text-[#64748B]">选择 Excel 文件，完成批量导入并查看实时日志</p>
          </div>
          <Button
            variant="ghost"
            className="h-8 w-8 p-0 text-[#94A3B8] hover:bg-[#F1F5F9] hover:text-[#0F172A]"
            onClick={closeImport}
            aria-label="关闭导入页面"
          >
            <X className="h-4 w-4" />
          </Button>
        </header>

        <div className={contentClass}>
          <section data-testid="import-left-pane" className="flex w-[48%] min-w-0 flex-col border-r border-[#E2E8F0] bg-white">
            <ScrollArea className="h-full">
              <div className="space-y-4 p-5">
                <div className="rounded-xl border border-[#E2E8F0] bg-[#F8FAFC] p-4">
                  <div className="text-sm font-semibold text-[#0F172A]">选择 Excel 文件</div>
                  <div className="mt-3 flex items-center gap-2">
                    <Input
                      ref={fileInputRef}
                      type="file"
                      accept=".xlsx,.xls"
                      className="h-9 border-[#E2E8F0] bg-white text-xs"
                      onChange={handleFileChange}
                    />
                    <Button
                      className="h-9 min-w-[100px] gap-1 bg-[#3B82F6] px-3 text-xs text-white hover:bg-[#2563EB]"
                      onClick={importCompleted ? finishImport : handleImport}
                      disabled={importing}
                    >
                      <Upload className="h-3.5 w-3.5" />
                      {importCompleted ? '进入仪表盘' : '开始导入'}
                    </Button>
                  </div>
                  {selectedFile && (
                    <div className="mt-2 text-xs text-[#64748B]">
                      已选择：<span className="text-[#0F172A]">{selectedFile.name}</span>
                    </div>
                  )}
                  {importError && <div className="mt-2 text-xs text-[#DC2626]">{importError}</div>}
                </div>

                <div className="rounded-xl border border-[#E2E8F0] bg-white p-4">
                  <div className="flex items-center justify-between gap-2">
                    <div>
                      <div className="text-sm font-semibold text-[#0F172A]">导入进度</div>
                      <div className="mt-1 text-xs text-[#94A3B8]">{importStage}</div>
                    </div>
                    <div className="flex items-center gap-2">
                      <Badge className="h-6 border-[#E2E8F0] bg-[#F8FAFC] px-2 text-[11px] text-[#475569]">
                        {importStatusText}
                      </Badge>
                      <Button
                        variant="ghost"
                        className="h-7 w-7 p-0 text-[#64748B] hover:bg-[#F1F5F9]"
                        onClick={() => void loadPreview()}
                      >
                        <RefreshCw className="h-3.5 w-3.5" />
                      </Button>
                    </div>
                  </div>

                  <div className="mt-3">
                    <Progress value={displayProgress} className="h-2" />
                    <div className="mt-2 grid grid-cols-2 gap-2 text-xs text-[#64748B]">
                      <div>总计 {liveSummary.total} 个 Sheet</div>
                      <div>完成 {liveSummary.imported + liveSummary.warn} 个</div>
                      <div>正常 {liveSummary.imported} 个</div>
                      <div>异常 {liveSummary.errors} 个</div>
                    </div>
                  </div>

                  <div className="mt-3 space-y-2">
                    {progressItems.length === 0 && <div className="text-xs text-[#94A3B8]">暂无进度</div>}

                    {progressItems.map((item) => (
                      <div key={item.sheetName} className="rounded-lg border border-[#E2E8F0] bg-white p-2.5">
                        <div className="flex items-center justify-between gap-2">
                          <div className="min-w-0">
                            <div className="truncate text-xs font-medium text-[#0F172A]">{item.sheetName}</div>
                            <div className="mt-1 text-[11px] text-[#94A3B8]">{sheetTypeLabel(item.sheetType)}</div>
                          </div>
                          <Badge className={`h-5 border px-2 text-[10px] ${statusBadgeClass(item.status)}`}>
                            {statusLabel(item.status)}
                          </Badge>
                        </div>
                        {typeof item.progress === 'number' && (
                          <div className="mt-2">
                            <Progress value={item.progress} className="h-1.5" />
                          </div>
                        )}
                        {item.message && <div className="mt-1 text-[11px] text-[#64748B]">{item.message}</div>}
                      </div>
                    ))}
                  </div>
                </div>
              </div>
            </ScrollArea>
          </section>

          <section data-testid="import-right-pane" className="flex min-w-0 flex-1 flex-col bg-white">
            <div className="flex items-center justify-between border-b border-[#E2E8F0] px-6 py-4">
              <div>
                <h2 className="text-lg font-semibold text-[#0F172A]">导入日志</h2>
                <p className="mt-1 text-xs text-[#64748B]">实时记录每个 Sheet 的解析和导入结果</p>
              </div>
              <Badge className="h-6 border-[#E2E8F0] bg-[#F8FAFC] px-2 text-[11px] text-[#475569]">
                {importLogs.length} 条
              </Badge>
            </div>

            <ScrollArea className="h-full">
              <div className="space-y-2 p-5">
                {importLogs.length === 0 && <div className="text-xs text-[#94A3B8]">暂无日志</div>}

                {importLogs.map((line) => (
                  <div key={line.id} className="flex items-start gap-2 rounded-lg border border-[#E2E8F0] bg-[#F8FAFC] px-3 py-2">
                    <span className="mt-0.5 w-[58px] shrink-0 font-mono text-[11px] text-[#64748B]">{line.time}</span>
                    <Badge className={`h-5 border px-2 text-[10px] ${logBadgeClass(line.level)}`}>
                      {line.level.toUpperCase()}
                    </Badge>
                    <span className="text-xs leading-5 text-[#334155]">{line.message}</span>
                  </div>
                ))}
                <div ref={logEndRef} />
              </div>
            </ScrollArea>
          </section>
        </div>
      </div>
    </div>
  )
}
