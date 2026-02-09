import { useEffect, useState } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

interface ConfigResponse {
  llmBaseUrl?: string
  llmModel?: string
  llmApiKey?: string
}

export default function ConfigPage() {
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [baseUrl, setBaseUrl] = useState('')
  const [model, setModel] = useState('')
  const [apiKey, setApiKey] = useState('')

  const load = async () => {
    setLoading(true)
    setError('')
    try {
      const response = await fetch('/api/config')
      if (!response.ok) {
        throw new Error('加载配置失败')
      }
      const data = (await response.json()) as ConfigResponse
      setBaseUrl(data.llmBaseUrl || '')
      setModel(data.llmModel || '')
      setApiKey(data.llmApiKey || '')
    } catch (loadErr) {
      setError(loadErr instanceof Error ? loadErr.message : '加载配置失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void load()
  }, [])

  const save = async () => {
    setSaving(true)
    setError('')
    try {
      const response = await fetch('/api/config', {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          updates: {
            llm_base_url: baseUrl.trim(),
            llm_model: model.trim(),
            llm_api_key: apiKey.trim(),
          },
        }),
      })
      const data = await response.json()
      if (!response.ok) {
        throw new Error(data?.error || '保存失败')
      }
    } catch (saveErr) {
      setError(saveErr instanceof Error ? saveErr.message : '保存失败')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="min-h-full bg-[#F8FAFC] px-8 py-6">
      <div className="space-y-5">
        <header className="flex h-12 items-center justify-between">
          <div className="space-y-0.5">
            <h1 className="text-[20px] font-semibold leading-[28px] text-[#0F172A]">全局配置</h1>
            <p className="text-[13px] text-[#64748B]">配置系统全局参数，包括 AI 大模型接口设置</p>
          </div>
          <Button
            onClick={save}
            disabled={saving || loading}
            className="h-9 rounded-lg bg-[#3B82F6] px-4 text-[13px] font-medium text-white hover:bg-[#2563EB]"
          >
            {saving ? '保存中...' : '保存配置'}
          </Button>
        </header>

        <section className="space-y-5 rounded-xl border border-[#E2E8F0] bg-white p-6">
          <h2 className="text-[15px] font-semibold text-[#0F172A]">AI 大模型配置</h2>

          <div className="space-y-4">
            <div className="space-y-1.5">
              <Label className="text-[13px] font-medium text-[#64748B]">API 地址</Label>
              <Input
                value={baseUrl}
                onChange={(event) => setBaseUrl(event.target.value)}
                placeholder="https://api.openai.com/v1"
                className="h-11 rounded-lg border-[#E2E8F0] px-3.5 text-[14px] text-[#0F172A]"
              />
            </div>

            <div className="space-y-1.5">
              <Label className="text-[13px] font-medium text-[#64748B]">模型名称</Label>
              <Input
                value={model}
                onChange={(event) => setModel(event.target.value)}
                placeholder="gpt-4o"
                className="h-11 rounded-lg border-[#E2E8F0] px-3.5 text-[14px] text-[#0F172A]"
              />
            </div>

            <div className="space-y-1.5">
              <Label className="text-[13px] font-medium text-[#64748B]">API Key</Label>
              <Input
                value={apiKey}
                onChange={(event) => setApiKey(event.target.value)}
                placeholder="sk-xxxxxxxxxxxxxxxx"
                className="h-11 rounded-lg border-[#E2E8F0] px-3.5 text-[14px] text-[#0F172A]"
              />
            </div>
          </div>
        </section>

        {error && <div className="text-xs text-[#DC2626]">{error}</div>}
      </div>
    </div>
  )
}
