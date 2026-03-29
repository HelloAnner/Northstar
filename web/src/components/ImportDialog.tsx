/**
 * 导入数据弹窗（精简三态设计：选择 → 导入 → 完成 + 验证）
 *
 * @author Anner
 * Created on 2026/3/28
 */

import { useCallback, useEffect, useRef, useState } from 'react'
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
import { Upload, CheckCircle, XCircle, AlertCircle, Loader2, ShieldCheck, ShieldAlert } from 'lucide-react'
import { useImportSSE } from '@/hooks/useImportSSE'

interface ImportDialogProps {
  open: boolean
  onClose: () => void
  onSuccess: () => void
}

interface VerifySheet {
  sheetName: string
  sheetType: string
  expectedRows: number
  actualRows: number
  match: boolean
}

interface VerifySummary {
  expectedTotal: number
  actualTotal: number
  wrExpected: number
  wrActual: number
  acExpected: number
  acActual: number
  allMatch: boolean
}

interface VerifyResult {
  filename: string
  sheets: VerifySheet[]
  summary: VerifySummary
}

const SHEET_TYPE_LABELS: Record<string, string> = {
  wholesale: '批发',
  retail: '零售',
  accommodation: '住宿',
  catering: '餐饮',
  wr_snapshot: '批零快照',
  ac_snapshot: '住餐快照',
  summary: '汇总',
}

export default function ImportDialog({ open, onClose, onSuccess }: ImportDialogProps) {
  const [file, setFile] = useState<File | null>(null)
  const [clearExisting, setClearExisting] = useState(true)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const sse = useImportSSE()

  const [verify, setVerify] = useState<VerifyResult | null>(null)
  const [verifying, setVerifying] = useState(false)

  const isIdle = sse.status === 'idle'
  const isDone = sse.status === 'done'
  const isError = sse.status === 'error'
  const isImporting = sse.status === 'importing'

  // 导入完成后自动触发验证
  const runVerify = useCallback(async () => {
    setVerifying(true)
    try {
      const res = await fetch('/api/import/verify')
      if (res.ok) {
        const data: VerifyResult = await res.json()
        setVerify(data)
      }
    } catch {
      // 验证失败不阻塞流程
    } finally {
      setVerifying(false)
    }
  }, [])

  useEffect(() => {
    if (isDone) {
      runVerify()
    }
  }, [isDone, runVerify])

  const handleImport = () => {
    if (!file) return
    sse.start(file, clearExisting)
  }

  const handleReset = () => {
    setFile(null)
    setVerify(null)
    setVerifying(false)
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

        {/* 完成态 + 验证 */}
        {isDone && (
          <div className="space-y-4">
            <div className="flex items-center gap-3 rounded-lg border border-emerald-500/20 bg-emerald-500/5 p-4">
              <CheckCircle className="h-5 w-5 text-emerald-400" />
              <div>
                <p className="text-sm font-medium">导入完成</p>
                <p className="text-xs text-muted-foreground">{summaryText()}</p>
              </div>
            </div>

            {/* 数据验证区域 */}
            {verifying && (
              <div className="flex items-center gap-2 rounded-lg border border-border/60 p-4">
                <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
                <span className="text-sm text-muted-foreground">正在验证数据库数据...</span>
              </div>
            )}

            {verify && (
              <div className="space-y-3">
                {/* 验证状态标题 */}
                <div className={`flex items-center gap-3 rounded-lg border p-4 ${
                  verify.summary.allMatch
                    ? 'border-emerald-500/20 bg-emerald-500/5'
                    : 'border-amber-500/20 bg-amber-500/5'
                }`}>
                  {verify.summary.allMatch
                    ? <ShieldCheck className="h-5 w-5 text-emerald-400" />
                    : <ShieldAlert className="h-5 w-5 text-amber-400" />
                  }
                  <div>
                    <p className="text-sm font-medium">
                      {verify.summary.allMatch ? '数据验证通过' : '数据存在差异'}
                    </p>
                    <p className="text-xs text-muted-foreground">
                      Excel 预期 {verify.summary.expectedTotal} 条，数据库实际 {verify.summary.actualTotal} 条
                    </p>
                  </div>
                </div>

                {/* 分类对比 */}
                <div className="rounded-lg border border-border/60">
                  <table className="w-full text-xs">
                    <thead>
                      <tr className="border-b border-border/60 bg-muted/30">
                        <th className="px-3 py-2 text-left font-medium text-muted-foreground">Sheet</th>
                        <th className="px-3 py-2 text-right font-medium text-muted-foreground">预期</th>
                        <th className="px-3 py-2 text-right font-medium text-muted-foreground">实际</th>
                        <th className="px-3 py-2 text-center font-medium text-muted-foreground">状态</th>
                      </tr>
                    </thead>
                    <tbody>
                      {verify.sheets.map((sheet) => (
                        <tr key={sheet.sheetName} className="border-b border-border/40 last:border-0">
                          <td className="px-3 py-2">
                            <span className="font-medium">{sheet.sheetName}</span>
                            <span className="ml-1.5 text-muted-foreground">
                              {SHEET_TYPE_LABELS[sheet.sheetType] || sheet.sheetType}
                            </span>
                          </td>
                          <td className="px-3 py-2 text-right font-mono">{sheet.expectedRows}</td>
                          <td className="px-3 py-2 text-right font-mono">{sheet.actualRows}</td>
                          <td className="px-3 py-2 text-center">
                            {sheet.match
                              ? <CheckCircle className="mx-auto h-3.5 w-3.5 text-emerald-400" />
                              : <XCircle className="mx-auto h-3.5 w-3.5 text-red-400" />
                            }
                          </td>
                        </tr>
                      ))}
                      {/* 汇总行 */}
                      {(verify.summary.wrExpected > 0 || verify.summary.acExpected > 0) && (
                        <>
                          {verify.summary.wrExpected > 0 && (
                            <tr className="border-b border-border/40 bg-muted/20">
                              <td className="px-3 py-2 font-medium">批零合计</td>
                              <td className="px-3 py-2 text-right font-mono">{verify.summary.wrExpected}</td>
                              <td className="px-3 py-2 text-right font-mono">{verify.summary.wrActual}</td>
                              <td className="px-3 py-2 text-center">
                                {verify.summary.wrExpected === verify.summary.wrActual
                                  ? <CheckCircle className="mx-auto h-3.5 w-3.5 text-emerald-400" />
                                  : <XCircle className="mx-auto h-3.5 w-3.5 text-red-400" />
                                }
                              </td>
                            </tr>
                          )}
                          {verify.summary.acExpected > 0 && (
                            <tr className="border-b border-border/40 bg-muted/20">
                              <td className="px-3 py-2 font-medium">住餐合计</td>
                              <td className="px-3 py-2 text-right font-mono">{verify.summary.acExpected}</td>
                              <td className="px-3 py-2 text-right font-mono">{verify.summary.acActual}</td>
                              <td className="px-3 py-2 text-center">
                                {verify.summary.acExpected === verify.summary.acActual
                                  ? <CheckCircle className="mx-auto h-3.5 w-3.5 text-emerald-400" />
                                  : <XCircle className="mx-auto h-3.5 w-3.5 text-red-400" />
                                }
                              </td>
                            </tr>
                          )}
                        </>
                      )}
                    </tbody>
                  </table>
                </div>
              </div>
            )}

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
