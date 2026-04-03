/**
 * AI 对话面板（统一对话模式，支持查询与调整）
 *
 * @author Anner
 * Created on 2026/3/14
 */

import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { toast } from 'sonner'
import { ArrowRight, BookPlus, BrainCircuit, CheckCircle2, ChevronDown, ChevronRight, Clock, Eraser, MessageCircle, Plus, Send, Sparkles, Square, Target, Trash2, TrendingUp, Wand2, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'

export interface AppliedRule {
  ruleId: string
  type: string
  indicatorId?: string
  ensureId?: string
  beforeValue?: number
  afterValue?: number
  beforeCount?: number
  afterCount?: number
  targetValue?: number
}

export interface ChatPanelMessage {
  id: string
  role: 'user' | 'assistant'
  content: string
  streaming?: boolean
  reasoning?: string
  reasoningDone?: boolean
  appliedRules?: AppliedRule[]
  ruleAdded?: RuleAddedPayload
  /** 本次调整涉及的指标 ID 列表 */
  changedIndicatorIds?: string[]
}

interface RuleAddedPayload {
  text: string
  status: string
}

interface StreamResultPayload {
  mode: string
  reply: string
  appliedRules?: AppliedRule[]
  ruleAdded?: RuleAddedPayload
  /** 直接调整的指标 ID 列表（后端返回，用于绿色高亮） */
  adjustedTargets?: string[]
}

interface StreamEvent {
  type: string
  content?: string
  result?: StreamResultPayload
  error?: string
}

interface ChatSession {
  id: string
  title: string
  mode: string
  createdAt: string
  updatedAt: string
}

interface SuggestionsData {
  chat: { title: string; content: string }[]
  adjust: { title: string; content: string }[]
}

interface ChatPanelProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  /** 调整完成后回调，传入变化的指标 ID 列表 */
  onAdjustApplied?: (changedIndicatorIds?: string[]) => void
  suggestions?: SuggestionsData | null
}

// ─── 规则转换轮询 ──────────────────────────────────────────

/**
 * 轮询规则转换状态，结束后回调最终状态。
 * 返回 cleanup 函数，调用方可在组件卸载时取消轮询。
 */
function pollRuleConvertStatus(onDone: (status: 'ok' | 'error') => void): () => void {
  let stopped = false
  const interval = setInterval(async () => {
    if (stopped) return
    try {
      const res = await fetch('/api/v1/rules/status')
      const data = await res.json()
      if (data.status === 'ok' || data.status === 'error') {
        stopped = true
        clearInterval(interval)
        clearTimeout(timeout)
        onDone(data.status)
      }
    } catch {
      // 网络错误继续重试
    }
  }, 1500)
  const timeout = setTimeout(() => {
    stopped = true
    clearInterval(interval)
  }, 90_000)
  return () => {
    stopped = true
    clearInterval(interval)
    clearTimeout(timeout)
  }
}

// ─── 常量 ───────────────────────────────────────────────

const defaultQuestions: { icon: ReactNode; title: string; content: string }[] = [
  {
    icon: <Sparkles className="h-3.5 w-3.5" />,
    title: '解释批发增速',
    content: '解释一下当前批发当月增速代表什么，以及它对整体指标有什么影响。',
  },
  {
    icon: <TrendingUp className="h-3.5 w-3.5" />,
    title: '分析零售走势',
    content: '当前零售业销售额增速偏低，可能是什么原因？',
  },
  {
    icon: <Target className="h-3.5 w-3.5" />,
    title: '调整批发增速',
    content: '把批发当月增速调到 15%',
  },
  {
    icon: <Wand2 className="h-3.5 w-3.5" />,
    title: '随机调整零售',
    content: '帮我将零售当月增速随机调整 5%',
  },
  {
    icon: <BookPlus className="h-3.5 w-3.5" />,
    title: '添加规则',
    content: '帮我加一条规则：批发当月增速不能超过 20%',
  },
]

const ASSISTANT_INTRO = '我是数据助手，可以帮你查看、分析经济指标，也能直接调整目标值或添加约束规则。'

// ─── 工具函数 ────────────────────────────────────────────

/**
 * 修复 Markdown 加粗语法在特殊字符前失效的问题
 */
function fixMarkdownBold(text: string): string {
  return text.replace(/\*\*([^\w\s*])/g, '**\u200B$1')
}

function formatTimeAgo(dateStr: string): string {
  const date = new Date(dateStr)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffMin = Math.floor(diffMs / 60000)
  if (diffMin < 1) return '刚刚'
  if (diffMin < 60) return `${diffMin} 分钟前`
  const diffHr = Math.floor(diffMin / 60)
  if (diffHr < 24) return `${diffHr} 小时前`
  const diffDay = Math.floor(diffHr / 24)
  if (diffDay < 7) return `${diffDay} 天前`
  return date.toLocaleDateString('zh-CN')
}

export function formatAppliedRuleDescription(rule: AppliedRule) {
  if (rule.type === 'clamp_target') {
    return `目标值从 ${Math.round(rule.beforeValue ?? 0)} 裁剪为 ${Math.round(rule.afterValue ?? 0)}`
  }
  if (rule.type === 'filter_allocation') {
    return `参与企业从 ${rule.beforeCount ?? 0} 过滤为 ${rule.afterCount ?? 0} 家`
  }
  if (rule.type === 'compensate') {
    return `联动补偿 ${rule.ensureId ?? ''} → ${Math.round(rule.targetValue ?? 0)}`
  }
  return `${rule.ruleId}：${rule.type}`
}

/** 从 appliedRules 中提取变化的指标 ID 列表 */
function extractChangedIndicatorIds(rules?: AppliedRule[]): string[] {
  if (!rules || rules.length === 0) return []
  const ids = new Set<string>()
  for (const rule of rules) {
    if (rule.indicatorId) ids.add(rule.indicatorId)
    if (rule.ensureId) ids.add(rule.ensureId)
  }
  return Array.from(ids)
}

// ─── 思考指示器 ─────────────────────────────────────────

function ThinkingIndicator() {
  return (
    <div className="flex items-start gap-3 px-1">
      <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-stone-100 dark:bg-stone-800">
        <Sparkles className="h-3.5 w-3.5 text-stone-500 dark:text-stone-400" />
      </div>
      <div className="flex items-center gap-1 pt-1.5">
        <span className="h-1.5 w-1.5 animate-bounce rounded-full bg-stone-400 dark:bg-stone-500" style={{ animationDelay: '0ms' }} />
        <span className="h-1.5 w-1.5 animate-bounce rounded-full bg-stone-400 dark:bg-stone-500" style={{ animationDelay: '150ms' }} />
        <span className="h-1.5 w-1.5 animate-bounce rounded-full bg-stone-400 dark:bg-stone-500" style={{ animationDelay: '300ms' }} />
      </div>
    </div>
  )
}

// ─── 滑动删除行 ─────────────────────────────────────────

function SwipeRow({
  children,
  onDelete,
}: {
  children: React.ReactNode
  onDelete: () => void
}) {
  const rowRef = useRef<HTMLDivElement>(null)
  const startX = useRef(0)
  const currentX = useRef(0)
  const swiping = useRef(false)
  const [offset, setOffset] = useState(0)
  const threshold = 70

  const handleStart = (x: number) => {
    startX.current = x
    currentX.current = x
    swiping.current = true
  }

  const handleMove = (x: number) => {
    if (!swiping.current) return
    currentX.current = x
    const dx = currentX.current - startX.current
    const clamped = Math.max(-threshold, Math.min(0, dx))
    setOffset(clamped)
  }

  const handleEnd = () => {
    if (!swiping.current) return
    swiping.current = false
    if (offset < -threshold / 2) {
      setOffset(-threshold)
    } else {
      setOffset(0)
    }
  }

  useEffect(() => {
    if (offset === 0) return
    const close = (e: MouseEvent) => {
      if (rowRef.current && !rowRef.current.contains(e.target as Node)) {
        setOffset(0)
      }
    }
    document.addEventListener('mousedown', close)
    return () => document.removeEventListener('mousedown', close)
  }, [offset])

  return (
    <div ref={rowRef} className="relative overflow-hidden rounded-lg">
      <div className="absolute inset-y-0 right-0 flex w-[70px] items-center justify-center bg-destructive text-destructive-foreground">
        <button
          className="flex h-full w-full items-center justify-center gap-1 text-xs font-medium"
          onClick={(e) => { e.stopPropagation(); onDelete() }}
        >
          <Trash2 className="h-4 w-4" />
          删除
        </button>
      </div>
      <div
        className="relative bg-background transition-transform duration-150 ease-out"
        style={{ transform: `translateX(${offset}px)`, transition: swiping.current ? 'none' : undefined }}
        onMouseDown={(e) => handleStart(e.clientX)}
        onMouseMove={(e) => { if (swiping.current) handleMove(e.clientX) }}
        onMouseUp={handleEnd}
        onMouseLeave={handleEnd}
        onTouchStart={(e) => handleStart(e.touches[0].clientX)}
        onTouchMove={(e) => handleMove(e.touches[0].clientX)}
        onTouchEnd={handleEnd}
      >
        {children}
      </div>
    </div>
  )
}

// ─── 历史列表视图 ────────────────────────────────────────

function HistoryListView({
  sessions,
  onSelect,
  onDelete,
  onBack,
}: {
  sessions: ChatSession[]
  onSelect: (id: string) => void
  onDelete: (id: string) => void
  onBack: () => void
}) {
  return (
    <div className="flex h-full flex-col">
      <div className="flex items-center justify-between border-b border-stone-200/60 px-5 py-3.5 dark:border-stone-700/40">
        <span className="text-sm font-medium text-stone-700 dark:text-stone-300">历史对话</span>
        <Button variant="ghost" size="sm" onClick={onBack} className="text-stone-500 hover:text-stone-700 dark:text-stone-400">
          返回
        </Button>
      </div>
      <ScrollArea className="flex-1">
        {sessions.length === 0 ? (
          <div className="px-5 py-12 text-center text-sm text-stone-400">暂无历史对话</div>
        ) : (
          <div className="space-y-0.5 p-2">
            {sessions.map((s) => (
              <SwipeRow key={s.id} onDelete={() => onDelete(s.id)}>
                <div
                  className="flex cursor-pointer items-center gap-3 rounded-lg px-3 py-2.5 transition-colors hover:bg-stone-50 dark:hover:bg-stone-800/50"
                  onClick={() => onSelect(s.id)}
                >
                  <MessageCircle className="h-4 w-4 shrink-0 text-stone-400" />
                  <div className="min-w-0 flex-1">
                    <div className="truncate text-sm text-stone-700 dark:text-stone-300">{s.title || '新对话'}</div>
                    <div className="text-xs text-stone-400">{formatTimeAgo(s.updatedAt)}</div>
                  </div>
                </div>
              </SwipeRow>
            ))}
          </div>
        )}
      </ScrollArea>
    </div>
  )
}

// ─── 调整结果卡片 ──────────────────────────────────────────

function AdjustmentCard({ rules }: { rules: AppliedRule[] }) {
  const [open, setOpen] = useState(false)
  if (!rules || rules.length === 0) return null

  return (
    <div className="rounded-xl border border-emerald-200/70 bg-gradient-to-b from-emerald-50/60 to-emerald-50/30 dark:border-emerald-800/40 dark:from-emerald-950/30 dark:to-emerald-950/10">
      <button
        onClick={() => setOpen((v) => !v)}
        className="flex w-full items-center gap-2.5 px-3.5 py-2.5"
      >
        <CheckCircle2 className="h-4 w-4 text-emerald-500" />
        <span className="text-xs font-semibold text-emerald-700 dark:text-emerald-400">
          已执行 {rules.length} 项调整
        </span>
        {open ? <ChevronDown className="ml-auto h-3.5 w-3.5 text-emerald-500" /> : <ChevronRight className="ml-auto h-3.5 w-3.5 text-emerald-500" />}
      </button>
      {open && (
        <div className="border-t border-emerald-200/50 dark:border-emerald-800/30">
          <table className="w-full text-xs">
            <thead>
              <tr className="text-emerald-600/70 dark:text-emerald-400/60">
                <th className="px-3.5 py-2 text-left font-medium">类型</th>
                <th className="px-3.5 py-2 text-left font-medium">指标</th>
                <th className="px-3.5 py-2 text-right font-medium">变更</th>
              </tr>
            </thead>
            <tbody>
              {rules.map((rule, i) => (
                <tr key={`${rule.ruleId}-${i}`} className="border-t border-emerald-100/60 dark:border-emerald-800/20">
                  <td className="px-3.5 py-2 text-emerald-600 dark:text-emerald-400">
                    {rule.type === 'clamp_target' && '裁剪'}
                    {rule.type === 'filter_allocation' && '过滤'}
                    {rule.type === 'compensate' && '联动'}
                    {!['clamp_target', 'filter_allocation', 'compensate'].includes(rule.type) && rule.type}
                  </td>
                  <td className="px-3.5 py-2 text-stone-600 dark:text-stone-400">
                    {rule.indicatorId || rule.ensureId || rule.ruleId}
                  </td>
                  <td className="px-3.5 py-2 text-right">
                    <RuleChangeValue rule={rule} />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}

function RuleChangeValue({ rule }: { rule: AppliedRule }) {
  if (rule.type === 'clamp_target') {
    return (
      <span className="inline-flex items-center gap-1 text-stone-600 dark:text-stone-400">
        <span>{Math.round(rule.beforeValue ?? 0)}</span>
        <ArrowRight className="h-3 w-3 text-emerald-500" />
        <span className="font-semibold text-emerald-700 dark:text-emerald-400">{Math.round(rule.afterValue ?? 0)}</span>
      </span>
    )
  }
  if (rule.type === 'filter_allocation') {
    return (
      <span className="inline-flex items-center gap-1 text-stone-600 dark:text-stone-400">
        <span>{rule.beforeCount ?? 0} 家</span>
        <ArrowRight className="h-3 w-3 text-emerald-500" />
        <span className="font-semibold text-emerald-700 dark:text-emerald-400">{rule.afterCount ?? 0} 家</span>
      </span>
    )
  }
  if (rule.type === 'compensate') {
    return (
      <span className="font-semibold text-emerald-700 dark:text-emerald-400">
        → {Math.round(rule.targetValue ?? 0)}
      </span>
    )
  }
  return <span className="text-stone-500">{rule.type}</span>
}

// ─── 规则添加卡片 ──────────────────────────────────────────

function RuleAddedCard({ ruleAdded }: { ruleAdded: RuleAddedPayload }) {
  const statusMap = {
    converting: { text: '转换中', color: 'text-amber-600 dark:text-amber-400', bg: 'bg-amber-50 dark:bg-amber-950/30', border: 'border-amber-200/60 dark:border-amber-800/30' },
    ok: { text: '已生效', color: 'text-emerald-600 dark:text-emerald-400', bg: 'bg-emerald-50 dark:bg-emerald-950/30', border: 'border-emerald-200/60 dark:border-emerald-800/30' },
    error: { text: '转换失败', color: 'text-red-600 dark:text-red-400', bg: 'bg-red-50 dark:bg-red-950/30', border: 'border-red-200/60 dark:border-red-800/30' },
  }
  const s = statusMap[ruleAdded.status as keyof typeof statusMap] ?? statusMap.converting

  return (
    <div className={`rounded-xl border ${s.border} ${s.bg} px-3.5 py-3`}>
      <div className="flex items-center gap-2">
        <BookPlus className={`h-4 w-4 shrink-0 ${s.color}`} />
        <span className={`text-xs font-semibold ${s.color}`}>新增规则 · {s.text}</span>
      </div>
      <p className="mt-1.5 text-xs leading-relaxed text-stone-600 dark:text-stone-400">{ruleAdded.text}</p>
    </div>
  )
}

// ─── 消息气泡 ────────────────────────────────────────────

const proseClasses = [
  'prose prose-sm max-w-none',
  'text-stone-700 dark:text-stone-300',
  // 段落间距
  'prose-p:my-2 prose-p:leading-[1.75]',
  // 列表
  'prose-ul:my-2.5 prose-ol:my-2.5 prose-li:my-0.5',
  // 标题
  'prose-headings:my-3 prose-headings:text-stone-800 prose-headings:font-semibold dark:prose-headings:text-stone-200',
  // 加粗
  'prose-strong:text-stone-800 dark:prose-strong:text-stone-200',
  // 分割线
  'prose-hr:my-4 prose-hr:border-stone-200 dark:prose-hr:border-stone-700',
  // 表格
  'prose-table:my-3 prose-th:px-3 prose-th:py-1.5 prose-th:text-left prose-th:font-medium prose-th:text-stone-600 dark:prose-th:text-stone-400',
  'prose-td:px-3 prose-td:py-1.5 prose-td:border-t prose-td:border-stone-100 dark:prose-td:border-stone-800',
].join(' ')

function MessageBubble({ message }: { message: ChatPanelMessage }) {
  const isUser = message.role === 'user'
  const changedCount = message.changedIndicatorIds?.length ?? 0
  const [reasoningOpen, setReasoningOpen] = useState(true)
  const [userToggled, setUserToggled] = useState(false)

  useEffect(() => {
    if (message.reasoningDone && !userToggled) {
      setReasoningOpen(false)
    }
  }, [message.reasoningDone])

  if (isUser) {
    return (
      <div className="flex justify-end">
        <div className="max-w-[85%] rounded-2xl rounded-br-md bg-primary/90 px-4 py-2.5 text-sm leading-relaxed text-primary-foreground shadow-sm">
          {message.content}
        </div>
      </div>
    )
  }

  return (
    <div className="flex items-start gap-3 px-1">
      <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-stone-100 dark:bg-stone-800">
        <Sparkles className="h-3.5 w-3.5 text-stone-500 dark:text-stone-400" />
      </div>
      <div className="min-w-0 flex-1 space-y-3">
        {/* 指标更新徽标 */}
        {changedCount > 0 && (
          <span className="inline-block rounded-full bg-emerald-50 px-2.5 py-0.5 text-[11px] font-medium text-emerald-600 dark:bg-emerald-900/30 dark:text-emerald-400">
            {changedCount} 项指标已更新
          </span>
        )}

        {/* 深度思考过程 */}
        {message.reasoning && (
          <div className="rounded-xl border border-amber-200/60 bg-gradient-to-b from-amber-50/50 to-amber-50/20 dark:border-amber-800/30 dark:from-amber-950/20 dark:to-transparent">
            <button
              onClick={() => { setReasoningOpen((v) => !v); setUserToggled(true) }}
              className="flex w-full items-center gap-2 px-3.5 py-2.5 text-xs font-semibold text-amber-700 dark:text-amber-400"
            >
              <BrainCircuit className="h-3.5 w-3.5" />
              <span>思考过程</span>
              {!message.reasoningDone && (
                <span className="ml-1 inline-block h-1.5 w-1.5 animate-pulse rounded-full bg-amber-500" />
              )}
              {reasoningOpen ? <ChevronDown className="ml-auto h-3.5 w-3.5" /> : <ChevronRight className="ml-auto h-3.5 w-3.5" />}
            </button>
            {reasoningOpen && (
              <div className="border-t border-amber-200/40 px-3.5 py-2.5 text-xs leading-[1.7] text-amber-800/70 dark:border-amber-800/20 dark:text-amber-300/60">
                <ReactMarkdown remarkPlugins={[remarkGfm]}>{message.reasoning}</ReactMarkdown>
              </div>
            )}
          </div>
        )}

        {/* 正文 */}
        {message.content.trim() && (
          <div className={proseClasses}>
            <ReactMarkdown remarkPlugins={[remarkGfm]}>{fixMarkdownBold(message.content)}</ReactMarkdown>
          </div>
        )}
        {message.streaming && <span className="inline-block h-4 w-0.5 animate-pulse bg-stone-400" />}

        {/* 调整结果卡片 */}
        {message.appliedRules && message.appliedRules.length > 0 && (
          <AdjustmentCard rules={message.appliedRules} />
        )}

        {/* 规则添加卡片 */}
        {message.ruleAdded && <RuleAddedCard ruleAdded={message.ruleAdded} />}
      </div>
    </div>
  )
}

// ─── 主组件 ──────────────────────────────────────────────

export default function ChatPanel({ open, onOpenChange, onAdjustApplied, suggestions }: ChatPanelProps) {
  const [messages, setMessages] = useState<ChatPanelMessage[]>([])
  const [sessionId, setSessionId] = useState<string>(crypto.randomUUID())
  const [streaming, setStreaming] = useState(false)
  const [thinking, setThinking] = useState(false)
  const [reasoning, setReasoning] = useState(false)
  const [reasoningSupported, setReasoningSupported] = useState(false)
  const [input, setInput] = useState('')
  const [showHistory, setShowHistory] = useState(false)
  const [sessions, setSessions] = useState<ChatSession[]>([])
  const endRef = useRef<HTMLDivElement | null>(null)
  const scrollRef = useRef<HTMLDivElement | null>(null)
  const inputRef = useRef<HTMLTextAreaElement | null>(null)
  const abortRef = useRef<AbortController | null>(null)
  const pollCleanupRef = useRef<(() => void) | null>(null)
  const userScrolledUp = useRef(false)

  useEffect(() => {
    return () => { pollCleanupRef.current?.() }
  }, [])

  // 只在用户未手动上滑时自动滚到底部
  useEffect(() => {
    if (!open || userScrolledUp.current) return
    endRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, open, thinking])

  // 用户发送新消息时重置滚动状态
  const resetScrollLock = () => { userScrolledUp.current = false }

  // 监听滚动区域，检测用户是否上滑
  const handleScrollAreaScroll = (e: React.UIEvent<HTMLDivElement>) => {
    const el = e.currentTarget
    const atBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 60
    userScrolledUp.current = !atBottom
  }

  useEffect(() => {
    if (open) {
      loadSessions()
      loadCapabilities()
      setTimeout(() => inputRef.current?.focus(), 100)
    }
  }, [open])

  const loadCapabilities = async () => {
    try {
      const res = await fetch('/api/config')
      const data = await res.json()
      setReasoningSupported(data.llmSupportsReasoning === true)
    } catch { /* ignore */ }
  }

  const loadSessions = async () => {
    try {
      const res = await fetch('/api/chat/sessions')
      const data = await res.json()
      setSessions(Array.isArray(data.items) ? data.items : [])
    } catch {
      /* ignore */
    }
  }

  const saveSession = async (msgs: ChatPanelMessage[]) => {
    if (msgs.length === 0) return
    try {
      await fetch('/api/chat/sessions', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          sessionId,
          mode: 'adjust',
          messages: msgs.map((msg) => ({
            id: msg.id,
            role: msg.role,
            content: msg.content,
            appliedRules: msg.appliedRules,
            ruleAdded: msg.ruleAdded,
          })),
        }),
      })
      loadSessions()
    } catch {
      /* ignore */
    }
  }

  const restoreSession = async (id: string) => {
    try {
      const res = await fetch(`/api/chat/sessions/${id}`)
      const data = await res.json()
      const msgs: ChatPanelMessage[] = (data.messages || []).map((m: any) => ({
        id: m.id,
        role: m.role,
        content: m.content,
        appliedRules: m.appliedRules,
        ruleAdded: m.ruleAdded,
      }))
      setSessionId(id)
      setMessages(msgs)
      setInput('')
      setShowHistory(false)
    } catch {
      toast.error('恢复会话失败')
    }
  }

  const deleteSession = async (id: string) => {
    try {
      await fetch(`/api/chat/sessions/${id}`, { method: 'DELETE' })
      setSessions((prev) => prev.filter((s) => s.id !== id))
      if (id === sessionId) {
        setMessages([])
        setSessionId(crypto.randomUUID())
      }
    } catch {
      toast.error('删除会话失败')
    }
  }

  const resetSession = () => {
    handleStop()
    setMessages([])
    setInput('')
    setThinking(false)
    setSessionId(crypto.randomUUID())
  }

  const handleStop = () => {
    if (abortRef.current) {
      abortRef.current.abort()
      abortRef.current = null
    }
    setThinking(false)
  }

  const handleSend = async (directContent?: string) => {
    const trimmed = (directContent ?? input).trim()
    if (!trimmed || streaming) return
    resetScrollLock()
    const nextMessage: ChatPanelMessage = { id: crypto.randomUUID(), role: 'user', content: trimmed }
    const nextMessages = [...messages, nextMessage]
    setMessages(nextMessages)
    setInput('')
    await startStream(nextMessages)
  }

  const startStream = async (history: ChatPanelMessage[]) => {
    setStreaming(true)
    setThinking(true)
    const controller = new AbortController()
    abortRef.current = controller

    try {
      const res = await fetch('/api/llm/chat/stream', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        signal: controller.signal,
        body: JSON.stringify({
          mode: 'adjust',
          reasoning,
          messages: history.map((message) => ({ role: message.role, content: message.content })),
        }),
      })
      if (!res.ok) {
        const payload = await res.json().catch(() => null)
        throw new Error(payload?.error || '对话请求失败')
      }
      if (!res.body) {
        throw new Error('对话请求失败')
      }

      const reader = res.body.getReader()
      const decoder = new TextDecoder('utf-8')
      let buffer = ''
      while (true) {
        const { value, done } = await reader.read()
        if (done) break
        buffer += decoder.decode(value, { stream: true })
        let index = buffer.indexOf('\n\n')
        while (index >= 0) {
          const raw = buffer.slice(0, index)
          buffer = buffer.slice(index + 2)
          handleStreamEvent(raw)
          index = buffer.indexOf('\n\n')
        }
      }
    } catch (error) {
      if (!controller.signal.aborted) {
        appendAssistantMessage(error instanceof Error ? error.message : '对话失败')
      }
    } finally {
      finalizeAssistantMessage()
      setStreaming(false)
      setThinking(false)
      if (abortRef.current === controller) {
        abortRef.current = null
      }
    }
  }

  // 流结束后自动保存
  useEffect(() => {
    if (!streaming && messages.length > 0 && messages[messages.length - 1]?.role === 'assistant') {
      saveSession(messages)
    }
  }, [streaming]) // eslint-disable-line react-hooks/exhaustive-deps

  const handleStreamEvent = (raw: string) => {
    const lines = raw.split('\n')
    for (const line of lines) {
      if (!line.startsWith('data:')) continue
      const payload = line.replace('data:', '').trim()
      if (!payload) continue
      try {
        const event = JSON.parse(payload) as StreamEvent
        applyStreamEvent(event)
      } catch (error) {
        console.error('Failed to parse stream event:', error)
      }
    }
  }

  const applyStreamEvent = (event: StreamEvent) => {
    if (event.type === 'thinking') {
      setThinking(true)
      return
    }
    if (event.type === 'reasoning_start') {
      setThinking(false)
      // 确保有一条 assistant 消息来承载 reasoning
      setMessages((prev) => {
        const last = prev[prev.length - 1]
        if (!last || last.role !== 'assistant') {
          return [...prev, { id: crypto.randomUUID(), role: 'assistant', content: '', streaming: true, reasoning: '', reasoningDone: false }]
        }
        return prev
      })
      return
    }
    if (event.type === 'reasoning_delta' && event.content) {
      setThinking(false)
      appendReasoningDelta(event.content)
      return
    }
    if (event.type === 'reasoning_done') {
      setMessages((prev) => {
        const last = prev[prev.length - 1]
        if (last?.role === 'assistant') {
          return [...prev.slice(0, -1), { ...last, reasoningDone: true }]
        }
        return prev
      })
      return
    }
    if (event.type === 'message_delta' && event.content) {
      setThinking(false)
      appendAssistantDelta(event.content)
      return
    }
    if (event.type === 'result' && event.result) {
      setThinking(false)
      applyResult(event.result)
      if (event.result.mode === 'adjust') {
        const fromTargets = event.result.adjustedTargets ?? []
        const fromRules = extractChangedIndicatorIds(event.result.appliedRules)
        const changedIds = [...new Set([...fromTargets, ...fromRules])]
        onAdjustApplied?.(changedIds)
      } else if (event.result.ruleAdded) {
        onAdjustApplied?.()
      }
      return
    }
    if (event.type === 'error' && event.error) {
      setThinking(false)
      appendAssistantMessage(`**错误**：${event.error}`)
    }
  }

  const appendReasoningDelta = (chunk: string) => {
    setMessages((prev) => {
      const last = prev[prev.length - 1]
      if (!last || last.role !== 'assistant') {
        return [...prev, { id: crypto.randomUUID(), role: 'assistant', content: '', streaming: true, reasoning: chunk, reasoningDone: false }]
      }
      return [...prev.slice(0, -1), { ...last, reasoning: (last.reasoning ?? '') + chunk }]
    })
  }

  const appendAssistantDelta = (chunk: string) => {
    setMessages((prev) => {
      const last = prev[prev.length - 1]
      if (!last || last.role !== 'assistant' || !last.streaming) {
        return [...prev, { id: crypto.randomUUID(), role: 'assistant', content: chunk, streaming: true }]
      }
      return [...prev.slice(0, -1), { ...last, content: last.content + chunk }]
    })
  }

  const appendAssistantMessage = (content: string) => {
    setMessages((prev) => [...prev, { id: crypto.randomUUID(), role: 'assistant', content }])
  }

  const finalizeAssistantMessage = () => {
    setMessages((prev) => {
      const last = prev[prev.length - 1]
      if (!last || last.role !== 'assistant' || !last.streaming) return prev
      return [...prev.slice(0, -1), { ...last, streaming: false }]
    })
  }

  const applyResult = (result: StreamResultPayload) => {
    if (result.ruleAdded) {
      if (result.ruleAdded.status === 'converting') {
        toast.success('规则添加成功，正在转换为 JSON…')
        pollCleanupRef.current?.()
        pollCleanupRef.current = pollRuleConvertStatus((finalStatus) => {
          setMessages((prev) =>
            prev.map((m) =>
              m.ruleAdded?.status === 'converting'
                ? { ...m, ruleAdded: { ...m.ruleAdded, status: finalStatus } }
                : m
            )
          )
          if (finalStatus === 'ok') {
            toast.success('规则转换成功，已生效')
          } else {
            toast.error('规则转换失败')
          }
        })
      } else {
        toast.error('规则添加失败')
      }
    }

    const changedIds = extractChangedIndicatorIds(result.appliedRules)

    setMessages((prev) => {
      const last = prev[prev.length - 1]
      if (!last || last.role !== 'assistant') {
        return [
          ...prev,
          {
            id: crypto.randomUUID(),
            role: 'assistant',
            content: result.reply,
            appliedRules: result.appliedRules ?? [],
            ruleAdded: result.ruleAdded,
            changedIndicatorIds: changedIds,
          },
        ]
      }
      return [
        ...prev.slice(0, -1),
        {
          ...last,
          content: last.content || result.reply,
          streaming: false,
          appliedRules: result.appliedRules ?? last.appliedRules,
          ruleAdded: result.ruleAdded ?? last.ruleAdded,
          changedIndicatorIds: changedIds.length > 0 ? changedIds : last.changedIndicatorIds,
        },
      ]
    })
  }

  // 自适应输入框高度
  const handleInputChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    setInput(e.target.value)
    const el = e.target
    el.style.height = 'auto'
    el.style.height = Math.min(el.scrollHeight, 120) + 'px'
  }

  if (!open) return null

  // 动态推荐优先，静态兜底
  const suggestionIcons = [
    <Sparkles className="h-3.5 w-3.5" />,
    <TrendingUp className="h-3.5 w-3.5" />,
    <Target className="h-3.5 w-3.5" />,
    <Wand2 className="h-3.5 w-3.5" />,
    <BookPlus className="h-3.5 w-3.5" />,
  ]
  const dynamicChat = suggestions?.chat ?? []
  const dynamicAdjust = suggestions?.adjust ?? []
  const dynamicAll = [...dynamicChat, ...dynamicAdjust]
  const questions = (dynamicAll.length > 0)
    ? dynamicAll.map((item, i) => ({ icon: suggestionIcons[i % suggestionIcons.length], ...item }))
    : defaultQuestions

  // 历史列表视图
  if (showHistory) {
    return (
      <div className="flex h-full w-[420px] shrink-0 flex-col border-l border-stone-200/60 bg-white shadow-xl dark:border-stone-700/40 dark:bg-stone-900">
        <HistoryListView
          sessions={sessions}
          onSelect={restoreSession}
          onDelete={deleteSession}
          onBack={() => setShowHistory(false)}
        />
      </div>
    )
  }

  // 对话视图
  return (
    <div className="flex h-full w-[420px] shrink-0 flex-col border-l border-stone-200/60 bg-white shadow-xl dark:border-stone-700/40 dark:bg-stone-900 overflow-hidden">
      {/* 顶部栏 */}
      <div className="flex items-center justify-between px-5 py-3.5">
        <span className="text-[15px] font-semibold tracking-tight text-stone-800 dark:text-stone-200">数据助手</span>
        <div className="flex items-center gap-0.5">
          <Button
            variant="ghost"
            size="icon"
            className="h-8 w-8 text-stone-400 hover:text-stone-600 dark:text-stone-500 dark:hover:text-stone-300"
            onClick={resetSession}
            disabled={streaming || messages.length === 0}
            title="清空对话"
          >
            <Eraser className="h-4 w-4" />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            className="h-8 w-8 text-stone-400 hover:text-stone-600 dark:text-stone-500 dark:hover:text-stone-300"
            onClick={resetSession}
            disabled={streaming}
            title="新建对话"
          >
            <Plus className="h-4 w-4" />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            className="h-8 w-8 text-stone-400 hover:text-stone-600 dark:text-stone-500 dark:hover:text-stone-300"
            onClick={() => setShowHistory(true)}
            title="历史对话"
          >
            <Clock className="h-4 w-4" />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            className="h-8 w-8 text-stone-400 hover:text-stone-600 dark:text-stone-500 dark:hover:text-stone-300"
            onClick={() => onOpenChange(false)}
          >
            <X className="h-4 w-4" />
          </Button>
        </div>
      </div>

      <div className="mx-5 border-b border-stone-100 dark:border-stone-800" />

      {/* 消息区 */}
      <div ref={scrollRef} className="flex-1 overflow-y-auto overflow-x-hidden" onScroll={handleScrollAreaScroll}>
        <div className="space-y-5 px-5 py-5">
          {messages.length === 0 && (
            <div className="space-y-5">
              {/* 角色介绍 */}
              <div className="flex items-start gap-3 px-1">
                <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-stone-100 dark:bg-stone-800">
                  <Sparkles className="h-3.5 w-3.5 text-stone-500 dark:text-stone-400" />
                </div>
                <p className="pt-0.5 text-[13px] leading-relaxed text-stone-500 dark:text-stone-400">
                  {ASSISTANT_INTRO}
                </p>
              </div>
              {/* 推荐问题 */}
              <div className="space-y-2 pl-10">
                {questions.map((question) => (
                  <button
                    key={question.title}
                    onClick={() => void handleSend(question.content)}
                    className="group flex w-full items-center gap-2.5 rounded-xl border border-stone-150 bg-stone-50/50 px-3.5 py-2.5 text-left transition-all hover:border-stone-300 hover:bg-stone-100/60 dark:border-stone-700/50 dark:bg-stone-800/30 dark:hover:border-stone-600 dark:hover:bg-stone-800/60"
                  >
                    <div className="text-stone-400 group-hover:text-stone-600 dark:text-stone-500 dark:group-hover:text-stone-300">{question.icon}</div>
                    <span className="text-[13px] text-stone-600 dark:text-stone-400">{question.title}</span>
                  </button>
                ))}
              </div>
            </div>
          )}

          {messages.map((message) => (
            <MessageBubble key={message.id} message={message} />
          ))}

          {thinking && <ThinkingIndicator />}

          <div ref={endRef} />
        </div>
      </div>

      {/* 输入区 */}
      <div className="border-t border-stone-100 px-4 py-3 dark:border-stone-800">
        <div className="flex items-end gap-2 rounded-xl border border-stone-200 bg-stone-50/50 px-3 py-2 transition-colors focus-within:border-stone-300 focus-within:bg-white dark:border-stone-700 dark:bg-stone-800/50 dark:focus-within:border-stone-600 dark:focus-within:bg-stone-800">
          <textarea
            ref={inputRef}
            value={input}
            onChange={handleInputChange}
            placeholder="输入问题或调整指令…"
            disabled={streaming}
            rows={1}
            onKeyDown={(event) => {
              if (event.key === 'Enter' && !event.shiftKey) {
                event.preventDefault()
                void handleSend()
              }
            }}
            className="flex-1 resize-none bg-transparent text-sm leading-relaxed text-stone-700 outline-none placeholder:text-stone-400 disabled:opacity-50 dark:text-stone-300 dark:placeholder:text-stone-500"
            style={{ maxHeight: '120px' }}
          />
          {streaming ? (
            <button
              onClick={handleStop}
              className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-red-500 text-white transition-colors hover:bg-red-600"
            >
              <Square className="h-3 w-3" />
            </button>
          ) : (
            <button
              onClick={() => void handleSend()}
              disabled={!input.trim()}
              className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-stone-800 text-white transition-all hover:bg-stone-700 disabled:opacity-30 dark:bg-stone-200 dark:text-stone-800 dark:hover:bg-stone-300"
            >
              <Send className="h-3.5 w-3.5" />
            </button>
          )}
        </div>
        <div className="mt-2 flex items-center justify-between">
          {reasoningSupported ? (
            <button
              onClick={() => setReasoning((v) => !v)}
              disabled={streaming}
              className={`flex items-center gap-1.5 rounded-full px-2.5 py-1 text-[11px] font-medium transition-all ${
                reasoning
                  ? 'bg-amber-100 text-amber-700 dark:bg-amber-900/40 dark:text-amber-400'
                  : 'text-stone-400 hover:bg-stone-100 hover:text-stone-500 dark:text-stone-500 dark:hover:bg-stone-800 dark:hover:text-stone-400'
              } disabled:opacity-50`}
            >
              <BrainCircuit className="h-3 w-3" />
              深度思考
            </button>
          ) : (
            <div />
          )}
          <span className="text-[11px] text-stone-400 dark:text-stone-600">Enter 发送 · Shift+Enter 换行</span>
        </div>
      </div>
    </div>
  )
}
