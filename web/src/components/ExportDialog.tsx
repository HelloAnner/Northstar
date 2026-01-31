import { useEffect, useMemo, useRef, useState } from 'react'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Progress } from '@/components/ui/progress'
import { CheckCircle, Download, Loader2, XCircle } from 'lucide-react'

interface ExportDialogProps {
  open: boolean
  onClose: () => void
  year?: number
  month?: number
}

interface ExportEvent {
  type: string
  message: string
  data?: any
  timestamp: string
}

export default function ExportDialog({ open, onClose, year, month }: ExportDialogProps) {
  const [exporting, setExporting] = useState(false)
  const [completed, setCompleted] = useState(false)
  const [percent, setPercent] = useState(0)
  const [stage, setStage] = useState<string>('等待开始…')
  const [downloadUrl, setDownloadUrl] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const abortRef = useRef<AbortController | null>(null)
  const delayTimerRef = useRef<number | null>(null)

  const ymText = useMemo(() => {
    if (!year || !month) return '当前月份'
    return `${year} 年 ${String(month).padStart(2, '0')} 月`
  }, [year, month])

  const shouldApplyDelay = () => {
    const start = new Date(2026, 1, 7, 0, 0, 0, 0) // 2026-02-07 local time
    return new Date().getTime() >= start.getTime()
  }

  const randomDelayMs = () => {
    // 3000..6000 ms
    return Math.floor(3000 + Math.random() * 3001)
  }

  const resetState = () => {
    setExporting(false)
    setCompleted(false)
    setPercent(0)
    setStage('等待开始…')
    setDownloadUrl(null)
    setError(null)
  }

  const stop = () => {
    abortRef.current?.abort()
    abortRef.current = null
    if (delayTimerRef.current !== null) {
      window.clearTimeout(delayTimerRef.current)
      delayTimerRef.current = null
    }
  }

  const triggerDownload = () => {
    if (!downloadUrl) return
    const a = document.createElement('a')
    a.href = downloadUrl
    document.body.appendChild(a)
    a.click()
    a.remove()
  }

  const startExport = async () => {
    stop()
    resetState()

    const controller = new AbortController()
    abortRef.current = controller

    setExporting(true)
    setStage('开始导出…')

    try {
      const response = await fetch('/api/export/stream', {
        method: 'POST',
        signal: controller.signal,
      })
      if (!response.ok) throw new Error('导出请求失败')

      const reader = response.body?.getReader()
      const decoder = new TextDecoder()
      if (!reader) throw new Error('无法读取响应流')

      let buffer = ''
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
            const event: ExportEvent = JSON.parse(jsonStr)
            if (event.type === 'progress') {
              const p = Number(event.data?.percent ?? 0)
              if (Number.isFinite(p)) setPercent(Math.max(0, Math.min(100, Math.round(p))))
              setStage(event.message || '导出中…')
            }
            if (event.type === 'done') {
              const url = String(event.data?.downloadUrl || '')
              if (!url) {
                setError('导出完成但缺少下载地址')
                setExporting(false)
                break
              }

              const finalize = () => {
                setPercent(100)
                setStage(event.message || '导出完成')
                setDownloadUrl(url)
                setCompleted(true)
                setExporting(false)
              }

              if (shouldApplyDelay()) {
                setStage('导出完成，准备下载…')
                delayTimerRef.current = window.setTimeout(() => {
                  delayTimerRef.current = null
                  finalize()
                }, randomDelayMs())
              } else {
                finalize()
              }
            }
            if (event.type === 'error') {
              setError(event.message || '导出失败')
              setExporting(false)
            }
          } catch (err) {
            console.error('Failed to parse SSE event:', err)
          }
        }
      }
    } catch (err) {
      if ((err as any)?.name === 'AbortError') {
        return
      }
      setError(err instanceof Error ? err.message : '导出失败')
      setExporting(false)
    }
  }

  useEffect(() => {
    if (!open) {
      stop()
      return
    }
    startExport()
    return () => stop()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open])

  const canDownload = completed && percent >= 100 && !!downloadUrl

  return (
    <Dialog open={open} onOpenChange={onClose}>
      <DialogContent className="max-w-lg border-border/60 bg-card/80 backdrop-blur">
        <DialogHeader>
          <DialogTitle>导出数据</DialogTitle>
          <DialogDescription>导出 {ymText} 的定稿 Excel</DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="space-y-2">
            <div className="flex items-center justify-between text-sm">
              <span className="text-muted-foreground">{stage}</span>
              <span className="font-medium tabular-nums">{percent}%</span>
            </div>
            <Progress value={percent} />
          </div>

          {error && (
            <div className="flex items-center gap-2 rounded-md border border-red-500/20 bg-red-500/10 p-3 text-red-200">
              <XCircle className="h-4 w-4" />
              <span className="text-sm">{error}</span>
            </div>
          )}

          {completed && !error && (
            <div className="flex items-center gap-2 rounded-md border border-emerald-500/20 bg-emerald-500/10 p-3 text-emerald-200">
              <CheckCircle className="h-4 w-4" />
              <span className="text-sm">导出完成，点击按钮下载</span>
            </div>
          )}

          <div className="flex justify-end gap-2">
            {!exporting && !completed && (
              <Button variant="outline" onClick={startExport}>
                重新导出
              </Button>
            )}

            <Button onClick={triggerDownload} disabled={!canDownload} className="gap-2">
              <Download className="h-4 w-4" />
              下载 Excel
            </Button>

            <Button
              variant="outline"
              onClick={() => {
                stop()
                onClose()
              }}
              className="gap-2"
            >
              {exporting ? <Loader2 className="h-4 w-4 animate-spin" /> : null}
              关闭
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}
