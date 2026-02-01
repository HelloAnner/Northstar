import { useEffect, useRef, useState } from 'react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { ScrollArea } from '@/components/ui/scroll-area'
import { X, MessageSquare, Sparkles, Target, Building2, TrendingUp } from 'lucide-react'

interface LlmChatDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onDataChanged: () => void
}

interface ChatMessage {
  id: string
  role: 'user' | 'assistant'
  content: string
  streaming?: boolean
}

interface StreamEvent {
  type: string
  content?: string
  summary?: {
    updatedCompanies: number
    targetIndicators: number
    optimized: boolean
    warnings?: string[]
  }
  error?: string
}

export default function LlmChatDialog({ open, onOpenChange, onDataChanged }: LlmChatDialogProps) {
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [input, setInput] = useState('')
  const [streaming, setStreaming] = useState(false)
  const [sessionId, setSessionId] = useState(() => crypto.randomUUID())
  const endRef = useRef<HTMLDivElement | null>(null)

  useEffect(() => {
    if (!open) return
    endRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, open])

  const resetSession = () => {
    setMessages([])
    setInput('')
    setSessionId(crypto.randomUUID())
  }

  const handleSend = async () => {
    const trimmed = input.trim()
    if (!trimmed || streaming) return
    const nextMessage: ChatMessage = { id: crypto.randomUUID(), role: 'user', content: trimmed }
    const nextMessages: ChatMessage[] = [...messages, nextMessage]
    setMessages(nextMessages)
    setInput('')
    await startStream(nextMessages)
  }

  const startStream = async (history: ChatMessage[]) => {
    setStreaming(true)
    try {
      const res = await fetch('/api/llm/chat/stream', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ sessionId, messages: history.map((m) => ({ role: m.role, content: m.content })) }),
      })
      if (!res.ok) {
        const data = await res.json().catch(() => null)
        throw new Error(data?.error || '对话请求失败')
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
        let idx = buffer.indexOf('\n\n')
        while (idx >= 0) {
          const raw = buffer.slice(0, idx)
          buffer = buffer.slice(idx + 2)
          handleStreamEvent(raw)
          idx = buffer.indexOf('\n\n')
        }
      }
    } catch (err) {
      appendAssistantMessage(err instanceof Error ? err.message : '对话失败')
    } finally {
      finalizeAssistantMessage()
      setStreaming(false)
    }
  }

  const handleStreamEvent = (raw: string) => {
    const lines = raw.split('\n')
    for (const line of lines) {
      if (!line.startsWith('data:')) continue
      const payload = line.replace('data:', '').trim()
      if (!payload) continue
      try {
        const event = JSON.parse(payload) as StreamEvent
        applyStreamEvent(event)
      } catch (err) {
        console.error('Failed to parse stream event:', err)
      }
    }
  }

  const applyStreamEvent = (event: StreamEvent) => {
    if (event.type === 'message_delta' && event.content) {
      appendAssistantDelta(event.content)
      return
    }
    if (event.type === 'tool_result' && event.summary) {
      appendAssistantMessage(formatSummary(event.summary))
      return
    }
    if (event.type === 'error' && event.error) {
      appendAssistantMessage(`**错误**：${event.error}`)
      return
    }
    if (event.type === 'final') {
      finalizeAssistantMessage()
      onDataChanged()
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

  const formatSummary = (summary: StreamEvent['summary']) => {
    if (!summary) return ''
    const parts = [
      `**已更新企业**：${summary.updatedCompanies} 家`,
      `**指标目标**：${summary.targetIndicators} 项`,
      summary.optimized ? '**智能调整**：已触发' : '**智能调整**：未触发',
    ]
    if (summary.warnings && summary.warnings.length > 0) {
      parts.push(`**提示**：${summary.warnings.join('；')}`)
    }
    return parts.join('  \n')
  }

  // 默认问题卡片
  const defaultQuestions = [
    {
      icon: <Target className="w-4 h-4" />,
      title: "调整限上社零额增速",
      content: "将限上社零额增速调整到 7.5%",
    },
    {
      icon: <Building2 className="w-4 h-4" />,
      title: "修改企业数据",
      content: "帮我调整所有小微企业的增速到 30% 左右",
    },
    {
      icon: <TrendingUp className="w-4 h-4" />,
      title: "调整行业指标",
      content: "将零售业销售额增速调整到 15%",
    },
    {
      icon: <Sparkles className="w-4 h-4" />,
      title: "智能优化",
      content: "请帮我优化所有指标，使社零总额增速达到预期目标",
    },
  ]

  const handleQuestionClick = (content: string) => {
    setInput(content)
  }

  if (!open) return null

  return (
    <>
      {/* 遮罩层 - 点击关闭 */}
      <div
        className="fixed inset-0 bg-black/20 z-40"
        onClick={() => onOpenChange(false)}
      />

      {/* 右侧侧边栏 */}
      <div
        data-llm-chat
        className="fixed right-0 top-0 h-full w-[420px] bg-background border-l border-border shadow-2xl z-50 flex flex-col"
      >
        {/* 头部 */}
        <div className="flex items-center justify-between px-4 py-3 border-b border-border bg-muted/30">
          <div className="flex items-center gap-2">
            <MessageSquare className="w-5 h-5 text-primary" />
            <span className="font-semibold text-sm">数据对话助手</span>
          </div>
          <div className="flex items-center gap-2">
            <Button variant="ghost" size="sm" onClick={resetSession} disabled={streaming}>
              新建会话
            </Button>
            <Button
              variant="ghost"
              size="icon"
              className="h-8 w-8"
              onClick={() => onOpenChange(false)}
            >
              <X className="w-4 h-4" />
            </Button>
          </div>
        </div>

        {/* 消息区域 */}
        <ScrollArea className="flex-1 px-4 py-3">
          <div className="space-y-3">
            {messages.length === 0 && (
              <div className="space-y-3">
                <p className="text-sm text-muted-foreground px-1">
                  可以直接描述你要调整的指标目标值或指定企业数据修改，也可以点击下方卡片快速提问：
                </p>
                {/* 默认问题卡片 */}
                <div className="grid grid-cols-1 gap-2">
                  {defaultQuestions.map((q, idx) => (
                    <button
                      key={idx}
                      onClick={() => handleQuestionClick(q.content)}
                      className="flex items-start gap-3 p-3 text-left rounded-lg border border-border bg-card hover:bg-accent hover:border-accent transition-colors group"
                    >
                      <div className="mt-0.5 text-muted-foreground group-hover:text-primary">
                        {q.icon}
                      </div>
                      <div className="flex-1 min-w-0">
                        <div className="text-sm font-medium text-foreground">{q.title}</div>
                        <div className="text-xs text-muted-foreground truncate">{q.content}</div>
                      </div>
                    </button>
                  ))}
                </div>
              </div>
            )}
            {messages.map((message) => (
              <div
                key={message.id}
                className={`flex ${message.role === 'user' ? 'justify-end' : 'justify-start'}`}
              >
                <div
                  className={`max-w-[85%] rounded-2xl px-4 py-2 text-sm leading-relaxed shadow-sm ${
                    message.role === 'user'
                      ? 'bg-primary text-primary-foreground'
                      : 'bg-muted text-foreground border border-border'
                  }`}
                >
                  <div className="text-inherit prose prose-sm dark:prose-invert max-w-none">
                    <ReactMarkdown remarkPlugins={[remarkGfm]}>{message.content}</ReactMarkdown>
                  </div>
                  {message.streaming && <span className="ml-1 animate-pulse">▍</span>}
                </div>
              </div>
            ))}
            <div ref={endRef} />
          </div>
        </ScrollArea>

        {/* 输入区域 */}
        <div className="border-t border-border p-3 bg-muted/30">
          <div className="flex gap-2">
            <Input
              value={input}
              onChange={(e) => setInput(e.target.value)}
              placeholder="输入你的调整需求…"
              disabled={streaming}
              onKeyDown={(e) => {
                if (e.key === 'Enter' && !e.shiftKey) {
                  e.preventDefault()
                  handleSend()
                }
              }}
              className="flex-1"
            />
            <Button onClick={handleSend} disabled={streaming || !input.trim()}>
              {streaming ? '…' : '发送'}
            </Button>
          </div>
          <div className="mt-2 text-xs text-muted-foreground text-center">
            按 Enter 发送，Shift+Enter 换行
          </div>
        </div>
      </div>
    </>
  )
}
