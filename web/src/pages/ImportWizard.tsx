import { useEffect, useMemo, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import Topbar from '@/components/app/Topbar'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { importApi } from '@/services/api'
import { useImportStore, useProjectStore, useDataStore } from '@/store'
import { ArrowLeft, ArrowRight, Upload } from 'lucide-react'

function StepDot({ n, active }: { n: number; active: boolean }) {
  return (
    <div
      className={[
        'flex h-7 w-7 items-center justify-center rounded-full border text-[12px] font-semibold',
        active ? 'border-[#FF6B35] bg-[#FF6B35] text-black' : 'border-white/10 bg-white/5 text-[#A3A3A3]',
      ].join(' ')}
    >
      {n}
    </div>
  )
}

function Stepper({ step }: { step: number }) {
  const items = [
    { n: 1, label: '文件上传' },
    { n: 2, label: '字段映射' },
    { n: 3, label: '生成规则' },
    { n: 4, label: '执行导入' },
  ]
  return (
    <div className="flex items-center gap-3">
      {items.map((it, idx) => {
        const active = step === it.n
        return (
          <div key={it.n} className="flex items-center gap-3">
            <div className="flex items-center gap-2">
              <StepDot n={it.n} active={active} />
              <div className="text-sm text-white">{it.label}</div>
            </div>
            {idx < items.length - 1 && <div className="h-px w-10 bg-white/10" />}
          </div>
        )
      })}
    </div>
  )
}

export default function ImportWizard() {
  const navigate = useNavigate()
  const fileInputRef = useRef<HTMLInputElement | null>(null)

  const refreshCurrent = useProjectStore((s) => s.refreshCurrent)
  const currentProjectId = useProjectStore((s) => s.current?.project?.projectId)
  const { fetchCompanies, fetchIndicators, fetchConfig } = useDataStore()
  const {
    step,
    fileId,
    fileName,
    sheets,
    selectedSheet,
    columns,
    mapping,
    generateHistory,
    currentMonth,
    setStep,
    setFileInfo,
    setSelectedSheet,
    setColumns,
    setMapping,
    setGenerateHistory,
    setCurrentMonth,
    reset,
  } = useImportStore()

  const [uploading, setUploading] = useState(false)
  const [mappingValid, setMappingValid] = useState(false)
  const [bootstrapped, setBootstrapped] = useState(false)

  useEffect(() => {
    ;(async () => {
      await refreshCurrent()
      setBootstrapped(true)
    })()
  }, [navigate, refreshCurrent])

  useEffect(() => {
    if (!bootstrapped) return
    if (!currentProjectId) {
      navigate('/')
      return
    }
    reset()
  }, [bootstrapped, currentProjectId, navigate, reset])

  useEffect(() => {
    if (!fileId || !selectedSheet) return
    ;(async () => {
      const res = await importApi.getColumns(fileId, selectedSheet)
      setColumns(res.columns, res.previewRows)
    })()
  }, [fileId, selectedSheet, setColumns])

  useEffect(() => {
    if (columns.length === 0) return

    const auto = (field: keyof typeof mapping, colName: string) => {
      if (mapping[field]) return
      if (columns.includes(colName)) {
        setMapping(field, colName)
      }
    }

    auto('companyName', '企业名称')
    auto('creditCode', '统一社会信用代码')
    auto('industryCode', '行业代码')
    auto('companyScale', '企业规模')
    auto('retailCurrentMonth', '本期零售额')
    auto('retailLastYearMonth', '上年同期零售额')
    auto('retailCurrentCumulative', '本期累计零售额')
    auto('retailLastYearCumulative', '上年累计零售额')
    auto('salesCurrentMonth', '本期销售额')
    auto('salesLastYearMonth', '上年同期销售额')
    auto('salesCurrentCumulative', '本期累计销售额')
    auto('salesLastYearCumulative', '上年累计销售额')
  }, [columns, mapping, setMapping])

  const mappingRows = useMemo(
    () => [
      { key: 'companyName' as const, name: '企业名称', desc: '用于展示与匹配', required: true },
      { key: 'creditCode' as const, name: '统一社会信用代码', desc: '用于去重/校验', required: true },
      { key: 'industryCode' as const, name: '行业代码', desc: '用于行业分组', required: true },
      { key: 'retailCurrentMonth' as const, name: '本期零售额', desc: '用于计算增速', required: true },
      { key: 'retailLastYearMonth' as const, name: '上年同期零售额', desc: '用于计算增速' },
      { key: 'retailCurrentCumulative' as const, name: '本期累计零售额', desc: '用于累计指标' },
      { key: 'retailLastYearCumulative' as const, name: '上年累计零售额', desc: '用于累计指标' },
      { key: 'salesCurrentMonth' as const, name: '本期销售额', desc: '用于校验零售额上限' },
      { key: 'salesLastYearMonth' as const, name: '上年同期销售额', desc: '用于行业增速' },
      { key: 'salesCurrentCumulative' as const, name: '本期累计销售额', desc: '用于行业累计增速' },
      { key: 'salesLastYearCumulative' as const, name: '上年累计销售额', desc: '用于行业累计增速' },
      { key: 'companyScale' as const, name: '企业规模', desc: '用于识别小微企业' },
    ],
    []
  )

  const requiredMappingReady = useMemo(() => {
    const requiredKeys = mappingRows.filter((r) => r.required).map((r) => r.key)
    return requiredKeys.every((k) => String(mapping[k] || '').trim().length > 0)
  }, [mapping, mappingRows])

  const canNext =
    (step === 1 && !!fileId && !!selectedSheet) ||
    (step === 2 && !!fileId && !!selectedSheet && requiredMappingReady) ||
    step === 3 ||
    step === 4

  return (
    <>
      <Topbar title="导入向导" statusText="等待导入" />

      <div className="flex-1 overflow-y-scroll p-6">
        <div className="flex items-start justify-between">
          <div>
            <div className="text-2xl font-semibold">数据导入与配置</div>
            <div className="mt-1 text-sm text-[#A3A3A3]">上传 Excel → 字段映射 → 生成规则 → 执行导入</div>
          </div>
        </div>

        <div className="mt-6">
          <Stepper step={step} />
        </div>

        <div className="mt-6 overflow-hidden">
          <div
            className="flex w-full transition-transform duration-300 ease-out"
            style={{ transform: `translateX(-${(step - 1) * 100}%)` }}
          >
            <div className="w-full shrink-0 pr-2">
              <Card className="border-white/10 bg-[#0D0D0D] p-4">
                <CardHeader className="space-y-1 p-0">
                  <div className="text-lg font-semibold text-white">1. 文件上传与选择工作表</div>
                  <div className="text-sm text-[#A3A3A3]">支持 .xlsx / .xls（建议 ≤ 10MB）</div>
                </CardHeader>
                <CardContent className="mt-4 space-y-4 p-0">
                  <div className="rounded-xl border border-dashed border-white/10 bg-white/5 p-6 text-center">
                    <div className="mx-auto flex h-10 w-10 items-center justify-center rounded-full border border-white/10 bg-[#0D0D0D]">
                      <Upload className="h-5 w-5 text-white" />
                    </div>
                    <div className="mt-3 text-sm text-white">点击上传 / 拖拽文件到此处</div>
                    <div className="mt-1 text-[12px] text-[#A3A3A3]">支持 XLS / XLSX</div>

                    <div className="mt-4 flex justify-center">
                      <Input
                        ref={fileInputRef}
                        type="file"
                        accept=".xlsx,.xls"
                        className="hidden"
                        onChange={async (e) => {
                          const file = e.target.files?.[0]
                          if (!file) return
                          setUploading(true)
                          try {
                            const res = await importApi.upload(file)
                            setFileInfo(res.fileId, res.fileName, res.sheets)
                            setMappingValid(false)
                            setStep(1)
                          } finally {
                            setUploading(false)
                          }
                        }}
                      />
                      <Button
                        className="bg-[#FF6B35] text-black hover:bg-[#FF6B35]/90"
                        disabled={uploading}
                        onClick={() => fileInputRef.current?.click()}
                      >
                        {uploading ? '上传中...' : '选择文件'}
                      </Button>
                    </div>
                  </div>

                  <div className="grid grid-cols-12 gap-4">
                    <div className="col-span-12 lg:col-span-6">
                      <Label className="text-sm text-[#A3A3A3]">选择工作表</Label>
                      <div className="mt-2">
                        <Select
                          value={selectedSheet ?? undefined}
                          onValueChange={(v) => {
                            setSelectedSheet(v)
                            setMappingValid(false)
                          }}
                          disabled={!fileId || sheets.length === 0}
                        >
                          <SelectTrigger className="border-white/10 bg-white/5 text-white">
                            <SelectValue placeholder="请选择工作表" />
                          </SelectTrigger>
                          <SelectContent className="border-white/10 bg-[#0D0D0D] text-white">
                            {sheets.map((s) => (
                              <SelectItem key={s.name} value={s.name}>
                                {s.name} - {s.rowCount} 行
                              </SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                      </div>
                    </div>

                    <div className="col-span-12 lg:col-span-6">
                      <div className="text-sm text-[#A3A3A3]">文件已上传</div>
                      <div className="mt-2 rounded-lg border border-white/10 bg-white/5 p-3 text-sm">
                        <div className="text-white">文件名：{fileName || '-'}</div>
                      </div>
                    </div>
                  </div>
                </CardContent>
              </Card>
            </div>

            <div className="w-full shrink-0 px-1">
              <Card className="border-white/10 bg-[#0D0D0D] p-4">
                <CardHeader className="space-y-1 p-0">
                  <div className="text-lg font-semibold text-white">2. 配置关键字段映射</div>
                  <div className="text-sm text-[#A3A3A3]">将 Excel 列映射到系统字段，确保导入准确</div>
                </CardHeader>
                <CardContent className="mt-4 p-0">
                  <div className="overflow-hidden rounded-lg border border-white/10">
                    <Table>
                    <TableHeader>
                      <TableRow className="border-white/10">
                        <TableHead className="text-[#A3A3A3]">系统字段</TableHead>
                        <TableHead className="text-[#A3A3A3]">描述</TableHead>
                        <TableHead className="text-[#A3A3A3]">Excel 列名</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {mappingRows.map((r) => (
                        <TableRow key={r.key} className="border-white/10">
                          <TableCell className="text-white">
                            <div className="flex items-center gap-2">
                              <div>{r.name}</div>
                              {r.required && (
                                <div className="rounded bg-[#FF6B35] px-2 py-0.5 text-[10px] font-semibold text-black">
                                  必填
                                </div>
                              )}
                            </div>
                          </TableCell>
                          <TableCell className="text-[#A3A3A3]">{r.desc}</TableCell>
                          <TableCell>
                            <Select
                              value={(mapping[r.key] as string) || undefined}
                              onValueChange={(v) => {
                                  setMapping(r.key, v)
                                  setMappingValid(false)
                                }}
                                disabled={!fileId || columns.length === 0}
                              >
                                <SelectTrigger className="h-9 border-white/10 bg-white/5 text-white">
                                  <SelectValue placeholder="选择列" />
                                </SelectTrigger>
                                <SelectContent className="border-white/10 bg-[#0D0D0D] text-white">
                                  {columns.map((c) => (
                                    <SelectItem key={c} value={c}>
                                      {c}
                                    </SelectItem>
                                  ))}
                                </SelectContent>
                              </Select>
                            </TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </div>
                </CardContent>
              </Card>
            </div>

            <div className="w-full shrink-0 px-1">
              <Card className="border-white/10 bg-[#0D0D0D] p-4">
                <CardHeader className="space-y-1 p-0">
                  <div className="text-lg font-semibold text-white">3. 生成规则</div>
                  <div className="text-sm text-white">是否生成历史数据与当前操作月份</div>
                </CardHeader>
                <CardContent className="mt-4 space-y-4 p-0">
                  <div className="flex items-center justify-between rounded-lg border border-white/10 bg-white/5 p-3">
                  <div>
                    <div className="text-sm text-white">生成历史数据</div>
                    <div className="text-[12px] text-[#A3A3A3]">用于补齐缺失月份（可选）</div>
                  </div>
                    <Switch checked={generateHistory} onCheckedChange={setGenerateHistory} />
                  </div>

                  <div className="flex items-center justify-between rounded-lg border border-white/10 bg-white/5 p-3">
                  <div>
                    <div className="text-sm text-white">当前操作月份</div>
                    <div className="text-[12px] text-[#A3A3A3]">用于导入后的配置更新</div>
                  </div>
                    <Input
                      type="number"
                      value={currentMonth}
                      onChange={(e) => setCurrentMonth(Number(e.target.value || 0))}
                      className="h-9 w-28 border-white/10 bg-white/5 text-right text-white"
                    />
                  </div>
                </CardContent>
              </Card>
            </div>

            <div className="w-full shrink-0 pl-2">
              <Card className="border-white/10 bg-[#0D0D0D] p-4">
                <CardHeader className="space-y-1 p-0">
                  <div className="text-lg font-semibold text-white">4. 执行导入</div>
                  <div className="text-sm text-white">确认配置后执行导入并进入仪表盘</div>
                </CardHeader>
                <CardContent className="mt-4 space-y-2 p-0 text-sm text-white">
                  <div>文件：{fileName || '-'}</div>
                  <div>工作表：{selectedSheet || '-'}</div>
                  <div>生成历史：{generateHistory ? '是' : '否'}</div>
                  <div>当前月份：{currentMonth}</div>
                </CardContent>
              </Card>
            </div>
          </div>
        </div>

        <div className="mt-6 flex items-center justify-end gap-2">
          <Button
            variant="secondary"
            className="border border-white/10 bg-white/5 text-white hover:bg-white/10"
            disabled={step === 1}
            onClick={() => setStep(Math.max(1, step - 1))}
          >
            <ArrowLeft className="mr-2 h-4 w-4" />
            上一步
          </Button>

          <Button
            className="bg-[#FF6B35] text-black hover:bg-[#FF6B35]/90"
            disabled={!canNext}
            onClick={async () => {
              if (step === 1) {
                setStep(2)
                return
              }
              if (step === 2) {
                if (!fileId || !selectedSheet) return
                await importApi.setMapping(fileId, selectedSheet, mapping)
                setMappingValid(true)
                setStep(3)
                return
              }
              if (step === 3) {
                setStep(4)
                return
              }
              if (step === 4) {
                if (!fileId || !selectedSheet) return
                await importApi.execute(fileId, selectedSheet, generateHistory, currentMonth)
                await Promise.all([fetchCompanies(), fetchIndicators(), fetchConfig()])
                reset()
                navigate('/dashboard')
              }
            }}
          >
            下一步
            <ArrowRight className="ml-2 h-4 w-4" />
          </Button>
        </div>

        {mappingValid && step >= 3 && (
          <div className="mt-3 text-sm text-[#A3A3A3]">
            提示：字段映射已提交，后端会根据映射解析并导入数据。
          </div>
        )}
      </div>
    </>
  )
}
