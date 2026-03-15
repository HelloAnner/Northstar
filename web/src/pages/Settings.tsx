/**
 * 设置页面
 *
 * @author Anner
 * Created on 2026/3/14
 */

import AIPreferenceForm from '@/components/AIPreferenceForm'
import RuleList from '@/components/RuleList'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'

export default function Settings() {
  return (
    <div className="min-h-screen bg-gradient-to-b from-background via-background to-muted/20 px-6 py-8">
      <div className="mx-auto flex w-full max-w-6xl flex-col gap-6">
        <div className="space-y-2">
          <div className="text-xs font-semibold uppercase tracking-[0.22em] text-muted-foreground">Settings</div>
          <h1 className="text-3xl font-semibold tracking-tight text-foreground">规则与系统偏好</h1>
          <p className="max-w-3xl text-sm leading-6 text-muted-foreground">
            在这里维护自然语言规则，系统会自动生成结构化角色规则并热重载到调整引擎。
          </p>
        </div>

        <Tabs defaultValue="rules" className="space-y-4">
          <TabsList>
            <TabsTrigger value="rules">调整规则</TabsTrigger>
            <TabsTrigger value="ai">AI 偏好</TabsTrigger>
          </TabsList>
          <TabsContent value="rules">
            <RuleList />
          </TabsContent>
          <TabsContent value="ai">
            <AIPreferenceForm />
          </TabsContent>
        </Tabs>
      </div>
    </div>
  )
}
