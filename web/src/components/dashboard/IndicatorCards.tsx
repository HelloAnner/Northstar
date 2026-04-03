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
  draftTargets: Record<string, string>
  highlightIndicators?: Record<string, boolean>
  changedIndicators?: Record<string, boolean>
  failedIndicators?: Record<string, boolean>
  disabled?: boolean
  compact?: boolean
  onDraftChange: (id: string, value: string) => void
  onEnterApply: (id: string, value: string) => void
  onPreview: (id: string) => void
}

const formatValue = (value: number, unit: string) => {
  if (unit.includes('%')) return Math.round(value).toString()
  return Math.round(value).toLocaleString()
}

export default function IndicatorCards(props: IndicatorCardsProps) {
  const groups = props.groups.slice(0, 4)
  const gridCols = props.compact ? 'lg:grid-cols-2' : 'lg:grid-cols-4'

  return (
    <div className={`grid grid-cols-1 gap-4 ${gridCols}`}>
      {groups.map((group) => (
        <Card
          key={group.name}
          className="border-border/60 bg-card/60 p-4 backdrop-blur supports-[backdrop-filter]:bg-card/50"
        >
          <div className="mb-3 text-[13px] font-medium text-muted-foreground">
            {group.name}
          </div>
          <div className="space-y-2">
            {group.indicators.map((indicator) => {
              const active = !!props.highlightIndicators?.[indicator.id]
              const changed = !!props.changedIndicators?.[indicator.id]
              const failed = !!props.failedIndicators?.[indicator.id]
              const rawDraft = props.draftTargets[indicator.id]
              const formatted = formatValue(indicator.value, indicator.unit)
              const display = rawDraft !== undefined ? rawDraft : formatted
              const dirty = rawDraft !== undefined && rawDraft !== formatted
              const isRate = indicator.unit.includes('%')
              const positive = indicator.value >= 0

              const tone = isRate
                ? positive
                  ? 'text-emerald-600 dark:text-emerald-400'
                  : 'text-rose-600 dark:text-rose-400'
                : 'text-foreground'

              // priority: green changed > yellow linkage > orange dirty > red failed > default
              const borderTone = changed
                ? 'border-emerald-400 ring-2 ring-emerald-400/40 transition-all duration-300'
                : active
                  ? 'border-amber-400 ring-1 ring-amber-400/40'
                  : dirty
                    ? 'border-orange-400/70 ring-1 ring-orange-400/30'
                    : failed
                      ? 'border-rose-400/70 ring-1 ring-rose-400/30'
                      : 'border-border/60'

              return (
                <div key={indicator.id} className="flex items-center justify-between gap-3">
                  <div className="flex min-w-0 items-center gap-1.5">
                    <div className="min-w-0 truncate text-[12px] text-muted-foreground">
                      {indicator.name}
                    </div>
                    {failed && (
                      <span
                        title="该指标命中规则约束"
                        className="inline-flex h-4 shrink-0 items-center rounded-full bg-rose-100 px-1.5 text-[10px] text-rose-700 dark:bg-rose-900/30 dark:text-rose-400"
                      >
                        违规
                      </span>
                    )}
                  </div>
                  <div className="flex shrink-0 items-center gap-2">
                    <Input
                      value={display}
                      disabled={props.disabled}
                      onClick={(e) => {
                        e.stopPropagation()
                        props.onPreview(indicator.id)
                      }}
                      onChange={(e) => props.onDraftChange(indicator.id, e.target.value)}
                      onKeyDown={(e) => {
                        if (e.key === 'Enter') {
                          e.preventDefault()
                          props.onEnterApply(indicator.id, display)
                        }
                      }}
                      className={`h-8 rounded-full px-3 text-right font-mono text-sm tabular-nums ${
                        isRate ? 'w-[110px]' : 'w-[150px]'
                      } ${tone} ${borderTone}`}
                    />
                    <span className="w-9 text-right text-[11px] text-muted-foreground">
                      {indicator.unit}
                    </span>
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
