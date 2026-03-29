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
import { BookPlus, Clock, Eraser, MessageCircle, Plus, Sparkles, Square, Target, Trash2, TrendingUp, Wand2, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
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
 * 轮询规则转换状态，结束后回调最终状态
 */
function pollRuleConvertStatus(onDone: (status: 'ok' | 'error') => void) {
  const interval = setInterval(async () => {
    try {
      const res = await fetch('/api/v1/rules/status')
      const data = await res.json()
      if (data.status === 'ok' || data.status === 'error') {
        clearInterval(interval)
        onDone(data.status)
      }
    } catch {
      // 网络错误继续重试
    }
  }, 1500)
  // 安全超时：90 秒后强制停止
  setTimeout(() => {
    clearInterval(interval)
  }, 90_000)
}

// ─── 常量 ───────────────────────────────────────────────

const defaultQuestions: { icon: ReactNode; title: string; content: string }[] = [
  {
    icon: <Sparkles className="h-4 w-4" />,
    title: '解释批发增速',
    content: '解释一下当前批发当月增速代表什么，以及它对整体指标有什么影响。',
  },
  {
    icon: <TrendingUp className="h-4 w-4" />,
    title: '分析零售走势',
    content: '当前零售业销售额增速偏低，可能是什么原因？',
  },
  {
    icon: <Target className="h-4 w-4" />,
    title: '调整批发增速',
    content: '把批发当月增速调到 15%',
  },
  {
    icon: <Wand2 className="h-4 w-4" />,
    title: '随机调整零售',
    content: '帮我将零售当月增速随机调整 5%',
  },
  {
    icon: <BookPlus className="h-4 w-4" />,
    title: '添加规则',
    content: '帮我加一条规则：批发当月增速不能超过 20%',
  },
]

const ASSISTANT_INTRO = '我是 Northstar 数据助手，可以帮你查看和分析当前经济指标数据，也可以直接帮你调整指标目标值或添加约束规则。你可以直接对我说需求，例如"当前零售增速为什么偏低？"或"把批发当月增速调到 15%"。'

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
      <div className="flex items-center justify-between border-b border-border bg-muted/30 px-4 py-3">
        <span className="text-sm font-semibold">历史对话</span>
        <Button variant="ghost" size="sm" onClick={onBack}>
          返回
        </Button>
      </div>
      <ScrollArea className="flex-1">
        {sessions.length === 0 ? (
          <div className="px-4 py-8 text-center text-sm text-muted-foreground">暂无历史对话</div>
        ) : (
          <div className="space-y-1 p-2">
            {sessions.map((s) => (
              <SwipeRow key={s.id} onDelete={() => onDelete(s.id)}>
                <div
                  className="flex cursor-pointer items-center gap-2 px-3 py-2.5 transition-colors hover:bg-muted"
                  onClick={() => onSelect(s.id)}
                >
                  <MessageCircle className="h-4 w-4 shrink-0 text-muted-foreground" />
                  <div className="min-w-0 flex-1">
                    <div className="truncate text-sm">{s.title || '新对话'}</div>
                    <div className="text-xs text-muted-foreground">{formatTimeAgo(s.updatedAt)}</div>
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

// ─── 消息气泡 ────────────────────────────────────────────

function MessageBubble({ message }: { message: ChatPanelMessage }) {
  const isUser = message.role === 'user'
  const changedCount = message.changedIndicatorIds?.length ?? 0

  return (
    <div className="flex justify-start">
      <div
        className={`max-w-[88%] rounded-xl px-4 py-3 text-sm leading-relaxed ${
          isUser
            ? 'bg-primary/10 text-foreground'
            : 'bg-muted/60 text-foreground'
        }`}
      >
        {/* 角色标签 + 指标更新徽标 */}
        <div className={`mb-1.5 flex items-center gap-2 text-xs font-medium ${isUser ? 'text-primary' : 'text-muted-foreground'}`}>
          <span>{isUser ? '我' : '助手'}</span>
          {!isUser && changedCount > 0 && (
            <span className="rounded-full bg-emerald-100 px-2 py-0.5 text-[10px] font-semibold text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-300">
              {changedCount} 项指标已更新
            </span>
          )}
        </div>

        <div className="prose prose-sm max-w-none break-words text-inherit prose-p:my-1.5 prose-ul:my-1.5 prose-ol:my-1.5 prose-li:my-0.5 prose-headings:my-2 prose-headings:text-inherit prose-strong:text-inherit dark:prose-invert">
          <ReactMarkdown remarkPlugins={[remarkGfm]}>{fixMarkdownBold(message.content)}</ReactMarkdown>
        </div>
        {message.streaming && <span className="ml-1 animate-pulse">▍</span>}

        {/* 调整结果：绿色高亮样式 */}
        {message.appliedRules && message.appliedRules.length > 0 && (
          <div className="mt-3 space-y-1.5 border-t border-emerald-200/60 pt-3 dark:border-emerald-800/40">
            <div className="text-xs font-medium text-emerald-600 dark:text-emerald-400">已执行调整：</div>
            {message.appliedRules.map((rule, index) => (
              <div
                key={`${rule.ruleId}-${index}`}
                className="rounded-lg border border-emerald-200 bg-emerald-50/80 px-3 py-1.5 text-xs text-emerald-700 dark:border-emerald-800/50 dark:bg-emerald-950/30 dark:text-emerald-300"
              >
                {formatAppliedRuleDescription(rule)}
              </div>
            ))}
          </div>
        )}

        {message.ruleAdded && (
          <div className="mt-3 border-t border-border/60 pt-3">
            <div className="flex items-center gap-2 rounded-lg border border-emerald-200 bg-emerald-50/80 px-3 py-1.5 text-xs text-emerald-700 dark:border-emerald-800/50 dark:bg-emerald-950/30 dark:text-emerald-300">
              <BookPlus className="h-3.5 w-3.5 shrink-0" />
              <span>
                已添加规则：{message.ruleAdded.text}
                {message.ruleAdded.status === 'converting' && ' — ⏳ 正在转换中…'}
                {message.ruleAdded.status === 'ok' && ' — ✅ 已生效'}
                {message.ruleAdded.status === 'error' && ' — ❌ 转换失败'}
              </span>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

// ─── 主组件 ──────────────────────────────────────────────

export default function ChatPanel({ open, onOpenChange, onAdjustApplied, suggestions }: ChatPanelProps) {
  const [messages, setMessages] = useState<ChatPanelMessage[]>([])
  const [sessionId, setSessionId] = useState<string>(crypto.randomUUID())
  const [streaming, setStreaming] = useState(false)
  const [input, setInput] = useState('')
  const [showHistory, setShowHistory] = useState(false)
  const [sessions, setSessions] = useState<ChatSession[]>([])
  const endRef = useRef<HTMLDivElement | null>(null)
  const abortRef = useRef<AbortController | null>(null)

  useEffect(() => {
    if (!open) return
    endRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, open])

  useEffect(() => {
    if (open) {
      loadSessions()
    }
  }, [open])

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
    setSessionId(crypto.randomUUID())
  }

  const handleStop = () => {
    if (abortRef.current) {
      abortRef.current.abort()
      abortRef.current = null
    }
  }

  const handleSend = async (directContent?: string) => {
    const trimmed = (directContent ?? input).trim()
    if (!trimmed || streaming) return
    const nextMessage: ChatPanelMessage = { id: crypto.randomUUID(), role: 'user', content: trimmed }
    const nextMessages = [...messages, nextMessage]
    setMessages(nextMessages)
    setInput('')
    await startStream(nextMessages)
  }

  const startStream = async (history: ChatPanelMessage[]) => {
    setStreaming(true)
    const controller = new AbortController()
    abortRef.current = controller

    try {
      const res = await fetch('/api/llm/chat/stream', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        signal: controller.signal,
        body: JSON.stringify({
          mode: 'adjust',
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
    if (event.type === 'message_delta' && event.content) {
      appendAssistantDelta(event.content)
      return
    }
    if (event.type === 'result' && event.result) {
      applyResult(event.result)
      if (event.result.mode === 'adjust') {
        // 合并：直接调整的目标 ID + 规则联动触发的 ID
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
      appendAssistantMessage(`**错误**：${event.error}`)
    }
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
        pollRuleConvertStatus((finalStatus) => {
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

  if (!open) return null

  // 动态推荐优先，静态兜底
  const suggestionIcons = [
    <Sparkles className="h-4 w-4" />,
    <TrendingUp className="h-4 w-4" />,
    <Target className="h-4 w-4" />,
    <Wand2 className="h-4 w-4" />,
    <BookPlus className="h-4 w-4" />,
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
      <div className="flex h-full w-[400px] shrink-0 flex-col border-l border-border bg-background shadow-lg">
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
    <div className="flex h-full w-[400px] shrink-0 flex-col border-l border-border bg-background shadow-lg overflow-hidden">
      {/* 顶部栏 */}
      <div className="flex items-center justify-between border-b border-border bg-muted/30 px-4 py-3">
        <div className="flex items-center gap-2">
          <MessageCircle className="h-5 w-5 text-primary" />
          <span className="text-sm font-semibold">数据助手</span>
        </div>
        <div className="flex items-center gap-1">
          <Button variant="ghost" size="icon" className="h-8 w-8" onClick={resetSession} disabled={streaming || messages.length === 0} title="清空对话">
            <Eraser className="h-4 w-4" />
          </Button>
          <Button variant="ghost" size="icon" className="h-8 w-8" onClick={resetSession} disabled={streaming} title="新建对话">
            <Plus className="h-4 w-4" />
          </Button>
          <Button variant="ghost" size="icon" className="h-8 w-8" onClick={() => setShowHistory(true)} title="历史对话">
            <Clock className="h-4 w-4" />
          </Button>
          <Button variant="ghost" size="icon" className="h-8 w-8" onClick={() => onOpenChange(false)}>
            <X className="h-4 w-4" />
          </Button>
        </div>
      </div>

      {/* 消息区 */}
      <ScrollArea className="flex-1 overflow-x-hidden">
        <div className="space-y-3 p-4">
          {messages.length === 0 && (
            <div className="space-y-4">
              {/* 角色介绍 */}
              <div className="rounded-xl bg-muted/40 px-4 py-3 text-sm leading-relaxed text-muted-foreground">
                {ASSISTANT_INTRO}
              </div>
              {/* 推荐问题 */}
              <div className="grid grid-cols-1 gap-2">
                {questions.map((question) => (
                  <button
                    key={question.title}
                    onClick={() => void handleSend(question.content)}
                    className="group flex items-start gap-3 rounded-lg border border-border bg-card p-3 text-left transition-colors hover:border-primary/40 hover:bg-primary/5"
                  >
                    <div className="mt-0.5 text-muted-foreground group-hover:text-primary">{question.icon}</div>
                    <div className="min-w-0 flex-1">
                      <div className="text-sm font-medium text-foreground">{question.title}</div>
                      <div className="truncate text-xs text-muted-foreground">{question.content}</div>
                    </div>
                  </button>
                ))}
              </div>
            </div>
          )}

          {messages.map((message) => (
            <MessageBubble key={message.id} message={message} />
          ))}
          <div ref={endRef} />
        </div>
      </ScrollArea>

      {/* 输入区 */}
      <div className="border-t border-border bg-muted/30 p-3">
        <div className="flex gap-2">
          <Input
            value={input}
            onChange={(event) => setInput(event.target.value)}
            placeholder="输入问题或调整指令…"
            disabled={streaming}
            onKeyDown={(event) => {
              if (event.key === 'Enter' && !event.shiftKey) {
                event.preventDefault()
                void handleSend()
              }
            }}
            className="flex-1"
          />
          {streaming ? (
            <Button variant="destructive" onClick={handleStop} className="shrink-0">
              <Square className="mr-1 h-3 w-3" />
              停止
            </Button>
          ) : (
            <Button onClick={() => void handleSend()} disabled={!input.trim()}>
              发送
            </Button>
          )}
        </div>
        <div className="mt-2 text-center text-xs text-muted-foreground">按 Enter 发送，Shift+Enter 换行</div>
      </div>
    </div>
  )
}
