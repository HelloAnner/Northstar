/**
 * AI 偏好表单
 *
 * @author Anner
 * Created on 2026/3/14
 */

import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { settingsApi } from '@/services/api'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Textarea } from '@/components/ui/textarea'

interface AIPreferenceFormViewProps {
  value: string
  saving: boolean
  onChange: (value: string) => void
  onSave: () => Promise<void>
}

export function AIPreferenceFormView({ value, saving, onChange, onSave }: AIPreferenceFormViewProps) {
  const count = Array.from(value).length
  const exceeded = count > 500

  return (
    <Card>
      <CardHeader>
        <CardTitle>AI 偏好</CardTitle>
        <CardDescription>
          这里的内容会作为第二层提示词追加到系统内置提示词后面，用于约束 AI 的表达风格和回答偏好。
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <Textarea
          value={value}
          onChange={(event) => onChange(event.target.value)}
          placeholder="例如：回答尽量直接，先说结论，再补充关键依据。"
          className="min-h-[180px]"
        />
        <div className="flex items-center justify-between text-sm">
          <span className={exceeded ? 'text-destructive' : 'text-muted-foreground'}>{count}/500</span>
          <Button onClick={() => void onSave()} disabled={saving || exceeded}>
            {saving ? '保存中…' : '保存偏好'}
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}

export default function AIPreferenceForm() {
  const [value, setValue] = useState('')
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    let active = true
    const load = async () => {
      try {
        const data = await settingsApi.getUserPrompt()
        if (active) {
          setValue(data.content ?? '')
        }
      } catch (error) {
        toast.error(error instanceof Error ? error.message : '加载 AI 偏好失败')
      }
    }
    void load()
    return () => {
      active = false
    }
  }, [])

  const handleSave = async () => {
    setSaving(true)
    try {
      const data = await settingsApi.updateUserPrompt(value)
      setValue(data.content ?? '')
      toast.success('AI 偏好已保存')
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '保存 AI 偏好失败')
    } finally {
      setSaving(false)
    }
  }

  return <AIPreferenceFormView value={value} saving={saving} onChange={setValue} onSave={handleSave} />
}
