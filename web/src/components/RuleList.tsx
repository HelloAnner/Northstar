/**
 * 规则列表
 *
 * @author Anner
 * Created on 2026/3/14
 */

import { useEffect, useMemo, useState } from 'react'
import { Plus, RefreshCw, Pencil, Trash2, AlertCircle } from 'lucide-react'
import { toast } from 'sonner'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Textarea } from '@/components/ui/textarea'
import { useRulesStore } from '@/store/rulesStore'
import type { RuleItem } from '@/services/api'

interface RuleListViewProps {
  rules: RuleItem[]
  status: 'idle' | 'running' | 'ok' | 'error'
  statusError: string
  statusUpdatedAt: string
  loading: boolean
  submitting: boolean
  onAdd: (text: string) => Promise<void>
  onEdit: (index: number, text: string) => Promise<void>
  onDelete: (index: number) => Promise<void>
  onRefresh: () => Promise<void>
}

export function RuleListView({
  rules,
  status,
  statusError,
  statusUpdatedAt,
  loading,
  submitting,
  onAdd,
  onEdit,
  onDelete,
  onRefresh,
}: RuleListViewProps) {
  const [dialogOpen, setDialogOpen] = useState(false)
  const [draft, setDraft] = useState('')
  const [editing, setEditing] = useState<RuleItem | null>(null)

  const statusMeta = useMemo(() => buildStatusMeta(status), [status])
  const formattedUpdatedAt = formatUpdatedAt(statusUpdatedAt)

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

  return (
    <>
      <Card data-testid="rule-list">
        <CardHeader className="gap-4 md:flex-row md:items-start md:justify-between">
          <div className="space-y-2">
            <CardTitle>调整规则</CardTitle>
            <CardDescription>每次新增、编辑或删除后都会自动触发规则转换与热重载。</CardDescription>
            <div className="flex flex-wrap items-center gap-3 text-sm text-muted-foreground">
              <Badge variant={statusMeta.variant}>{statusMeta.label}</Badge>
              <span>最后转换：{formattedUpdatedAt}</span>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <Button variant="outline" onClick={() => void onRefresh()} disabled={loading || submitting}>
              <RefreshCw className="h-4 w-4" />
              刷新状态
            </Button>
            <Button onClick={openCreate} disabled={submitting}>
              <Plus className="h-4 w-4" />
              新增规则
            </Button>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          {status === 'error' && statusError && (
            <div className="rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive">
              <div className="flex items-center gap-2 font-medium">
                <AlertCircle className="h-4 w-4" />
                转换失败
              </div>
              <div className="mt-2 whitespace-pre-wrap break-words">{statusError}</div>
            </div>
          )}

          <ScrollArea className="max-h-[480px]">
            <div className="space-y-3">
              {rules.length === 0 && (
                <div className="rounded-lg border border-dashed px-4 py-10 text-center text-sm text-muted-foreground">
                  还没有任何调整规则，可以先新增一条自然语言规则。
                </div>
              )}
              {rules.map((item) => (
                <div
                  key={item.index}
                  className="flex items-start justify-between gap-4 rounded-lg border bg-muted/20 px-4 py-4"
                >
                  <div className="min-w-0 space-y-1">
                    <div className="text-xs font-medium uppercase tracking-[0.18em] text-muted-foreground">
                      Rule {item.index}
                    </div>
                    <div className="text-sm leading-6 text-foreground">{item.text}</div>
                  </div>
                  <div className="flex shrink-0 items-center gap-2">
                    <Button variant="outline" size="sm" onClick={() => openEdit(item)} disabled={submitting}>
                      <Pencil className="h-4 w-4" />
                      编辑
                    </Button>
                    <Button variant="outline" size="sm" onClick={() => void handleDelete(item.index)} disabled={submitting}>
                      <Trash2 className="h-4 w-4" />
                      删除
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          </ScrollArea>
        </CardContent>
      </Card>

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
  const loading = useRulesStore((state) => state.loading)
  const submitting = useRulesStore((state) => state.submitting)
  const loadRules = useRulesStore((state) => state.loadRules)
  const loadStatus = useRulesStore((state) => state.loadStatus)
  const addRule = useRulesStore((state) => state.addRule)
  const updateRule = useRulesStore((state) => state.updateRule)
  const deleteRule = useRulesStore((state) => state.deleteRule)
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
      loading={loading}
      submitting={submitting}
      onAdd={addRule}
      onEdit={updateRule}
      onDelete={deleteRule}
      onRefresh={loadStatus}
    />
  )
}

function buildStatusMeta(status: RuleListViewProps['status']) {
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
