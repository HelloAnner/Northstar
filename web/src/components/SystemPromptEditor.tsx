/**
 * 系统提示词编辑器
 *
 * @author Anner
 * Created on 2026/4/6
 */

import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { settingsApi } from '@/services/api'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Textarea } from '@/components/ui/textarea'

const MAX_LENGTH = 5000

export default function SystemPromptEditor({ embedded = false }: { embedded?: boolean }) {
  const [value, setValue] = useState('')
  const [saving, setSaving] = useState(false)
  const [loaded, setLoaded] = useState(false)

  useEffect(() => {
    let active = true
    const load = async () => {
      try {
        const data = await settingsApi.getSystemPrompt()
        if (active) {
          setValue(data.content ?? '')
          setLoaded(true)
        }
      } catch (error) {
        toast.error(error instanceof Error ? error.message : '加载系统提示词失败')
      }
    }
    void load()
    return () => { active = false }
  }, [])

  const handleSave = async () => {
    setSaving(true)
    try {
      const data = await settingsApi.updateSystemPrompt(value)
      setValue(data.content ?? '')
      toast.success('系统提示词已保存')
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '保存系统提示词失败')
    } finally {
      setSaving(false)
    }
  }

  const count = Array.from(value).length
  const exceeded = count > MAX_LENGTH

  const content = (
    <div className="flex flex-col flex-1 min-h-0 space-y-3">
      {!embedded && (
        <p className="text-sm text-muted-foreground">
          定义 AI 助手的业务背景知识和回答规则。修改后对所有新对话生效。
        </p>
      )}
      <Textarea
        value={value}
        onChange={(e) => setValue(e.target.value)}
        placeholder={loaded ? '' : '加载中…'}
        className="flex-1 min-h-[300px] font-mono text-sm leading-relaxed"
        disabled={!loaded}
      />
      <div className="flex items-center justify-between text-sm shrink-0">
        <span className={exceeded ? 'text-destructive' : 'text-muted-foreground'}>
          {count}/{MAX_LENGTH}
        </span>
        <Button onClick={() => void handleSave()} disabled={saving || exceeded || !loaded}>
          {saving ? '保存中…' : '保存'}
        </Button>
      </div>
    </div>
  )

  if (embedded) return content

  return (
    <Card>
      <CardHeader>
        <CardTitle>系统提示词</CardTitle>
        <CardDescription>
          定义 AI 助手的业务背景知识和回答规则。修改后对所有新对话生效。
        </CardDescription>
      </CardHeader>
      <CardContent>{content}</CardContent>
    </Card>
  )
}
