/**
 * 规则列表 — 硬约束 + 自然语言规则（支持搜索和分页）
 *
 * @author Anner
 * Created on 2026/3/14
 */

import { useEffect, useMemo, useState } from 'react'
import { Plus, Pencil, Trash2, ChevronLeft, ChevronRight, Search } from 'lucide-react'
import { toast } from 'sonner'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'
import { useRulesStore } from '@/store/rulesStore'
import type { AdjustmentConstraint, NaturalRule } from '@/services/api'

const PAGE_SIZE = 6

// --- 指标选项 ---

const INDICATOR_OPTIONS = [
  { value: 'limitAbove_month_value', label: '限上社零额月值' },
  { value: 'limitAbove_month_rate', label: '限上社零额月增速' },
  { value: 'limitAbove_cumulative_value', label: '限上社零额累计值' },
  { value: 'limitAbove_cumulative_rate', label: '限上社零额累计增速' },
  { value: 'eatWearUse_month_rate', label: '吃穿用月增速' },
  { value: 'microSmall_month_rate', label: '小微企业月增速' },
  { value: 'wholesale_month_rate', label: '批发业月增速' },
  { value: 'wholesale_cumulative_rate', label: '批发业累计增速' },
  { value: 'retail_month_rate', label: '零售业月增速' },
  { value: 'retail_cumulative_rate', label: '零售业累计增速' },
  { value: 'accommodation_month_rate', label: '住宿业月增速' },
  { value: 'accommodation_cumulative_rate', label: '住宿业累计增速' },
  { value: 'catering_month_rate', label: '餐饮业月增速' },
  { value: 'catering_cumulative_rate', label: '餐饮业累计增速' },
  { value: 'totalSocial_cumulative_value', label: '社零总额累计值' },
  { value: 'totalSocial_cumulative_rate', label: '社零总额累计增速' },
]

const FILTER_OPTIONS = [
  { value: 'positive_current', label: '仅正增长企业' },
  { value: 'negative_current', label: '仅负增长企业' },
  { value: 'large_scale_only', label: '仅大型企业' },
  { value: 'exclude_small_micro', label: '排除小微企业' },
]

const CONSTRAINT_TYPE_LABELS: Record<string, string> = {
  clamp_target: '值域约束',
  filter_allocation: '过滤分配',
  compensate: '联动补偿',
}

// --- 通用分页组件 ---

function Pagination({ current, total, onChange }: { current: number; total: number; onChange: (p: number) => void }) {
  if (total <= 1) return null
  return (
    <div className="flex items-center justify-center gap-2 pt-2 text-sm text-muted-foreground">
      <Button variant="outline" size="sm" disabled={current <= 1} onClick={() => onChange(current - 1)}>
        <ChevronLeft className="h-3.5 w-3.5" />
      </Button>
      <span>{current} / {total}</span>
      <Button variant="outline" size="sm" disabled={current >= total} onClick={() => onChange(current + 1)}>
        <ChevronRight className="h-3.5 w-3.5" />
      </Button>
    </div>
  )
}

// --- 搜索匹配 ---

function matchConstraint(c: AdjustmentConstraint, query: string): boolean {
  const q = query.toLowerCase()
  const indicator = INDICATOR_OPTIONS.find((o) => o.value === c.indicatorId)?.label ?? c.indicatorId ?? ''
  const trigger = INDICATOR_OPTIONS.find((o) => o.value === c.triggerId)?.label ?? c.triggerId ?? ''
  const ensure = INDICATOR_OPTIONS.find((o) => o.value === c.ensureId)?.label ?? c.ensureId ?? ''
  const filter = FILTER_OPTIONS.find((o) => o.value === c.filterMode)?.label ?? c.filterMode ?? ''
  const typeLabel = CONSTRAINT_TYPE_LABELS[c.type] ?? c.type
  return [indicator, trigger, ensure, filter, typeLabel, c.indicatorId, c.triggerId, c.ensureId, c.filterMode, c.type]
    .filter(Boolean)
    .some((s) => s!.toLowerCase().includes(q))
}

function matchNaturalRule(r: NaturalRule, query: string): boolean {
  return r.text.toLowerCase().includes(query.toLowerCase())
}

// --- 硬约束区 ---

function ConstraintSection({
  constraints,
  submitting,
  onAdd,
  onUpdate,
  onDelete,
}: {
  constraints: AdjustmentConstraint[]
  submitting: boolean
  onAdd: (data: Omit<AdjustmentConstraint, 'id'>) => Promise<void>
  onUpdate: (id: number, data: Omit<AdjustmentConstraint, 'id'>) => Promise<void>
  onDelete: (id: number) => Promise<void>
}) {
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editing, setEditing] = useState<AdjustmentConstraint | null>(null)
  const [page, setPage] = useState(1)

  const totalPages = Math.max(1, Math.ceil(constraints.length / PAGE_SIZE))
  const paged = useMemo(() => {
    const start = (page - 1) * PAGE_SIZE
    return constraints.slice(start, start + PAGE_SIZE)
  }, [constraints, page])

  useEffect(() => {
    if (page > totalPages) setPage(totalPages)
  }, [totalPages, page])

  return (
    <>
      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <div className="space-y-1">
            <h3 className="text-sm font-medium">硬约束</h3>
            <p className="text-xs text-muted-foreground">
              确定性执行，每次调整自动生效。共 {constraints.length} 条。
            </p>
          </div>
          <Button size="sm" onClick={() => { setEditing(null); setDialogOpen(true) }} disabled={submitting}>
            <Plus className="h-3.5 w-3.5" />
            新增约束
          </Button>
        </div>

        {constraints.length === 0 && (
          <div className="rounded-lg border border-dashed px-4 py-6 text-center text-sm text-muted-foreground">
            暂无硬约束
          </div>
        )}

        <div className="space-y-2">
          {paged.map((item) => (
            <div
              key={item.id}
              className="flex items-center justify-between gap-4 rounded-lg border bg-muted/20 px-4 py-2.5"
            >
              <div className="min-w-0 flex items-center gap-2">
                <Badge variant="outline" className="text-[10px] shrink-0">
                  {CONSTRAINT_TYPE_LABELS[item.type] ?? item.type}
                </Badge>
                <span className="text-xs text-muted-foreground truncate">
                  {formatConstraintSummary(item)}
                </span>
              </div>
              <div className="flex shrink-0 items-center gap-1">
                <Button variant="ghost" size="sm" onClick={() => { setEditing(item); setDialogOpen(true) }} disabled={submitting}>
                  <Pencil className="h-3.5 w-3.5" />
                </Button>
                <Button variant="ghost" size="sm" onClick={() => void onDelete(item.id)} disabled={submitting}>
                  <Trash2 className="h-3.5 w-3.5" />
                </Button>
              </div>
            </div>
          ))}
        </div>

        <Pagination current={page} total={totalPages} onChange={setPage} />
      </div>

      <ConstraintDialog
        open={dialogOpen}
        editing={editing}
        submitting={submitting}
        onClose={() => { setDialogOpen(false); setEditing(null) }}
        onSave={async (data) => {
          if (editing) await onUpdate(editing.id, data)
          else await onAdd(data)
          setDialogOpen(false)
          setEditing(null)
        }}
      />
    </>
  )
}

function formatConstraintSummary(c: AdjustmentConstraint): string {
  const indicator = INDICATOR_OPTIONS.find((o) => o.value === c.indicatorId)?.label ?? c.indicatorId ?? ''
  if (c.type === 'clamp_target') {
    const parts: string[] = [indicator]
    if (c.minValue != null) parts.push(`≥ ${c.minValue}`)
    if (c.maxValue != null) parts.push(`≤ ${c.maxValue}`)
    return parts.join(' ')
  }
  if (c.type === 'filter_allocation') {
    const filter = FILTER_OPTIONS.find((o) => o.value === c.filterMode)?.label ?? c.filterMode
    return `${indicator} → ${filter}`
  }
  if (c.type === 'compensate') {
    const trigger = INDICATOR_OPTIONS.find((o) => o.value === c.triggerId)?.label ?? c.triggerId
    const ensure = INDICATOR_OPTIONS.find((o) => o.value === c.ensureId)?.label ?? c.ensureId
    return `${trigger} → ${ensure} (${c.relation}, ±${c.tolerance})`
  }
  return ''
}

// --- 约束编辑弹窗 ---

function ConstraintDialog({
  open,
  editing,
  submitting,
  onClose,
  onSave,
}: {
  open: boolean
  editing: AdjustmentConstraint | null
  submitting: boolean
  onClose: () => void
  onSave: (data: Omit<AdjustmentConstraint, 'id'>) => Promise<void>
}) {
  const [type, setType] = useState<AdjustmentConstraint['type']>('clamp_target')
  const [indicatorId, setIndicatorId] = useState('')
  const [minValue, setMinValue] = useState('')
  const [maxValue, setMaxValue] = useState('')
  const [filterMode, setFilterMode] = useState('')
  const [triggerId, setTriggerId] = useState('')
  const [ensureId, setEnsureId] = useState('')
  const [relation, setRelation] = useState('gte')
  const [tolerance, setTolerance] = useState('0')

  useEffect(() => {
    if (editing) {
      setType(editing.type)
      setIndicatorId(editing.indicatorId ?? '')
      setMinValue(editing.minValue != null ? String(editing.minValue) : '')
      setMaxValue(editing.maxValue != null ? String(editing.maxValue) : '')
      setFilterMode(editing.filterMode ?? '')
      setTriggerId(editing.triggerId ?? '')
      setEnsureId(editing.ensureId ?? '')
      setRelation(editing.relation ?? 'gte')
      setTolerance(String(editing.tolerance ?? 0))
    } else {
      setType('clamp_target')
      setIndicatorId('')
      setMinValue('')
      setMaxValue('')
      setFilterMode('')
      setTriggerId('')
      setEnsureId('')
      setRelation('gte')
      setTolerance('0')
    }
  }, [editing, open])

  const handleSave = async () => {
    const data: Omit<AdjustmentConstraint, 'id'> = {
      type,
      indicatorId: type !== 'compensate' ? indicatorId : undefined,
      minValue: type === 'clamp_target' && minValue !== '' ? Number(minValue) : null,
      maxValue: type === 'clamp_target' && maxValue !== '' ? Number(maxValue) : null,
      filterMode: type === 'filter_allocation' ? filterMode : undefined,
      triggerId: type === 'compensate' ? triggerId : undefined,
      ensureId: type === 'compensate' ? ensureId : undefined,
      relation: type === 'compensate' ? relation : undefined,
      tolerance: type === 'compensate' ? Number(tolerance) : 0,
      enabled: true,
    }
    try {
      await onSave(data)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '保存失败')
    }
  }

  return (
    <Dialog open={open} onOpenChange={(v) => !v && onClose()}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>{editing ? '编辑约束' : '新增约束'}</DialogTitle>
          <DialogDescription>硬约束在每次数据调整时自动执行。</DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="space-y-2">
            <Label>约束类型</Label>
            <Select value={type} onValueChange={(v) => setType(v as AdjustmentConstraint['type'])}>
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem value="clamp_target">值域约束（Clamp）</SelectItem>
                <SelectItem value="filter_allocation">过滤分配（Filter）</SelectItem>
                <SelectItem value="compensate">联动补偿（Compensate）</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {type === 'clamp_target' && (
            <>
              <div className="space-y-2">
                <Label>指标</Label>
                <Select value={indicatorId} onValueChange={setIndicatorId}>
                  <SelectTrigger><SelectValue placeholder="选择指标" /></SelectTrigger>
                  <SelectContent>
                    {INDICATOR_OPTIONS.map((o) => (
                      <SelectItem key={o.value} value={o.value}>{o.label}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-2">
                  <Label>最小值</Label>
                  <Input type="number" value={minValue} onChange={(e) => setMinValue(e.target.value)} placeholder="不限" />
                </div>
                <div className="space-y-2">
                  <Label>最大值</Label>
                  <Input type="number" value={maxValue} onChange={(e) => setMaxValue(e.target.value)} placeholder="不限" />
                </div>
              </div>
            </>
          )}

          {type === 'filter_allocation' && (
            <>
              <div className="space-y-2">
                <Label>指标</Label>
                <Select value={indicatorId} onValueChange={setIndicatorId}>
                  <SelectTrigger><SelectValue placeholder="选择指标" /></SelectTrigger>
                  <SelectContent>
                    {INDICATOR_OPTIONS.map((o) => (
                      <SelectItem key={o.value} value={o.value}>{o.label}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label>过滤模式</Label>
                <Select value={filterMode} onValueChange={setFilterMode}>
                  <SelectTrigger><SelectValue placeholder="选择过滤模式" /></SelectTrigger>
                  <SelectContent>
                    {FILTER_OPTIONS.map((o) => (
                      <SelectItem key={o.value} value={o.value}>{o.label}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            </>
          )}

          {type === 'compensate' && (
            <>
              <div className="space-y-2">
                <Label>触发指标</Label>
                <Select value={triggerId} onValueChange={setTriggerId}>
                  <SelectTrigger><SelectValue placeholder="选择触发指标" /></SelectTrigger>
                  <SelectContent>
                    {INDICATOR_OPTIONS.map((o) => (
                      <SelectItem key={o.value} value={o.value}>{o.label}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label>保障指标</Label>
                <Select value={ensureId} onValueChange={setEnsureId}>
                  <SelectTrigger><SelectValue placeholder="选择保障指标" /></SelectTrigger>
                  <SelectContent>
                    {INDICATOR_OPTIONS.map((o) => (
                      <SelectItem key={o.value} value={o.value}>{o.label}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-2">
                  <Label>关系</Label>
                  <Select value={relation} onValueChange={setRelation}>
                    <SelectTrigger><SelectValue /></SelectTrigger>
                    <SelectContent>
                      <SelectItem value="gte">≥（不低于）</SelectItem>
                      <SelectItem value="lte">≤（不高于）</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-2">
                  <Label>容差</Label>
                  <Input type="number" value={tolerance} onChange={(e) => setTolerance(e.target.value)} />
                </div>
              </div>
            </>
          )}
        </div>

        <DialogFooter className="gap-2">
          <Button variant="outline" onClick={onClose} disabled={submitting}>取消</Button>
          <Button onClick={() => void handleSave()} disabled={submitting}>
            {submitting ? '保存中…' : '保存'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

// --- 自然语言规则区 ---

function NaturalRuleSection({
  rules,
  submitting,
  onAdd,
  onUpdate,
  onDelete,
}: {
  rules: NaturalRule[]
  submitting: boolean
  onAdd: (text: string) => Promise<void>
  onUpdate: (id: number, text: string) => Promise<void>
  onDelete: (id: number) => Promise<void>
}) {
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editing, setEditing] = useState<NaturalRule | null>(null)
  const [draft, setDraft] = useState('')
  const [page, setPage] = useState(1)

  const totalPages = Math.max(1, Math.ceil(rules.length / PAGE_SIZE))
  const paged = useMemo(() => {
    const start = (page - 1) * PAGE_SIZE
    return rules.slice(start, start + PAGE_SIZE)
  }, [rules, page])

  useEffect(() => {
    if (page > totalPages) setPage(totalPages)
  }, [totalPages, page])

  const handleSave = async () => {
    const text = draft.trim()
    if (!text) { toast.error('规则内容不能为空'); return }
    try {
      if (editing) await onUpdate(editing.id, text)
      else await onAdd(text)
      setDialogOpen(false)
      setEditing(null)
      setDraft('')
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '保存失败')
    }
  }

  return (
    <>
      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <div className="space-y-1">
            <h3 className="text-sm font-medium">自然语言规则</h3>
            <p className="text-xs text-muted-foreground">
              作为 AI 对话上下文生效，由大模型理解和遵守。共 {rules.length} 条。
            </p>
          </div>
          <Button size="sm" onClick={() => { setEditing(null); setDraft(''); setDialogOpen(true) }} disabled={submitting}>
            <Plus className="h-3.5 w-3.5" />
            新增规则
          </Button>
        </div>

        {rules.length === 0 && (
          <div className="rounded-lg border border-dashed px-4 py-6 text-center text-sm text-muted-foreground">
            暂无自然语言规则
          </div>
        )}

        <div className="space-y-2">
          {paged.map((item) => (
            <div key={item.id} className="flex items-start justify-between gap-4 rounded-lg border bg-muted/20 px-4 py-2.5">
              <div className="min-w-0">
                <div className="text-sm leading-6 text-foreground">{item.text}</div>
              </div>
              <div className="flex shrink-0 items-center gap-1">
                <Button variant="ghost" size="sm" onClick={() => { setEditing(item); setDraft(item.text); setDialogOpen(true) }} disabled={submitting}>
                  <Pencil className="h-3.5 w-3.5" />
                </Button>
                <Button variant="ghost" size="sm" onClick={() => void onDelete(item.id)} disabled={submitting}>
                  <Trash2 className="h-3.5 w-3.5" />
                </Button>
              </div>
            </div>
          ))}
        </div>

        <Pagination current={page} total={totalPages} onChange={setPage} />
      </div>

      <Dialog open={dialogOpen} onOpenChange={(v) => { if (!v) { setDialogOpen(false); setEditing(null); setDraft('') } }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{editing ? '编辑规则' : '新增规则'}</DialogTitle>
            <DialogDescription>用自然语言描述规则，AI 在对话调整时会参考这些规则。</DialogDescription>
          </DialogHeader>
          <Textarea
            value={draft}
            onChange={(e) => setDraft(e.target.value)}
            placeholder="例如：调整零售业时，尽量避免过大的波动。"
            disabled={submitting}
          />
          <DialogFooter className="gap-2">
            <Button variant="outline" onClick={() => setDialogOpen(false)} disabled={submitting}>取消</Button>
            <Button onClick={() => void handleSave()} disabled={submitting}>
              {submitting ? '保存中…' : '保存'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}

// --- 主组件 ---

export default function RuleList() {
  const constraints = useRulesStore((s) => s.constraints)
  const naturalRules = useRulesStore((s) => s.naturalRules)
  const submitting = useRulesStore((s) => s.submitting)
  const loadConstraints = useRulesStore((s) => s.loadConstraints)
  const loadNaturalRules = useRulesStore((s) => s.loadNaturalRules)
  const addConstraint = useRulesStore((s) => s.addConstraint)
  const updateConstraint = useRulesStore((s) => s.updateConstraint)
  const deleteConstraint = useRulesStore((s) => s.deleteConstraint)
  const addNaturalRule = useRulesStore((s) => s.addNaturalRule)
  const updateNaturalRule = useRulesStore((s) => s.updateNaturalRule)
  const deleteNaturalRule = useRulesStore((s) => s.deleteNaturalRule)

  const [search, setSearch] = useState('')

  useEffect(() => {
    void loadConstraints()
    void loadNaturalRules()
  }, [loadConstraints, loadNaturalRules])

  const query = search.trim()
  const filteredConstraints = useMemo(
    () => query ? constraints.filter((c) => matchConstraint(c, query)) : constraints,
    [constraints, query],
  )
  const filteredNaturalRules = useMemo(
    () => query ? naturalRules.filter((r) => matchNaturalRule(r, query)) : naturalRules,
    [naturalRules, query],
  )

  return (
    <div data-testid="rule-list" className="space-y-6">
      {/* 搜索栏 */}
      <div className="relative">
        <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        <Input
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="搜索约束或规则…"
          className="pl-9"
        />
      </div>

      <ConstraintSection
        constraints={filteredConstraints}
        submitting={submitting}
        onAdd={addConstraint}
        onUpdate={updateConstraint}
        onDelete={deleteConstraint}
      />

      <div className="border-t" />

      <NaturalRuleSection
        rules={filteredNaturalRules}
        submitting={submitting}
        onAdd={addNaturalRule}
        onUpdate={updateNaturalRule}
        onDelete={deleteNaturalRule}
      />
    </div>
  )
}
