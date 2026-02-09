import { useEffect, useMemo, useState } from 'react'
import { Plus, Save, Trash2 } from 'lucide-react'
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
import { Switch } from '@/components/ui/switch'
import FormulaEditor from '@/components/design/FormulaEditor'
import { designApi } from '@/services/designApi'
import type { IndicatorDefinition, UpsertIndicatorDefinitionPayload } from '@/types/design'

const formulaReferenceTokens = [
  '批零零售额_当月汇总',
  '批零零售额_上年当月汇总',
  '批零零售额_累计汇总',
  '批零零售额_上年累计汇总',
  '住餐折算零售额_当月汇总',
  '住餐折算零售额_上年当月汇总',
  '住餐折算零售额_累计汇总',
  '住餐折算零售额_上年累计汇总',
  '吃穿用零售额_当月汇总',
  '吃穿用零售额_上年当月汇总',
  '小微零售额_当月汇总',
  '小微零售额_上年当月汇总',
  '批发销售额_当月汇总',
  '批发销售额_上年当月汇总',
  '批发销售额_累计汇总',
  '批发销售额_上年累计汇总',
  '零售销售额_当月汇总',
  '零售销售额_上年当月汇总',
  '零售销售额_累计汇总',
  '零售销售额_上年累计汇总',
  '住宿营业额_当月汇总',
  '住宿营业额_上年当月汇总',
  '住宿营业额_累计汇总',
  '住宿营业额_上年累计汇总',
  '餐饮营业额_当月汇总',
  '餐饮营业额_上年当月汇总',
  '餐饮营业额_累计汇总',
  '餐饮营业额_上年累计汇总',
  '小微增速_上月配置',
  '吃穿用增速_上月配置',
  '抽样增速_上月配置',
  '小微增速_本月配置',
  '吃穿用增速_本月配置',
  '抽样增速_本月配置',
  '小微权重_配置',
  '吃穿用权重_配置',
  '抽样权重_配置',
  '全省限下增速变动量_配置',
  '限下累计估算_上年值',
]

const groupByCode = (items: IndicatorDefinition[]) => {
  const order: string[] = []
  const map = new Map<string, { name: string; order: number; items: IndicatorDefinition[] }>()
  for (const item of items) {
    if (!map.has(item.groupCode)) {
      map.set(item.groupCode, { name: item.groupName, order: item.groupOrder, items: [] })
      order.push(item.groupCode)
    }
    map.get(item.groupCode)?.items.push(item)
  }

  return order
    .map((code) => ({ code, ...map.get(code)! }))
    .sort((left, right) => left.order - right.order)
    .map((group) => ({
      ...group,
      items: [...group.items].sort((left, right) => left.displayOrder - right.displayOrder),
    }))
}

const createIndicatorDraft = (): IndicatorDefinition => ({
  code: '',
  name: '',
  groupCode: '自定义指标',
  groupName: '自定义指标',
  groupOrder: 99,
  description: '',
  formula: '',
  unit: '%',
  floatMin: -5,
  floatMax: 5,
  displayOrder: 999,
  enabled: true,
})

const toPayload = (item: IndicatorDefinition): UpsertIndicatorDefinitionPayload => ({
  name: item.name.trim(),
  groupCode: item.groupCode.trim() || item.groupName.trim(),
  groupName: item.groupName.trim(),
  groupOrder: item.groupOrder,
  description: item.description,
  formula: item.formula,
  unit: item.unit,
  floatMin: item.floatMin,
  floatMax: item.floatMax,
  displayOrder: item.displayOrder,
  enabled: item.enabled,
})

export default function IndicatorsPage() {
  const [loading, setLoading] = useState(true)
  const [savingName, setSavingName] = useState('')
  const [items, setItems] = useState<IndicatorDefinition[]>([])
  const [error, setError] = useState('')

  const [createOpen, setCreateOpen] = useState(false)
  const [createDraft, setCreateDraft] = useState<IndicatorDefinition>(() => createIndicatorDraft())

  const load = async () => {
    setLoading(true)
    setError('')
    try {
      const definitions = await designApi.listIndicatorDefinitions(false)
      setItems(definitions)
    } catch (loadErr) {
      setError(loadErr instanceof Error ? loadErr.message : '加载指标定义失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void load()
  }, [])

  const grouped = useMemo(() => groupByCode(items), [items])

  const formulaTokens = useMemo(() => {
    const all = new Set<string>(formulaReferenceTokens)
    for (const item of items) {
      all.add(item.name)
    }
    return [...all]
  }, [items])

  const saveDefinition = async (marker: string, definition: IndicatorDefinition) => {
    const name = definition.name.trim()
    if (!name) {
      setError('指标名称不能为空')
      return
    }
    if (!definition.formula.trim()) {
      setError('计算公式不能为空')
      return
    }

    setSavingName(name)
    setError('')
    try {
      await designApi.upsertIndicatorDefinition(marker || name, toPayload({ ...definition, code: name, name }))
      await load()
    } catch (saveErr) {
      setError(saveErr instanceof Error ? saveErr.message : '保存指标定义失败')
    } finally {
      setSavingName('')
    }
  }

  const disableDefinition = async (definition: IndicatorDefinition) => {
    await saveDefinition(definition.code || definition.name, {
      ...definition,
      enabled: false,
    })
  }

  const createDefinition = async () => {
    const name = createDraft.name.trim()
    if (!name) {
      setError('请输入指标名称')
      return
    }
    if (!createDraft.formula.trim()) {
      setError('请输入公式')
      return
    }

    setSavingName(name)
    setError('')
    try {
      const groupName = createDraft.groupName.trim() || '自定义指标'
      await designApi.upsertIndicatorDefinition(name, toPayload({
        ...createDraft,
        code: name,
        name,
        groupName,
        groupCode: groupName,
      }))
      setCreateOpen(false)
      setCreateDraft(createIndicatorDraft())
      await load()
    } catch (saveErr) {
      setError(saveErr instanceof Error ? saveErr.message : '创建指标失败')
    } finally {
      setSavingName('')
    }
  }

  const onFieldChange = <K extends keyof IndicatorDefinition>(
    marker: string,
    key: K,
    value: IndicatorDefinition[K],
  ) => {
    setItems((previous) => {
      return previous.map((item) => {
        if ((item.code || item.name) !== marker) {
          return item
        }
        return {
          ...item,
          [key]: value,
        }
      })
    })
  }

  return (
    <div className="min-h-full bg-[#F8FAFC] px-8 py-6">
      <div className="space-y-5">
        <header className="flex h-12 items-center justify-between">
          <div className="space-y-0.5">
            <h1 className="text-[20px] font-semibold leading-[28px] text-[#0F172A]">指标中心</h1>
            <p className="text-[13px] text-[#64748B]">管理指标定义、公式与浮动区间，名称为唯一标识</p>
          </div>
          <Button
            className="h-9 gap-1.5 rounded-lg bg-[#3B82F6] px-4 text-[13px] font-medium text-white hover:bg-[#2563EB]"
            onClick={() => {
              setCreateDraft(createIndicatorDraft())
              setCreateOpen(true)
            }}
          >
            <Plus className="h-4 w-4" />
            新建指标
          </Button>
        </header>

        {loading && <div className="text-sm text-[#64748B]">加载中...</div>}

        {!loading && grouped.length === 0 && <div className="text-sm text-[#64748B]">暂无指标定义</div>}

        {!loading &&
          grouped.map((group) => (
            <section key={group.code} className="space-y-3">
              <div className="flex items-center gap-2">
                <h2 className="text-[13px] font-semibold text-[#0F172A]">{group.name}</h2>
                <Badge variant="secondary" className="h-5 rounded-full bg-[#F1F5F9] px-2.5 text-[11px] text-[#475569]">
                  {group.items.length} 项
                </Badge>
              </div>

              <div className="grid gap-3">
                {group.items.map((item) => {
                  const marker = item.code || item.name
                  return (
                    <article key={marker} className="rounded-xl border border-[#E2E8F0] bg-white px-4 py-3 shadow-sm">
                      <div className="flex items-start justify-between gap-3">
                        <div className="space-y-1">
                          <div className="text-[13px] font-semibold text-[#0F172A]">{item.name}</div>
                          <div className="text-[12px] text-[#64748B]">{item.description || '暂无说明'}</div>
                        </div>

                        <div className="flex items-center gap-2">
                          <Button
                            variant="outline"
                            className="h-8 gap-1 rounded-lg border-[#E2E8F0] px-3 text-xs"
                            disabled={savingName === item.name}
                            onClick={() => saveDefinition(marker, item)}
                          >
                            <Save className="h-3.5 w-3.5" />
                            {savingName === item.name ? '保存中' : '保存'}
                          </Button>
                          <Button
                            variant="ghost"
                            className="h-8 gap-1 rounded-lg px-2 text-xs text-[#EF4444] hover:bg-[#FEF2F2]"
                            onClick={() => disableDefinition(item)}
                          >
                            <Trash2 className="h-3.5 w-3.5" />
                            停用
                          </Button>
                        </div>
                      </div>

                      <div className="mt-3 grid gap-3">
                        <div className="space-y-1.5">
                          <Label className="text-xs text-[#94A3B8]">描述</Label>
                          <Input
                            value={item.description}
                            onChange={(event) => onFieldChange(marker, 'description', event.target.value)}
                            className="h-9 rounded-lg border-[#E2E8F0] px-3 text-[13px]"
                          />
                        </div>

                        <div className="space-y-1.5">
                          <Label className="text-xs text-[#94A3B8]">计算公式（支持模糊搜索与自动提示）</Label>
                          <FormulaEditor
                            value={item.formula}
                            onChange={(next) => onFieldChange(marker, 'formula', next)}
                            indicatorNames={items.map((current) => current.name)}
                            extraTokens={formulaTokens}
                          />
                        </div>

                        <div className="flex items-end gap-2.5">
                          <div className="space-y-1.5">
                            <Label className="text-xs text-[#94A3B8]">下限</Label>
                            <Input
                              value={item.floatMin}
                              onChange={(event) => onFieldChange(marker, 'floatMin', Number(event.target.value) || 0)}
                              className="h-7 w-24 rounded-md border-[#E2E8F0] px-2 text-center text-[12px]"
                            />
                          </div>

                          <div className="pb-1 text-xs text-[#94A3B8]">~</div>

                          <div className="space-y-1.5">
                            <Label className="text-xs text-[#94A3B8]">上限</Label>
                            <Input
                              value={item.floatMax}
                              onChange={(event) => onFieldChange(marker, 'floatMax', Number(event.target.value) || 0)}
                              className="h-7 w-24 rounded-md border-[#E2E8F0] px-2 text-center text-[12px]"
                            />
                          </div>

                          <div className="space-y-1.5">
                            <Label className="text-xs text-[#94A3B8]">单位</Label>
                            <Input
                              value={item.unit}
                              onChange={(event) => onFieldChange(marker, 'unit', event.target.value)}
                              className="h-7 w-20 rounded-md border-[#E2E8F0] px-2 text-center text-[12px]"
                            />
                          </div>

                          <div className="ml-auto flex items-center gap-2 pb-1">
                            <Label className="text-xs text-[#94A3B8]">启用</Label>
                            <Switch
                              checked={item.enabled}
                              onCheckedChange={(checked) => onFieldChange(marker, 'enabled', checked)}
                            />
                          </div>
                        </div>
                      </div>
                    </article>
                  )
                })}
              </div>
            </section>
          ))}

        {error && <div className="text-xs text-[#DC2626]">{error}</div>}
      </div>

      <Dialog open={createOpen} onOpenChange={setCreateOpen}>
        <DialogContent className="max-w-[760px] rounded-2xl border-[#E2E8F0] p-0">
          <DialogHeader className="border-b border-[#E2E8F0] px-6 py-4">
            <DialogTitle className="text-[#0F172A]">新建指标定义</DialogTitle>
            <DialogDescription>新增后立即进入数据库定义，并参与计算与规则联动。</DialogDescription>
          </DialogHeader>

          <div className="grid grid-cols-2 gap-4 px-6 py-5">
            <div className="col-span-2 space-y-1.5">
              <Label className="text-xs text-[#64748B]">指标名称（唯一标识）</Label>
              <Input
                value={createDraft.name}
                onChange={(event) => {
                  const value = event.target.value
                  setCreateDraft((prev) => ({ ...prev, name: value, code: value }))
                }}
                placeholder="请输入中文指标名称"
                className="h-9 text-xs"
              />
            </div>

            <div className="space-y-1.5">
              <Label className="text-xs text-[#64748B]">分组名称</Label>
              <Input
                value={createDraft.groupName}
                onChange={(event) => {
                  const value = event.target.value
                  setCreateDraft((prev) => ({ ...prev, groupName: value, groupCode: value }))
                }}
                className="h-9 text-xs"
              />
            </div>

            <div className="space-y-1.5">
              <Label className="text-xs text-[#64748B]">分组顺序</Label>
              <Input
                value={createDraft.groupOrder}
                onChange={(event) => setCreateDraft((prev) => ({ ...prev, groupOrder: Number(event.target.value) || 0 }))}
                className="h-9 text-xs"
              />
            </div>

            <div className="col-span-2 space-y-1.5">
              <Label className="text-xs text-[#64748B]">计算公式</Label>
              <FormulaEditor
                value={createDraft.formula}
                onChange={(next) => setCreateDraft((prev) => ({ ...prev, formula: next }))}
                indicatorNames={items.map((item) => item.name)}
                extraTokens={formulaTokens}
                placeholder="例如：同比增速(限上社零额_累计值, 批零零售额_上年累计汇总 + 住餐折算零售额_上年累计汇总)"
              />
            </div>

            <div className="space-y-1.5">
              <Label className="text-xs text-[#64748B]">单位</Label>
              <Input
                value={createDraft.unit}
                onChange={(event) => setCreateDraft((prev) => ({ ...prev, unit: event.target.value }))}
                className="h-9 text-xs"
              />
            </div>

            <div className="space-y-1.5">
              <Label className="text-xs text-[#64748B]">说明</Label>
              <Input
                value={createDraft.description}
                onChange={(event) => setCreateDraft((prev) => ({ ...prev, description: event.target.value }))}
                className="h-9 text-xs"
              />
            </div>

            <div className="space-y-1.5">
              <Label className="text-xs text-[#64748B]">下限</Label>
              <Input
                value={createDraft.floatMin}
                onChange={(event) => setCreateDraft((prev) => ({ ...prev, floatMin: Number(event.target.value) || 0 }))}
                className="h-9 text-xs"
              />
            </div>
            <div className="space-y-1.5">
              <Label className="text-xs text-[#64748B]">上限</Label>
              <Input
                value={createDraft.floatMax}
                onChange={(event) => setCreateDraft((prev) => ({ ...prev, floatMax: Number(event.target.value) || 0 }))}
                className="h-9 text-xs"
              />
            </div>
          </div>

          <DialogFooter className="h-16 border-t border-[#E2E8F0] px-6">
            <Button variant="outline" onClick={() => setCreateOpen(false)}>
              取消
            </Button>
            <Button
              onClick={createDefinition}
              disabled={savingName === createDraft.name.trim()}
              className="gap-1.5 bg-[#3B82F6] text-white hover:bg-[#2563EB]"
            >
              <Save className="h-4 w-4" />
              {savingName === createDraft.name.trim() ? '保存中' : '创建并保存'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
