/**
 * 导入全屏页面（进度看板 + 日志追踪）
 *
 * @author Anner
 * Created on 2026/2/5
 */

import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Progress } from '@/components/ui/progress'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Separator } from '@/components/ui/separator'
import { Upload, RefreshCw, AlertTriangle } from 'lucide-react'
import { type ColumnMappingItem } from './importMapping'
import { applyImportEvent, type ImportTask, type ImportEvent } from './importProgress'

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

export default function ImportV3() {
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
  const navigate = useNavigate()
  const logEndRef = useRef<HTMLDivElement>(null)
  const sheets = preview?.sheets ?? []
  const importLog = preview?.importLog
  const totalSheets = importLog?.totalSheets ?? sheets.length
  const importedSheets =
    importLog?.importedSheets ?? sheets.filter((sheet) => sheet.status === 'imported').length
  const warnSheets = Math.max(totalSheets - importedSheets, 0)
  const progressValue = totalSheets > 0 ? Math.round((importedSheets / totalSheets) * 100) : 0
  const errorSheets = sheets.filter((sheet) => sheet.status === 'error').length

  const liveSummary = useMemo(() => {
    if (importing || importTotalSheets !== null || importTasks.length > 0) {
      const total = importTotalSheets ?? Math.max(importTasks.length, 0)
      const imported = importTasks.filter((task) => task.status === 'imported').length
      const warn = importTasks.filter((task) => task.status === 'skipped' || task.status === 'error').length
      const errors = importTasks.filter((task) => task.status === 'error').length
      return { total, imported, warn, errors, live: true }
    }
    return { total: totalSheets, imported: importedSheets, warn: warnSheets, errors: errorSheets, live: false }
  }, [importing, importTotalSheets, importTasks, totalSheets, importedSheets, warnSheets, errorSheets])

  const loadPreview = useCallback(async () => {
    try {
      const res = await fetch('/api/import/preview')
      const data = (await res.json()) as ImportPreview
      setPreview(data)
    } catch (err) {
      console.error(err)
    }
  }, [])

  useEffect(() => {
    setPreview(null)
    setSelectedFile(null)
    setImporting(false)
    setImportError(null)
    setImportStage('等待选择文件')
    setImportTotalSheets(null)
    setDoneSheets(0)
    setImportTasks([])
    setImportCompleted(false)
    setImportLogs([])
    if (fileInputRef.current) {
      fileInputRef.current.value = ''
    }
  }, [])

  const importPercent = useMemo(() => {
    if (!importTotalSheets || importTotalSheets <= 0) return 0
    const pct = Math.round((doneSheets / importTotalSheets) * 100)
    return Math.min(100, Math.max(0, pct))
  }, [doneSheets, importTotalSheets])

  const displayProgress = importing ? importPercent : progressValue

  const appendLog = useCallback((line: ImportLogLine) => {
    setImportLogs((prev) => {
      const next = [...prev, line]
      if (next.length > 600) {
        return next.slice(next.length - 600)
      }
      return next
    })
  }, [])

  useEffect(() => {
    if (!logEndRef.current) return
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
    setImportStage('开始导入…')
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
        if (done) break
        buffer += decoder.decode(value, { stream: true })
        const lines = buffer.split('\n')
        buffer = lines.pop() || ''

        for (const line of lines) {
          if (!line.startsWith('data: ')) continue
          const jsonStr = line.slice(6)
          try {
            const event = JSON.parse(jsonStr) as ImportEvent
            const now = new Date().toLocaleTimeString('zh-CN', { hour12: false })
            if (event.type === 'sheet_start') {
              const sheetName = String(event.data?.sheet_name || '')
              appendLog({
                id: `${now}-${Math.random()}`,
                time: now,
                level: 'info',
                message: sheetName ? `开始解析 Sheet：${sheetName}` : (event.message || '开始解析 Sheet'),
              })
            } else if (event.type === 'sheet_done') {
              const sheetName = String(event.data?.sheetName || '')
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
              const name = String(event.data.sheetName || '')
              if (name && !doneSet.has(name)) {
                doneSet.add(name)
                setDoneSheets(doneSet.size)
              }
            }
            if (event.type === 'sheet_start' || event.type === 'sheet_done') {
              setImportTasks((prev) => applyImportEvent(prev, event))
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
          } catch (err) {
            console.error('Failed to parse import event:', err)
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
    } catch (err) {
      setImportError(err instanceof Error ? err.message : '导入失败')
      setImporting(false)
      const now = new Date().toLocaleTimeString('zh-CN', { hour12: false })
      appendLog({
        id: `${now}-${Math.random()}`,
        time: now,
        level: 'error',
        message: err instanceof Error ? err.message : '导入失败',
      })
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

  const statusVariant = (status?: string) => {
    if (status === 'error') return 'destructive'
    if (status === 'warn') return 'secondary'
    if (status === 'running') return 'secondary'
    if (status === 'pending') return 'outline'
    if (status === 'skipped') return 'outline'
    return 'secondary'
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
        progress:
          sheet.status === 'imported' || sheet.status === 'skipped'
            ? 100
            : sheet.status === 'warn'
              ? 70
              : 30,
        message: sheet.errors,
        sheetType: sheet.sheetType,
      }))
    }
    return []
  }, [importTasks, sheets])

  const importStatusText = useMemo(() => {
    if (importing) return '导入中'
    if (importCompleted) return '已完成'
    if (selectedFile) return '待导入'
    return '未开始'
  }, [importing, importCompleted, selectedFile])

  const importStatusVariant = importing ? 'secondary' : importCompleted ? 'outline' : 'secondary'

  const logLevelBadge = (level: ImportLogLevel) => {
    if (level === 'error') return 'destructive'
    if (level === 'warn') return 'secondary'
    if (level === 'success') return 'outline'
    return 'secondary'
  }


  return (
    <div className="h-full w-full overflow-hidden bg-muted/20">
      <div className="flex h-full flex-col">
        <header className="shrink-0 border-b border-border/60 bg-background/80 px-6 py-4 backdrop-blur">
          <div className="flex items-center justify-between gap-4">
            <div>
              <h1 className="text-lg font-semibold text-foreground">导入数据</h1>
              <p className="text-sm text-muted-foreground">全量解析 · 进度看板 · 日志追踪</p>
            </div>
            <div className="flex items-center gap-2">
              <Button variant="outline" size="sm" className="gap-2" onClick={loadPreview}>
                <RefreshCw className="h-4 w-4" />
                重新解析
              </Button>
              <Button
                size="sm"
                className="gap-2"
                onClick={importCompleted ? () => navigate('/') : handleImport}
                disabled={importing}
              >
                <Upload className="h-4 w-4" />
                {importCompleted ? '进入仪表盘' : '开始导入'}
              </Button>
            </div>
          </div>
          <div className="mt-4 flex flex-wrap items-center gap-3 rounded-lg border border-border/60 bg-background/70 px-4 py-3">
            <div className="text-xs text-muted-foreground">选择文件</div>
            <Input
              ref={fileInputRef}
              type="file"
              accept=".xlsx,.xls"
              className="h-8 w-[260px]"
              onChange={handleFileChange}
            />
            <div className="text-xs text-muted-foreground">
              导入方式: <span className="text-foreground">覆盖导入（默认）</span>
            </div>
            <div className="text-xs text-muted-foreground">{importStage}</div>
            {selectedFile && (
              <div className="text-xs text-muted-foreground">
                已选择: <span className="text-foreground">{selectedFile.name}</span>
              </div>
            )}
            {importError && (
              <div className="text-xs text-destructive">{importError}</div>
            )}
          </div>
          <div className="mt-4 grid grid-cols-4 gap-3">
            <Card className="border-border/60 bg-card/60 p-3">
              <div className="text-xs text-muted-foreground">识别结果</div>
              <div className="mt-1 flex items-center gap-2 text-sm">
                <Badge variant="secondary">{liveSummary.total} 个 Sheet</Badge>
                <Badge variant="secondary">{liveSummary.imported} 正常</Badge>
                <Badge variant="secondary">{liveSummary.warn} 警告</Badge>
              </div>
            </Card>
            <Card className="border-border/60 bg-card/60 p-3">
              <div className="text-xs text-muted-foreground">进度</div>
              <div className="mt-2">
                <Progress value={displayProgress} className="h-2" />
              </div>
            </Card>
            <Card className="border-border/60 bg-card/60 p-3">
              <div className="text-xs text-muted-foreground">异常</div>
              <div className="mt-1 flex items-center gap-2 text-sm">
                <AlertTriangle className="h-4 w-4 text-amber-400" />
                {liveSummary.errors > 0 ? `${liveSummary.errors} 个 Sheet 需关注` : '暂无异常'}
              </div>
            </Card>
          </div>
        </header>

        <div className="flex h-full min-h-0">
          <section
            data-testid="import-left-pane"
            className="flex h-full w-1/2 flex-col border-r border-border/60 bg-background"
          >
            <div className="shrink-0 border-b border-border/60 px-6 py-4">
              <div className="flex items-center justify-between">
                <div>
                  <h2 className="text-sm font-semibold text-foreground">导入进度</h2>
                  <p className="text-xs text-muted-foreground">实时进度 + Sheet 执行状态</p>
                </div>
                <Badge variant="secondary">{progressItems.length} 项</Badge>
              </div>
            </div>

            <ScrollArea className="h-full">
              <div className="p-6 space-y-4">
                <Card className="border-border/60 bg-card/60 p-4">
                  <div className="flex items-center justify-between">
                    <div className="text-xs font-medium text-muted-foreground">当前阶段</div>
                    <Badge variant={importStatusVariant}>{importStatusText}</Badge>
                  </div>
                  <div className="mt-2 text-sm font-medium text-foreground">
                    {importStage}
                  </div>
                  <div className="mt-3">
                    <Progress value={displayProgress} className="h-2" />
                    <div className="mt-2 flex flex-wrap items-center gap-3 text-xs text-muted-foreground">
                      <span>已完成 {liveSummary.imported + liveSummary.warn}/{liveSummary.total} 个 Sheet</span>
                      <span>异常 {liveSummary.errors} 个</span>
                    </div>
                  </div>
                </Card>

                <Card className="border-border/60 bg-card/60 p-4">
                  <div className="flex items-center justify-between">
                    <div className="text-xs font-medium text-muted-foreground">Sheet 任务</div>
                    <Badge variant="secondary">{progressItems.length} 项</Badge>
                  </div>
                  <Separator className="my-3" />
                  {progressItems.length === 0 ? (
                    <div className="text-xs text-muted-foreground">暂无进度</div>
                  ) : (
                    <div className="space-y-3">
                      {progressItems.map((item) => (
                        <div
                          key={item.sheetName}
                          className="rounded-md border border-border/60 bg-background/60 p-3"
                        >
                          <div className="flex items-center justify-between gap-3">
                            <div className="space-y-1">
                              <div className="text-sm font-medium text-foreground">
                                {item.sheetName}
                              </div>
                              <div className="flex items-center gap-2 text-xs text-muted-foreground">
                                <span>{sheetTypeLabel(item.sheetType)}</span>
                                <span>·</span>
                                <span>{statusLabel(item.status)}</span>
                              </div>
                            </div>
                            <Badge variant={statusVariant(item.status)}>
                              {statusLabel(item.status)}
                            </Badge>
                          </div>
                          {typeof item.progress === 'number' && (
                            <div className="mt-2">
                              <Progress value={item.progress} className="h-2" />
                            </div>
                          )}
                          {item.message && (
                            <div className="mt-2 text-xs text-muted-foreground">
                              {item.message}
                            </div>
                          )}
                        </div>
                      ))}
                    </div>
                  )}
                </Card>
              </div>
            </ScrollArea>
          </section>

          <section
            data-testid="import-right-pane"
            className="flex h-full w-1/2 flex-col bg-background"
          >
            <div className="shrink-0 border-b border-border/60 px-6 py-4">
              <div className="flex items-center justify-between">
                <div>
                  <h2 className="text-sm font-semibold text-foreground">导入日志</h2>
                  <p className="text-xs text-muted-foreground">实时滚动输出 · 自动跟随</p>
                </div>
                <Badge variant="secondary">{importLogs.length} 条</Badge>
              </div>
            </div>

            <ScrollArea className="h-full">
              <div className="p-6">
                <Card className="border-border/60 bg-card/60 p-4">
                  <div className="text-xs font-medium text-muted-foreground">详细日志</div>
                  <Separator className="my-3" />
                  {importLogs.length === 0 ? (
                    <div className="text-xs text-muted-foreground">暂无日志</div>
                  ) : (
                    <div className="space-y-2">
                      {importLogs.map((line) => (
                        <div
                          key={line.id}
                          className="flex items-start gap-3 rounded-md border border-border/60 bg-background/60 px-3 py-2 text-xs"
                        >
                          <span className="w-[70px] shrink-0 font-mono text-muted-foreground">
                            {line.time}
                          </span>
                          <Badge variant={logLevelBadge(line.level)} className="h-5 px-2 text-[10px]">
                            {line.level.toUpperCase()}
                          </Badge>
                          <span className="text-foreground">{line.message}</span>
                        </div>
                      ))}
                      <div ref={logEndRef} />
                    </div>
                  )}
                </Card>
              </div>
            </ScrollArea>
          </section>
        </div>
      </div>
    </div>
  )
}
