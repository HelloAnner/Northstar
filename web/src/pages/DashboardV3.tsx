import { useState, useEffect, useRef, useCallback } from 'react'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { Badge } from '@/components/ui/badge'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Download, MessageCircle, Settings2, Upload } from 'lucide-react'
import ChatPanel from '@/components/ChatPanel'
import ExportDialog from '@/components/ExportDialog'
import ImportDialog from '@/components/ImportDialog'
import CompaniesTable, { type IndicatorGroup } from '@/components/CompaniesTable'
import IndicatorCards from '@/components/dashboard/IndicatorCards'
import ThemeToggle from '@/components/app/ThemeToggle'
import NorthstarIcon from '@/components/app/NorthstarIcon'
import GlobalConfigDialog from '@/components/GlobalConfigDialog'
import { toast } from 'sonner'
import { buildCompanySnapshot, buildUndoChanges, buildUndoUpdates, type CompanySnapshot } from '@/lib/undo'
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
  suggestMin?: number
}

export default function DashboardV3() {
  const [status, setStatus] = useState<SystemStatus | null>(null)
  const [groups, setGroups] = useState<IndicatorGroup[]>([])
  const [loading, setLoading] = useState(true)
  const [tableSaving, setTableSaving] = useState(false)
  const [optimizing, setOptimizing] = useState(false)
  const [reloadToken, setReloadToken] = useState(0)
  const [draftTargets, setDraftTargets] = useState<Record<string, string>>({})
  const [months, setMonths] = useState<YearMonthStat[]>([])
  const [monthsLoading, setMonthsLoading] = useState(false)
  const [undoing, setUndoing] = useState(false)
  const undoStack = useUndoStore((state) => state.stack)
  const pushUndoStep = useUndoStore((state) => state.push)
  const popUndoStep = useUndoStore((state) => state.pop)
  const clearUndo = useUndoStore((state) => state.clear)
  const [showExportDialog, setShowExportDialog] = useState(false)
  const [showConfigDialog, setShowConfigDialog] = useState(false)
  const [showImportDialog, setShowImportDialog] = useState(false)
  const [showChatDialog, setShowChatDialog] = useState(false)
  const [highlightCells, setHighlightCells] = useState<Record<string, boolean>>({})
  const [highlightIndicators, setHighlightIndicators] = useState<Record<string, boolean>>({})
  const [failedIndicators, setFailedIndicators] = useState<Record<string, boolean>>({})
  const [suggestions, setSuggestions] = useState<{ chat: { title: string; content: string }[]; adjust: { title: string; content: string }[] } | null>(null)
  const [changedIndicators, setChangedIndicators] = useState<Record<string, boolean>>({})
  const changedTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const flashChangedIndicators = useCallback((ids: string[]) => {
    if (ids.length === 0) return
    const next: Record<string, boolean> = {}
    for (const id of ids) next[id] = true
    setChangedIndicators(next)
    if (changedTimerRef.current) clearTimeout(changedTimerRef.current)
    changedTimerRef.current = setTimeout(() => {
      setChangedIndicators({})
      changedTimerRef.current = null
    }, 3000)
  }, [])

  const deriveFailedIndicators = (notices?: OptimizeNotice[]) => {
    if (!Array.isArray(notices) || notices.length === 0) {
      setFailedIndicators({})
      return
    }
    const failed: Record<string, boolean> = {}
    for (const n of notices) {
      if (n.level === 'warn' || n.level === 'error') {
        failed[n.indicatorId] = true
      }
    }
    setFailedIndicators(failed)
  }

  const showOptimizeNotices = (notices?: OptimizeNotice[]) => {
    if (!Array.isArray(notices) || notices.length === 0) return
    for (const notice of notices) {
      const title = notice.indicatorName
        ? `${notice.indicatorName}：${notice.message}`
        : notice.message || '智能调整提示'

      const descParts: string[] = []
      if (notice.target !== undefined && notice.before !== undefined) {
        descParts.push(`目标 ${notice.target}，调整前 ${notice.before} → 调整后 ${notice.after}`)
      }
      if (notice.suggestion) {
        descParts.push(notice.suggestion)
      }
      const description = descParts.length > 0 ? descParts.join('。') : undefined

      const duration = notice.level === 'error' ? 6000 : 4000
      const payload = { description, duration }
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
    try {
      const res = await fetch('/api/status')
      const data = await res.json()
      setStatus(data)
    } catch (err) {
      console.error('Failed to load status:', err)
    }
  }

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

  const loadSuggestions = async () => {
    try {
      const res = await fetch('/api/llm/suggestions')
      if (res.ok) {
        const data = await res.json()
        if (data.chat?.length > 0 || data.adjust?.length > 0) {
          setSuggestions(data)
        }
      }
    } catch {
      /* silent */
    }
  }

  useEffect(() => {
    loadStatus()
    loadIndicators()
    loadMonths()
    loadSuggestions()
  }, [])

  useEffect(() => {
    const handleClear = () => {
      setHighlightCells({})
      setHighlightIndicators({})
    }
    document.addEventListener('click', handleClear)
    return () => document.removeEventListener('click', handleClear)
  }, [])

  const fetchCompaniesSnapshot = async (): Promise<CompanySnapshot> => {
    const q = new URLSearchParams()
    q.set('page', '1')
    q.set('pageSize', '2000')
    const res = await fetch(`/api/companies?${q.toString()}`)
    if (!res.ok) throw new Error('加载企业数据失败')
    const data = (await res.json()) as { items: any[] }
    return buildCompanySnapshot(Array.isArray(data.items) ? data.items : [])
  }

  const pushUndoFromSnapshots = (type: 'indicator' | 'optimize', summary: string, before: CompanySnapshot, after: CompanySnapshot) => {
    const changes = buildUndoChanges(before, after)
    if (changes.length === 0) return
    pushUndoStep({
      type,
      summary,
      changes,
      createdAt: Date.now(),
    })
  }

  const applyOptimize = async (
    targets: Record<string, number>,
    clearIds?: string[],
    undoMeta?: { type: 'indicator' | 'optimize'; summary: string }
  ) => {
    setOptimizing(true)
    let beforeSnapshot: CompanySnapshot | null = null
    let succeeded = false
    try {
      if (undoMeta) {
        beforeSnapshot = await fetchCompaniesSnapshot()
      }
      const res = await fetch('/api/optimize', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ targets }),
      })
      const data = await res.json()
      const notices = data?.notices as OptimizeNotice[] | undefined
      showOptimizeNotices(notices)
      deriveFailedIndicators(notices)

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
          for (const id of clearIds) delete next[id]
          return next
        })
      }
      setReloadToken((x) => x + 1)
      succeeded = true
    } finally {
      setOptimizing(false)
    }

    if (undoMeta && beforeSnapshot && succeeded) {
      try {
        const afterSnapshot = await fetchCompaniesSnapshot()
        pushUndoFromSnapshots(undoMeta.type, undoMeta.summary, beforeSnapshot, afterSnapshot)
      } catch (err) {
        console.error(err)
      }
    }
  }

  const handleSmartAdjust = async () => {
    const entries = Object.entries(draftTargets)
    if (entries.length === 0) return

    const targets: Record<string, number> = {}
    for (const [id, raw] of entries) {
      const v = Number(String(raw).replaceAll(',', '').trim())
      if (Number.isFinite(v)) targets[id] = v
    }
    if (Object.keys(targets).length === 0) return

    try {
      await applyOptimize(targets, undefined, { type: 'optimize', summary: '智能调整' })
      flashChangedIndicators(Object.keys(targets))
    } catch (err) {
      console.error(err)
    }
  }

  const applySingle = async (id: string, raw: string) => {
    try {
      const v = Number(String(raw).replaceAll(',', '').trim())
      if (!Number.isFinite(v)) return
      await applyOptimize({ [id]: v }, [id], { type: 'indicator', summary: '指标调整' })
      flashChangedIndicators([id])
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

      if (data.status) setStatus(data.status as SystemStatus)
      if (Array.isArray(data.groups)) setGroups(data.groups as IndicatorGroup[])
      setDraftTargets({})
      setFailedIndicators({})
      setReloadToken((x) => x + 1)
      clearUndo()
    } catch (err) {
      console.error(err)
    }
  }

  const handleChatDataChanged = (changedIndicatorIds?: string[]) => {
    loadIndicators()
    setReloadToken((x) => x + 1)
    if (changedIndicatorIds && changedIndicatorIds.length > 0) {
      flashChangedIndicators(changedIndicatorIds)
    }
  }

  const previewLinkage = async (anchor: any) => {
    try {
      const res = await fetch('/api/linkage/preview', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ anchor }),
      })
      const data = await res.json()
      if (!res.ok) throw new Error(data?.error || '联动预览失败')
      const nodes = Array.isArray(data.nodes) ? data.nodes : []
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
    } catch (err) {
      console.error(err)
      setHighlightCells({})
      setHighlightIndicators({})
    }
  }

  // 空状态：自动弹出导入弹窗
  useEffect(() => {
    if (status && !status.initialized) {
      setShowImportDialog(true)
    }
  }, [status])

  const hasDraft = Object.keys(draftTargets).length > 0
  const canUndo = undoStack.length > 0 && !undoing

  const handleUndo = async () => {
    if (!canUndo) return
    const step = undoStack[undoStack.length - 1]
    if (!step) return

    setUndoing(true)
    try {
      const updates = buildUndoUpdates(step.changes).filter((item) => Object.keys(item.patch).length > 0)
      if (updates.length === 0) {
        popUndoStep()
        return
      }
      const res = await fetch('/api/companies/batch', {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ updates }),
      })
      const data = await res.json()
      if (!res.ok) throw new Error(data?.error || '撤销失败')
      if (Array.isArray(data.groups)) {
        setGroups(data.groups as IndicatorGroup[])
      }
      setDraftTargets({})
      setFailedIndicators({})
      setReloadToken((x) => x + 1)
      popUndoStep()
    } catch (err) {
      console.error(err)
      toast.error('撤销失败', { description: err instanceof Error ? err.message : '请稍后重试' })
    } finally {
      setUndoing(false)
    }
  }

  const monthSelector = (
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
  )

  return (
    <div className="flex h-screen overflow-hidden bg-gradient-to-b from-background via-background to-muted/20">
      <div className="min-w-0 flex-1 space-y-6 overflow-y-auto p-6">
        {/* 顶部栏 */}
        <div className="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
          <div>
            <h1 className="flex items-center gap-2 text-3xl font-semibold tracking-tight">
              <NorthstarIcon className="h-7 w-7" />
              Northstar
            </h1>
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

            {monthSelector}

            <Button
              disabled={!hasDraft || optimizing}
              className="gap-2 bg-orange-500 text-white hover:bg-orange-600"
              onClick={handleSmartAdjust}
              title={hasDraft ? '按输入值反推并写回企业数据' : '先在指标输入框里填入目标值'}
            >
              智能调整
            </Button>

            <Button onClick={handleUndo} variant="outline" className="gap-2" disabled={!canUndo}>
              撤销
            </Button>

            <Button onClick={() => setShowImportDialog(true)} variant="outline" className="gap-2">
              <Upload className="h-4 w-4" />
              导入
            </Button>

            <Button onClick={() => setShowExportDialog(true)} variant="outline" className="gap-2" disabled={saving}>
              <Download className="h-4 w-4" />
              导出
            </Button>

            <Button onClick={() => setShowConfigDialog(true)} variant="outline" className="gap-2">
              <Settings2 className="h-4 w-4" />
              全局配置
            </Button>
            <ThemeToggle />
          </div>
        </div>

        {/* 指标面板 */}
        {loading ? (
          <div className={`grid grid-cols-1 gap-4 ${showChatDialog ? 'lg:grid-cols-2' : 'lg:grid-cols-4'}`}>
            {[...Array(4)].map((_, i) => (
              <Card key={i} className="border-border/60 bg-card/60 backdrop-blur">
                <CardHeader className="pb-3">
                  <Skeleton className="h-4 w-32" />
                </CardHeader>
                <CardContent className="space-y-2">
                  {[...Array(4)].map((__, j) => (
                    <Skeleton key={j} className="h-8 w-full" />
                  ))}
                </CardContent>
              </Card>
            ))}
          </div>
        ) : (
          <IndicatorCards
            groups={groups}
            draftTargets={draftTargets}
            highlightIndicators={highlightIndicators}
            changedIndicators={changedIndicators}
            failedIndicators={failedIndicators}
            disabled={optimizing}
            compact={showChatDialog}
            onDraftChange={(id, v) => setDraftTargets((prev) => ({ ...prev, [id]: v }))}
            onEnterApply={applySingle}
            onPreview={(id) => previewLinkage({ indicatorId: id })}
          />
        )}

        {/* 明细表 */}
        <CompaniesTable
          onIndicatorsUpdate={(next) => setGroups(next)}
          onSavingChange={(s) => setTableSaving(s)}
          onCellPreview={(rowId, columnKey) => previewLinkage({ ui: { rowId, columnKey } })}
          highlightCells={highlightCells}
          monthSelector={monthSelector}
          reloadToken={reloadToken}
        />

        {!showChatDialog && (
          <Button
            onClick={() => setShowChatDialog(true)}
            size="icon"
            className="fixed bottom-6 right-6 z-40 h-12 w-12 rounded-full shadow-lg transition hover:scale-105"
          >
            <MessageCircle className="h-5 w-5" />
          </Button>
        )}

        {showExportDialog && (
          <ExportDialog
            open={showExportDialog}
            onClose={() => setShowExportDialog(false)}
            year={status?.currentYear}
            month={status?.currentMonth}
          />
        )}

        <GlobalConfigDialog open={showConfigDialog} onOpenChange={setShowConfigDialog} />

        <ImportDialog
          open={showImportDialog}
          onClose={() => setShowImportDialog(false)}
          onSuccess={() => {
            setShowImportDialog(false)
            window.location.reload()
          }}
        />
      </div>

      {showChatDialog && (
        <ChatPanel open={showChatDialog} onOpenChange={setShowChatDialog} onAdjustApplied={handleChatDataChanged} suggestions={suggestions} />
      )}
    </div>
  )
}
