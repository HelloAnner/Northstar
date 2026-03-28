/**
 * 导入数据弹窗（精简三态设计：选择 → 导入 → 完成）
 *
 * @author Anner
 * Created on 2026/3/28
 */

import { useRef, useState } from 'react'
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
import { useImportSSE } from '@/hooks/useImportSSE'

interface ImportDialogProps {
  open: boolean
  onClose: () => void
  onSuccess: () => void
}

export default function ImportDialog({ open, onClose, onSuccess }: ImportDialogProps) {
  const [file, setFile] = useState<File | null>(null)
  const [clearExisting, setClearExisting] = useState(true)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const sse = useImportSSE()

  const isIdle = sse.status === 'idle'
  const isDone = sse.status === 'done'
  const isError = sse.status === 'error'
  const isImporting = sse.status === 'importing'

  const handleImport = () => {
    if (!file) return
    sse.start(file, clearExisting)
  }

  const handleReset = () => {
    setFile(null)
    sse.reset()
    if (fileInputRef.current) fileInputRef.current.value = ''
  }

  const handleClose = () => {
    if (!isImporting) {
      handleReset()
      onClose()
    }
  }

  const handleDone = () => {
    handleReset()
    onSuccess()
  }

  const eventIcon = (type?: string) => {
    if (type === 'error') return <XCircle className="mt-0.5 h-3.5 w-3.5 shrink-0 text-red-400" />
    if (type === 'done' || type === 'sheet_done') return <CheckCircle className="mt-0.5 h-3.5 w-3.5 shrink-0 text-emerald-400" />
    if (type === 'warning') return <AlertCircle className="mt-0.5 h-3.5 w-3.5 shrink-0 text-amber-400" />
    return <Loader2 className="mt-0.5 h-3.5 w-3.5 shrink-0 animate-spin text-muted-foreground" />
  }

  // 结果摘要
  const summaryText = () => {
    if (!sse.result) return '导入完成'
    const { importedSheets, skippedSheets, importedRows } = sse.result
    const parts = [`${importedSheets} 个 Sheet 成功`]
    if (skippedSheets > 0) parts.push(`${skippedSheets} 个跳过`)
    parts.push(`共 ${importedRows} 条记录`)
    return parts.join('，')
  }

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="max-w-lg border-border/60">
        <DialogHeader>
          <DialogTitle>导入数据</DialogTitle>
          <DialogDescription>上传 Excel 文件，自动识别并导入企业数据</DialogDescription>
        </DialogHeader>

        {/* 选择态 */}
        {isIdle && (
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="import-file">选择文件</Label>
              <Input
                id="import-file"
                ref={fileInputRef}
                type="file"
                accept=".xlsx,.xls"
                onChange={(e) => setFile(e.target.files?.[0] || null)}
              />
              {file && (
                <p className="text-xs text-muted-foreground">
                  {file.name} · {(file.size / 1024 / 1024).toFixed(2)} MB
                </p>
              )}
            </div>

            <div className="flex items-center justify-between rounded-lg border border-border/60 p-3">
              <div>
                <p className="text-sm font-medium">清空现有数据</p>
                <p className="text-xs text-muted-foreground">清空当前月份数据后重新导入</p>
              </div>
              <Switch checked={clearExisting} onCheckedChange={setClearExisting} />
            </div>

            <div className="flex gap-2">
              <Button onClick={handleImport} disabled={!file} className="flex-1">
                <Upload className="mr-2 h-4 w-4" />
                开始导入
              </Button>
              <Button variant="outline" onClick={handleClose}>
                取消
              </Button>
            </div>
          </div>
        )}

        {/* 导入态 */}
        {isImporting && (
          <div className="space-y-4">
            <div className="space-y-1.5">
              <div className="flex items-center justify-between text-sm">
                <span className="text-muted-foreground">导入进度</span>
                <span className="font-mono text-xs text-muted-foreground">{sse.progress}%</span>
              </div>
              <Progress value={sse.progress} className="h-2" />
            </div>

            <ScrollArea className="h-52 rounded-md border border-border/60">
              <div className="space-y-1 p-3">
                {sse.events.map((event, i) => (
                  <div key={i} className="flex items-start gap-2 text-xs">
                    {eventIcon(event.type)}
                    <span className="text-foreground/80">{event.message}</span>
                  </div>
                ))}
              </div>
            </ScrollArea>

            {sse.stale && (
              <p className="text-xs text-amber-400">导入仍在进行，等待后端响应...</p>
            )}
          </div>
        )}

        {/* 完成态 */}
        {isDone && (
          <div className="space-y-4">
            <div className="flex items-center gap-3 rounded-lg border border-emerald-500/20 bg-emerald-500/5 p-4">
              <CheckCircle className="h-5 w-5 text-emerald-400" />
              <div>
                <p className="text-sm font-medium">导入完成</p>
                <p className="text-xs text-muted-foreground">{summaryText()}</p>
              </div>
            </div>
            <Button onClick={handleDone} className="w-full">完成</Button>
          </div>
        )}

        {/* 错误态 */}
        {isError && (
          <div className="space-y-4">
            <div className="flex items-center gap-3 rounded-lg border border-red-500/20 bg-red-500/5 p-4">
              <XCircle className="h-5 w-5 text-red-400" />
              <div>
                <p className="text-sm font-medium">导入失败</p>
                <p className="text-xs text-muted-foreground">{sse.error}</p>
              </div>
            </div>
            <div className="flex gap-2">
              <Button onClick={handleReset} className="flex-1">重试</Button>
              <Button variant="outline" onClick={handleClose}>关闭</Button>
            </div>
          </div>
        )}
      </DialogContent>
    </Dialog>
  )
}
