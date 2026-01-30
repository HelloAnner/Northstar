import { useState, useRef } from 'react'
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

interface SheetResult {
  sheetName: string
  sheetType: string
  status: string
  importedRows: number
  errorRows?: number
  errors?: string[]
}

export default function ImportDialog({ open, onClose, onSuccess }: ImportDialogProps) {
  const [file, setFile] = useState<File | null>(null)
  const [clearExisting, setClearExisting] = useState(true)
  const [importing, setImporting] = useState(false)
  const [progress, setProgress] = useState<ProgressEvent[]>([])
  const [sheetResults, setSheetResults] = useState<SheetResult[]>([])
  const [completed, setCompleted] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const selectedFile = e.target.files?.[0]
    if (selectedFile) {
      setFile(selectedFile)
      setError(null)
    }
  }

  const handleImport = async () => {
    if (!file) {
      setError('请选择文件')
      return
    }

    setImporting(true)
    setProgress([])
    setSheetResults([])
    setCompleted(false)
    setError(null)

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

              // 更新 Sheet 结果
              if (event.type === 'sheet_done') {
                const sheetData = event.data as SheetResult
                setSheetResults((prev) => [...prev, sheetData])
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
    setSheetResults([])
    setCompleted(false)
    setError(null)
    setImporting(false)
    if (fileInputRef.current) {
      fileInputRef.current.value = ''
    }
  }

  return (
    <Dialog open={open} onOpenChange={onClose}>
      <DialogContent className="max-w-3xl max-h-[80vh] overflow-y-auto">
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
                  <p className="text-sm text-gray-500">
                    已选择: {file.name} ({(file.size / 1024).toFixed(2)} KB)
                  </p>
                )}
              </div>

              <div className="flex items-center justify-between">
                <Label htmlFor="clear" className="cursor-pointer">
                  清空现有数据后导入
                </Label>
                <Switch
                  id="clear"
                  checked={clearExisting}
                  onCheckedChange={setClearExisting}
                />
              </div>

              {error && (
                <div className="flex items-center gap-2 p-3 bg-red-50 text-red-700 rounded-md">
                  <XCircle className="w-4 h-4" />
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
              {/* 进度日志 */}
              <div className="bg-gray-50 rounded-md p-4 max-h-60 overflow-y-auto space-y-2">
                {progress.map((event, index) => (
                  <div
                    key={index}
                    className="flex items-start gap-2 text-sm"
                  >
                    {event.type === 'error' && (
                      <XCircle className="w-4 h-4 text-red-500 mt-0.5" />
                    )}
                    {event.type === 'warning' && (
                      <AlertCircle className="w-4 h-4 text-yellow-500 mt-0.5" />
                    )}
                    {event.type === 'done' && (
                      <CheckCircle className="w-4 h-4 text-green-500 mt-0.5" />
                    )}
                    {event.type === 'info' && (
                      <Loader2 className="w-4 h-4 text-blue-500 mt-0.5 animate-spin" />
                    )}
                    <span className="text-gray-700">{event.message}</span>
                  </div>
                ))}
              </div>

              {/* Sheet 结果列表 */}
              {sheetResults.length > 0 && (
                <div className="space-y-2">
                  <h3 className="font-semibold text-sm text-gray-700">
                    Sheet 解析结果
                  </h3>
                  <div className="border rounded-md divide-y">
                    {sheetResults.map((sheet, index) => (
                      <div
                        key={index}
                        className="flex items-center justify-between p-3"
                      >
                        <div className="flex items-center gap-2">
                          {sheet.status === 'imported' && (
                            <CheckCircle className="w-4 h-4 text-green-500" />
                          )}
                          {sheet.status === 'skipped' && (
                            <AlertCircle className="w-4 h-4 text-yellow-500" />
                          )}
                          {sheet.status === 'error' && (
                            <XCircle className="w-4 h-4 text-red-500" />
                          )}
                          <div>
                            <p className="font-medium text-sm">
                              {sheet.sheetName}
                            </p>
                            <p className="text-xs text-gray-500">
                              {sheet.sheetType}
                            </p>
                          </div>
                        </div>
                        <div className="text-right">
                          {sheet.status === 'imported' && (
                            <p className="text-sm text-green-600">
                              ✓ {sheet.importedRows} 行
                            </p>
                          )}
                          {sheet.status === 'skipped' && (
                            <p className="text-sm text-yellow-600">跳过</p>
                          )}
                          {sheet.status === 'error' && (
                            <p className="text-sm text-red-600">失败</p>
                          )}
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              )}

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
