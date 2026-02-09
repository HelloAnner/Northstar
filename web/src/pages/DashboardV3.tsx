import { useEffect, useMemo, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { Download, Upload } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Dialog, DialogContent } from '@/components/ui/dialog'
import { toast } from 'sonner'
import ExportDialog from '@/components/ExportDialog'
import LlmChatDialog from '@/components/LlmChatDialog'
import IndicatorCards, { type IndicatorGroup } from '@/components/dashboard/IndicatorCards'
import EnterpriseDataTable from '@/components/dashboard/EnterpriseDataTable'
import ImportV3 from '@/pages/ImportV3'
import { buildCompanySnapshot, buildUndoChanges, buildUndoUpdates, type CompanySnapshot, type CompanySnapshotRow } from '@/lib/undo'
import { useUndoStore } from '@/store/undoStore'

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

interface OptimizeNotice {
  indicatorId: string
  indicatorName: string
  target: number
  before: number
  after: number
  code: string
  level: 'info' | 'warn' | 'error'
  message: string
  suggestion?: string
}

type LinkageAnchor =
  | { indicatorId: string }
  | { ui: { rowId: string; columnKey: string } }

interface RuleEvaluation {
  status: 'pass' | 'fail' | 'skipped'
  failedIndicators?: Array<{
    indicatorCode?: string
  }>
}

interface LinkageNode {
  indicatorId?: string
  ui?: {
    rowId?: string
    columnKey?: string
  }
}

const isRateGroup = (groupName: string) => {
  return groupName.includes('增速')
}

export default function DashboardV3() {
  const [searchParams, setSearchParams] = useSearchParams()

  const [status, setStatus] = useState<SystemStatus | null>(null)
  const [groups, setGroups] = useState<IndicatorGroup[]>([])
  const [months, setMonths] = useState<YearMonthStat[]>([])
  const [monthsLoading, setMonthsLoading] = useState(false)
  const [loading, setLoading] = useState(true)
  const [tableSaving, setTableSaving] = useState(false)
  const [optimizing, setOptimizing] = useState(false)
  const [undoing, setUndoing] = useState(false)
  const [reloadToken, setReloadToken] = useState(0)

  const [draftTargets, setDraftTargets] = useState<Record<string, string>>({})
  const [highlightCells, setHighlightCells] = useState<Record<string, boolean>>({})
  const [highlightIndicators, setHighlightIndicators] = useState<Record<string, boolean>>({})
  const [ruleFailedIndicators, setRuleFailedIndicators] = useState<Record<string, boolean>>({})

  const [showExportDialog, setShowExportDialog] = useState(false)
  const [showChatDialog, setShowChatDialog] = useState(false)
  const [showImportDialog, setShowImportDialog] = useState(false)
  const [autoImportPrompted, setAutoImportPrompted] = useState(false)

  const undoStack = useUndoStore((state) => state.stack)
  const pushUndoStep = useUndoStore((state) => state.push)
  const popUndoStep = useUndoStore((state) => state.pop)
  const clearUndo = useUndoStore((state) => state.clear)

  const hasDraft = Object.keys(draftTargets).length > 0
  const canUndo = undoStack.length > 0 && !undoing

  const saving = tableSaving || optimizing

  const fetchCompaniesSnapshot = async (): Promise<CompanySnapshot> => {
    const query = new URLSearchParams()
    query.set('page', '1')
    query.set('pageSize', '2000')
    const response = await fetch(`/api/companies?${query.toString()}`)
    if (!response.ok) {
      throw new Error('加载企业数据失败')
    }
    const data = (await response.json()) as { items?: CompanySnapshotRow[] }
    return buildCompanySnapshot(Array.isArray(data.items) ? data.items : [])
  }

  const showOptimizeNotices = (notices?: OptimizeNotice[]) => {
    if (!Array.isArray(notices) || notices.length === 0) {
      return
    }
    for (const notice of notices) {
      const title = notice.indicatorName ? `${notice.indicatorName}：${notice.message}` : notice.message || '智能调整提示'
      const payload = {
        description: notice.suggestion,
        duration: 3000,
      }
      if (notice.level === 'error') {
        toast.error(title, payload)
      } else if (notice.level === 'warn') {
        toast.warning(title, payload)
      } else {
        toast(title, payload)
      }
    }
  }

  const loadStatus = async () => {
    const response = await fetch('/api/status')
    const data = (await response.json()) as SystemStatus
    setStatus(data)
  }

  const loadIndicators = async () => {
    setLoading(true)
    try {
      const response = await fetch('/api/indicators')
      const data = (await response.json()) as { groups?: IndicatorGroup[] }
      setGroups(Array.isArray(data.groups) ? data.groups : [])
    } finally {
      setLoading(false)
    }
  }

  const loadRuleEvaluations = async () => {
    try {
      const response = await fetch('/api/rules/evaluate?enabledOnly=true')
      if (!response.ok) {
        throw new Error('加载规则校验失败')
      }
      const data = (await response.json()) as { items?: RuleEvaluation[] }
      const next: Record<string, boolean> = {}
      for (const item of Array.isArray(data.items) ? data.items : []) {
        if (item?.status !== 'fail' || !Array.isArray(item.failedIndicators)) {
          continue
        }
        for (const indicator of item.failedIndicators) {
          if (indicator?.indicatorCode) {
            next[String(indicator.indicatorCode)] = true
          }
        }
      }
      setRuleFailedIndicators(next)
    } catch {
      setRuleFailedIndicators({})
    }
  }

  const loadMonths = async () => {
    setMonthsLoading(true)
    try {
      const response = await fetch('/api/months')
      if (!response.ok) {
        throw new Error('加载月份失败')
      }
      const data = (await response.json()) as { items?: YearMonthStat[] }
      setMonths(Array.isArray(data.items) ? data.items : [])
    } catch {
      setMonths([])
    } finally {
      setMonthsLoading(false)
    }
  }

  useEffect(() => {
    Promise.all([loadStatus(), loadIndicators(), loadMonths(), loadRuleEvaluations()]).catch(() => {
      // ignore
    })
  }, [])

  useEffect(() => {
    const next = new URLSearchParams(searchParams)
    let changed = false

    if (searchParams.get('chat') === '1') {
      setShowChatDialog(true)
      next.delete('chat')
      changed = true
    }

    if (searchParams.get('import') === '1') {
      setShowImportDialog(true)
      next.delete('import')
      changed = true
    }

    if (changed) {
      setSearchParams(next, { replace: true })
    }
  }, [searchParams, setSearchParams])
  useEffect(() => {
    const handleDocumentClick = () => {
      setHighlightCells({})
      setHighlightIndicators({})
    }
    document.addEventListener('click', handleDocumentClick)
    return () => document.removeEventListener('click', handleDocumentClick)
  }, [])

  useEffect(() => {
    if (!status || autoImportPrompted) {
      return
    }
    if (!status.initialized) {
      setShowImportDialog(true)
      setAutoImportPrompted(true)
    }
  }, [autoImportPrompted, status])

  const selectMonth = async (key: string) => {
    const [yearRaw, monthRaw] = key.split('-')
    const year = Number(yearRaw)
    const month = Number(monthRaw)
    if (!Number.isFinite(year) || !Number.isFinite(month)) {
      return
    }

    const response = await fetch('/api/months/select', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ year, month }),
    })
    const data = await response.json()
    if (!response.ok) {
      throw new Error(data?.error || '切换月份失败')
    }

    if (data.status) {
      setStatus(data.status as SystemStatus)
    }
    if (Array.isArray(data.groups)) {
      setGroups(data.groups as IndicatorGroup[])
    }
    setDraftTargets({})
    setReloadToken((value) => value + 1)
    clearUndo()
    void loadRuleEvaluations()
  }

  const previewLinkage = async (anchor: LinkageAnchor) => {
    try {
      const response = await fetch('/api/linkage/preview', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ anchor }),
      })
      const data = await response.json()
      if (!response.ok) {
        throw new Error(data?.error || '联动预览失败')
      }

      const nodes = (Array.isArray(data.nodes) ? data.nodes : []) as LinkageNode[]
      const nextCells: Record<string, boolean> = {}
      const nextIndicators: Record<string, boolean> = {}

      for (const node of nodes) {
        if (node?.ui?.rowId && node?.ui?.columnKey) {
          nextCells[`${node.ui.rowId}|${node.ui.columnKey}`] = true
        }
        if (node?.indicatorId) {
          nextIndicators[String(node.indicatorId)] = true
        }
      }

      setHighlightCells(nextCells)
      setHighlightIndicators(nextIndicators)
    } catch {
      setHighlightCells({})
      setHighlightIndicators({})
    }
  }

  const openImportDialog = () => {
    setShowImportDialog(true)
  }

  const closeImportDialog = () => {
    setShowImportDialog(false)
  }

  const handleImported = async () => {
    setShowImportDialog(false)
    await Promise.all([loadStatus(), loadIndicators(), loadMonths(), loadRuleEvaluations()])
    setReloadToken((value) => value + 1)
    clearUndo()
    setDraftTargets({})
  }

  const applyOptimize = async (
    targets: Record<string, number>,
    clearIds?: string[],
    undoMeta?: { type: 'indicator' | 'optimize'; summary: string }
  ) => {
    setOptimizing(true)

    let beforeSnapshot: CompanySnapshot | null = null
    if (undoMeta) {
      beforeSnapshot = await fetchCompaniesSnapshot()
    }

    try {
      const response = await fetch('/api/optimize', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ targets }),
      })
      const data = await response.json()
      if (Array.isArray(data.notices)) {
        showOptimizeNotices(data.notices as OptimizeNotice[])
      }
      if (!response.ok) {
        throw new Error(data?.error || '智能调整失败')
      }

      if (Array.isArray(data.groups)) {
        setGroups(data.groups as IndicatorGroup[])
      }
      void loadRuleEvaluations()
      if (!clearIds || clearIds.length === 0) {
        setDraftTargets({})
      } else {
        setDraftTargets((previous) => {
          const next = { ...previous }
          for (const id of clearIds) {
            delete next[id]
          }
          return next
        })
      }
      setReloadToken((value) => value + 1)

      if (undoMeta && beforeSnapshot) {
        const afterSnapshot = await fetchCompaniesSnapshot()
        const changes = buildUndoChanges(beforeSnapshot, afterSnapshot)
        if (changes.length > 0) {
          pushUndoStep({
            type: undoMeta.type,
            summary: undoMeta.summary,
            changes,
            createdAt: Date.now(),
          })
        }
      }
    } finally {
      setOptimizing(false)
    }
  }

  const applySingle = async (indicatorId: string) => {
    const value = Number(String(draftTargets[indicatorId] || '').replaceAll(',', '').trim())
    if (!Number.isFinite(value)) {
      return
    }
    await applyOptimize({ [indicatorId]: value }, [indicatorId], { type: 'indicator', summary: '指标调整' })
  }

  const handleSmartAdjust = async () => {
    const targets: Record<string, number> = {}
    for (const [id, raw] of Object.entries(draftTargets)) {
      const value = Number(String(raw).replaceAll(',', '').trim())
      if (Number.isFinite(value)) {
        targets[id] = value
      }
    }
    if (Object.keys(targets).length === 0) {
      return
    }
    await applyOptimize(targets, undefined, { type: 'optimize', summary: '智能调整' })
  }

  const handleUndo = async () => {
    if (!canUndo) {
      return
    }
    const step = undoStack[undoStack.length - 1]
    if (!step) {
      return
    }

    setUndoing(true)
    try {
      const updates = buildUndoUpdates(step.changes).filter((item) => Object.keys(item.patch).length > 0)
      if (updates.length === 0) {
        popUndoStep()
        return
      }

      const response = await fetch('/api/companies/batch', {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ updates }),
      })
      const data = await response.json()
      if (!response.ok) {
        throw new Error(data?.error || '撤销失败')
      }

      if (Array.isArray(data.groups)) {
        setGroups(data.groups as IndicatorGroup[])
      }
      void loadRuleEvaluations()
      popUndoStep()
      setDraftTargets({})
      setReloadToken((value) => value + 1)
    } finally {
      setUndoing(false)
    }
  }

  const monthKey = status
    ? `${status.currentYear}-${String(status.currentMonth).padStart(2, '0')}`
    : ''

  const summaryText = status
    ? `${status.currentYear}年${status.currentMonth}月 · 共 ${status.totalCompanies} 家企业（批零 ${status.wrCount} + 住餐 ${status.acCount}）`
    : '加载中...'

  const mergedHighlightIndicators = useMemo(() => {
    return {
      ...highlightIndicators,
    }
  }, [highlightIndicators])

  const visibleGroups = useMemo(() => {
    if (groups.length === 0) {
      return []
    }
    return groups.map((group) => {
      const indicators = group.indicators.map((indicator) => {
        return {
          ...indicator,
          unit: indicator.unit || (isRateGroup(group.name) ? '%' : '万元'),
        }
      })
      return {
        ...group,
        indicators,
      }
    })
  }, [groups])

  return (
    <div className="min-h-full bg-[#F8FAFC] px-6 py-5">
      <div className="space-y-5">
        <header className="flex flex-wrap items-start justify-between gap-3 px-1">
          <div>
            <h1 className="text-[20px] font-semibold leading-[28px] text-[#0F172A]">经济数据统计</h1>
            <p className="mt-0.5 text-xs text-[#64748B]">{summaryText}</p>
          </div>

          <div className="flex flex-wrap items-center gap-2">
            <Select value={monthKey || undefined} onValueChange={selectMonth} disabled={monthsLoading || saving}>
              <SelectTrigger className="h-9 w-[140px] border-[#E2E8F0] text-xs">
                <SelectValue placeholder={monthsLoading ? '加载中...' : '选择月份'} />
              </SelectTrigger>
              <SelectContent>
                {months.map((month) => {
                  const value = `${month.year}-${String(month.month).padStart(2, '0')}`
                  return (
                    <SelectItem key={value} value={value}>
                      {month.year}年{month.month}月
                    </SelectItem>
                  )
                })}
              </SelectContent>
            </Select>

            <Button variant="outline" className="h-9 gap-1 px-3 text-xs" onClick={openImportDialog}>
              <Upload className="h-3.5 w-3.5" />
              导入
            </Button>
            <Button variant="outline" className="h-9 gap-1 px-3 text-xs" onClick={() => setShowExportDialog(true)}>
              <Download className="h-3.5 w-3.5" />
              导出
            </Button>
          </div>
        </header>

        {loading ? (
          <div className="rounded-xl border border-[#E2E8F0] bg-white p-6 text-sm text-[#64748B]">正在加载指标数据...</div>
        ) : (
          <IndicatorCards
            groups={visibleGroups}
            draftTargets={draftTargets}
            highlightIndicators={mergedHighlightIndicators}
            onDraftChange={(id, value) => setDraftTargets((prev) => ({ ...prev, [id]: value }))}
            onEnterApply={applySingle}
            onPreview={(id) => previewLinkage({ indicatorId: id })}
          />
        )}

        <EnterpriseDataTable
          reloadToken={reloadToken}
          highlightCells={highlightCells}
          onCellPreview={(rowId, columnKey) => previewLinkage({ ui: { rowId, columnKey } })}
          onIndicatorsUpdate={(next) => setGroups(next)}
          onSavingChange={setTableSaving}
          onSmartAdjust={handleSmartAdjust}
          onUndo={handleUndo}
          smartAdjustDisabled={!hasDraft || optimizing}
          undoDisabled={!canUndo}
          busy={saving}
        />

        <Dialog open={showImportDialog} onOpenChange={setShowImportDialog}>
          <DialogContent className="max-h-[92vh] w-[min(1220px,96vw)] max-w-none overflow-hidden border border-[#E2E8F0] p-0">
            <ImportV3 modal onClose={closeImportDialog} onImported={() => void handleImported()} />
          </DialogContent>
        </Dialog>

        {showExportDialog && (
          <ExportDialog
            open={showExportDialog}
            onClose={() => setShowExportDialog(false)}
            year={status?.currentYear}
            month={status?.currentMonth}
          />
        )}
        <LlmChatDialog
          open={showChatDialog}
          onOpenChange={setShowChatDialog}
          onDataChanged={() => {
            loadIndicators()
            setReloadToken((value) => value + 1)
          }}
          onPreviewImpact={(impact) => {
            const nextCells: Record<string, boolean> = {}
            const nextIndicators: Record<string, boolean> = {}
            for (const cell of impact.cells) {
              if (cell?.rowId && cell?.columnKey) {
                nextCells[`${cell.rowId}|${cell.columnKey}`] = true
              }
            }
            for (const id of impact.indicators) {
              if (id) {
                nextIndicators[String(id)] = true
              }
            }
            setHighlightCells(nextCells)
            setHighlightIndicators(nextIndicators)
          }}
        />
      </div>
    </div>
  )
}
