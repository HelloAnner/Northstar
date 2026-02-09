import { useMemo, useRef, useState } from 'react'
import { Input } from '@/components/ui/input'
import { cn } from '@/lib/utils'

type TokenType = 'indicator' | 'operator' | 'function' | 'number' | 'plain'

type SuggestionItem = {
  value: string
  type: TokenType
}

type FormulaEditorProps = {
  value: string
  onChange: (next: string) => void
  indicatorNames: string[]
  extraTokens?: string[]
  placeholder?: string
  inputClassName?: string
}

const operatorTokens = ['+', '-', '*', '/', '(', ')', ',', '&&', '||', '<=', '>=', '==', '!=', '<', '>']
const functionTokens = ['同比增速', '绝对值', '最小值', '最大值', '四舍五入']

const tokenClassName: Record<TokenType, string> = {
  indicator: 'rounded bg-[#DBEAFE] px-1.5 py-0.5 text-[#1D4ED8]',
  operator: 'rounded bg-[#E2E8F0] px-1.5 py-0.5 text-[#475569]',
  function: 'rounded bg-[#DCFCE7] px-1.5 py-0.5 text-[#166534]',
  number: 'rounded bg-[#FEF3C7] px-1.5 py-0.5 text-[#92400E]',
  plain: 'text-[#0F172A]',
}

const fuzzyMatch = (source: string, keyword: string) => {
  if (!keyword) {
    return true
  }
  const sourceRunes = Array.from(source.toLowerCase())
  const keywordRunes = Array.from(keyword.toLowerCase())
  let index = 0
  for (const char of keywordRunes) {
    while (index < sourceRunes.length && sourceRunes[index] !== char) {
      index += 1
    }
    if (index >= sourceRunes.length) {
      return false
    }
    index += 1
  }
  return true
}

const tokenizeFormula = (value: string) => {
  return value.match(/\s+|\&\&|\|\||<=|>=|==|!=|[()+\-*/,<>]|[^\s()+\-*/,<>=!&|]+/g) ?? []
}

const detectTokenType = (token: string, indicators: Set<string>, extras: Set<string>): TokenType => {
  const pure = token.trim()
  if (!pure) {
    return 'plain'
  }
  if (operatorTokens.includes(pure)) {
    return 'operator'
  }
  if (!Number.isNaN(Number(pure))) {
    return 'number'
  }
  if (functionTokens.includes(pure)) {
    return 'function'
  }
  if (indicators.has(pure) || extras.has(pure)) {
    return 'indicator'
  }
  return 'plain'
}

export default function FormulaEditor(props: FormulaEditorProps) {
  const { value, onChange, indicatorNames, extraTokens = [], placeholder, inputClassName } = props
  const inputRef = useRef<HTMLInputElement | null>(null)
  const [focus, setFocus] = useState(false)

  const indicatorSet = useMemo(() => new Set(indicatorNames), [indicatorNames])
  const extraSet = useMemo(() => new Set(extraTokens), [extraTokens])

  const allSuggestions = useMemo(() => {
    const items: SuggestionItem[] = []
    for (const item of indicatorNames) {
      items.push({ value: item, type: 'indicator' })
    }
    for (const item of extraTokens) {
      items.push({ value: item, type: 'indicator' })
    }
    for (const item of functionTokens) {
      items.push({ value: item, type: 'function' })
    }
    for (const item of operatorTokens) {
      items.push({ value: item, type: 'operator' })
    }
    return items
  }, [extraTokens, indicatorNames])

  const tokenMeta = useMemo(() => {
    const tokens = tokenizeFormula(value)
    return tokens.map((item) => ({ token: item, type: detectTokenType(item, indicatorSet, extraSet) }))
  }, [extraSet, indicatorSet, value])

  const selectionState = useMemo(() => {
    const input = inputRef.current
    if (!input) {
      return { keyword: '', start: 0, end: 0 }
    }
    const cursor = input.selectionStart ?? value.length
    const left = value.slice(0, cursor)
    const right = value.slice(cursor)

    const leftMatch = left.match(/[^\s()+\-*/,<>=!&|]+$/)
    const rightMatch = right.match(/^[^\s()+\-*/,<>=!&|]+/)
    const leftToken = leftMatch?.[0] ?? ''
    const rightToken = rightMatch?.[0] ?? ''

    return {
      keyword: `${leftToken}${rightToken}`,
      start: cursor - leftToken.length,
      end: cursor + rightToken.length,
    }
  }, [value])

  const suggestions = useMemo(() => {
    const keyword = selectionState.keyword
    if (!focus) {
      return []
    }
    if (!keyword) {
      return allSuggestions.slice(0, 8)
    }
    return allSuggestions
      .filter((item) => fuzzyMatch(item.value, keyword))
      .slice(0, 8)
  }, [allSuggestions, focus, selectionState.keyword])

  const applySuggestion = (item: SuggestionItem) => {
    const start = selectionState.start
    const end = selectionState.end
    const nextValue = `${value.slice(0, start)}${item.value}${value.slice(end)}`
    onChange(nextValue)

    requestAnimationFrame(() => {
      if (!inputRef.current) {
        return
      }
      const nextCursor = start + item.value.length
      inputRef.current.focus()
      inputRef.current.setSelectionRange(nextCursor, nextCursor)
    })
  }

  return (
    <div className="space-y-2">
      <div className="relative">
        <Input
          ref={inputRef}
          value={value}
          placeholder={placeholder}
          className={cn('h-9 rounded-lg border-[#E2E8F0] px-3 text-[13px] text-[#0F172A]', inputClassName)}
          onChange={(event) => onChange(event.target.value)}
          onFocus={() => setFocus(true)}
          onBlur={() => setTimeout(() => setFocus(false), 120)}
          onKeyDown={(event) => {
            if (event.key === 'Enter' && suggestions.length > 0) {
              event.preventDefault()
              applySuggestion(suggestions[0])
            }
          }}
        />

        {suggestions.length > 0 && (
          <div className="absolute z-20 mt-1 max-h-56 w-full overflow-y-auto rounded-lg border border-[#E2E8F0] bg-white p-1 shadow-lg">
            {suggestions.map((item) => (
              <button
                type="button"
                key={`${item.type}-${item.value}`}
                className="flex h-8 w-full items-center justify-between rounded-md px-2 text-left hover:bg-[#F8FAFC]"
                onMouseDown={(event) => {
                  event.preventDefault()
                  applySuggestion(item)
                }}
              >
                <span className="text-xs text-[#0F172A]">{item.value}</span>
                <span className={cn('text-[11px]', tokenClassName[item.type])}>
                  {item.type === 'indicator' ? '指标' : item.type === 'function' ? '函数' : '运算符'}
                </span>
              </button>
            ))}
          </div>
        )}
      </div>

      <div className="flex min-h-10 flex-wrap gap-1 rounded-lg border border-dashed border-[#E2E8F0] bg-[#F8FAFC] px-2 py-2 text-xs leading-5">
        {tokenMeta.length === 0 && <span className="text-[#94A3B8]">输入公式后将在这里展示彩色语法预览</span>}
        {tokenMeta.map((item, index) => (
          <span key={`${item.token}-${index}`} className={tokenClassName[item.type]}>
            {item.token === ' ' ? '\u00a0' : item.token}
          </span>
        ))}
      </div>
    </div>
  )
}
