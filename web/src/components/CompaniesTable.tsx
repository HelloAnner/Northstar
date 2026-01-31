import { useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Loader2, Save, Search } from 'lucide-react'

export interface IndicatorGroup {
  name: string
  indicators: { id: string; name: string; value: number; unit: string }[]
}

interface CompanyRow {
  id: string
  kind: 'wr' | 'ac'
  creditCode?: string
  name: string
  industryCode?: string
  industryType?: string
  companyScale?: number
  isSmallMicro?: number
  isEatWearUse?: number
  sourceSheet?: string

  salesPrevMonth?: number
  salesCurrentMonth?: number
  salesLastYearMonth?: number
  salesCurrentCumulative?: number
  salesLastYearCumulative?: number
  salesMonthRate?: number | null
  salesCumulativeRate?: number | null

  retailPrevMonth?: number
  retailCurrentMonth?: number
  retailLastYearMonth?: number
  retailCurrentCumulative?: number
  retailLastYearCumulative?: number
  retailMonthRate?: number | null
  retailCumulativeRate?: number | null
  retailRatio?: number | null

  revenuePrevMonth?: number
  revenueCurrentMonth?: number
  revenueLastYearMonth?: number
  revenueCurrentCumulative?: number
  revenueLastYearCumulative?: number
  revenueMonthRate?: number | null
  revenueCumulativeRate?: number | null

  roomPrevMonth?: number
  roomCurrentMonth?: number
  roomLastYearMonth?: number
  roomCurrentCumulative?: number
  roomLastYearCumulative?: number

  foodPrevMonth?: number
  foodCurrentMonth?: number
  foodLastYearMonth?: number
  foodCurrentCumulative?: number
  foodLastYearCumulative?: number

  goodsPrevMonth?: number
  goodsCurrentMonth?: number
  goodsLastYearMonth?: number
  goodsCurrentCumulative?: number
  goodsLastYearCumulative?: number
}

type EditableField =
  | 'salesCurrentMonth'
  | 'salesLastYearMonth'
  | 'salesCurrentCumulative'
  | 'salesLastYearCumulative'
  | 'retailCurrentMonth'
  | 'retailLastYearMonth'
  | 'retailCurrentCumulative'
  | 'retailLastYearCumulative'
  | 'salesMonthRate'
  | 'salesCumulativeRate'
  | 'retailMonthRate'
  | 'retailCumulativeRate'
  | 'revenueCurrentMonth'
  | 'revenueLastYearMonth'
  | 'revenueCurrentCumulative'
  | 'revenueLastYearCumulative'
  | 'revenueMonthRate'
  | 'revenueCumulativeRate'

type ColumnKey =
  | 'companyScale'
  | 'flags'
  | 'salesPrevMonth'
  | 'salesCurrentMonth'
  | 'salesLastYearMonth'
  | 'salesYoYDiff'
  | 'salesMoMDiff'
  | 'salesMoMRate'
  | 'salesMonthRate'
  | 'salesCurrentCumulative'
  | 'salesLastYearCumulative'
  | 'salesCumulativeYoYDiff'
  | 'salesCumulativeRate'
  | 'retailPrevMonth'
  | 'retailCurrentMonth'
  | 'retailLastYearMonth'
  | 'retailYoYDiff'
  | 'retailMoMDiff'
  | 'retailMoMRate'
  | 'retailMonthRate'
  | 'retailCurrentCumulative'
  | 'retailLastYearCumulative'
  | 'retailCumulativeYoYDiff'
  | 'retailCumulativeRate'
  | 'retailRatio'
  | 'sourceSheet'

interface ColumnDef {
  key: ColumnKey
  label: string
  widthClass?: string
  align?: 'left' | 'right' | 'center'
  kind?: 'wr' | 'ac' | 'both'
  editable?: boolean
}

const ALL_COLUMNS: ColumnDef[] = [
  { key: 'companyScale', label: '规模', widthClass: 'w-[72px]', align: 'center' },
  { key: 'flags', label: '标记', widthClass: 'w-[120px]' },

  { key: 'salesPrevMonth', label: '本年-上月', widthClass: 'w-[150px]', align: 'right', kind: 'both' },
  { key: 'salesCurrentMonth', label: '本年-本月', widthClass: 'w-[160px]', align: 'right', kind: 'both', editable: true },
  { key: 'salesLastYearMonth', label: '上年-本月', widthClass: 'w-[160px]', align: 'right', kind: 'both', editable: true },
  { key: 'salesYoYDiff', label: '同比增量(当月)', widthClass: 'w-[160px]', align: 'right', kind: 'both' },
  { key: 'salesMoMDiff', label: '环比增量(当月)', widthClass: 'w-[160px]', align: 'right', kind: 'both' },
  { key: 'salesMoMRate', label: '环比增速(当月)', widthClass: 'w-[160px]', align: 'right', kind: 'both' },
  { key: 'salesMonthRate', label: '同比增速(当月)', widthClass: 'w-[160px]', align: 'right', kind: 'both', editable: true },
  { key: 'salesCurrentCumulative', label: '本年-1—本月', widthClass: 'w-[170px]', align: 'right', kind: 'both', editable: true },
  { key: 'salesLastYearCumulative', label: '上年-1—本月', widthClass: 'w-[170px]', align: 'right', kind: 'both', editable: true },
  { key: 'salesCumulativeYoYDiff', label: '累计同比增量', widthClass: 'w-[160px]', align: 'right', kind: 'both' },
  { key: 'salesCumulativeRate', label: '累计同比增速', widthClass: 'w-[160px]', align: 'right', kind: 'both', editable: true },

  { key: 'retailPrevMonth', label: '零售额;本年-上月', widthClass: 'w-[150px]', align: 'right', kind: 'wr' },
  { key: 'retailCurrentMonth', label: '零售额;本年-本月', widthClass: 'w-[160px]', align: 'right', kind: 'both', editable: true },
  { key: 'retailLastYearMonth', label: '零售额;上年-本月', widthClass: 'w-[160px]', align: 'right', kind: 'both', editable: true },
  { key: 'retailYoYDiff', label: '零售额;同比增量(当月)', widthClass: 'w-[170px]', align: 'right', kind: 'both' },
  { key: 'retailMoMDiff', label: '零售额;环比增量(当月)', widthClass: 'w-[170px]', align: 'right', kind: 'both' },
  { key: 'retailMoMRate', label: '零售额;环比增速(当月)', widthClass: 'w-[170px]', align: 'right', kind: 'both' },
  { key: 'retailMonthRate', label: '零售额;同比增速(当月)', widthClass: 'w-[170px]', align: 'right', kind: 'wr', editable: true },
  { key: 'retailCurrentCumulative', label: '零售额;本年-1—本月', widthClass: 'w-[170px]', align: 'right', kind: 'wr', editable: true },
  { key: 'retailLastYearCumulative', label: '零售额;上年-1—本月', widthClass: 'w-[170px]', align: 'right', kind: 'wr', editable: true },
  { key: 'retailCumulativeYoYDiff', label: '零售额;累计同比增量', widthClass: 'w-[170px]', align: 'right', kind: 'wr' },
  { key: 'retailCumulativeRate', label: '零售额;累计同比增速', widthClass: 'w-[170px]', align: 'right', kind: 'wr', editable: true },
  { key: 'retailRatio', label: '零销比(%)', widthClass: 'w-[110px]', align: 'right', kind: 'wr' },

  { key: 'sourceSheet', label: '来源表', widthClass: 'w-[120px]' },
]

export default function CompaniesTable(props: {
  onIndicatorsUpdate: (groups: IndicatorGroup[]) => void
  onSavingChange: (saving: boolean) => void
  monthSelector?: ReactNode
  reloadToken: number
}) {
  const [loading, setLoading] = useState(false)
  const [items, setItems] = useState<CompanyRow[]>([])
  const [total, setTotal] = useState(0)

  const [industryType, setIndustryType] = useState<'all' | 'wholesale' | 'retail' | 'accommodation' | 'catering'>(
    'all'
  )
  const [keyword, setKeyword] = useState('')

  const searchTimer = useRef<number | null>(null)
  const [savingCount, setSavingCount] = useState(0)

  useEffect(() => {
    props.onSavingChange(savingCount > 0)
  }, [savingCount, props])

  const columns = useMemo(() => ALL_COLUMNS, [])

  const load = async (opts?: { keyword?: string; industryType?: string }) => {
    setLoading(true)
    try {
      const q = new URLSearchParams()
      q.set('page', '1')
      q.set('pageSize', '2000')
      const it = opts?.industryType ?? industryType
      const kw = opts?.keyword ?? keyword
      if (it && it !== 'all') q.set('industryType', it)
      if (kw) q.set('keyword', kw)

      const res = await fetch(`/api/companies?${q.toString()}`)
      if (!res.ok) throw new Error('加载企业数据失败')
      const data = (await res.json()) as { items: CompanyRow[]; total: number }
      setItems(data.items || [])
      setTotal(data.total || 0)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    load()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [industryType, props.reloadToken])

  const handleKeywordChange = (v: string) => {
    setKeyword(v)
    if (searchTimer.current) {
      window.clearTimeout(searchTimer.current)
    }
    searchTimer.current = window.setTimeout(() => {
      load({ keyword: v })
    }, 250)
  }

  const updateCompany = async (id: string, patch: Partial<Record<EditableField, number>>) => {
    setSavingCount((c) => c + 1)
    try {
      const res = await fetch(`/api/companies/${encodeURIComponent(id)}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(patch),
      })
      if (!res.ok) throw new Error('保存失败')
      const data = (await res.json()) as { company: CompanyRow; groups: IndicatorGroup[] }
      setItems((prev) => prev.map((x) => (x.id === id ? { ...x, ...data.company } : x)))
      if (Array.isArray(data.groups)) {
        props.onIndicatorsUpdate(data.groups)
      }
    } finally {
      setSavingCount((c) => Math.max(0, c - 1))
    }
  }

  return (
    <Card className="border-border/60 bg-card/60 backdrop-blur supports-[backdrop-filter]:bg-card/50">
      <CardHeader className="space-y-3">
        <div className="flex items-center justify-between gap-3">
          <div>
            <CardTitle className="text-base">企业数据微调</CardTitle>
            <p className="mt-1 text-xs text-muted-foreground">
              支持搜索 / 筛选 / 排序；修改后自动保存并联动更新 16 项指标
            </p>
          </div>
          <div className="flex items-center gap-2">
            <Badge variant="secondary" className="hidden sm:inline-flex">
              <Save className="mr-1 h-3.5 w-3.5" />
              {savingCount > 0 ? `自动保存中 · ${savingCount}` : '自动保存'}
            </Badge>
          </div>
        </div>

        <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
          <Tabs value={industryType} onValueChange={(v) => setIndustryType(v as any)}>
            <TabsList className="w-full lg:w-auto">
              <TabsTrigger value="all">全部</TabsTrigger>
              <TabsTrigger value="wholesale">批发</TabsTrigger>
              <TabsTrigger value="retail">零售</TabsTrigger>
              <TabsTrigger value="accommodation">住宿</TabsTrigger>
              <TabsTrigger value="catering">餐饮</TabsTrigger>
            </TabsList>
          </Tabs>

          <div className="flex flex-1 items-center gap-2 lg:justify-end">
            {props.monthSelector}
            <div className="relative w-full lg:max-w-[360px]">
              <Search className="pointer-events-none absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
              <Input
                value={keyword}
                onChange={(e) => handleKeywordChange(e.target.value)}
                placeholder="按企业名称/信用代码搜索…"
                className="pl-9"
              />
            </div>
          </div>
        </div>
      </CardHeader>

      <CardContent className="pt-0">
        <div className="rounded-lg border border-border/60">
          <ScrollArea className="h-[520px] w-full">
            <div className="min-w-[1200px]">
              <Table>
                <TableHeader className="bg-card/90">
                  <TableRow>
                    <TableHead className="sticky top-0 z-20 w-[260px] whitespace-nowrap bg-card/90 text-center backdrop-blur">
                      企业名称
                    </TableHead>
                    {columns.map((col) => (
                      <TableHead
                        key={col.key}
                        className={`sticky top-0 z-20 bg-card/90 whitespace-nowrap backdrop-blur ${col.widthClass ?? ''} ${
                          col.align === 'right' ? 'text-right' : col.align === 'left' ? 'text-left' : 'text-center'
                        }`}
                        style={{
                          textAlign: col.align ?? 'center',
                        }}
                      >
                        {col.label}
                      </TableHead>
                    ))}
                  </TableRow>
                </TableHeader>

                <TableBody>
                  {loading && (
                    <TableRow>
                      <TableCell colSpan={columns.length + 1} className="py-10 text-center text-muted-foreground">
                        <Loader2 className="mr-2 inline-block h-4 w-4 animate-spin" />
                        加载中…
                      </TableCell>
                    </TableRow>
                  )}

                  {!loading && items.length === 0 && (
                    <TableRow>
                      <TableCell colSpan={columns.length + 1} className="py-10 text-center text-muted-foreground">
                        暂无数据
                      </TableCell>
                    </TableRow>
                  )}

                  {!loading &&
                    items.map((row) => (
                      <TableRow key={row.id} className="hover:bg-muted/30">
                        <TableCell className="w-[260px] text-center">
                          <div className="space-y-1 text-center">
                            <div className="flex items-center justify-center gap-2">
                              <div className="truncate font-medium">{row.name}</div>
                              {industryType === 'all' && <IndustryBadge type={row.industryType} />}
                            </div>
                            <div className="truncate font-mono text-xs text-muted-foreground">{row.creditCode || '-'}</div>
                          </div>
                        </TableCell>

                        {columns.map((col) => (
                          <TableCell
                            key={col.key}
                            className={`align-middle whitespace-nowrap ${
                              col.align === 'right' ? 'text-right' : col.align === 'left' ? 'text-left' : 'text-center'
                            }`}
                            style={{ textAlign: col.align ?? 'center' }}
                          >
                            <Cell
                              row={row}
                              col={col}
                              onUpdate={(field, value) => updateCompany(row.id, { [field]: value } as any)}
                            />
                          </TableCell>
                        ))}
                      </TableRow>
                    ))}
                </TableBody>
              </Table>
            </div>
          </ScrollArea>
        </div>

        <div className="mt-3 flex items-center justify-between text-xs text-muted-foreground">
          <span>共 {total} 家企业</span>
          <Button variant="ghost" size="sm" onClick={() => load()} className="h-8 px-2">
            刷新
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}

function IndustryBadge({ type }: { type?: string }) {
  const label = (() => {
    switch (type) {
      case 'wholesale':
        return '批发'
      case 'retail':
        return '零售'
      case 'accommodation':
        return '住宿'
      case 'catering':
        return '餐饮'
      default:
        return '未知'
    }
  })()

  const variant = type === 'unknown' ? 'outline' : 'secondary'
  return (
    <Badge variant={variant} className="h-5 px-2 text-[11px] font-medium">
      {label}
    </Badge>
  )
}

function Cell(props: { row: CompanyRow; col: ColumnDef; onUpdate: (field: EditableField, value: number) => void }) {
  const { row, col } = props
  const key = col.key

  if (key === 'companyScale') {
    return <span className="text-sm">{row.companyScale ?? '-'}</span>
  }
  if (key === 'flags') {
    return (
      <div className="flex flex-wrap justify-center gap-1">
        {row.isSmallMicro === 1 && (
          <Badge variant="secondary" className="h-5 px-2 text-[11px]">
            小微
          </Badge>
        )}
        {row.isEatWearUse === 1 && (
          <Badge variant="secondary" className="h-5 px-2 text-[11px]">
            吃穿用
          </Badge>
        )}
        {row.isSmallMicro !== 1 && row.isEatWearUse !== 1 && <span className="text-muted-foreground">-</span>}
      </div>
    )
  }
  if (key === 'sourceSheet') {
    return <span className="text-xs text-muted-foreground">{row.sourceSheet || '-'}</span>
  }

  const editable = col.editable && isEditableForRow(row, key)
  if (editable) {
    const field = resolveEditableField(row, key)
    const current = (row as any)[field] as number | undefined
    return <EditableNumber value={current} onCommit={(v) => props.onUpdate(field, v)} />
  }

  const calcRatePercent = (cur?: number | null, last?: number | null) => {
    if (cur === null || cur === undefined) return null
    if (last === null || last === undefined) return null
    if (last === 0) return -100
    return (cur / last - 1) * 100
  }

  const calcDiff = (cur?: number | null, base?: number | null) => {
    if (cur === null || cur === undefined) return null
    if (base === null || base === undefined) return null
    return cur - base
  }

  const salesPrev = row.kind === 'ac' ? row.revenuePrevMonth : row.salesPrevMonth
  const salesCur = row.kind === 'ac' ? row.revenueCurrentMonth : row.salesCurrentMonth
  const salesLast = row.kind === 'ac' ? row.revenueLastYearMonth : row.salesLastYearMonth
  const salesCurCum = row.kind === 'ac' ? row.revenueCurrentCumulative : row.salesCurrentCumulative
  const salesLastCum = row.kind === 'ac' ? row.revenueLastYearCumulative : row.salesLastYearCumulative

  if (key === 'salesYoYDiff') {
    const v = calcDiff(salesCur, salesLast)
    return v === null ? <span className="text-muted-foreground">-</span> : <NumberValue value={v} />
  }
  if (key === 'salesMoMDiff') {
    const v = calcDiff(salesCur, salesPrev)
    return v === null ? <span className="text-muted-foreground">-</span> : <NumberValue value={v} />
  }
  if (key === 'salesMoMRate') {
    const v = calcRatePercent(salesCur, salesPrev)
    return v === null ? <span className="text-muted-foreground">-</span> : <RateValue value={v} />
  }
  if (key === 'salesCumulativeYoYDiff') {
    const v = calcDiff(salesCurCum, salesLastCum)
    return v === null ? <span className="text-muted-foreground">-</span> : <NumberValue value={v} />
  }

  if (key === 'retailYoYDiff') {
    const v = calcDiff(row.retailCurrentMonth, row.retailLastYearMonth)
    return v === null ? <span className="text-muted-foreground">-</span> : <NumberValue value={v} />
  }
  if (key === 'retailMoMDiff') {
    const v = calcDiff(row.retailCurrentMonth, row.retailPrevMonth)
    return v === null ? <span className="text-muted-foreground">-</span> : <NumberValue value={v} />
  }
  if (key === 'retailMoMRate') {
    const v = calcRatePercent(row.retailCurrentMonth, row.retailPrevMonth)
    return v === null ? <span className="text-muted-foreground">-</span> : <RateValue value={v} />
  }
  if (key === 'retailCumulativeYoYDiff') {
    const v = calcDiff(row.retailCurrentCumulative, row.retailLastYearCumulative)
    return v === null ? <span className="text-muted-foreground">-</span> : <NumberValue value={v} />
  }

  let v = (row as any)[key] as number | null | undefined
  if ((v === null || v === undefined) && row.kind === 'ac') {
    // “销售额”列在住餐口径下对应 “营业额”
    if (key === 'salesPrevMonth') {
      v = row.revenuePrevMonth
    }
    if (key === 'salesCurrentMonth') {
      v = row.revenueCurrentMonth
    }
    if (key === 'salesLastYearMonth') {
      v = row.revenueLastYearMonth
    }
    if (key === 'salesCurrentCumulative') {
      v = row.revenueCurrentCumulative
    }
    if (key === 'salesLastYearCumulative') {
      v = row.revenueLastYearCumulative
    }
    if (key === 'salesMonthRate') {
      v = row.revenueMonthRate
    }
    if (key === 'salesCumulativeRate') {
      v = row.revenueCumulativeRate
    }
  }
  if (v === null || v === undefined) {
    return <span className="text-muted-foreground">-</span>
  }
  if (String(key).toLowerCase().includes('rate') || key === 'retailRatio') {
    return <RateValue value={v} />
  }
  return <NumberValue value={v} />
}

function isEditableForRow(row: CompanyRow, key: ColumnKey) {
  const allowWR: Partial<Record<ColumnKey, boolean>> = {
    salesCurrentMonth: true,
    salesLastYearMonth: true,
    salesCurrentCumulative: true,
    salesLastYearCumulative: true,
    salesMonthRate: true,
    salesCumulativeRate: true,
    retailCurrentMonth: true,
    retailLastYearMonth: true,
    retailMonthRate: row.kind === 'wr',
    retailCurrentCumulative: row.kind === 'wr',
    retailLastYearCumulative: row.kind === 'wr',
    retailCumulativeRate: row.kind === 'wr',
  }
  const allowAC: Partial<Record<ColumnKey, boolean>> = {
    retailCurrentMonth: true,
    retailLastYearMonth: true,
  }
  return Boolean(allowWR[key] || allowAC[key])
}

function resolveEditableField(row: CompanyRow, key: ColumnKey): EditableField {
  if (row.kind !== 'ac') {
    return key as EditableField
  }
  switch (key) {
    case 'salesCurrentMonth':
      return 'revenueCurrentMonth'
    case 'salesLastYearMonth':
      return 'revenueLastYearMonth'
    case 'salesCurrentCumulative':
      return 'revenueCurrentCumulative'
    case 'salesLastYearCumulative':
      return 'revenueLastYearCumulative'
    case 'salesMonthRate':
      return 'revenueMonthRate'
    case 'salesCumulativeRate':
      return 'revenueCumulativeRate'
    default:
      return key as EditableField
  }
}

function EditableNumber(props: { value?: number; onCommit: (v: number) => void }) {
  const [draft, setDraft] = useState(() => (props.value ?? 0).toString())
  const [busy, setBusy] = useState(false)
  const lastValue = useRef<number | undefined>(props.value)

  useEffect(() => {
    if (busy) return
    if (props.value !== lastValue.current) {
      lastValue.current = props.value
      setDraft((props.value ?? 0).toString())
    }
  }, [props.value, busy])

  const commit = async () => {
    const parsed = Number(draft)
    if (!Number.isFinite(parsed)) {
      setDraft((props.value ?? 0).toString())
      return
    }
    const v = Math.round(parsed)
    if (props.value === v) {
      if (draft !== String(v)) {
        setDraft(String(v))
      }
      return
    }
    setBusy(true)
    try {
      setDraft(String(v))
      await Promise.resolve(props.onCommit(v))
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="flex items-center justify-center gap-2">
      <Input
        value={draft}
        onChange={(e) => setDraft(e.target.value)}
        onBlur={commit}
        onKeyDown={(e) => {
          if (e.key === 'Enter') {
            e.currentTarget.blur()
          }
        }}
        className="h-8 w-[120px] bg-muted/20 text-center"
      />
      {busy && <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />}
    </div>
  )
}

function NumberValue({ value }: { value: number }) {
  const s = Math.round(value).toLocaleString()
  return <span className="font-mono text-sm tabular-nums">{s}</span>
}

function RateValue({ value }: { value: number }) {
  const positive = value >= 0
  const s = `${Math.round(value)}%`
  return (
    <span className={`font-mono text-sm tabular-nums ${positive ? 'text-emerald-400' : 'text-rose-400'}`}>
      {s}
    </span>
  )
}
