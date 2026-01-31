import { useEffect, useMemo, useRef, useState } from 'react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Progress } from '@/components/ui/progress'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Upload, CheckCircle, XCircle, AlertCircle, Loader2 } from 'lucide-react'

interface ImportDialogProps {
  open: boolean
  onClose: () => void
  onSuccess: () => void
}

interface ProgressEvent {
  type: string
  message: string
  data?: any
  timestamp: string
}

export default function ImportDialog({ open, onClose, onSuccess }: ImportDialogProps) {
  const [file, setFile] = useState<File | null>(null)
  const [clearExisting, setClearExisting] = useState(true)
  const [importing, setImporting] = useState(false)
  const [progress, setProgress] = useState<ProgressEvent[]>([])
  const [completed, setCompleted] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [totalSheets, setTotalSheets] = useState<number | null>(null)
  const [doneSheets, setDoneSheets] = useState(0)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const logEndRef = useRef<HTMLDivElement>(null)

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const selectedFile = e.target.files?.[0]
    if (selectedFile) {
      setFile(selectedFile)
      setError(null)
    }
  }

  useEffect(() => {
    logEndRef.current?.scrollIntoView({ behavior: 'smooth', block: 'end' })
  }, [progress.length])

  const percent = useMemo(() => {
    if (!totalSheets || totalSheets <= 0) return 0
    const pct = Math.round((doneSheets / totalSheets) * 100)
    return Math.min(100, Math.max(0, pct))
  }, [doneSheets, totalSheets])

  const handleImport = async () => {
    if (!file) {
      setError('请选择文件')
      return
    }

    setImporting(true)
    setProgress([])
    setCompleted(false)
    setError(null)
    setTotalSheets(null)
    setDoneSheets(0)

    try {
      const formData = new FormData()
      formData.append('file', file)
      formData.append('clearExisting', clearExisting ? 'true' : 'false')
      formData.append('updateConfigYM', 'true')

      const response = await fetch('/api/import', {
        method: 'POST',
        body: formData,
      })

      if (!response.ok) {
        throw new Error('导入请求失败')
      }

      // 读取 SSE 流
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

        // 按行分割
        const lines = buffer.split('\n')
        buffer = lines.pop() || '' // 保留最后不完整的行

        for (const line of lines) {
          if (line.startsWith('data: ')) {
            const jsonStr = line.slice(6)
            try {
              const event: ProgressEvent = JSON.parse(jsonStr)
              setProgress((prev) => [...prev, event])

              if (event.type === 'info' && event.data && typeof event.data === 'object') {
                const ts = (event.data as any).total_sheets as number | undefined
                if (typeof ts === 'number' && ts > 0) {
                  setTotalSheets(ts)
                }
              }

              if (event.type === 'sheet_done' && event.data && typeof event.data === 'object') {
                const name = String((event.data as any).sheetName || '')
                if (name && !doneSet.has(name)) {
                  doneSet.add(name)
                  setDoneSheets(doneSet.size)
                }
              }

              // 完成
              if (event.type === 'done') {
                setCompleted(true)
                setImporting(false)
                // 延迟调用成功回调，让用户看到完成状态
                setTimeout(() => {
                  onSuccess()
                }, 2000)
              }

              // 错误
              if (event.type === 'error') {
                setError(event.message)
                setImporting(false)
              }
            } catch (err) {
              console.error('Failed to parse SSE event:', err)
            }
          }
        }
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : '导入失败')
      setImporting(false)
    }
  }

  const handleReset = () => {
    setFile(null)
    setProgress([])
    setCompleted(false)
    setError(null)
    setImporting(false)
    setTotalSheets(null)
    setDoneSheets(0)
    if (fileInputRef.current) {
      fileInputRef.current.value = ''
    }
  }

  return (
    <Dialog open={open} onOpenChange={onClose}>
      <DialogContent className="max-w-2xl max-h-[84vh] overflow-y-auto border-border/60 bg-card/80 backdrop-blur">
        <DialogHeader>
          <DialogTitle>导入数据</DialogTitle>
          <DialogDescription>
            上传 Excel 文件，自动识别并导入企业数据
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          {/* 文件选择 */}
          {!importing && !completed && (
            <>
              <div className="space-y-2">
                <Label htmlFor="file">选择文件</Label>
                <Input
                  id="file"
                  ref={fileInputRef}
                  type="file"
                  accept=".xlsx,.xls"
                  onChange={handleFileChange}
                />
                {file && (
                  <p className="text-sm text-muted-foreground">
                    已选择: <span className="font-medium text-foreground">{file.name}</span> ·{' '}
                    {(file.size / 1024 / 1024).toFixed(2)} MB
                  </p>
                )}
              </div>

              <div className="flex items-center justify-between rounded-lg border border-border/60 bg-muted/20 p-4">
                <div>
                  <p className="text-sm font-medium">清空现有数据</p>
                  <p className="text-xs text-muted-foreground">清空当前月份数据后重新导入</p>
                </div>
                <Switch checked={clearExisting} onCheckedChange={setClearExisting} />
              </div>

              {error && (
                <div className="flex items-center gap-2 rounded-md border border-red-500/20 bg-red-500/10 p-3 text-red-200">
                  <XCircle className="h-4 w-4" />
                  <span className="text-sm">{error}</span>
                </div>
              )}

              <div className="flex gap-2">
                <Button
                  onClick={handleImport}
                  disabled={!file || importing}
                  className="flex-1"
                >
                  <Upload className="w-4 h-4 mr-2" />
                  开始导入
                </Button>
                <Button variant="outline" onClick={onClose}>
                  取消
                </Button>
              </div>
            </>
          )}

          {/* 导入进度 */}
          {(importing || completed) && (
            <div className="space-y-4">
              <div className="space-y-2">
                <div className="flex items-center justify-between text-sm">
                  <span className="text-muted-foreground">导入进度</span>
                  <span className="font-mono text-xs text-muted-foreground">
                    {totalSheets ? `${doneSheets}/${totalSheets}` : `${doneSheets}/?`} · {percent}%
                  </span>
                </div>
                <Progress value={percent} className="h-2" />
              </div>

              <div className="rounded-lg border border-border/60 bg-muted/20">
                <ScrollArea className="h-60 w-full">
                  <div className="space-y-2 p-4">
                    {progress.map((event, index) => (
                      <div key={index} className="flex items-start gap-2 text-sm">
                        {(event.type === 'error') && <XCircle className="mt-0.5 h-4 w-4 text-red-300" />}
                        {(event.type === 'warning') && <AlertCircle className="mt-0.5 h-4 w-4 text-amber-300" />}
                        {(event.type === 'done') && <CheckCircle className="mt-0.5 h-4 w-4 text-emerald-300" />}
                        {(event.type === 'info' || event.type === 'sheet_start') && (
                          <Loader2 className="mt-0.5 h-4 w-4 animate-spin text-sky-300" />
                        )}
                        {(event.type === 'sheet_done') && <CheckCircle className="mt-0.5 h-4 w-4 text-emerald-300" />}
                        <span className="text-foreground/90">{event.message}</span>
                      </div>
                    ))}
                    <div ref={logEndRef} />
                  </div>
                </ScrollArea>
              </div>

              {/* 完成或重试 */}
              {completed && (
                <div className="flex gap-2">
                  <Button onClick={onSuccess} className="flex-1">
                    <CheckCircle className="w-4 h-4 mr-2" />
                    完成
                  </Button>
                </div>
              )}

              {error && !completed && (
                <div className="flex gap-2">
                  <Button onClick={handleReset} className="flex-1">
                    重试
                  </Button>
                  <Button variant="outline" onClick={onClose}>
                    关闭
                  </Button>
                </div>
              )}
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}
