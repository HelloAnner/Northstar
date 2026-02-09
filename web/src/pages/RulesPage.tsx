import { useEffect, useMemo, useState } from 'react'
import { Edit3, GitBranch, Plus, Save, Trash2 } from 'lucide-react'
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
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { designApi } from '@/services/designApi'
import type {
  IndicatorDefinition,
  RuleDetail,
  RuleEvaluation,
  RuleIndicatorLink,
  RuleSeverity,
  UpsertRulePayload,
} from '@/types/design'

type RuleDraft = {
  code: string
  name: string
  description: string
  expression: string
  severity: RuleSeverity
  suggestion: string
  preferenceJson: string
  displayOrder: number
  enabled: boolean
  links: RuleIndicatorLink[]
}

const buildRuleDraft = (detail?: RuleDetail): RuleDraft => {
  if (!detail) {
    return {
      code: '',
      name: '',
      description: '',
      expression: '',
      severity: 'warn',
      suggestion: '',
      preferenceJson: '{}',
      displayOrder: 999,
      enabled: true,
      links: [],
    }
  }
  return {
    code: detail.rule.ruleCode,
    name: detail.rule.name,
    description: detail.rule.description,
    expression: detail.rule.expression,
    severity: detail.rule.severity,
    suggestion: detail.rule.suggestion,
    preferenceJson: detail.rule.preferenceJson,
    displayOrder: detail.rule.displayOrder,
    enabled: detail.rule.enabled,
    links: detail.links.map((link) => ({ ...link })),
  }
}

const toPayload = (draft: RuleDraft): UpsertRulePayload => ({
  name: draft.name,
  description: draft.description,
  expression: draft.expression,
  severity: draft.severity,
  suggestion: draft.suggestion,
  preferenceJson: draft.preferenceJson,
  displayOrder: draft.displayOrder,
  enabled: draft.enabled,
  links: draft.links.map((link, index) => ({
    ...link,
    ruleCode: draft.name,
    displayOrder: (index + 1) * 10,
  })),
})

const severityLabel: Record<RuleSeverity, string> = {
  info: '提示',
  warn: '预警',
  error: '错误',
}

const severityBadgeClass: Record<RuleSeverity, string> = {
  info: 'bg-[#E0F2FE] text-[#0369A1]',
  warn: 'bg-[#FEF3C7] text-[#92400E]',
  error: 'bg-[#FEE2E2] text-[#991B1B]',
}

const linkBadgePalette = [
  'bg-[#DBEAFE] text-[#3B82F6]',
  'bg-[#D1FAE5] text-[#10B981]',
  'bg-[#FEF3C7] text-[#F59E0B]',
]


const ruleStatusLabel: Record<string, string> = {
  pass: '通过',
  fail: '违规',
  skipped: '待补充',
}

const ruleStatusBadgeClass: Record<string, string> = {
  pass: 'bg-[#DCFCE7] text-[#15803D]',
  fail: 'bg-[#FEE2E2] text-[#B91C1C]',
  skipped: 'bg-[#E2E8F0] text-[#475569]',
}

export default function RulesPage() {
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [items, setItems] = useState<RuleDetail[]>([])
  const [indicators, setIndicators] = useState<IndicatorDefinition[]>([])
  const [evaluations, setEvaluations] = useState<Record<string, RuleEvaluation>>({})
  const [error, setError] = useState('')

  const [editorOpen, setEditorOpen] = useState(false)
  const [editorDraft, setEditorDraft] = useState<RuleDraft>(() => buildRuleDraft())

  const load = async () => {
    setLoading(true)
    setError('')
    try {
      const [rules, defs, checks] = await Promise.all([
        designApi.listRules(false),
        designApi.listIndicatorDefinitions(false),
        designApi.listRuleEvaluations(false),
      ])
      setItems(rules)
      setIndicators(defs)
      const map: Record<string, RuleEvaluation> = {}
      for (const item of checks) {
        if (item?.ruleCode) {
          map[item.ruleCode] = item
        }
      }
      setEvaluations(map)
    } catch (loadErr) {
      setError(loadErr instanceof Error ? loadErr.message : '加载规则失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void load()
  }, [])

  const sorted = useMemo(() => {
    return [...items].sort((left, right) => left.rule.displayOrder - right.rule.displayOrder)
  }, [items])

  const indicatorOptions = useMemo(() => {
    return indicators.map((item) => ({ code: item.code || item.name, label: item.name }))
  }, [indicators])


  const evaluationSummary = useMemo(() => {
    const values = Object.values(evaluations)
    const pass = values.filter((item) => item.status === 'pass').length
    const fail = values.filter((item) => item.status === 'fail').length
    const skipped = values.filter((item) => item.status === 'skipped').length
    return { pass, fail, skipped }
  }, [evaluations])

  const openCreate = () => {
    setEditorDraft(buildRuleDraft())
    setEditorOpen(true)
  }

  const openEdit = (item: RuleDetail) => {
    setEditorDraft(buildRuleDraft(item))
    setEditorOpen(true)
  }

  const saveRule = async () => {
    const name = editorDraft.name.trim()
    if (!name || !editorDraft.expression.trim()) {
      setError('规则名称和表达式不能为空')
      return
    }

    setSaving(true)
    setError('')
    try {
      await designApi.upsertRule(name, toPayload({ ...editorDraft, code: name, name }))
      setEditorOpen(false)
      await load()
    } catch (saveErr) {
      setError(saveErr instanceof Error ? saveErr.message : '保存规则失败')
    } finally {
      setSaving(false)
    }
  }

  const disableRule = async (detail: RuleDetail) => {
    setSaving(true)
    setError('')
    try {
      await designApi.upsertRule(detail.rule.ruleCode, {
        name: detail.rule.name,
        description: detail.rule.description,
        expression: detail.rule.expression,
        severity: detail.rule.severity,
        suggestion: detail.rule.suggestion,
        preferenceJson: detail.rule.preferenceJson,
        displayOrder: detail.rule.displayOrder,
        enabled: false,
        links: detail.links,
      })
      await load()
    } catch (saveErr) {
      setError(saveErr instanceof Error ? saveErr.message : '停用规则失败')
    } finally {
      setSaving(false)
    }
  }

  const addLink = () => {
    const fallbackCode = indicatorOptions[0]?.code || ''
    if (!fallbackCode) {
      return
    }
    setEditorDraft((previous) => ({
      ...previous,
      links: [
        ...previous.links,
        {
          ruleCode: previous.name,
          indicatorCode: fallbackCode,
          relationLabel: '',
          weight: 0.5,
          displayOrder: (previous.links.length + 1) * 10,
        },
      ],
    }))
  }

  const updateLink = (index: number, patch: Partial<RuleIndicatorLink>) => {
    setEditorDraft((previous) => ({
      ...previous,
      links: previous.links.map((link, idx) => {
        if (idx !== index) {
          return link
        }
        return {
          ...link,
          ...patch,
        }
      }),
    }))
  }

  const removeLink = (index: number) => {
    setEditorDraft((previous) => ({
      ...previous,
      links: previous.links.filter((_, idx) => idx !== index),
    }))
  }

  const upsertPreferenceWeight = (indicatorCode: string, relationLabel: string, weight: number, order: number) => {
    setEditorDraft((previous) => {
      const nextLinks = [...previous.links]
      const index = nextLinks.findIndex((item) => item.indicatorCode === indicatorCode)
      if (index >= 0) {
        nextLinks[index] = {
          ...nextLinks[index],
          weight,
          relationLabel: nextLinks[index].relationLabel || relationLabel,
        }
      } else {
        nextLinks.push({
          ruleCode: previous.code,
          indicatorCode,
          relationLabel,
          weight,
          displayOrder: order,
        })
      }
      return {
        ...previous,
        links: nextLinks,
      }
    })
  }

  const getWeight = (indicatorCode: string, fallback: number) => {
    const target = editorDraft.links.find((item) => item.indicatorCode === indicatorCode)
    return target ? target.weight : fallback
  }

  return (
    <div className="min-h-full bg-[#F8FAFC] px-8 py-6">
      <div className="space-y-5">
        <header className="flex h-12 items-center justify-between">
          <div className="space-y-0.5">
            <h1 className="text-[20px] font-semibold leading-[28px] text-[#0F172A]">计算规则配置</h1>
            <p className="text-[13px] text-[#64748B]">规则定义、联动关系与权重参数统一落库，支持新增与修改</p>
          </div>
          <Button className="h-9 gap-1.5 bg-[#3B82F6] px-4 text-[13px] font-medium text-white hover:bg-[#2563EB]" onClick={openCreate}>
            <Plus className="h-4 w-4" />
            新建规则
          </Button>
        </header>

        {loading && <div className="text-sm text-[#64748B]">加载中...</div>}


        {!loading && (
          <div className="flex items-center gap-2 text-xs text-[#64748B]">
            <span>通过 {evaluationSummary.pass}</span>
            <span>违规 {evaluationSummary.fail}</span>
            <span>待补充 {evaluationSummary.skipped}</span>
          </div>
        )}

        {!loading &&
          sorted.map((item) => {
            const check = evaluations[item.rule.ruleCode]
            const status = check?.status || 'skipped'
            return (
            <article key={item.rule.ruleCode} className="space-y-3 rounded-xl border border-[#E2E8F0] bg-white p-5">
              <div className="flex items-center justify-between gap-3">
                <div className="flex min-w-0 items-center gap-2.5">
                  <h3 className="truncate text-[14px] font-semibold text-[#0F172A]">{item.rule.name}</h3>
                  <Badge className="h-5 rounded-full bg-[#DCFCE7] px-2 text-[11px] text-[#16A34A]">
                    {item.rule.enabled ? '已启用' : '已停用'}
                  </Badge>
                  <Badge className={`h-5 rounded-full px-2 text-[11px] ${severityBadgeClass[item.rule.severity]}`}>
                    {severityLabel[item.rule.severity]}
                  </Badge>
                  <Badge className={`h-5 rounded-full px-2 text-[11px] ${ruleStatusBadgeClass[status] || ruleStatusBadgeClass.skipped}`}>
                    {ruleStatusLabel[status] || ruleStatusLabel.skipped}
                  </Badge>
                </div>

                <div className="flex items-center gap-2">
                  <Button
                    variant="outline"
                    className="h-8 w-8 rounded-md border-[#E2E8F0] p-0 text-[#64748B]"
                    onClick={() => openEdit(item)}
                    title="编辑"
                  >
                    <Edit3 className="h-3.5 w-3.5" />
                  </Button>
                  <Button
                    variant="outline"
                    className="h-8 w-8 rounded-md border-[#FEE2E2] p-0 text-[#EF4444] hover:bg-[#FEF2F2]"
                    onClick={() => disableRule(item)}
                    disabled={!item.rule.enabled || saving}
                    title="停用"
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                  </Button>
                </div>
              </div>

              <p className="text-[13px] text-[#64748B]">{item.rule.description || '暂无描述'}</p>
              <p className="text-[12px] text-[#94A3B8]">表达式：{item.rule.expression || '-'}</p>


              {check && (
                <div className="rounded-lg bg-[#F8FAFC] px-3 py-2 text-[12px] text-[#475569]">
                  <div>{check.message}</div>
                  {check.skippedReason && <div className="mt-0.5 text-[#94A3B8]">原因：{check.skippedReason}</div>}
                </div>
              )}



              {check?.failedIndicators && check.failedIndicators.length > 0 && (
                <div className="flex flex-wrap gap-2">
                  {check.failedIndicators.map((indicator) => (
                    <Badge key={`${item.rule.ruleCode}-${indicator.indicatorCode}`} className="h-6 rounded-md bg-[#FEE2E2] px-2 text-[11px] text-[#B91C1C]">
                      {indicator.indicatorName || indicator.indicatorCode}：{indicator.value}
                    </Badge>
                  ))}
                </div>
              )}

              {item.links.length > 0 && (
                <div className="space-y-2">
                  {item.links.slice(0, 3).map((link, index) => (
                    <div
                      key={`${item.rule.ruleCode}-${link.indicatorCode}-${link.displayOrder}`}
                      className="flex h-12 items-center justify-between rounded-lg border border-[#E2E8F0] px-3"
                    >
                      <div className="flex items-center gap-2 text-[13px] text-[#0F172A]">
                        <GitBranch className="h-4 w-4 text-[#3B82F6]" />
                        {link.relationLabel || link.indicatorCode}
                      </div>
                      <span
                        className={`rounded px-2 py-1 text-[11px] ${linkBadgePalette[index % linkBadgePalette.length]}`}
                      >
                        权重 {link.weight}
                      </span>
                    </div>
                  ))}
                </div>
              )}
            </article>
            )
          })}

        {error && <div className="text-xs text-[#DC2626]">{error}</div>}
      </div>

      <Dialog open={editorOpen} onOpenChange={setEditorOpen}>
        <DialogContent className="max-w-[720px] overflow-hidden rounded-2xl border border-[#E2E8F0] p-0">
          <DialogHeader className="border-b border-[#E2E8F0] px-5 py-4">
            <DialogTitle className="text-[16px] font-semibold text-[#0F172A]">编辑计算规则</DialogTitle>
            <DialogDescription>规则定义将作为系统默认规则参与智能调整与提示策略。</DialogDescription>
          </DialogHeader>

          <div className="max-h-[580px] space-y-6 overflow-y-auto px-6 py-6">
            <section className="space-y-4">
              <h3 className="text-[14px] font-semibold text-[#0F172A]">基本信息</h3>

              <div className="grid grid-cols-[90px_1fr] items-center gap-3">
                <Label className="text-[13px] text-[#64748B]">规则名称</Label>
                <Input
                  value={editorDraft.name}
                  onChange={(event) => setEditorDraft((prev) => ({ ...prev, name: event.target.value }))}
                  className="h-10 rounded-lg border-[#E2E8F0] text-[13px]"
                />
              </div>
              <div className="grid grid-cols-[90px_1fr] items-center gap-3">
                <Label className="text-[13px] text-[#64748B]">规则描述</Label>
                <Input
                  value={editorDraft.description}
                  onChange={(event) => setEditorDraft((prev) => ({ ...prev, description: event.target.value }))}
                  className="h-10 rounded-lg border-[#E2E8F0] text-[13px]"
                />
              </div>

              <div className="grid grid-cols-[90px_1fr] items-center gap-3">
                <Label className="text-[13px] text-[#64748B]">规则表达式</Label>
                <Input
                  value={editorDraft.expression}
                  onChange={(event) => setEditorDraft((prev) => ({ ...prev, expression: event.target.value }))}
                  className="h-10 rounded-lg border-[#E2E8F0] text-[13px]"
                />
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1.5">
                  <Label className="text-xs text-[#64748B]">严重级别</Label>
                  <Select
                    value={editorDraft.severity}
                    onValueChange={(value: RuleSeverity) => setEditorDraft((prev) => ({ ...prev, severity: value }))}
                  >
                    <SelectTrigger className="h-10 text-xs">
                      <SelectValue placeholder="选择级别" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="info">提示</SelectItem>
                      <SelectItem value="warn">预警</SelectItem>
                      <SelectItem value="error">错误</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-1.5">
                  <Label className="text-xs text-[#64748B]">显示顺序</Label>
                  <Input
                    value={editorDraft.displayOrder}
                    onChange={(event) => setEditorDraft((prev) => ({ ...prev, displayOrder: Number(event.target.value) || 0 }))}
                    className="h-10 text-xs"
                  />
                </div>
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1.5">
                  <Label className="text-xs text-[#64748B]">建议文案</Label>
                  <Input
                    value={editorDraft.suggestion}
                    onChange={(event) => setEditorDraft((prev) => ({ ...prev, suggestion: event.target.value }))}
                    className="h-10 text-xs"
                  />
                </div>
                <div className="space-y-1.5">
                  <Label className="text-xs text-[#64748B]">偏好参数 JSON</Label>
                  <Input
                    value={editorDraft.preferenceJson}
                    onChange={(event) => setEditorDraft((prev) => ({ ...prev, preferenceJson: event.target.value }))}
                    className="h-10 text-xs"
                  />
                </div>
              </div>

              <div className="flex items-center justify-between rounded-lg border border-[#E2E8F0] px-3 py-2.5">
                <div>
                  <div className="text-xs font-medium text-[#0F172A]">规则启用</div>
                  <div className="text-[11px] text-[#94A3B8]">停用后规则仍保留定义，但不参与默认执行。</div>
                </div>
                <Switch
                  checked={editorDraft.enabled}
                  onCheckedChange={(checked) => setEditorDraft((prev) => ({ ...prev, enabled: checked }))}
                />
              </div>
            </section>

            <section className="space-y-4">
              <h3 className="text-[14px] font-semibold text-[#0F172A]">计算偏好</h3>
              <div className="flex flex-wrap items-end gap-3">
                <div className="space-y-1.5">
                  <Label className="text-[13px] text-[#64748B]">小微权重</Label>
                  <Input
                    value={getWeight('小微企业增速_当月', 0.3)}
                    onChange={(event) => {
                      const value = Number(event.target.value) || 0
                      upsertPreferenceWeight('小微企业增速_当月', '小微增速 → 限下累计估算', value, 10)
                    }}
                    className="h-10 w-[120px] rounded-lg border-[#E2E8F0] text-[13px]"
                  />
                </div>

                <div className="space-y-1.5">
                  <Label className="text-[13px] text-[#64748B]">吃穿用权重</Label>
                  <Input
                    value={getWeight('吃穿用增速_当月', 0.3)}
                    onChange={(event) => {
                      const value = Number(event.target.value) || 0
                      upsertPreferenceWeight('吃穿用增速_当月', '吃穿用增速 → 限下累计估算', value, 20)
                    }}
                    className="h-10 w-[120px] rounded-lg border-[#E2E8F0] text-[13px]"
                  />
                </div>

                <div className="space-y-1.5">
                  <Label className="text-[13px] text-[#64748B]">抽样权重</Label>
                  <Input
                    value={getWeight('限上社零额增速_当月', 0.4)}
                    onChange={(event) => {
                      const value = Number(event.target.value) || 0
                      upsertPreferenceWeight('限上社零额增速_当月', '抽样增速代理 → 限下累计估算', value, 30)
                    }}
                    className="h-10 w-[120px] rounded-lg border-[#E2E8F0] text-[13px]"
                  />
                </div>
              </div>
            </section>

            <section className="space-y-3">
              <div>
                <h3 className="text-[14px] font-semibold text-[#0F172A]">指标联动关系</h3>
                <p className="mt-0.5 text-[12px] text-[#94A3B8]">配置该规则影响的指标及其依赖关系</p>
              </div>

              <div className="space-y-2">
                {editorDraft.links.length === 0 && <div className="text-xs text-[#94A3B8]">暂无联动项</div>}

                {editorDraft.links.map((link, index) => (
                  <div
                    key={`${link.indicatorCode}-${index}`}
                    className="grid grid-cols-[1.2fr_1fr_90px_32px] items-center gap-2 rounded-lg border border-[#E2E8F0] p-2"
                  >
                    <Select
                      value={link.indicatorCode}
                      onValueChange={(value) => updateLink(index, { indicatorCode: value })}
                    >
                      <SelectTrigger className="h-9 text-xs">
                        <SelectValue placeholder="关联指标" />
                      </SelectTrigger>
                      <SelectContent>
                        {indicatorOptions.map((option) => (
                          <SelectItem key={option.code} value={option.code}>
                            {option.label}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>

                    <Input
                      value={link.relationLabel}
                      onChange={(event) => updateLink(index, { relationLabel: event.target.value })}
                      placeholder="关系说明"
                      className="h-9 text-xs"
                    />

                    <Input
                      value={link.weight}
                      onChange={(event) => updateLink(index, { weight: Number(event.target.value) || 0 })}
                      placeholder="权重"
                      className="h-9 text-xs"
                    />

                    <Button
                      variant="ghost"
                      className="h-8 w-8 p-0 text-[#EF4444] hover:bg-[#FEF2F2]"
                      onClick={() => removeLink(index)}
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                    </Button>
                  </div>
                ))}
              </div>

              <Button variant="outline" className="h-8 gap-1.5 px-3 text-xs" onClick={addLink}>
                <Plus className="h-3.5 w-3.5" />
                新增联动
              </Button>
            </section>
          </div>

          <DialogFooter className="h-16 border-t border-[#E2E8F0] px-5">
            <Button variant="outline" className="h-10" onClick={() => setEditorOpen(false)}>
              取消
            </Button>
            <Button onClick={saveRule} disabled={saving} className="h-10 gap-1.5 bg-[#3B82F6] px-5 text-white hover:bg-[#2563EB]">
              <Save className="h-4 w-4" />
              {saving ? '保存中' : '保存规则'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
