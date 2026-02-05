/**
 * 导入全屏页面（左右对比）
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
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Upload, RefreshCw, AlertTriangle } from 'lucide-react'
import { type ColumnMappingItem } from './importMapping'
import { applyImportEvent, type ImportTask, type ImportEvent } from './importProgress'
import { applyPreviewUpdate } from './importPreview'

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

type SheetColumn = {
  colIdx: number
  headerText: string
}

type SheetRow = {
  rowIdx: number
}

type SheetCell = {
  rowIdx: number
  colIdx: number
  rawValue: string
  calcValue: string
  formula: string
  isMerged: number
  mergeRange: string
}

type SheetPreview = {
  sheetName: string
  columns: SheetColumn[]
  rows: SheetRow[]
  cells: SheetCell[]
}

export default function ImportV3() {
  const [preview, setPreview] = useState<ImportPreview | null>(null)
  const [selectedSheet, setSelectedSheet] = useState('')
  const [sheetPreview, setSheetPreview] = useState<SheetPreview | null>(null)
  const [loadingPreview, setLoadingPreview] = useState(false)
  const [loadingSheet, setLoadingSheet] = useState(false)
  const [selectedFile, setSelectedFile] = useState<File | null>(null)
  const [importing, setImporting] = useState(false)
  const [importError, setImportError] = useState<string | null>(null)
  const [importStage, setImportStage] = useState('等待选择文件')
  const [importTotalSheets, setImportTotalSheets] = useState<number | null>(null)
  const [doneSheets, setDoneSheets] = useState(0)
  const [importTasks, setImportTasks] = useState<ImportTask[]>([])
  const [importCompleted, setImportCompleted] = useState(false)
  const [previewRefreshToken, setPreviewRefreshToken] = useState(0)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const navigate = useNavigate()
  const selectedSheetRef = useRef('')
  const previewTokenRef = useRef(0)
  const sheets = preview?.sheets ?? []
  const importLog = preview?.importLog
  const totalSheets = importLog?.totalSheets ?? sheets.length
  const importedSheets =
    importLog?.importedSheets ?? sheets.filter((sheet) => sheet.status === 'imported').length
  const warnSheets = Math.max(totalSheets - importedSheets, 0)
  const progressValue = totalSheets > 0 ? Math.round((importedSheets / totalSheets) * 100) : 0
  const errorSheets = sheets.filter((sheet) => sheet.status === 'error').length

  const activeSheet = useMemo(
    () => sheets.find((sheet) => sheet.sheetName === selectedSheet),
    [sheets, selectedSheet]
  )

  useEffect(() => {
    selectedSheetRef.current = selectedSheet
  }, [selectedSheet])

  useEffect(() => {
    previewTokenRef.current = previewRefreshToken
  }, [previewRefreshToken])

  const loadPreview = useCallback(async () => {
    setLoadingPreview(true)
    try {
      const res = await fetch('/api/import/preview')
      const data = (await res.json()) as ImportPreview
      setPreview(data)
      const selection = applyPreviewUpdate(
        selectedSheetRef.current,
        previewTokenRef.current,
        data.sheets ?? []
      )
      setSelectedSheet(selection.selectedSheet)
      setPreviewRefreshToken(selection.refreshToken)
      if (!selection.selectedSheet) {
        setSheetPreview(null)
      }
    } catch (err) {
      console.error(err)
    } finally {
      setLoadingPreview(false)
    }
  }, [])

  useEffect(() => {
    loadPreview()
  }, [loadPreview])

  useEffect(() => {
    const fetchSheet = async () => {
      if (!selectedSheet) return
      setLoadingSheet(true)
      try {
        const res = await fetch(`/api/import/sheet?name=${encodeURIComponent(selectedSheet)}&limitRows=50&limitCols=40`)
        const data = (await res.json()) as SheetPreview
        setSheetPreview(data)
      } catch (err) {
        console.error(err)
      } finally {
        setLoadingSheet(false)
      }
    }
    fetchSheet()
  }, [selectedSheet, previewRefreshToken])

  const previewHeaders = useMemo(() => {
    if (!sheetPreview?.columns?.length) return []
    return sheetPreview.columns.map((col) => col.headerText || `列${col.colIdx}`)
  }, [sheetPreview])

  const previewRows = useMemo(() => {
    if (!sheetPreview?.rows?.length || !sheetPreview?.columns?.length) return []
    const columns = sheetPreview.columns
    const cellMap = new Map<string, SheetCell>()
    for (const cell of sheetPreview.cells) {
      cellMap.set(`${cell.rowIdx}:${cell.colIdx}`, cell)
    }
    const rows = sheetPreview.rows.filter((row) => row.rowIdx > 1).slice(0, 20)
    return rows.map((row) => {
      return columns.map((col) => {
        const cell = cellMap.get(`${row.rowIdx}:${col.colIdx}`)
        if (!cell) return ''
        const value = cell.rawValue || cell.calcValue || cell.formula || ''
        return cell.isMerged ? `${value} (合并)` : value
      })
    })
  }, [sheetPreview])

  const importPercent = useMemo(() => {
    if (!importTotalSheets || importTotalSheets <= 0) return 0
    const pct = Math.round((doneSheets / importTotalSheets) * 100)
    return Math.min(100, Math.max(0, pct))
  }, [doneSheets, importTotalSheets])

  const displayProgress = importing ? importPercent : progressValue

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
          }
        }
      }
    } catch (err) {
      setImportError(err instanceof Error ? err.message : '导入失败')
      setImporting(false)
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
    if (status === 'skipped') return 'outline'
    return 'secondary'
  }


  return (
    <div className="h-full w-full overflow-hidden bg-muted/20">
      <div className="flex h-full flex-col">
        <header className="shrink-0 border-b border-border/60 bg-background/80 px-6 py-4 backdrop-blur">
          <div className="flex items-center justify-between gap-4">
            <div>
              <h1 className="text-lg font-semibold text-foreground">导入数据</h1>
              <p className="text-sm text-muted-foreground">全量解析 · 左右对比 · 100% 高度展示</p>
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
            <Card className="p-3">
              <div className="text-xs text-muted-foreground">文件</div>
              <div className="mt-1 text-sm font-medium">
                {loadingPreview ? '加载中...' : importLog?.filename ?? '暂无导入记录'}
              </div>
            </Card>
            <Card className="p-3">
              <div className="text-xs text-muted-foreground">识别结果</div>
              <div className="mt-1 flex items-center gap-2 text-sm">
                <Badge variant="secondary">{totalSheets} 个 Sheet</Badge>
                <Badge variant="secondary">{importedSheets} 正常</Badge>
                <Badge variant="secondary">{warnSheets} 警告</Badge>
              </div>
            </Card>
            <Card className="p-3">
              <div className="text-xs text-muted-foreground">进度</div>
              <div className="mt-2">
                <Progress value={displayProgress} className="h-2" />
              </div>
            </Card>
            <Card className="p-3">
              <div className="text-xs text-muted-foreground">异常</div>
              <div className="mt-1 flex items-center gap-2 text-sm">
                <AlertTriangle className="h-4 w-4 text-amber-400" />
                {errorSheets > 0 ? `${errorSheets} 个 Sheet 需关注` : '暂无异常'}
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
                  <h2 className="text-sm font-semibold text-foreground">原始预览</h2>
                  <p className="text-xs text-muted-foreground">多 Sheet 结构 + 列宽 + 合并提示</p>
                </div>
                <Input className="h-8 w-40" placeholder="搜索 Sheet" />
              </div>
              <div className="mt-3 flex flex-wrap gap-2">
                {sheets.length === 0 ? (
                  <span className="text-xs text-muted-foreground">暂无 Sheet</span>
                ) : (
                  sheets.map((sheet) => (
                    <button
                      key={sheet.sheetName}
                      onClick={() => setSelectedSheet(sheet.sheetName)}
                      className={`rounded-full border px-3 py-1 text-xs transition ${
                        selectedSheet === sheet.sheetName
                          ? 'border-primary bg-primary/10 text-primary'
                          : 'border-border/60 text-muted-foreground'
                      }`}
                    >
                      {sheet.sheetName}
                    </button>
                  ))
                )}
              </div>
            </div>

            <ScrollArea className="h-full">
              <div className="p-6">
                <div className="mb-4 flex items-center gap-2">
                  <Badge variant="secondary">{sheetTypeLabel(activeSheet?.sheetType)}</Badge>
                  <Badge variant={statusVariant(activeSheet?.status)}>
                    {statusLabel(activeSheet?.status)}
                  </Badge>
                  <Badge variant="secondary">
                    {Math.round((activeSheet?.confidence ?? 0) * 100)}%
                  </Badge>
                </div>
                <Card className="overflow-hidden">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        {previewHeaders.length > 0 ? (
                          previewHeaders.map((header) => (
                            <TableHead key={header} className="whitespace-nowrap">
                              {header}
                            </TableHead>
                          ))
                        ) : (
                          <TableHead className="whitespace-nowrap">暂无预览数据</TableHead>
                        )}
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {loadingSheet ? (
                        <TableRow>
                          <TableCell colSpan={Math.max(previewHeaders.length, 1)}>
                            加载中...
                          </TableCell>
                        </TableRow>
                      ) : previewRows.length === 0 ? (
                        <TableRow>
                          <TableCell colSpan={Math.max(previewHeaders.length, 1)}>
                            暂无预览数据
                          </TableCell>
                        </TableRow>
                      ) : (
                        previewRows.map((row, idx) => (
                          <TableRow key={idx}>
                            {row.map((cell, cellIdx) => (
                              <TableCell key={cellIdx} className="whitespace-nowrap">
                                {cell}
                              </TableCell>
                            ))}
                          </TableRow>
                        ))
                      )}
                    </TableBody>
                  </Table>
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
                  <h2 className="text-sm font-semibold text-foreground">导入进度</h2>
                  <p className="text-xs text-muted-foreground">逐 Sheet 展示实时进度</p>
                </div>
                <Badge variant="secondary">{importTasks.length} 项</Badge>
              </div>
            </div>

            <ScrollArea className="h-full">
              <div className="p-6">
                <Card className="p-4">
                  <div className="text-xs font-medium text-muted-foreground">进度详情</div>
                  <Separator className="my-3" />
                  {importTasks.length === 0 ? (
                    <div className="text-xs text-muted-foreground">暂无进度</div>
                  ) : (
                    <div className="space-y-3">
                      {importTasks.map((task) => (
                        <div
                          key={task.sheetName}
                          className="rounded-md border border-border/60 p-3"
                        >
                          <div className="flex items-center justify-between">
                            <div className="text-sm font-medium text-foreground">
                              {task.sheetName}
                            </div>
                            <Badge
                              variant={task.status === 'error' ? 'destructive' : 'secondary'}
                            >
                              {task.status}
                            </Badge>
                          </div>
                          <div className="mt-2">
                            <Progress value={task.progress} className="h-2" />
                          </div>
                          {task.message && (
                            <div className="mt-2 text-xs text-muted-foreground">
                              {task.message}
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
        </div>
      </div>
    </div>
  )
}
