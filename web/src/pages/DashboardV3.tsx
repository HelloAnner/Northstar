import { useState, useEffect } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Download, RefreshCw, Upload } from 'lucide-react'
import ImportDialog from '@/components/ImportDialog'
import CompaniesTable, { type IndicatorGroup } from '@/components/CompaniesTable'

type Indicator = IndicatorGroup['indicators'][number]

interface SystemStatus {
  initialized: boolean
  currentYear: number
  currentMonth: number
  totalCompanies: number
  wrCount: number
  acCount: number
}

interface YearMonthStat {
  year: number
  month: number
  wrCount: number
  acCount: number
  totalCompanies: number
}

export default function DashboardV3() {
  const [status, setStatus] = useState<SystemStatus | null>(null)
  const [groups, setGroups] = useState<IndicatorGroup[]>([])
  const [loading, setLoading] = useState(true)
  const [showImportDialog, setShowImportDialog] = useState(false)
  const [tableSaving, setTableSaving] = useState(false)
  const [optimizing, setOptimizing] = useState(false)
  const [reloadToken, setReloadToken] = useState(0)
  const [draftTargets, setDraftTargets] = useState<Record<string, string>>({})
  const [months, setMonths] = useState<YearMonthStat[]>([])
  const [monthsLoading, setMonthsLoading] = useState(false)

  // 加载系统状态
  const loadStatus = async () => {
    try {
      const res = await fetch('/api/status')
      const data = await res.json()
      setStatus(data)
    } catch (err) {
      console.error('Failed to load status:', err)
    }
  }

  // 加载可用月份（用于切换整个平台年月）
  const loadMonths = async () => {
    setMonthsLoading(true)
    try {
      const res = await fetch('/api/months')
      if (!res.ok) throw new Error('加载月份失败')
      const data = (await res.json()) as { items?: YearMonthStat[] }
      setMonths(Array.isArray(data.items) ? data.items : [])
    } catch (err) {
      console.error(err)
      setMonths([])
    } finally {
      setMonthsLoading(false)
    }
  }

  // 加载指标数据
  const loadIndicators = async () => {
    setLoading(true)
    try {
      const res = await fetch('/api/indicators')
      const data = await res.json()
      setGroups(data.groups || [])
    } catch (err) {
      console.error('Failed to load indicators:', err)
    } finally {
      setLoading(false)
    }
  }

  // 初始加载
  useEffect(() => {
    loadStatus()
    loadIndicators()
    loadMonths()
  }, [])

  // 导入完成回调
  const handleImportSuccess = () => {
    setShowImportDialog(false)
    loadStatus()
    loadIndicators()
    loadMonths()
    setReloadToken((x) => x + 1)
  }

  const handleResetAll = async () => {
    try {
      const res = await fetch('/api/companies/reset', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({}),
      })
      if (!res.ok) throw new Error('重置失败')
      const data = await res.json()
      if (data.groups) {
        setGroups(data.groups)
      }
      setDraftTargets({})
      setReloadToken((x) => x + 1)
    } catch (err) {
      console.error(err)
    }
  }

  const applyOptimize = async (targets: Record<string, number>, clearIds?: string[]) => {
    setOptimizing(true)
    try {
      const res = await fetch('/api/optimize', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ targets }),
      })
      const data = await res.json()
      if (!res.ok) {
        throw new Error(data?.error || '智能调整失败')
      }
      if (data.groups) {
        setGroups(data.groups)
      }
      if (!clearIds || clearIds.length === 0) {
        setDraftTargets({})
      } else {
        setDraftTargets((prev) => {
          const next = { ...prev }
          for (const id of clearIds) {
            delete next[id]
          }
          return next
        })
      }
      setReloadToken((x) => x + 1)
    } finally {
      setOptimizing(false)
    }
  }

  const handleSmartAdjust = async () => {
    const entries = Object.entries(draftTargets)
    if (entries.length === 0) return

    const targets: Record<string, number> = {}
    for (const [id, raw] of entries) {
      const v = Number(String(raw).replaceAll(',', '').trim())
      if (Number.isFinite(v)) {
        targets[id] = v
      }
    }
    if (Object.keys(targets).length === 0) return

    try {
      await applyOptimize(targets)
    } catch (err) {
      console.error(err)
    }
  }

  const saving = tableSaving || optimizing
  const saveText = optimizing ? '智能调整中…' : '自动保存中…'

  const currentMonthKey =
    status && status.currentYear > 0 && status.currentMonth > 0
      ? `${status.currentYear}-${String(status.currentMonth).padStart(2, '0')}`
      : ''
  const canSelectMonth = !saving && !monthsLoading && months.length > 0

  const selectMonth = async (key: string) => {
    const [yRaw, mRaw] = key.split('-')
    const year = Number(yRaw)
    const month = Number(mRaw)
    if (!Number.isFinite(year) || !Number.isFinite(month)) return

    try {
      const res = await fetch('/api/months/select', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ year, month }),
      })
      const data = await res.json()
      if (!res.ok) throw new Error(data?.error || '切换月份失败')

      if (data.status) {
        setStatus(data.status as SystemStatus)
      }
      if (Array.isArray(data.groups)) {
        setGroups(data.groups as IndicatorGroup[])
      }
      setDraftTargets({})
      setReloadToken((x) => x + 1)
    } catch (err) {
      console.error(err)
    }
  }

  const handleExport = async () => {
    try {
      const res = await fetch('/api/export', { method: 'POST' })
      if (!res.ok) throw new Error('导出失败')
      const blob = await res.blob()
      const url = window.URL.createObjectURL(blob)
      const a = document.createElement('a')
      const y = status?.currentYear ?? ''
      const m = status?.currentMonth ?? ''
      a.href = url
      a.download = `月报-${y}-${String(m).padStart(2, '0')}.xlsx`
      document.body.appendChild(a)
      a.click()
      a.remove()
      window.URL.revokeObjectURL(url)
    } catch (err) {
      console.error(err)
    }
  }

  // 空状态
  if (status && !status.initialized) {
    return (
      <div className="flex h-screen items-center justify-center bg-gradient-to-b from-background via-background to-muted/20">
        <div className="text-center space-y-4">
          <div className="mx-auto w-[520px] max-w-[92vw] rounded-2xl border border-border/60 bg-card/60 p-8 text-left shadow-2xl backdrop-blur">
            <div className="flex items-start justify-between gap-4">
              <div>
                <h2 className="text-2xl font-semibold">仪表盘</h2>
                <p className="mt-1 text-sm text-muted-foreground">关键指标总览 + 企业数据微调</p>
              </div>
              <Badge variant="secondary" className="mt-1">
                未导入
              </Badge>
            </div>

            <div className="mt-6 space-y-2">
              <p className="text-sm text-muted-foreground">
                请选择日常使用的预估表 Excel（例如：<span className="font-mono">12月月报（预估）_补全企业名称社会代码_20260129.xlsx</span>）。
              </p>
            </div>

            <div className="mt-6 flex gap-2">
              <Button onClick={() => setShowImportDialog(true)} size="lg" className="flex-1">
                <Upload className="mr-2 h-4 w-4" />
                导入数据
              </Button>
              <Button variant="outline" size="lg" onClick={() => loadStatus()}>
                <RefreshCw className="mr-2 h-4 w-4" />
                刷新
              </Button>
            </div>
          </div>
        </div>

        {showImportDialog && (
          <ImportDialog
            open={showImportDialog}
            onClose={() => setShowImportDialog(false)}
            onSuccess={handleImportSuccess}
          />
        )}
      </div>
    )
  }

  const hasDraft = Object.keys(draftTargets).length > 0
  const applySingle = async (id: string, raw: string) => {
    try {
      const v = Number(String(raw).replaceAll(',', '').trim())
      if (!Number.isFinite(v)) return
      await applyOptimize({ [id]: v }, [id])
    } catch (err) {
      console.error(err)
    }
  }

  return (
    <div className="min-h-screen bg-gradient-to-b from-background via-background to-muted/20">
      <div className="mx-auto w-full max-w-none space-y-6 p-6">
        {/* 顶部栏 */}
        <div className="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
          <div>
            <h1 className="text-3xl font-semibold tracking-tight">仪表盘</h1>
            {status && (
              <p className="mt-1 text-sm text-muted-foreground">
                {status.currentYear}年{status.currentMonth}月 · 共 {status.totalCompanies} 家（批零 {status.wrCount} + 住餐{' '}
                {status.acCount}）
              </p>
            )}
          </div>

          <div className="flex flex-wrap items-center justify-start gap-2 lg:justify-end">
            <Badge variant="secondary" className="gap-2">
              {saving ? (
                <>
                  <span className="inline-block h-2 w-2 animate-pulse rounded-full bg-emerald-400" />
                  {saveText}
                </>
              ) : (
                <>
                  <span className="inline-block h-2 w-2 rounded-full bg-muted-foreground/60" />
                  已保存
                </>
              )}
            </Badge>

            <div className="flex items-center gap-2">
              <span className="hidden text-xs text-muted-foreground lg:inline">月份</span>
              <Select value={currentMonthKey || undefined} onValueChange={selectMonth} disabled={!canSelectMonth}>
                <SelectTrigger className="h-9 w-[180px]">
                  <SelectValue placeholder={monthsLoading ? '加载中…' : '选择月份'} />
                </SelectTrigger>
                <SelectContent>
                  {months.map((it) => {
                    const key = `${it.year}-${String(it.month).padStart(2, '0')}`
                    return (
                      <SelectItem key={key} value={key}>
                        {it.year}年{it.month}月 · {it.totalCompanies} 家
                      </SelectItem>
                    )
                  })}
                </SelectContent>
              </Select>
            </div>

            <Button
              disabled={!hasDraft || optimizing}
              className="gap-2 bg-orange-500 text-black opacity-80 hover:bg-orange-400"
              onClick={handleSmartAdjust}
              title={hasDraft ? '按输入值反推并写回企业数据' : '先在指标输入框里填入目标值'}
            >
              智能调整
            </Button>

            <Button onClick={handleResetAll} variant="outline" className="gap-2">
              重置
            </Button>

            <Button onClick={() => setShowImportDialog(true)} variant="outline" className="gap-2">
              <Upload className="h-4 w-4" />
              导入
            </Button>

            <Button onClick={handleExport} variant="outline" className="gap-2" disabled={saving}>
              <Download className="h-4 w-4" />
              导出
            </Button>

            <Button onClick={() => loadIndicators()} variant="outline" className="gap-2">
              <RefreshCw className="h-4 w-4" />
              刷新
            </Button>
          </div>
        </div>

        {/* 指标面板 */}
        {loading ? (
          <div className="grid grid-cols-1 gap-4 lg:grid-cols-4">
            {[...Array(4)].map((_, i) => (
              <Card key={i} className="border-border/60 bg-card/60 backdrop-blur">
                <CardHeader className="pb-3">
                  <Skeleton className="h-4 w-32" />
                </CardHeader>
                <CardContent className="space-y-2">
                  {[...Array(4)].map((__, j) => (
                    <Skeleton key={j} className="h-9 w-full" />
                  ))}
                </CardContent>
              </Card>
            ))}
          </div>
        ) : (
          <div className="grid grid-cols-1 gap-4 lg:grid-cols-4">
            {groups.map((g) => (
              <IndicatorGroupCard
                key={g.name}
                group={g}
                draftTargets={draftTargets}
                onDraftChange={(id, v) => setDraftTargets((prev) => ({ ...prev, [id]: v }))}
                onApplySingle={applySingle}
                disabled={optimizing}
              />
            ))}
          </div>
        )}

        {/* 明细表 */}
        <CompaniesTable
          onIndicatorsUpdate={(next) => setGroups(next)}
          onSavingChange={(s) => {
            setTableSaving(s)
          }}
          monthSelector={
            <div className="flex items-center gap-2">
              <span className="text-xs text-muted-foreground">月份</span>
              <Select value={currentMonthKey || undefined} onValueChange={selectMonth} disabled={!canSelectMonth}>
                <SelectTrigger className="h-9 w-[180px]">
                  <SelectValue placeholder={monthsLoading ? '加载中…' : '选择月份'} />
                </SelectTrigger>
                <SelectContent>
                  {months.map((it) => {
                    const key = `${it.year}-${String(it.month).padStart(2, '0')}`
                    return (
                      <SelectItem key={key} value={key}>
                        {it.year}年{it.month}月 · {it.totalCompanies} 家
                      </SelectItem>
                    )
                  })}
                </SelectContent>
              </Select>
            </div>
          }
          reloadToken={reloadToken}
        />

        {/* 导入弹窗 */}
        {showImportDialog && (
          <ImportDialog
            open={showImportDialog}
            onClose={() => setShowImportDialog(false)}
            onSuccess={handleImportSuccess}
          />
        )}
      </div>
    </div>
  )
}

function IndicatorGroupCard(props: {
  group: IndicatorGroup
  disabled?: boolean
  draftTargets: Record<string, string>
  onDraftChange: (id: string, value: string) => void
  onApplySingle: (id: string, value: string) => Promise<void>
}) {
  return (
    <Card className="border-border/60 bg-card/60 backdrop-blur supports-[backdrop-filter]:bg-card/50">
      <CardHeader className="pb-3">
        <CardTitle className="text-sm font-medium text-muted-foreground">{props.group.name}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-2">
        {props.group.indicators.map((it) => (
          <MetricRow
            key={it.id}
            label={it.name}
            indicator={it}
            type={String(it.unit).includes('%') ? 'rate' : 'value'}
            draftTargets={props.draftTargets}
            onDraftChange={props.onDraftChange}
            onApplySingle={props.onApplySingle}
            disabled={props.disabled}
          />
        ))}
      </CardContent>
    </Card>
  )
}

function MetricRow(props: {
  label: string
  indicator?: Indicator
  type?: 'rate' | 'value'
  disabled?: boolean
  draftTargets: Record<string, string>
  onDraftChange: (id: string, value: string) => void
  onApplySingle: (id: string, value: string) => Promise<void>
}) {
  const value = props.indicator?.value ?? 0
  const type = props.type ?? 'value'
  const unit = props.indicator?.unit ?? (type === 'rate' ? '%' : '')
  const id = props.indicator?.id

  const text = type === 'rate' ? `${Math.round(value)}` : Math.round(value).toLocaleString()
  const positive = value >= 0
  const tone = type === 'rate' ? (positive ? 'text-emerald-300' : 'text-rose-300') : 'text-foreground'

  const draft = id ? props.draftTargets[id] : undefined
  const displayValue = draft !== undefined ? draft : text
  const dirty = draft !== undefined && draft !== text

  return (
    <div className="flex items-center gap-3">
      <div className="min-w-0 flex-1 whitespace-normal break-words text-xs text-muted-foreground">{props.label}</div>
      <div className="flex shrink-0 items-center gap-2">
        <Input
          value={displayValue}
          disabled={props.disabled || !id}
          onChange={(e) => {
            if (!id) return
            props.onDraftChange(id, e.target.value)
          }}
          onKeyDown={async (e) => {
            if (!id) return
            if (e.key !== 'Enter') return
            e.preventDefault()
            await props.onApplySingle(id, displayValue)
          }}
          className={`h-9 w-[150px] rounded-full bg-muted/25 text-right font-mono text-sm tabular-nums ${tone} ${
            dirty ? 'border-orange-400/70 ring-1 ring-orange-400/40' : 'border-border/60'
          }`}
        />
        <span className="w-10 text-right text-xs text-muted-foreground">{unit}</span>
      </div>
    </div>
  )
}
