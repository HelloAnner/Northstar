import { useEffect, useState } from 'react'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
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

function LLMConfigPanel({ open, onOpenChange }: { open: boolean; onOpenChange: (open: boolean) => void }) {
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [showApiKey, setShowApiKey] = useState(false)
  const [error, setError] = useState('')
  const [form, setForm] = useState<ConfigFormState>({ baseUrl: '', model: '', apiKey: '' })

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
      onOpenChange(false)
    } catch (err) {
      if (err instanceof Error && err.message === 'Failed to fetch') {
        setError('保存失败：后端不可达或请求被拦截（Failed to fetch）')
      } else {
        setError(err instanceof Error ? err.message : '保存失败')
      }
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="flex flex-col flex-1 min-h-0">
      <div className="space-y-4">
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
          {saving ? '保存中…' : '保存'}
        </Button>
      </div>
    </div>
  )
}
