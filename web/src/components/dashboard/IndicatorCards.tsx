import { Card } from '@/components/ui/card'
import { Input } from '@/components/ui/input'

export interface IndicatorItem {
  id: string
  name: string
  value: number
  unit: string
}

export interface IndicatorGroup {
  name: string
  indicators: IndicatorItem[]
}

interface IndicatorCardsProps {
  groups: IndicatorGroup[]
  highlightIndicators?: Record<string, boolean>
  draftTargets: Record<string, string>
  onDraftChange: (id: string, value: string) => void
  onEnterApply: (id: string) => void
  onPreview: (id: string) => void
}

const formatValue = (value: number, unit: string) => {
  if (unit.includes('%')) {
    return Math.round(value).toString()
  }
  return Math.round(value).toLocaleString()
}

export default function IndicatorCards(props: IndicatorCardsProps) {
  const groups = props.groups.slice(0, 4)

  return (
    <div className="grid grid-cols-1 gap-4 2xl:grid-cols-4">
      {groups.map((group) => (
        <Card key={group.name} className="rounded-xl border border-[#E2E8F0] bg-white p-4 shadow-none">
          <div className="mb-3 text-[13px] font-medium text-[#64748B]">{group.name}</div>
          <div className="space-y-2">
            {group.indicators.map((indicator) => {
              const active = !!props.highlightIndicators?.[indicator.id]
              const rawDraft = props.draftTargets[indicator.id]
              const display = rawDraft !== undefined ? rawDraft : formatValue(indicator.value, indicator.unit)

              return (
                <div key={indicator.id} className="flex items-center justify-between gap-3">
                  <div className="min-w-0 text-[12px] text-[#64748B]">{indicator.name}</div>
                  <div className="flex items-center gap-2">
                    <Input
                      value={display}
                      onClick={(event) => {
                        event.stopPropagation()
                        props.onPreview(indicator.id)
                      }}
                      onChange={(event) => props.onDraftChange(indicator.id, event.target.value)}
                      onKeyDown={(event) => {
                        if (event.key === 'Enter') {
                          event.preventDefault()
                          props.onEnterApply(indicator.id)
                        }
                      }}
                      className={`h-7 w-[88px] rounded-full px-2 text-right text-[13px] font-medium tabular-nums transition-colors ${
                        active
                          ? 'border-yellow-400 ring-1 ring-yellow-400/50 bg-[#FFFBEB] text-[#92400E]'
                          : 'border-[#E2E8F0] bg-white text-[#0F172A]'
                      }`}
                    />
                    <span className="w-9 text-right text-[11px] text-[#94A3B8]">{indicator.unit}</span>
                  </div>
                </div>
              )
            })}
          </div>
        </Card>
      ))}
    </div>
  )
}
