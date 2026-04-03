import { useEffect, useState } from 'react'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Check, Loader2, X, Zap, BrainCircuit, Radio, Wrench } from 'lucide-react'
import RuleList from '@/components/RuleList'

interface GlobalConfigDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

interface ConfigFormState {
  baseUrl: string
  model: string
  apiKey: string
}

interface ProbeResult {
  connected: boolean
  streaming: boolean
  tools: boolean
  reasoning: boolean
  error?: string
  latency: number
}

interface Capabilities {
  streaming: boolean
  tools: boolean
  reasoning: boolean
}

export default function GlobalConfigDialog({ open, onOpenChange }: GlobalConfigDialogProps) {
  const [activeTab, setActiveTab] = useState('llm')

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-4xl h-[70vh] overflow-hidden flex flex-col">
        <DialogHeader>
          <DialogTitle>全局配置</DialogTitle>
        </DialogHeader>

        <Tabs value={activeTab} onValueChange={setActiveTab} className="flex-1 min-h-0 flex flex-col">
          <TabsList className="w-fit shrink-0">
            <TabsTrigger value="llm">模型配置</TabsTrigger>
            <TabsTrigger value="rules">规则列表</TabsTrigger>
          </TabsList>

          <TabsContent value="llm" className="flex-1 min-h-0 mt-4 data-[state=active]:flex flex-col">
            <LLMConfigPanel open={open} onOpenChange={onOpenChange} />
          </TabsContent>

          <TabsContent value="rules" className="flex-1 min-h-0 overflow-auto mt-4 data-[state=active]:flex flex-col">
            <RuleList />
          </TabsContent>
        </Tabs>
      </DialogContent>
    </Dialog>
  )
}

// ─── 能力标签 ────────────────────────────────────────────

function CapabilityBadges({ capabilities }: { capabilities: Capabilities | null }) {
  if (!capabilities) return null
  return (
    <div className="flex items-center gap-2">
      <Badge active={capabilities.streaming} icon={<Radio className="h-3 w-3" />} label="流式输出" />
      <Badge active={capabilities.tools} icon={<Wrench className="h-3 w-3" />} label="工具调用" />
      <Badge active={capabilities.reasoning} icon={<BrainCircuit className="h-3 w-3" />} label="深度思考" />
    </div>
  )
}

function Badge({ active, icon, label }: { active: boolean; icon: React.ReactNode; label: string }) {
  if (!active) return null
  return (
    <span className="inline-flex items-center gap-1 rounded-full bg-emerald-50 px-2.5 py-0.5 text-xs font-medium text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400">
      {icon}
      {label}
    </span>
  )
}

// ─── 测试结果弹窗 ────────────────────────────────────────

type TestPhase = 'idle' | 'testing' | 'done'

interface TestItem {
  key: string
  label: string
  icon: React.ReactNode
  status: 'pending' | 'testing' | 'pass' | 'fail'
}

function ProbeDialog({
  open,
  phase,
  items,
  result,
  onClose,
}: {
  open: boolean
  phase: TestPhase
  items: TestItem[]
  result: ProbeResult | null
  onClose: () => void
}) {
  if (!open) return null

  return (
    <Dialog open={open} onOpenChange={() => phase === 'done' && onClose()}>
      <DialogContent className="max-w-sm">
        <DialogHeader>
          <DialogTitle className="text-base">模型能力检测</DialogTitle>
        </DialogHeader>
        <div className="space-y-3 py-2">
          {items.map((item) => (
            <div key={item.key} className="flex items-center gap-3">
              <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-stone-100 text-stone-500 dark:bg-stone-800 dark:text-stone-400">
                {item.icon}
              </div>
              <span className="flex-1 text-sm text-stone-700 dark:text-stone-300">{item.label}</span>
              <TestStatus status={item.status} />
            </div>
          ))}

          {result?.error && (
            <div className="rounded-lg bg-red-50 px-3 py-2 text-xs text-red-600 dark:bg-red-950/30 dark:text-red-400">
              {result.error}
            </div>
          )}

          {result && result.connected && (
            <div className="text-xs text-stone-400 text-center pt-1">
              响应延迟 {result.latency}ms
            </div>
          )}
        </div>

        {phase === 'done' && (
          <div className="flex justify-end pt-1">
            <Button size="sm" onClick={onClose}>确定</Button>
          </div>
        )}
      </DialogContent>
    </Dialog>
  )
}

function TestStatus({ status }: { status: TestItem['status'] }) {
  switch (status) {
    case 'pending':
      return <div className="h-4 w-4 rounded-full border-2 border-stone-200 dark:border-stone-700" />
    case 'testing':
      return <Loader2 className="h-4 w-4 animate-spin text-primary" />
    case 'pass':
      return (
        <div className="flex h-5 w-5 items-center justify-center rounded-full bg-emerald-100 dark:bg-emerald-900/40">
          <Check className="h-3 w-3 text-emerald-600 dark:text-emerald-400" />
        </div>
      )
    case 'fail':
      return (
        <div className="flex h-5 w-5 items-center justify-center rounded-full bg-red-100 dark:bg-red-900/40">
          <X className="h-3 w-3 text-red-500 dark:text-red-400" />
        </div>
      )
  }
}

// ─── 配置面板 ────────────────────────────────────────────

function LLMConfigPanel({ open, onOpenChange }: { open: boolean; onOpenChange: (open: boolean) => void }) {
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [showApiKey, setShowApiKey] = useState(false)
  const [error, setError] = useState('')
  const [form, setForm] = useState<ConfigFormState>({ baseUrl: '', model: '', apiKey: '' })
  const [capabilities, setCapabilities] = useState<Capabilities | null>(null)

  // 测试状态
  const [probeOpen, setProbeOpen] = useState(false)
  const [probePhase, setProbePhase] = useState<TestPhase>('idle')
  const [probeResult, setProbeResult] = useState<ProbeResult | null>(null)
  const [probeItems, setProbeItems] = useState<TestItem[]>([
    { key: 'connect', label: '连通性', icon: <Zap className="h-4 w-4" />, status: 'pending' },
    { key: 'stream', label: '流式输出', icon: <Radio className="h-4 w-4" />, status: 'pending' },
    { key: 'tools', label: '工具调用', icon: <Wrench className="h-4 w-4" />, status: 'pending' },
    { key: 'reason', label: '深度思考', icon: <BrainCircuit className="h-4 w-4" />, status: 'pending' },
  ])

  useEffect(() => {
    if (!open) return
    setError('')
    loadConfig()
  }, [open])

  const loadConfig = async () => {
    setLoading(true)
    try {
      const res = await fetch('/api/config')
      if (!res.ok) throw new Error('加载配置失败')
      const data = await res.json()
      setForm({
        baseUrl: data.llmBaseUrl ?? '',
        model: data.llmModel ?? '',
        apiKey: data.llmApiKey ?? '',
      })
      // 加载已保存的能力标签
      if (data.llmSupportsStreaming || data.llmSupportsTools || data.llmSupportsReasoning) {
        setCapabilities({
          streaming: data.llmSupportsStreaming ?? false,
          tools: data.llmSupportsTools ?? false,
          reasoning: data.llmSupportsReasoning ?? false,
        })
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : '加载配置失败')
    } finally {
      setLoading(false)
    }
  }

  const updateField = (key: keyof ConfigFormState, value: string) => {
    setForm((prev) => ({ ...prev, [key]: value }))
  }

  const handleSave = async () => {
    setSaving(true)
    setError('')
    try {
      const res = await fetch('/api/config', {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          updates: {
            llm_base_url: form.baseUrl.trim(),
            llm_model: form.model.trim(),
            llm_api_key: form.apiKey.trim(),
          },
        }),
      })
      const data = await res.json()
      if (!res.ok) {
        throw new Error(data?.error || '保存失败')
      }
      // 保存成功后自动开始测试
      runProbe()
    } catch (err) {
      if (err instanceof Error && err.message === 'Failed to fetch') {
        setError('保存失败：后端不可达或请求被拦截')
      } else {
        setError(err instanceof Error ? err.message : '保存失败')
      }
    } finally {
      setSaving(false)
    }
  }

  const runProbe = async () => {
    // 重置测试状态
    const items: TestItem[] = [
      { key: 'connect', label: '连通性', icon: <Zap className="h-4 w-4" />, status: 'testing' },
      { key: 'stream', label: '流式输出', icon: <Radio className="h-4 w-4" />, status: 'pending' },
      { key: 'reason', label: '深度思考', icon: <BrainCircuit className="h-4 w-4" />, status: 'pending' },
    ]
    setProbeItems(items)
    setProbeResult(null)
    setProbePhase('testing')
    setProbeOpen(true)

    try {
      const res = await fetch('/api/config/llm/test', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          baseUrl: form.baseUrl.trim(),
          model: form.model.trim(),
          apiKey: form.apiKey.trim(),
        }),
      })
      const result: ProbeResult = await res.json()
      setProbeResult(result)

      // 按顺序更新每个测试项状态
      const updated: TestItem[] = [
        { ...items[0], status: result.connected ? 'pass' : 'fail' },
        { ...items[1], status: result.connected ? (result.streaming ? 'pass' : 'fail') : 'fail' },
        { ...items[2], status: result.connected ? (result.tools ? 'pass' : 'fail') : 'fail' },
        { ...items[3], status: result.connected ? (result.reasoning ? 'pass' : 'fail') : 'fail' },
      ]

      // 逐步显示结果，每项间隔 300ms
      for (let i = 0; i < updated.length; i++) {
        await new Promise((r) => setTimeout(r, 300))
        setProbeItems((prev) => prev.map((item, idx) => (idx <= i ? updated[idx] : { ...item, status: idx === i + 1 ? 'testing' : 'pending' })))
      }

      // 更新能力标签
      if (result.connected) {
        setCapabilities({ streaming: result.streaming, tools: result.tools, reasoning: result.reasoning })
      }
    } catch {
      setProbeItems((prev) => prev.map((item) => ({ ...item, status: 'fail' as const })))
      setProbeResult({ connected: false, streaming: false, tools: false, reasoning: false, error: '测试请求失败', latency: 0 })
    } finally {
      setProbePhase('done')
    }
  }

  return (
    <div className="flex flex-col flex-1 min-h-0">
      <div className="space-y-4">
        {/* 能力标签 */}
        {capabilities && (
          <div className="flex items-center gap-3 rounded-lg bg-stone-50 px-4 py-2.5 dark:bg-stone-800/50">
            <span className="text-xs text-stone-500 dark:text-stone-400">模型能力</span>
            <CapabilityBadges capabilities={capabilities} />
            {!capabilities.streaming && !capabilities.tools && !capabilities.reasoning && (
              <span className="text-xs text-stone-400">暂无检测到的特殊能力</span>
            )}
          </div>
        )}

        <div className="space-y-2">
          <Label htmlFor="llm-base-url">接口地址</Label>
          <Input
            id="llm-base-url"
            value={form.baseUrl}
            onChange={(e) => updateField('baseUrl', e.target.value)}
            placeholder="例如：https://api.openai.com/v1"
            disabled={loading}
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="llm-model">模型名称</Label>
          <Input
            id="llm-model"
            value={form.model}
            onChange={(e) => updateField('model', e.target.value)}
            placeholder="例如：gpt-4o-mini"
            disabled={loading}
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="llm-api-key">API 密钥</Label>
          <div className="flex gap-2">
            <Input
              id="llm-api-key"
              type={showApiKey ? 'text' : 'password'}
              value={form.apiKey}
              onChange={(e) => updateField('apiKey', e.target.value)}
              placeholder="例如：sk-..."
              disabled={loading}
            />
            <Button
              type="button"
              variant="outline"
              onClick={() => setShowApiKey((prev) => !prev)}
              disabled={loading}
            >
              {showApiKey ? '隐藏' : '显示'}
            </Button>
          </div>
        </div>

        {error && <div className="text-sm text-destructive">{error}</div>}
      </div>

      <div className="flex justify-end gap-2 pt-4 mt-auto">
        <Button variant="outline" onClick={() => onOpenChange(false)} disabled={saving}>
          取消
        </Button>
        <Button onClick={handleSave} disabled={saving || loading}>
          {saving ? '保存中…' : '保存并测试'}
        </Button>
      </div>

      {/* 测试结果弹窗 */}
      <ProbeDialog
        open={probeOpen}
        phase={probePhase}
        items={probeItems}
        result={probeResult}
        onClose={() => {
          setProbeOpen(false)
          // 测试通过后关闭配置面板
          if (probeResult?.connected) {
            onOpenChange(false)
          }
        }}
      />
    </div>
  )
}
