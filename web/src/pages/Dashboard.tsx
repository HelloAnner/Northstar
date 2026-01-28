import { useEffect, useMemo, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import Topbar from '@/components/app/Topbar'
import { useDataStore, useProjectStore } from '@/store'
import { cn, formatCurrency, formatPercent } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from '@/components/ui/dropdown-menu'
import { exportApi, indicatorsApi, optimizeApi, projectsApi } from '@/services/api'
import type { IndustryType } from '@/types'
import { ArrowUpDown, ArrowLeft, ArrowRight, Filter, Search, Sparkles, Download, RotateCcw, Undo2 } from 'lucide-react'
import { Skeleton } from '@/components/ui/skeleton'

function parseNumber(text: string) {
  const v = text.trim().replace(/,/g, '').replace(/%/g, '')
  const n = Number(v)
  return Number.isFinite(n) ? n : 0
}

function parseRate(text: string) {
  // UI 输入为 “百分比数值”，如 8.5 表示 8.5%
  const n = parseNumber(text)
  if (Math.abs(n) > 1) return n / 100
  return n
}

function clampAmount(v: number) {
  if (!Number.isFinite(v)) return 0
  if (v < 0) return 0
  if (v > 1_000_000_000) return 1_000_000_000
  return v
}

function clampRate(v: number) {
  if (!Number.isFinite(v)) return 0
  if (v < -0.5) return -0.5
  if (v > 1) return 1
  return v
}

function formatDateTime(iso: string | undefined) {
  if (!iso) return '-'
  const d = new Date(iso)
  if (Number.isNaN(d.getTime()) || d.getFullYear() <= 2000) return '-'
  return d.toLocaleString('zh-CN', { hour12: false })
}

function MetricInput({
  value,
  kind,
  onCommit,
  className,
}: {
  value: number
  kind: 'value' | 'rate'
  onCommit: (v: number) => Promise<void> | void
  className?: string
}) {
  const display = kind === 'rate' ? formatPercent(value) : formatCurrency(value)
  const editingRef = useRef(false)
  const timerRef = useRef<number | null>(null)
  const inputRef = useRef<HTMLInputElement | null>(null)

  useEffect(() => {
    if (!editingRef.current && inputRef.current) {
      inputRef.current.value = display
    }
  }, [display])

  useEffect(() => {
    return () => {
      if (timerRef.current) {
        window.clearTimeout(timerRef.current)
      }
    }
  }, [])

  const commitDraft = async (nextDraft: string) => {
    const parsed = kind === 'rate' ? parseRate(nextDraft) : parseNumber(nextDraft)
    await onCommit(parsed)
  }

  const scheduleCommit = (nextDraft: string) => {
    if (timerRef.current) {
      window.clearTimeout(timerRef.current)
    }
    timerRef.current = window.setTimeout(() => {
      void commitDraft(nextDraft)
    }, 450)
  }

  const isNegativeRate = kind === 'rate' && value < 0

  return (
    <Input
      onChange={(e) => {
        const nextDraft = e.target.value
        scheduleCommit(nextDraft)
      }}
      ref={inputRef}
      defaultValue={display}
      onFocus={() => {
        editingRef.current = true
      }}
      onBlur={async () => {
        editingRef.current = false
        if (timerRef.current) {
          window.clearTimeout(timerRef.current)
        }
        const text = inputRef.current?.value ?? display
        await commitDraft(text)
        if (inputRef.current) {
          const parsed = kind === 'rate' ? parseRate(text) : parseNumber(text)
          inputRef.current.value = kind === 'rate' ? formatPercent(parsed) : formatCurrency(parsed)
        }
      }}
      onKeyDown={(e) => {
        if (e.key === 'Enter') {
          e.currentTarget.blur()
        }
      }}
      className={cn(
        'h-9 w-[220px] rounded-lg border-white/10 bg-white/5 text-right text-[12px] font-semibold placeholder:text-[#A3A3A3]',
        isNegativeRate ? 'text-emerald-400' : 'text-white',
        className,
      )}
    />
  )
}

function MetricRow({
  label,
  value,
  kind,
  onCommit,
  layout = 'row',
}: {
  label: string
  value: number
  kind: 'value' | 'rate'
  onCommit: (v: number) => Promise<void> | void
  layout?: 'row' | 'stack'
}) {
  if (layout === 'stack') {
    return (
      <div className="flex flex-col gap-1 text-sm">
        <div className="truncate text-[#A3A3A3]">{label}</div>
        <MetricInput value={value} kind={kind} onCommit={onCommit} className="w-full" />
      </div>
    )
  }

  return (
    <div className="flex items-center justify-between gap-4 text-sm">
      <div className="text-[#A3A3A3]">{label}</div>
      <MetricInput value={value} kind={kind} onCommit={onCommit} />
    </div>
  )
}

function IndustryColumn({
  title,
  monthRate,
  cumulativeRate,
  onCommitMonth,
  onCommitCumulative,
}: {
  title: string
  monthRate: number
  cumulativeRate: number
  onCommitMonth: (v: number) => Promise<void> | void
  onCommitCumulative: (v: number) => Promise<void> | void
}) {
  return (
    <div className="flex flex-col gap-2">
      <div className="text-center text-sm font-medium text-white">{title}</div>
      <div className="flex items-center gap-2 text-xs">
        <div className="w-9 text-[#A3A3A3]">当月</div>
        <MetricInput value={monthRate} kind="rate" onCommit={onCommitMonth} />
      </div>
      <div className="flex items-center gap-2 text-xs">
        <div className="w-9 text-[#A3A3A3]">累计</div>
        <MetricInput value={cumulativeRate} kind="rate" onCommit={onCommitCumulative} />
      </div>
    </div>
  )
}

function EditableNumber({
  value,
  format,
  onCommit,
  className,
}: {
  value: number
  format: (v: number) => string
  onCommit: (v: number) => Promise<void> | void
  className?: string
}) {
  const editingRef = useRef(false)
  const timerRef = useRef<number | null>(null)
  const inputRef = useRef<HTMLInputElement | null>(null)
  const display = format(value)

  useEffect(() => {
    if (!editingRef.current && inputRef.current) {
      inputRef.current.value = display
    }
  }, [display])

  useEffect(() => {
    return () => {
      if (timerRef.current) {
        window.clearTimeout(timerRef.current)
      }
    }
  }, [])

  const commit = async (nextDraft: string, formatAfterCommit: boolean) => {
    const next = clampAmount(parseNumber(nextDraft))
    await onCommit(next)
    if (formatAfterCommit && inputRef.current) {
      inputRef.current.value = format(next)
    }
  }

  const scheduleCommit = (nextDraft: string) => {
    if (timerRef.current) {
      window.clearTimeout(timerRef.current)
    }
    timerRef.current = window.setTimeout(() => {
      void commit(nextDraft, false)
    }, 450)
  }

  return (
    <Input
      onChange={(e) => {
        const nextDraft = e.target.value
        scheduleCommit(nextDraft)
      }}
      ref={inputRef}
      defaultValue={display}
      onFocus={() => {
        editingRef.current = true
      }}
      onBlur={async () => {
        editingRef.current = false
        if (timerRef.current) {
          window.clearTimeout(timerRef.current)
        }
        const text = inputRef.current?.value ?? display
        await commit(text, true)
      }}
      onKeyDown={async (e) => {
        if (e.key === 'Enter') {
          e.currentTarget.blur()
        }
      }}
      className={cn(
        'h-9 w-full rounded-lg border-white/10 bg-white/5 text-center text-[12px] font-semibold text-white placeholder:text-[#A3A3A3]',
        className,
      )}
    />
  )
}

function EditableText({
  value,
  onCommit,
  className,
}: {
  value: string
  onCommit: (v: string) => Promise<void> | void
  className?: string
}) {
  const editingRef = useRef(false)
  const timerRef = useRef<number | null>(null)
  const inputRef = useRef<HTMLInputElement | null>(null)

  useEffect(() => {
    if (!editingRef.current && inputRef.current) {
      inputRef.current.value = value
    }
  }, [value])

  useEffect(() => {
    return () => {
      if (timerRef.current) {
        window.clearTimeout(timerRef.current)
      }
    }
  }, [])

  const commit = async (nextDraft: string, formatAfterCommit: boolean) => {
    const next = nextDraft.trim()
    if (!next) return
    if (next === value) return
    await onCommit(next)
    if (formatAfterCommit && inputRef.current) {
      inputRef.current.value = next
    }
  }

  const scheduleCommit = (nextDraft: string) => {
    if (timerRef.current) {
      window.clearTimeout(timerRef.current)
    }
    timerRef.current = window.setTimeout(() => {
      void commit(nextDraft, false)
    }, 600)
  }

  return (
    <Input
      onChange={(e) => {
        const nextDraft = e.target.value
        scheduleCommit(nextDraft)
      }}
      ref={inputRef}
      defaultValue={value}
      onFocus={() => {
        editingRef.current = true
      }}
      onBlur={async () => {
        editingRef.current = false
        if (timerRef.current) {
          window.clearTimeout(timerRef.current)
        }
        const text = inputRef.current?.value ?? value
        await commit(text, true)
      }}
      onKeyDown={(e) => {
        if (e.key === 'Enter') {
          e.currentTarget.blur()
        }
      }}
      className={cn(
        'h-9 w-full rounded-lg border-white/10 bg-white/5 text-center text-[12px] font-semibold text-white placeholder:text-[#A3A3A3]',
        className,
      )}
    />
  )
}

function EditableGrowthRate({
  lastYearRetailMonth,
  salesCurrentMonth,
  value,
  onCommitRetailCurrentMonth,
}: {
  lastYearRetailMonth: number
  salesCurrentMonth: number
  value: number
  onCommitRetailCurrentMonth: (v: number) => Promise<void> | void
}) {
  const display = formatPercent(value)
  const editingRef = useRef(false)
  const timerRef = useRef<number | null>(null)
  const inputRef = useRef<HTMLInputElement | null>(null)

  useEffect(() => {
    if (!editingRef.current && inputRef.current) {
      inputRef.current.value = display
    }
  }, [display])

  useEffect(() => {
    return () => {
      if (timerRef.current) {
        window.clearTimeout(timerRef.current)
      }
    }
  }, [])

  const commit = async (nextDraft: string, formatAfterCommit: boolean) => {
    const rate = clampRate(parseRate(nextDraft))
    if (lastYearRetailMonth <= 0) return
    let next = lastYearRetailMonth * (1 + rate)
    next = clampAmount(next)
    if (salesCurrentMonth > 0) {
      next = Math.min(next, salesCurrentMonth)
    }
    await onCommitRetailCurrentMonth(next)
    if (formatAfterCommit && inputRef.current) {
      inputRef.current.value = formatPercent(rate)
    }
  }

  const scheduleCommit = (nextDraft: string) => {
    if (timerRef.current) {
      window.clearTimeout(timerRef.current)
    }
    timerRef.current = window.setTimeout(() => {
      void commit(nextDraft, false)
    }, 450)
  }

  const isNegative = value < 0

  return (
    <Input
      onChange={(e) => {
        const nextDraft = e.target.value
        scheduleCommit(nextDraft)
      }}
      ref={inputRef}
      defaultValue={display}
      onFocus={() => {
        editingRef.current = true
      }}
      onBlur={async () => {
        editingRef.current = false
        if (timerRef.current) {
          window.clearTimeout(timerRef.current)
        }
        const text = inputRef.current?.value ?? display
        await commit(text, true)
      }}
      onKeyDown={async (e) => {
        if (e.key === 'Enter') {
          e.currentTarget.blur()
        }
      }}
      className={cn(
        'h-9 w-full rounded-lg border-white/10 bg-white/5 text-center text-[12px] font-bold placeholder:text-[#A3A3A3]',
        isNegative ? 'text-emerald-400' : 'text-white',
      )}
    />
  )
}

export default function Dashboard() {
  const navigate = useNavigate()
  const pageScrollRef = useRef<HTMLDivElement | null>(null)
  const pendingScrollTopRef = useRef<number | null>(null)
  const refreshSavedAtTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const refreshCurrent = useProjectStore((s) => s.refreshCurrent)
  const currentProjectId = useProjectStore((s) => s.current?.project?.projectId)
  const currentHasData = useProjectStore((s) => s.current?.hasData)
  const currentUpdatedAt = useProjectStore((s) => s.current?.project?.updatedAt)
  const {
    indicators,
    companies,
    totalCompanies,
    currentPage,
    pageSize,
    setPage,
    industryFilter,
    scaleFilter,
    sortBy,
    sortDir,
    loading,
    fetchIndicators,
    fetchCompanies,
    fetchConfig,
    updateCompany,
    resetCompanies,
    setSearchKeyword,
    setIndustryFilter,
    setScaleFilter,
    setSort,
    searchKeyword,
  } = useDataStore()

  const [bootstrapped, setBootstrapped] = useState(false)
  const [undoing, setUndoing] = useState(false)

  useEffect(() => {
    ;(async () => {
      await refreshCurrent()
      setBootstrapped(true)
    })()
  }, [refreshCurrent])

  useEffect(() => {
    if (!bootstrapped) return
    if (!currentProjectId) {
      navigate('/')
      return
    }
    if (!currentHasData) {
      navigate('/import')
      return
    }
    Promise.all([fetchIndicators(), fetchConfig(), fetchCompanies()])
  }, [bootstrapped, currentHasData, currentProjectId, fetchCompanies, fetchConfig, fetchIndicators, navigate])

  const scheduleRefreshSavedAt = () => {
    if (refreshSavedAtTimerRef.current) {
      clearTimeout(refreshSavedAtTimerRef.current)
    }
    // 后端 debounce 保存为 1000ms，这里稍微留出余量
    refreshSavedAtTimerRef.current = setTimeout(() => {
      refreshCurrent()
    }, 1500)
  }

  useEffect(() => {
    fetchCompanies()
  }, [fetchCompanies, searchKeyword, currentPage, pageSize, industryFilter, scaleFilter, sortBy, sortDir])

  useEffect(() => {
    if (loading) return
    if (pendingScrollTopRef.current == null) return
    const el = pageScrollRef.current
    if (!el) return
    el.scrollTop = pendingScrollTopRef.current
    pendingScrollTopRef.current = null
  }, [loading, companies.length])

  const adjust = async (key: string, value: number) => {
    const next = await indicatorsApi.adjust(key, value)
    useDataStore.getState().setIndicators(next)
    await Promise.all([fetchCompanies(), fetchConfig()])
    scheduleRefreshSavedAt()
  }

  const undoLast = async () => {
    if (undoing) return
    setUndoing(true)
    try {
      const res = await projectsApi.undoCurrent()
      useDataStore.getState().setIndicators(res.indicators)
      await Promise.all([fetchCompanies(), fetchConfig()])
      scheduleRefreshSavedAt()
    } catch (e) {
      console.warn('undo failed', e)
    } finally {
      setUndoing(false)
    }
  }

  const statusText = useMemo(() => {
    const ts = formatDateTime(currentUpdatedAt)
    return `上次保存时间 ${ts}`
  }, [currentUpdatedAt])

  const industryCards = useMemo(() => {
    const industries: { key: IndustryType; title: string }[] = [
      { key: 'wholesale', title: '批发业' },
      { key: 'retail', title: '零售业' },
      { key: 'accommodation', title: '住宿业' },
      { key: 'catering', title: '餐饮业' },
    ]
    return industries
  }, [])

  return (
    <>
      <Topbar title="仪表盘" statusText={statusText} />

      <div ref={pageScrollRef} className="flex-1 overflow-y-scroll p-6">
        <div className="flex items-start justify-between">
          <div>
            <div className="text-2xl font-semibold">仪表盘</div>
            <div className="mt-1 text-sm text-[#A3A3A3]">关键指标总览 + 企业数据微调</div>
          </div>

          <div className="flex max-w-full flex-1 items-center justify-end gap-3 overflow-x-auto whitespace-nowrap">
            <div className="flex shrink-0 items-center gap-2">
              <div className="shrink-0 text-[11px] text-[#A3A3A3]">限上社零额增速（当月）</div>
              <MetricInput
                value={indicators.limitAboveMonthRate}
                kind="rate"
                onCommit={(v) => adjust('limitAboveMonthRate', v)}
                className="w-[160px]"
              />
            </div>

            <div className="flex shrink-0 items-center gap-2">
              <div className="shrink-0 text-[11px] text-[#A3A3A3]">限上社零额增速（累计）</div>
              <MetricInput
                value={indicators.limitAboveCumulativeRate}
                kind="rate"
                onCommit={(v) => adjust('limitAboveCumulativeRate', v)}
                className="w-[160px]"
              />
            </div>

            <div className="flex shrink-0 items-center gap-2">
              <Button
                className="bg-[#FF6B35] text-black hover:bg-[#FF6B35]/90"
                onClick={async () => {
                  // 默认目标：当前累计增速 + 1 个百分点
                  const target = indicators.limitAboveCumulativeRate + 0.01
                  await optimizeApi.run(target)
                  await Promise.all([fetchCompanies(), fetchIndicators(), fetchConfig()])
                  scheduleRefreshSavedAt()
                }}
              >
                <Sparkles className="mr-2 h-4 w-4" />
                智能调整
              </Button>
              <Button
                variant="secondary"
                className="border border-white/10 bg-white/5 text-white hover:bg-white/10"
                onClick={async () => {
                  await resetCompanies()
                  await fetchIndicators()
                  scheduleRefreshSavedAt()
                }}
              >
                <RotateCcw className="mr-2 h-4 w-4" />
                重置
              </Button>
              <Button
                variant="secondary"
                disabled={undoing}
                className="border border-white/10 bg-white/5 text-white hover:bg-white/10"
                onClick={undoLast}
              >
                <Undo2 className="mr-2 h-4 w-4" />
                撤销
              </Button>
              <Button
                variant="outline"
                className="border-white/10 bg-transparent text-white hover:bg-white/5"
                onClick={async () => {
                  const res = await exportApi.export({ format: 'xlsx', includeIndicators: true, includeChanges: true })
                  window.location.href = res.downloadUrl
                }}
              >
                <Download className="mr-2 h-4 w-4" />
                导出
              </Button>
            </div>
          </div>
        </div>

        <div className="mt-6 grid grid-cols-12 gap-4">
          <Card className="col-span-12 border-white/10 bg-[#0D0D0D] p-4 lg:col-span-3">
            <CardHeader className="space-y-1 p-0">
              <div className="text-lg font-semibold text-white">限上社零额</div>
              <div className="text-[11px] text-[#A3A3A3]">单位：万元 / %</div>
            </CardHeader>
            <CardContent className="mt-4 space-y-3 p-0">
              <MetricRow
                label="当期零售额（万元）"
                value={indicators.limitAboveMonthValue}
                kind="value"
                onCommit={(v) => adjust('limitAboveMonthValue', v)}
              />
              <MetricRow
                label="当月增速（%）"
                value={indicators.limitAboveMonthRate}
                kind="rate"
                onCommit={(v) => adjust('limitAboveMonthRate', v)}
              />
              <MetricRow
                label="累计零售额（万元）"
                value={indicators.limitAboveCumulativeValue}
                kind="value"
                onCommit={(v) => adjust('limitAboveCumulativeValue', v)}
              />
              <MetricRow
                label="累计增速（%）"
                value={indicators.limitAboveCumulativeRate}
                kind="rate"
                onCommit={(v) => adjust('limitAboveCumulativeRate', v)}
              />
            </CardContent>
          </Card>

          <Card className="col-span-12 border-white/10 bg-[#0D0D0D] p-4 lg:col-span-3">
            <CardHeader className="space-y-1 p-0">
              <div className="text-lg font-semibold text-white">专项增速</div>
              <div className="text-[11px] text-[#A3A3A3]">当月口径</div>
            </CardHeader>
            <CardContent className="mt-4 space-y-3 p-0">
              <MetricRow
                label="吃穿用增速（当月）"
                value={indicators.eatWearUseMonthRate}
                kind="rate"
                onCommit={(v) => adjust('eatWearUseMonthRate', v)}
                layout="stack"
              />
              <MetricRow
                label="小微企业增速（当月）"
                value={indicators.microSmallMonthRate}
                kind="rate"
                onCommit={(v) => adjust('microSmallMonthRate', v)}
                layout="stack"
              />
            </CardContent>
          </Card>

          <Card className="col-span-12 border-white/10 bg-[#0D0D0D] p-4 lg:col-span-4">
            <CardHeader className="space-y-1 p-0">
              <div className="text-lg font-semibold text-white">四大行业增速</div>
              <div className="text-[11px] text-[#A3A3A3]">当月 / 累计</div>
            </CardHeader>
            <CardContent className="mt-4 grid grid-cols-2 gap-4 p-0">
              {industryCards.map((it) => (
                <IndustryColumn
                  key={it.key}
                  title={it.title}
                  monthRate={indicators.industryRates[it.key]?.monthRate ?? 0}
                  cumulativeRate={indicators.industryRates[it.key]?.cumulativeRate ?? 0}
                  onCommitMonth={(v) => adjust(`industry.${it.key}.monthRate`, v)}
                  onCommitCumulative={(v) => adjust(`industry.${it.key}.cumulativeRate`, v)}
                />
              ))}
            </CardContent>
          </Card>

          <Card className="col-span-12 border-white/10 bg-[#0D0D0D] p-4 lg:col-span-2">
            <CardHeader className="space-y-1 p-0">
              <div className="text-lg font-semibold text-white">社零总额估算</div>
              <div className="text-[11px] text-[#A3A3A3]">估算值 + 增速</div>
            </CardHeader>
            <CardContent className="mt-4 space-y-3 p-0">
              <MetricRow
                label="社零总额（估算）"
                value={indicators.totalSocialCumulativeValue}
                kind="value"
                onCommit={(v) => adjust('totalSocialCumulativeValue', v)}
                layout="stack"
              />
              <MetricRow
                label="累计增速（%）"
                value={indicators.totalSocialCumulativeRate}
                kind="rate"
                onCommit={(v) => adjust('totalSocialCumulativeRate', v)}
                layout="stack"
              />
            </CardContent>
          </Card>
        </div>

        <Card className="mt-6 border-white/10 bg-[#0D0D0D] p-4">
          <CardHeader className="space-y-1 p-0">
            <div className="text-lg font-semibold text-white">企业数据微调</div>
            <div className="text-sm text-[#A3A3A3]">支持搜索 / 筛选 / 排序；修改后自动保存</div>
          </CardHeader>

          <CardContent className="mt-4 p-0">
            <div className="flex items-center justify-between gap-3">
              <div className="flex h-10 flex-1 items-center gap-2 rounded-lg border border-white/10 bg-white/5 px-3 text-[12px] text-[#A3A3A3]">
                <Search className="h-4 w-4" />
                <Input
                  value={searchKeyword}
                  onChange={(e) => setSearchKeyword(e.target.value)}
                  placeholder="按行业 / 规模搜索企业…"
                  className="h-8 border-0 bg-transparent p-0 text-white placeholder:text-[#A3A3A3] focus-visible:ring-0"
                />
              </div>

              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button variant="outline" className="border-white/10 bg-transparent text-white hover:bg-white/5">
                    <Filter className="mr-2 h-4 w-4" />
                    筛选
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent className="border-white/10 bg-[#0D0D0D] text-white">
                  <DropdownMenuItem
                    className="text-[12px] text-white"
                    onSelect={() => {
                      pendingScrollTopRef.current = pageScrollRef.current?.scrollTop ?? null
                      setIndustryFilter('')
                      setScaleFilter('')
                    }}
                  >
                    清除筛选
                  </DropdownMenuItem>
                  <DropdownMenuItem className="text-[12px] text-[#A3A3A3]">行业</DropdownMenuItem>
                  <DropdownMenuItem
                    className={cn('text-[12px]', !industryFilter ? 'text-white' : 'text-[#A3A3A3]')}
                    onSelect={() => {
                      pendingScrollTopRef.current = pageScrollRef.current?.scrollTop ?? null
                      setIndustryFilter('')
                    }}
                  >
                    全部行业
                  </DropdownMenuItem>
                  <DropdownMenuItem
                    className={cn('text-[12px]', industryFilter === 'retail' ? 'text-white' : 'text-[#A3A3A3]')}
                    onSelect={() => {
                      pendingScrollTopRef.current = pageScrollRef.current?.scrollTop ?? null
                      setIndustryFilter('retail')
                    }}
                  >
                    零售业
                  </DropdownMenuItem>
                  <DropdownMenuItem
                    className={cn('text-[12px]', industryFilter === 'wholesale' ? 'text-white' : 'text-[#A3A3A3]')}
                    onSelect={() => {
                      pendingScrollTopRef.current = pageScrollRef.current?.scrollTop ?? null
                      setIndustryFilter('wholesale')
                    }}
                  >
                    批发业
                  </DropdownMenuItem>
                  <DropdownMenuItem
                    className={cn('text-[12px]', industryFilter === 'accommodation' ? 'text-white' : 'text-[#A3A3A3]')}
                    onSelect={() => {
                      pendingScrollTopRef.current = pageScrollRef.current?.scrollTop ?? null
                      setIndustryFilter('accommodation')
                    }}
                  >
                    住宿业
                  </DropdownMenuItem>
                  <DropdownMenuItem
                    className={cn('text-[12px]', industryFilter === 'catering' ? 'text-white' : 'text-[#A3A3A3]')}
                    onSelect={() => {
                      pendingScrollTopRef.current = pageScrollRef.current?.scrollTop ?? null
                      setIndustryFilter('catering')
                    }}
                  >
                    餐饮业
                  </DropdownMenuItem>

                  <DropdownMenuItem className="text-[12px] text-[#A3A3A3]">规模</DropdownMenuItem>
                  <DropdownMenuItem
                    className={cn('text-[12px]', !scaleFilter ? 'text-white' : 'text-[#A3A3A3]')}
                    onSelect={() => {
                      pendingScrollTopRef.current = pageScrollRef.current?.scrollTop ?? null
                      setScaleFilter('')
                    }}
                  >
                    全部规模
                  </DropdownMenuItem>
                  <DropdownMenuItem
                    className={cn('text-[12px]', scaleFilter === '1' ? 'text-white' : 'text-[#A3A3A3]')}
                    onSelect={() => {
                      pendingScrollTopRef.current = pageScrollRef.current?.scrollTop ?? null
                      setScaleFilter('1')
                    }}
                  >
                    大型
                  </DropdownMenuItem>
                  <DropdownMenuItem
                    className={cn('text-[12px]', scaleFilter === '2' ? 'text-white' : 'text-[#A3A3A3]')}
                    onSelect={() => {
                      pendingScrollTopRef.current = pageScrollRef.current?.scrollTop ?? null
                      setScaleFilter('2')
                    }}
                  >
                    中型
                  </DropdownMenuItem>
                  <DropdownMenuItem
                    className={cn('text-[12px]', scaleFilter === '3,4' ? 'text-white' : 'text-[#A3A3A3]')}
                    onSelect={() => {
                      pendingScrollTopRef.current = pageScrollRef.current?.scrollTop ?? null
                      setScaleFilter('3,4')
                    }}
                  >
                    小微
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>

              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button variant="outline" className="border-white/10 bg-transparent text-white hover:bg-white/5">
                    <ArrowUpDown className="mr-2 h-4 w-4" />
                    排序
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent className="border-white/10 bg-[#0D0D0D] text-white">
                  <DropdownMenuItem
                    className="text-[12px] text-white"
                    onSelect={() => {
                      pendingScrollTopRef.current = pageScrollRef.current?.scrollTop ?? null
                      setSort('name', 'asc')
                    }}
                  >
                    清除排序
                  </DropdownMenuItem>
                  <DropdownMenuItem
                    className={cn('text-[12px]', sortBy === 'name' && sortDir === 'asc' ? 'text-white' : 'text-[#A3A3A3]')}
                    onSelect={() => {
                      pendingScrollTopRef.current = pageScrollRef.current?.scrollTop ?? null
                      setSort('name', 'asc')
                    }}
                  >
                    企业名称 ↑
                  </DropdownMenuItem>
                  <DropdownMenuItem
                    className={cn('text-[12px]', sortBy === 'name' && sortDir === 'desc' ? 'text-white' : 'text-[#A3A3A3]')}
                    onSelect={() => {
                      pendingScrollTopRef.current = pageScrollRef.current?.scrollTop ?? null
                      setSort('name', 'desc')
                    }}
                  >
                    企业名称 ↓
                  </DropdownMenuItem>
                  <DropdownMenuItem
                    className={cn(
                      'text-[12px]',
                      sortBy === 'salesCurrentMonth' && sortDir === 'desc' ? 'text-white' : 'text-[#A3A3A3]'
                    )}
                    onSelect={() => {
                      pendingScrollTopRef.current = pageScrollRef.current?.scrollTop ?? null
                      setSort('salesCurrentMonth', 'desc')
                    }}
                  >
                    总销售额 ↓
                  </DropdownMenuItem>
                  <DropdownMenuItem
                    className={cn(
                      'text-[12px]',
                      sortBy === 'retailCurrentMonth' && sortDir === 'desc' ? 'text-white' : 'text-[#A3A3A3]'
                    )}
                    onSelect={() => {
                      pendingScrollTopRef.current = pageScrollRef.current?.scrollTop ?? null
                      setSort('retailCurrentMonth', 'desc')
                    }}
                  >
                    本期零售额 ↓
                  </DropdownMenuItem>
                  <DropdownMenuItem
                    className={cn(
                      'text-[12px]',
                      sortBy === 'retailLastYearMonth' && sortDir === 'desc' ? 'text-white' : 'text-[#A3A3A3]'
                    )}
                    onSelect={() => {
                      pendingScrollTopRef.current = pageScrollRef.current?.scrollTop ?? null
                      setSort('retailLastYearMonth', 'desc')
                    }}
                  >
                    同期零售额 ↓
                  </DropdownMenuItem>
                  <DropdownMenuItem
                    className={cn(
                      'text-[12px]',
                      sortBy === 'monthGrowthRate' && sortDir === 'desc' ? 'text-white' : 'text-[#A3A3A3]'
                    )}
                    onSelect={() => {
                      pendingScrollTopRef.current = pageScrollRef.current?.scrollTop ?? null
                      setSort('monthGrowthRate', 'desc')
                    }}
                  >
                    增速 ↓
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </div>

            <div className="mt-4 overflow-hidden rounded-lg border border-white/10">
              <Table>
                <TableHeader>
                  <TableRow className="border-white/10">
                    <TableHead className="text-center text-[#A3A3A3]">企业名称</TableHead>
                    <TableHead className="text-center text-[#A3A3A3]">总销售额</TableHead>
                    <TableHead className="text-center text-[#A3A3A3]">本期零售额</TableHead>
                    <TableHead className="text-center text-[#A3A3A3]">同期零售额</TableHead>
                    <TableHead className="text-center text-[#A3A3A3]">增速</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {Array.from({ length: pageSize }).map((_, idx) => {
                    if (loading) {
                      return (
                        <TableRow key={`loading-${idx}`} className="border-white/10">
                          <TableCell className="min-w-[220px]">
                            <Skeleton className="h-9 w-full bg-white/5" />
                          </TableCell>
                          <TableCell className="min-w-[180px]">
                            <Skeleton className="h-9 w-full bg-white/5" />
                          </TableCell>
                          <TableCell className="min-w-[180px]">
                            <Skeleton className="h-9 w-full bg-white/5" />
                          </TableCell>
                          <TableCell className="min-w-[180px]">
                            <Skeleton className="h-9 w-full bg-white/5" />
                          </TableCell>
                          <TableCell className="min-w-[140px]">
                            <Skeleton className="h-9 w-full bg-white/5" />
                          </TableCell>
                        </TableRow>
                      )
                    }

                    const c = companies[idx]
                    if (!c) {
                      if (totalCompanies === 0 && idx === 0) {
                        return (
                          <TableRow key="empty-message" className="border-white/10">
                            <TableCell colSpan={5} className="py-6 text-center text-sm text-[#A3A3A3]">
                              暂无数据，请先导入数据
                            </TableCell>
                          </TableRow>
                        )
                      }
                      return (
                        <TableRow key={`placeholder-${idx}`} className="border-white/10">
                          <TableCell className="min-w-[220px]">
                            <div className="h-9" />
                          </TableCell>
                          <TableCell className="min-w-[180px]">
                            <div className="h-9" />
                          </TableCell>
                          <TableCell className="min-w-[180px]">
                            <div className="h-9" />
                          </TableCell>
                          <TableCell className="min-w-[180px]">
                            <div className="h-9" />
                          </TableCell>
                          <TableCell className="min-w-[140px]">
                            <div className="h-9" />
                          </TableCell>
                        </TableRow>
                      )
                    }

                    const growth = c.monthGrowthRate ?? 0
                    const hasError = c.validation?.hasError
                    return (
                      <TableRow key={c.id} className="border-white/10">
                        <TableCell className="min-w-[220px] text-center">
                          <EditableText
                            value={c.name}
                            onCommit={async (name) => {
                              await updateCompany(c.id, { name })
                              scheduleRefreshSavedAt()
                            }}
                          />
                        </TableCell>
                        <TableCell className="min-w-[180px] text-center">
                          <EditableNumber
                            value={c.salesCurrentMonth}
                            format={formatCurrency}
                            onCommit={async (v) => {
                              const nextSales = clampAmount(v)
                              const nextRetail = nextSales > 0 && c.retailCurrentMonth > nextSales ? nextSales : c.retailCurrentMonth
                              await updateCompany(c.id, {
                                salesCurrentMonth: nextSales,
                                retailCurrentMonth: nextRetail,
                              })
                              scheduleRefreshSavedAt()
                            }}
                          />
                        </TableCell>
                        <TableCell className="min-w-[180px] text-center">
                          <EditableNumber
                            value={c.retailCurrentMonth}
                            format={formatCurrency}
                            onCommit={async (v) => {
                              let next = clampAmount(v)
                              if (c.salesCurrentMonth > 0) {
                                next = Math.min(next, c.salesCurrentMonth)
                              }
                              await updateCompany(c.id, { retailCurrentMonth: next })
                              scheduleRefreshSavedAt()
                            }}
                            className={hasError ? 'border-[#EF4444]/60 bg-[#EF4444]/10 text-[#EF4444]' : ''}
                          />
                        </TableCell>
                        <TableCell className="min-w-[180px] text-center">
                          <EditableNumber
                            value={c.retailLastYearMonth}
                            format={formatCurrency}
                            onCommit={async (v) => {
                              await updateCompany(c.id, { retailLastYearMonth: clampAmount(v) })
                              scheduleRefreshSavedAt()
                            }}
                          />
                        </TableCell>
                        <TableCell className="min-w-[140px] text-center">
                          <EditableGrowthRate
                            lastYearRetailMonth={c.retailLastYearMonth}
                            salesCurrentMonth={c.salesCurrentMonth}
                            value={growth}
                            onCommitRetailCurrentMonth={async (next) => {
                              await updateCompany(c.id, { retailCurrentMonth: next })
                              scheduleRefreshSavedAt()
                            }}
                          />
                        </TableCell>
                      </TableRow>
                    )
                  })}
                </TableBody>
              </Table>
            </div>

            {totalCompanies > 0 && (
              <div className="mt-4 flex items-center justify-between">
                <div className="text-sm text-[#A3A3A3]">
                  共 {totalCompanies} 家企业，每页 {pageSize} 家
                </div>
                <div className="flex items-center gap-2">
                  <Button
                    size="sm"
                    variant="outline"
                    className="h-8 w-8 border-white/10 bg-[#2D2D2D] p-0 text-white hover:bg-white/10"
                    disabled={currentPage <= 1}
                    onClick={() => {
                      pendingScrollTopRef.current = pageScrollRef.current?.scrollTop ?? null
                      setPage(Math.max(1, currentPage - 1))
                    }}
                  >
                    <ArrowLeft className="h-4 w-4" />
                  </Button>
                  <div className="text-sm text-white">
                    {currentPage} / {Math.max(1, Math.ceil(totalCompanies / pageSize))}
                  </div>
                  <Button
                    size="sm"
                    variant="outline"
                    className="h-8 w-8 border-white/10 bg-[#2D2D2D] p-0 text-white hover:bg-white/10"
                    disabled={currentPage >= Math.max(1, Math.ceil(totalCompanies / pageSize))}
                    onClick={() => {
                      pendingScrollTopRef.current = pageScrollRef.current?.scrollTop ?? null
                      setPage(Math.min(Math.ceil(totalCompanies / pageSize), currentPage + 1))
                    }}
                  >
                    <ArrowRight className="h-4 w-4" />
                  </Button>
                </div>
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </>
  )
}
