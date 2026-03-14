/**
 * AI 对话面板
 *
 * @author Anner
 * Created on 2026/3/14
 */

import { useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { MessageCircle, Sparkles, Target, TrendingUp, Wand2, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'

export type ChatMode = 'chat' | 'adjust'

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
}

interface StreamResultPayload {
  mode: ChatMode
  reply: string
  appliedRules?: AppliedRule[]
}

interface StreamEvent {
  type: string
  content?: string
  result?: StreamResultPayload
  error?: string
}

interface ChatPanelProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onAdjustApplied?: () => void
}

interface ChatPanelViewProps {
  open: boolean
  mode: ChatMode
  input: string
  streaming: boolean
  messages: ChatPanelMessage[]
  onClose: () => void
  onInputChange: (value: string) => void
  onModeChange: (mode: ChatMode) => void
  onReset: () => void
  onSend: () => void
}

const defaultQuestions: Record<ChatMode, { icon: ReactNode; title: string; content: string }[]> = {
  chat: [
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
  ],
  adjust: [
    {
      icon: <Target className="h-4 w-4" />,
      title: '调整批发增速',
      content: '把批发当月增速调到 15%',
    },
    {
      icon: <Wand2 className="h-4 w-4" />,
      title: '调整限上社零',
      content: '把限上社零额当月增速调到 8%',
    },
  ],
}

export function ChatPanelView({
  open,
  mode,
  input,
  streaming,
  messages,
  onClose,
  onInputChange,
  onModeChange,
  onReset,
  onSend,
}: ChatPanelViewProps) {
  const questions = useMemo(() => defaultQuestions[mode], [mode])
  if (!open) {
    return null
  }

  return (
    <>
      <div className="fixed inset-0 z-40 bg-black/20" onClick={onClose} />
      <div className="fixed right-0 top-0 z-50 flex h-full w-[420px] flex-col border-l border-border bg-background shadow-2xl">
        <div className="flex items-center justify-between border-b border-border bg-muted/30 px-4 py-3">
          <div className="flex items-center gap-2">
            <MessageCircle className="h-5 w-5 text-primary" />
            <span className="text-sm font-semibold">数据对话助手</span>
          </div>
          <div className="flex items-center gap-2">
            <Button variant="ghost" size="sm" onClick={onReset} disabled={streaming}>
              新建会话
            </Button>
            <Button variant="ghost" size="icon" className="h-8 w-8" onClick={onClose}>
              <X className="h-4 w-4" />
            </Button>
          </div>
        </div>

        <div className="border-b border-border px-4 py-3">
          <Tabs value={mode} onValueChange={(value) => onModeChange(value as ChatMode)}>
            <TabsList className="grid w-full grid-cols-2">
              <TabsTrigger value="chat">聊天</TabsTrigger>
              <TabsTrigger value="adjust">调整</TabsTrigger>
            </TabsList>
          </Tabs>
        </div>

        <ScrollArea className="flex-1 px-4 py-3">
          <div className="space-y-3">
            {messages.length === 0 && (
              <div className="space-y-3">
                <p className="px-1 text-sm text-muted-foreground">
                  {mode === 'adjust'
                    ? '直接描述要调整到的指标目标值，系统会先解析意图，再调用调整引擎执行。'
                    : '直接提问当前指标含义、走势分析或规则影响，AI 会基于当前指标和规则结果回答。'}
                </p>
                <div className="grid grid-cols-1 gap-2">
                  {questions.map((question) => (
                    <button
                      key={question.title}
                      onClick={() => onInputChange(question.content)}
                      className="group flex items-start gap-3 rounded-lg border border-border bg-card p-3 text-left transition-colors hover:border-accent hover:bg-accent"
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
              <div key={message.id} className={`flex ${message.role === 'user' ? 'justify-end' : 'justify-start'}`}>
                <div
                  className={`max-w-[85%] rounded-2xl px-4 py-3 text-sm leading-relaxed shadow-sm ${
                    message.role === 'user'
                      ? 'bg-primary text-primary-foreground'
                      : 'border border-border bg-muted text-foreground'
                  }`}
                >
                  <div className="prose prose-sm max-w-none text-inherit dark:prose-invert">
                    <ReactMarkdown remarkPlugins={[remarkGfm]}>{message.content}</ReactMarkdown>
                  </div>
                  {message.streaming && <span className="ml-1 animate-pulse">▍</span>}
                  {message.appliedRules && message.appliedRules.length > 0 && (
                    <div className="mt-3 space-y-2 border-t border-border/60 pt-3">
                      {message.appliedRules.map((rule, index) => (
                        <div key={`${rule.ruleId}-${index}`} className="rounded-xl bg-background/70 px-3 py-2 text-xs text-muted-foreground">
                          {formatAppliedRuleDescription(rule)}
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              </div>
            ))}
          </div>
        </ScrollArea>

        <div className="border-t border-border bg-muted/30 p-3">
          <div className="flex gap-2">
            <Input
              value={input}
              onChange={(event) => onInputChange(event.target.value)}
              placeholder={mode === 'adjust' ? '输入你的调整需求…' : '输入你的咨询问题…'}
              disabled={streaming}
              onKeyDown={(event) => {
                if (event.key === 'Enter' && !event.shiftKey) {
                  event.preventDefault()
                  onSend()
                }
              }}
              className="flex-1"
            />
            <Button onClick={onSend} disabled={streaming || !input.trim()}>
              {streaming ? '…' : '发送'}
            </Button>
          </div>
          <div className="mt-2 text-center text-xs text-muted-foreground">按 Enter 发送，Shift+Enter 换行</div>
        </div>
      </div>
    </>
  )
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

export default function ChatPanel({ open, onOpenChange, onAdjustApplied }: ChatPanelProps) {
  const [messages, setMessages] = useState<ChatPanelMessage[]>([])
  const [input, setInput] = useState('')
  const [streaming, setStreaming] = useState(false)
  const [mode, setMode] = useState<ChatMode>('chat')
  const endRef = useRef<HTMLDivElement | null>(null)

  useEffect(() => {
    if (!open) {
      return
    }
    endRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, open])

  const resetSession = () => {
    setMessages([])
    setInput('')
    setMode('chat')
  }

  const handleSend = async () => {
    const trimmed = input.trim()
    if (!trimmed || streaming) {
      return
    }
    const nextMessage: ChatPanelMessage = { id: crypto.randomUUID(), role: 'user', content: trimmed }
    const nextMessages = [...messages, nextMessage]
    setMessages(nextMessages)
    setInput('')
    await startStream(nextMessages)
  }

  const startStream = async (history: ChatPanelMessage[]) => {
    setStreaming(true)
    try {
      const res = await fetch('/api/llm/chat/stream', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          mode,
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
        if (done) {
          break
        }
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
      appendAssistantMessage(error instanceof Error ? error.message : '对话失败')
    } finally {
      finalizeAssistantMessage()
      setStreaming(false)
    }
  }

  const handleStreamEvent = (raw: string) => {
    const lines = raw.split('\n')
    for (const line of lines) {
      if (!line.startsWith('data:')) {
        continue
      }
      const payload = line.replace('data:', '').trim()
      if (!payload) {
        continue
      }
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
      if (!last || last.role !== 'assistant' || !last.streaming) {
        return prev
      }
      return [...prev.slice(0, -1), { ...last, streaming: false }]
    })
  }

  const applyResult = (result: StreamResultPayload) => {
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
        },
      ]
    })
  }

  return (
    <div ref={endRef}>
      <ChatPanelView
        open={open}
        mode={mode}
        input={input}
        streaming={streaming}
        messages={messages}
        onClose={() => onOpenChange(false)}
        onInputChange={setInput}
        onModeChange={setMode}
        onReset={resetSession}
        onSend={() => void handleSend()}
      />
    </div>
  )
}
