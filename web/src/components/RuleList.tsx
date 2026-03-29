/**
 * 规则列表
 *
 * @author Anner
 * Created on 2026/3/14
 */

import { useEffect, useMemo, useState } from 'react'
import { Plus, RefreshCw, Pencil, Trash2, AlertCircle, ChevronLeft, ChevronRight, Loader2 } from 'lucide-react'
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
import { Progress } from '@/components/ui/progress'
import { Textarea } from '@/components/ui/textarea'
import { useRulesStore } from '@/store/rulesStore'
import type { RuleItem } from '@/services/api'

const PAGE_SIZE = 8

/** 转换步骤 → 进度百分比与中文标签 */
const STEP_META: Record<string, { progress: number; label: string }> = {
  initializing: { progress: 10, label: '初始化' },
  calling_llm: { progress: 30, label: '调用大模型' },
  validating: { progress: 70, label: '校验规则' },
  writing_json: { progress: 85, label: '写入规则' },
  reloading: { progress: 95, label: '热重载引擎' },
}

interface RuleListViewProps {
  rules: RuleItem[]
  status: 'idle' | 'pending' | 'running' | 'ok' | 'error'
  statusError: string
  statusUpdatedAt: string
  statusStep: string
  statusAttempt: string
  loading: boolean
  submitting: boolean
  onAdd: (text: string) => Promise<void>
  onEdit: (index: number, text: string) => Promise<void>
  onDelete: (index: number) => Promise<void>
  onRefresh: () => Promise<void>
  onConvert: () => Promise<void>
}

export function RuleListView({
  rules,
  status,
  statusError,
  statusUpdatedAt,
  statusStep,
  statusAttempt,
  loading,
  submitting,
  onAdd,
  onEdit,
  onDelete,
  onRefresh,
  onConvert,
}: RuleListViewProps) {
  const [dialogOpen, setDialogOpen] = useState(false)
  const [draft, setDraft] = useState('')
  const [editing, setEditing] = useState<RuleItem | null>(null)
  const [currentPage, setCurrentPage] = useState(1)

  const statusMeta = useMemo(() => buildStatusMeta(status), [status])
  const formattedUpdatedAt = formatUpdatedAt(statusUpdatedAt)

  const totalPages = Math.max(1, Math.ceil(rules.length / PAGE_SIZE))
  const pagedRules = useMemo(() => {
    const start = (currentPage - 1) * PAGE_SIZE
    return rules.slice(start, start + PAGE_SIZE)
  }, [rules, currentPage])

  useEffect(() => {
    if (currentPage > totalPages) {
      setCurrentPage(totalPages)
    }
  }, [totalPages, currentPage])

  const handleSubmit = async () => {
    const text = draft.trim()
    if (!text) {
      toast.error('规则内容不能为空')
      return
    }
    try {
      if (editing) {
        await onEdit(editing.index, text)
        toast.success('规则已更新')
      } else {
        await onAdd(text)
        toast.success('规则已新增')
      }
      setDialogOpen(false)
      setDraft('')
      setEditing(null)
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '保存规则失败')
    }
  }

  const handleDelete = async (index: number) => {
    try {
      await onDelete(index)
      toast.success('规则已删除')
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '删除规则失败')
    }
  }

  const handleConvert = async () => {
    try {
      await onConvert()
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '触发规则转换失败')
    }
  }

  const openCreate = () => {
    setEditing(null)
    setDraft('')
    setDialogOpen(true)
  }

  const openEdit = (item: RuleItem) => {
    setEditing(item)
    setDraft(item.text)
    setDialogOpen(true)
  }

  // 进度条计算
  const stepInfo = STEP_META[statusStep] ?? null
  const progressValue = status === 'running' ? (stepInfo?.progress ?? 15) : 0

  return (
    <>
      <div data-testid="rule-list" className="flex h-full flex-col">
        {/* 头部 */}
        <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
          <div className="space-y-1.5">
            <p className="text-sm text-muted-foreground">
              编辑完成后点击「转换规则」按钮统一生效。
            </p>
            <div className="flex flex-wrap items-center gap-3 text-sm text-muted-foreground">
              <Badge variant={statusMeta.variant}>{statusMeta.label}</Badge>
              <span>最后转换：{formattedUpdatedAt}</span>
              <span className="text-xs">共 {rules.length} 条</span>
            </div>
          </div>
          <div className="flex items-center gap-2 shrink-0">
            <Button size="sm" variant="outline" onClick={() => void onRefresh()} disabled={loading || submitting}>
              <RefreshCw className="h-3.5 w-3.5" />
              刷新
            </Button>
            <Button
              size="sm"
              variant={status === 'pending' ? 'default' : 'outline'}
              onClick={() => void handleConvert()}
              disabled={submitting || status === 'running'}
            >
              <RefreshCw className="h-3.5 w-3.5" />
              {status === 'pending' ? '转换规则' : '重新转换'}
            </Button>
            <Button size="sm" onClick={openCreate} disabled={submitting}>
              <Plus className="h-3.5 w-3.5" />
              新增规则
            </Button>
          </div>
        </div>

        {/* 转换进度条 */}
        {status === 'running' && (
          <div className="mt-3 space-y-1.5">
            <div className="flex items-center gap-2 text-xs text-muted-foreground">
              <Loader2 className="h-3 w-3 animate-spin" />
              <span>
                {stepInfo?.label ?? '准备中'}
                {statusAttempt ? ` (第 ${statusAttempt} 次)` : ''}
                …
              </span>
            </div>
            <Progress value={progressValue} className="h-1.5" />
          </div>
        )}

        {/* 错误详情 */}
        {status === 'error' && statusError && (
          <details className="mt-3 rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive">
            <summary className="flex cursor-pointer list-none items-center gap-2 font-medium">
              <AlertCircle className="h-4 w-4" />
              转换失败，查看错误详情
            </summary>
            <div className="mt-2 whitespace-pre-wrap break-words">{statusError}</div>
          </details>
        )}

        {/* 规则列表（固定高度区域） */}
        <div className="mt-3 min-h-0 flex-1 overflow-auto">
          <div className="space-y-2">
            {rules.length === 0 && (
              <div className="rounded-lg border border-dashed px-4 py-10 text-center text-sm text-muted-foreground">
                还没有任何调整规则，可以先新增一条自然语言规则。
              </div>
            )}
            {pagedRules.map((item) => (
              <div
                key={item.index}
                className="flex items-start justify-between gap-4 rounded-lg border bg-muted/20 px-4 py-3"
              >
                <div className="min-w-0 space-y-0.5">
                  <div className="text-xs font-medium uppercase tracking-[0.18em] text-muted-foreground">
                    Rule {item.index}
                  </div>
                  <div className="text-sm leading-6 text-foreground">{item.text}</div>
                </div>
                <div className="flex shrink-0 items-center gap-1.5">
                  <Button variant="ghost" size="sm" onClick={() => openEdit(item)} disabled={submitting}>
                    <Pencil className="h-3.5 w-3.5" />
                  </Button>
                  <Button variant="ghost" size="sm" onClick={() => void handleDelete(item.index)} disabled={submitting}>
                    <Trash2 className="h-3.5 w-3.5" />
                  </Button>
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* 分页 */}
        {totalPages > 1 && (
          <div className="flex items-center justify-center gap-2 mt-3 pt-2 border-t text-sm text-muted-foreground">
            <Button
              variant="outline"
              size="sm"
              disabled={currentPage <= 1}
              onClick={() => setCurrentPage((p) => p - 1)}
            >
              <ChevronLeft className="h-3.5 w-3.5" />
            </Button>
            <span>
              {currentPage} / {totalPages}
            </span>
            <Button
              variant="outline"
              size="sm"
              disabled={currentPage >= totalPages}
              onClick={() => setCurrentPage((p) => p + 1)}
            >
              <ChevronRight className="h-3.5 w-3.5" />
            </Button>
          </div>
        )}
      </div>

      {/* 新增/编辑弹窗 */}
      <Dialog
        open={dialogOpen}
        onOpenChange={(open) => {
          setDialogOpen(open)
          if (!open) {
            setEditing(null)
            setDraft('')
          }
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{editing ? `编辑规则 ${editing.index}` : '新增规则'}</DialogTitle>
            <DialogDescription>使用一条清晰、可执行的自然语言规则，系统会自动转换为结构化规则。</DialogDescription>
          </DialogHeader>
          <Textarea
            value={draft}
            onChange={(event) => setDraft(event.target.value)}
            placeholder="例如：调整零售业当月增速时，仅允许使用正增长企业参与分配。"
            disabled={submitting}
          />
          <DialogFooter className="gap-2">
            <Button variant="outline" onClick={() => setDialogOpen(false)} disabled={submitting}>
              取消
            </Button>
            <Button onClick={() => void handleSubmit()} disabled={submitting}>
              {submitting ? '提交中…' : '保存'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}

export default function RuleList() {
  const rules = useRulesStore((state) => state.rules)
  const status = useRulesStore((state) => state.status)
  const statusError = useRulesStore((state) => state.statusError)
  const statusUpdatedAt = useRulesStore((state) => state.statusUpdatedAt)
  const statusStep = useRulesStore((state) => state.statusStep)
  const statusAttempt = useRulesStore((state) => state.statusAttempt)
  const loading = useRulesStore((state) => state.loading)
  const submitting = useRulesStore((state) => state.submitting)
  const loadRules = useRulesStore((state) => state.loadRules)
  const loadStatus = useRulesStore((state) => state.loadStatus)
  const addRule = useRulesStore((state) => state.addRule)
  const updateRule = useRulesStore((state) => state.updateRule)
  const deleteRule = useRulesStore((state) => state.deleteRule)
  const convertRules = useRulesStore((state) => state.convertRules)
  const stopPolling = useRulesStore((state) => state.stopPolling)

  useEffect(() => {
    void loadRules()
    void loadStatus()
    return () => {
      stopPolling()
    }
  }, [loadRules, loadStatus, stopPolling])

  return (
    <RuleListView
      rules={rules}
      status={status}
      statusError={statusError}
      statusUpdatedAt={statusUpdatedAt}
      statusStep={statusStep}
      statusAttempt={statusAttempt}
      loading={loading}
      submitting={submitting}
      onAdd={addRule}
      onEdit={updateRule}
      onDelete={deleteRule}
      onRefresh={loadStatus}
      onConvert={convertRules}
    />
  )
}

function buildStatusMeta(status: RuleListViewProps['status']) {
  if (status === 'idle') {
    return { label: '待转换', variant: 'outline' as const }
  }
  if (status === 'pending') {
    return { label: '规则已修改', variant: 'secondary' as const }
  }
  if (status === 'running') {
    return { label: '转换中', variant: 'secondary' as const }
  }
  if (status === 'error') {
    return { label: '转换失败', variant: 'destructive' as const }
  }
  return { label: '已生效', variant: 'default' as const }
}

function formatUpdatedAt(value: string) {
  if (!value) {
    return '未完成'
  }
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }
  return `${date.getMonth() + 1}-${date.getDate()} ${String(date.getHours()).padStart(2, '0')}:${String(
    date.getMinutes()
  ).padStart(2, '0')}`
}
