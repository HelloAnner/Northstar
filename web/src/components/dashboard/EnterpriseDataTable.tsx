import { useCallback, useEffect, useMemo, useState } from 'react'
import { Undo2, Wand2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'

type CompanyRow = {
  id: string
  kind: 'wr' | 'ac'
  name: string
  creditCode?: string
  industryType?: string
  companyScale?: number
  isSmallMicro?: number
  isEatWearUse?: number
  salesPrevMonth?: number
  salesCurrentMonth?: number
  salesLastYearMonth?: number
  salesMonthRate?: number
  salesCurrentCumulative?: number
  salesCumulativeRate?: number
  retailCurrentMonth?: number
  sourceSheet?: string
}

type ColumnKey =
  | 'companyScale'
  | 'flags'
  | 'salesPrevMonth'
  | 'salesCurrentMonth'
  | 'salesLastYearMonth'
  | 'salesMonthRate'
  | 'salesCurrentCumulative'
  | 'salesCumulativeRate'
  | 'retailCurrentMonth'
  | 'sourceSheet'

type IndustryTab = 'all' | 'wholesale' | 'retail' | 'accommodation' | 'catering'

const headerColumns: Array<{ key: ColumnKey; title: string; width: number; align?: 'left' | 'center' | 'right' }> = [
  { key: 'companyScale', title: '规模', width: 60, align: 'center' },
  { key: 'flags', title: '标记', width: 90, align: 'center' },
  { key: 'salesPrevMonth', title: '本年-上月', width: 95, align: 'right' },
  { key: 'salesCurrentMonth', title: '本年-本月', width: 95, align: 'right' },
  { key: 'salesLastYearMonth', title: '上年-本月', width: 95, align: 'right' },
  { key: 'salesMonthRate', title: '同比增速(当月)', width: 100, align: 'center' },
  { key: 'salesCurrentCumulative', title: '本年-1—本月', width: 110, align: 'right' },
  { key: 'salesCumulativeRate', title: '累计同比增速', width: 100, align: 'center' },
  { key: 'retailCurrentMonth', title: '零售额;本年-本月', width: 120, align: 'right' },
  { key: 'sourceSheet', title: '来源表', width: 90, align: 'center' },
]

const tabOptions: Array<{ value: IndustryTab; label: string }> = [
  { value: 'all', label: '全部' },
  { value: 'wholesale', label: '批发' },
  { value: 'retail', label: '零售' },
  { value: 'accommodation', label: '住宿' },
  { value: 'catering', label: '餐饮' },
]

type EditableField =
  | 'salesCurrentMonth'
  | 'salesLastYearMonth'
  | 'salesCurrentCumulative'
  | 'retailCurrentMonth'
  | 'revenueCurrentMonth'
  | 'revenueLastYearMonth'
  | 'revenueCurrentCumulative'

interface EnterpriseDataTableProps {
  reloadToken: number
  highlightCells?: Record<string, boolean>
  onCellPreview?: (rowId: string, columnKey: ColumnKey) => void
  onIndicatorsUpdate: (groups: Array<{ name: string; indicators: Array<{ id: string; name: string; value: number; unit: string }> }>) => void
  onSavingChange: (saving: boolean) => void
  onSmartAdjust: () => void
  onUndo: () => void
  smartAdjustDisabled: boolean
  undoDisabled: boolean
  busy?: boolean
}

const formatNumber = (value?: number) => {
  if (typeof value !== 'number' || Number.isNaN(value)) {
    return '-'
  }
  return Math.round(value).toLocaleString()
}

const formatRate = (value?: number) => {
  if (typeof value !== 'number' || Number.isNaN(value)) {
    return '-'
  }
  const rounded = Math.round(value)
  return `${rounded > 0 ? '+' : ''}${rounded}%`
}

const industryLabel = (industryType?: string) => {
  switch (industryType) {
    case 'wholesale':
      return '批发'
    case 'retail':
      return '零售'
    case 'accommodation':
      return '住宿'
    case 'catering':
      return '餐饮'
    default:
      return '-'
  }
}

const flagText = (row: CompanyRow) => {
  if (row.isSmallMicro === 1) {
    return '小微'
  }
  if (row.isEatWearUse === 1) {
    return '吃穿用'
  }
  return '-'
}

const resolveEditableField = (row: CompanyRow, key: ColumnKey): EditableField | null => {
  if (row.kind === 'ac') {
    if (key === 'salesCurrentMonth') {
      return 'revenueCurrentMonth'
    }
    if (key === 'salesLastYearMonth') {
      return 'revenueLastYearMonth'
    }
    if (key === 'salesCurrentCumulative') {
      return 'revenueCurrentCumulative'
    }
    if (key === 'retailCurrentMonth') {
      return 'retailCurrentMonth'
    }
    return null
  }

  if (key === 'salesCurrentMonth') {
    return 'salesCurrentMonth'
  }
  if (key === 'salesLastYearMonth') {
    return 'salesLastYearMonth'
  }
  if (key === 'salesCurrentCumulative') {
    return 'salesCurrentCumulative'
  }
  if (key === 'retailCurrentMonth') {
    return 'retailCurrentMonth'
  }
  return null
}

export default function EnterpriseDataTable(props: EnterpriseDataTableProps) {
  const {
    reloadToken,
    highlightCells,
    onCellPreview,
    onIndicatorsUpdate,
    onSavingChange,
    onSmartAdjust,
    onUndo,
    smartAdjustDisabled,
    undoDisabled,
    busy,
  } = props

  const [items, setItems] = useState<CompanyRow[]>([])
  const [loading, setLoading] = useState(false)
  const [savingCount, setSavingCount] = useState(0)
  const [activeTab, setActiveTab] = useState<IndustryTab>('all')
  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)

  const pageSize = 20
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const query = new URLSearchParams()
      query.set('page', String(page))
      query.set('pageSize', String(pageSize))
      query.set('industryType', activeTab)
      if (search.trim()) {
        query.set('keyword', search.trim())
      }

      const response = await fetch(`/api/companies?${query.toString()}`)
      const data = await response.json()
      if (!response.ok) {
        throw new Error(data?.error || '加载企业数据失败')
      }

      const nextItems = Array.isArray(data.items) ? data.items : []
      const nextTotal = typeof data.total === 'number' ? data.total : 0
      const safeTotalPages = Math.max(1, Math.ceil(nextTotal / pageSize))

      if (nextTotal > 0 && page > safeTotalPages) {
        setPage(safeTotalPages)
        return
      }

      setItems(nextItems)
      setTotal(nextTotal)
    } finally {
      setLoading(false)
    }
  }, [activeTab, page, pageSize, search])

  useEffect(() => {
    void load()
  }, [load, reloadToken])

  useEffect(() => {
    onSavingChange(savingCount > 0)
  }, [onSavingChange, savingCount])

  const updateField = async (row: CompanyRow, key: ColumnKey, nextValue: string) => {
    const field = resolveEditableField(row, key)
    if (!field) {
      return
    }

    const parsed = Number(nextValue)
    if (!Number.isFinite(parsed)) {
      return
    }

    setSavingCount((count) => count + 1)
    try {
      const response = await fetch(`/api/companies/${encodeURIComponent(row.id)}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ [field]: Math.round(parsed) }),
      })
      const data = await response.json()
      if (!response.ok) {
        throw new Error(data?.error || '保存失败')
      }

      const updated = data.company as Partial<CompanyRow>
      setItems((previous) => previous.map((item) => (item.id === row.id ? { ...item, ...updated } : item)))
      if (Array.isArray(data.groups)) {
        onIndicatorsUpdate(data.groups)
      }
    } finally {
      setSavingCount((count) => Math.max(0, count - 1))
    }
  }

  const pageText = useMemo(() => {
    return `第 ${page}/${totalPages} 页 · 每页 ${pageSize} 条`
  }, [page, pageSize, totalPages])

  return (
    <section className="overflow-hidden rounded-xl border border-[#E2E8F0] bg-white">
      <div className="border-b border-[#E2E8F0] px-4">
        <div className="flex h-[52px] items-center justify-between">
          <div className="text-[15px] font-semibold text-[#0F172A]">企业明细数据</div>

          <div className="flex items-center gap-2">
            <Button
              onClick={onSmartAdjust}
              disabled={smartAdjustDisabled}
              className="h-9 gap-1 bg-[#F59E0B] px-3 text-xs font-medium text-white hover:bg-[#D97706]"
            >
              <Wand2 className="h-3.5 w-3.5" />
              {busy ? '调整中' : '智能调整'}
            </Button>

            <Button onClick={onUndo} disabled={undoDisabled} variant="outline" className="h-9 gap-1 px-3 text-xs">
              <Undo2 className="h-3.5 w-3.5" />
              撤销
            </Button>
          </div>
        </div>

        <div className="flex flex-wrap items-center justify-between gap-3 pb-3">
          <Tabs
            value={activeTab}
            onValueChange={(value) => {
              setActiveTab(value as IndustryTab)
              setPage(1)
            }}
          >
            <TabsList className="h-8 rounded-lg bg-[#F8FAFC] p-1">
              {tabOptions.map((tab) => (
                <TabsTrigger
                  key={tab.value}
                  value={tab.value}
                  className="h-6 rounded-md px-3 text-xs data-[state=active]:bg-white"
                >
                  {tab.label}
                </TabsTrigger>
              ))}
            </TabsList>
          </Tabs>

          <div className="flex items-center gap-2">
            <Input
              value={search}
              onChange={(event) => {
                setSearch(event.target.value)
                setPage(1)
              }}
              placeholder="按企业名称/信用代码搜索…"
              className="h-8 w-[280px] rounded-lg border-[#E2E8F0] text-xs"
            />
            <span className="text-xs text-[#64748B]">共 {total} 家企业</span>
          </div>
        </div>
      </div>

      <div className="overflow-x-auto">
        <table className="w-full min-w-[1260px] border-collapse text-xs">
          <thead>
            <tr className="h-11 border-b border-[#E2E8F0] bg-[#F8FAFC] text-[#64748B]">
              <th className="w-[220px] px-4 text-left text-[12px] font-medium">企业名称</th>
              <th className="w-[70px] text-center text-[12px] font-medium">行业</th>
              {headerColumns.map((column) => (
                <th
                  key={column.key}
                  className="text-[12px] font-medium"
                  style={{ width: `${column.width}px`, textAlign: column.align ?? 'center' }}
                >
                  {column.title}
                </th>
              ))}
            </tr>
          </thead>

          <tbody>
            {loading && (
              <tr>
                <td colSpan={headerColumns.length + 2} className="h-16 text-center text-[#94A3B8]">
                  加载中...
                </td>
              </tr>
            )}

            {!loading && total === 0 && (
              <tr>
                <td colSpan={headerColumns.length + 2} className="h-16 text-center text-[#94A3B8]">
                  暂无数据
                </td>
              </tr>
            )}

            {!loading && total > 0 && items.length === 0 && (
              <tr>
                <td colSpan={headerColumns.length + 2} className="h-16 text-center text-[#94A3B8]">
                  当前页无数据
                </td>
              </tr>
            )}

            {!loading &&
              items.map((row) => (
                <tr key={row.id} className="h-12 border-b border-[#F1F5F9] text-[#0F172A] hover:bg-[#F8FAFC]">
                  <td className="px-4">
                    <div className="truncate text-[12px] font-medium">{row.name}</div>
                    <div className="mt-0.5 truncate font-mono text-[10px] text-[#94A3B8]">{row.creditCode || '-'}</div>
                    <span className="mt-1 inline-flex h-5 items-center rounded-md bg-[#EFF6FF] px-1.5 text-[10px] text-[#2563EB]">
                      {industryLabel(row.industryType)}
                    </span>
                  </td>
                  <td className="text-center text-[12px]">{industryLabel(row.industryType)}</td>

                  {headerColumns.map((column) => {
                    const cellKey = `${row.id}|${column.key}`
                    const active = !!highlightCells?.[cellKey]
                    const editableField = resolveEditableField(row, column.key)
                    const rawValue = (row as unknown as Record<string, number | undefined>)[column.key]
                    const showRate = column.key === 'salesMonthRate' || column.key === 'salesCumulativeRate'

                    return (
                      <td
                        key={column.key}
                        onClick={(event) => {
                          event.stopPropagation()
                          onCellPreview?.(row.id, column.key)
                        }}
                        className={`px-2 ${active ? 'bg-[#FEF3C7] ring-1 ring-inset ring-yellow-400/80' : ''}`}
                        style={{ textAlign: column.align ?? 'center' }}
                      >
                        {column.key === 'companyScale' && <span>{row.companyScale ?? '-'}</span>}
                        {column.key === 'flags' && <span>{flagText(row)}</span>}
                        {column.key === 'sourceSheet' && <span>{row.sourceSheet || '-'}</span>}

                        {column.key !== 'companyScale' && column.key !== 'flags' && column.key !== 'sourceSheet' && editableField && (
                          <Input
                            defaultValue={showRate ? formatRate(rawValue) : formatNumber(rawValue)}
                            onBlur={(event) =>
                              updateField(row, column.key, event.target.value.replace('%', '').replace('+', ''))
                            }
                            className="h-7 w-full rounded border border-transparent bg-transparent px-1 text-right font-mono tabular-nums outline-none hover:border-[#E2E8F0] focus:border-[#3B82F6]"
                          />
                        )}

                        {column.key !== 'companyScale' &&
                          column.key !== 'flags' &&
                          column.key !== 'sourceSheet' &&
                          !editableField && (
                            <span className="font-mono tabular-nums">{showRate ? formatRate(rawValue) : formatNumber(rawValue)}</span>
                          )}
                      </td>
                    )
                  })}
                </tr>
              ))}
          </tbody>
        </table>
      </div>

      <div className="flex h-11 items-center justify-between border-t border-[#E2E8F0] bg-[#F8FAFC] px-4">
        <span className="text-xs text-[#64748B]">{pageText}</span>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            className="h-7 px-2 text-xs"
            disabled={page <= 1 || loading}
            onClick={() => setPage((prev) => Math.max(1, prev - 1))}
          >
            上一页
          </Button>
          <Button
            variant="outline"
            className="h-7 px-2 text-xs"
            disabled={page >= totalPages || loading}
            onClick={() => setPage((prev) => Math.min(totalPages, prev + 1))}
          >
            下一页
          </Button>
        </div>
      </div>
    </section>
  )
}
