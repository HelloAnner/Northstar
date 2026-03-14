/**
 * AI 对话面板测试
 *
 * @author Anner
 * Created on 2026/3/14
 */

import { describe, expect, it } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import { ChatPanelView } from './ChatPanel'

describe('ChatPanelView', () => {
  it('renders adjust mode and applied rule bubbles', () => {
    const html = renderToStaticMarkup(
      <ChatPanelView
        open
        mode="adjust"
        input="把批发当月增速调到 15%"
        streaming={false}
        messages={[
          { id: 'u1', role: 'user', content: '把批发当月增速调到 15%' },
          {
            id: 'a1',
            role: 'assistant',
            content: '已完成调整',
            appliedRules: [
              { ruleId: 'c1', type: 'clamp_target', indicatorId: 'wholesale_month_rate', beforeValue: 15, afterValue: 10 },
              { ruleId: 'f1', type: 'filter_allocation', indicatorId: 'retail_month_rate', beforeCount: 8, afterCount: 3 },
              { ruleId: 'cp1', type: 'compensate', ensureId: 'wholesale_month_rate', targetValue: 12 },
            ],
          },
        ]}
        onClose={() => {}}
        onInputChange={() => {}}
        onModeChange={() => {}}
        onReset={() => {}}
        onSend={() => {}}
      />
    )

    expect(html).toContain('数据对话助手')
    expect(html).toContain('调整')
    expect(html).toContain('目标值从 15 裁剪为 10')
    expect(html).toContain('参与企业从 8 过滤为 3 家')
    expect(html).toContain('联动补偿 wholesale_month_rate → 12')
  })
})
