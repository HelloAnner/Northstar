/**
 * 规则列表组件测试
 *
 * @author Anner
 * Created on 2026/3/14
 */

import { describe, expect, it, vi } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import { RuleListView } from './RuleList'

describe('RuleListView', () => {
  it('renders rules and error status details', () => {
    const html = renderToStaticMarkup(
      <RuleListView
        rules={[
          { index: 1, text: '零售增速不超过 15%' },
          { index: 2, text: '批发增速不低于 0%' },
        ]}
        status="error"
        statusError="indicator 不合法"
        statusUpdatedAt="2026-03-14T10:30:00Z"
        loading={false}
        submitting={false}
        onAdd={async () => {}}
        onEdit={async () => {}}
        onDelete={async () => {}}
        onRefresh={async () => {}}
        onConvert={async () => {}}
      />
    )

    expect(html).toContain('调整规则')
    expect(html).toContain('零售增速不超过 15%')
    expect(html).toContain('批发增速不低于 0%')
    expect(html).toContain('转换失败')
    expect(html).toContain('indicator 不合法')
  })

  it('renders idle state and manual convert action', () => {
    const onConvert = vi.fn(async () => {})
    const html = renderToStaticMarkup(
      <RuleListView
        rules={[]}
        status="idle"
        statusError=""
        statusUpdatedAt=""
        loading={false}
        submitting={false}
        onAdd={async () => {}}
        onEdit={async () => {}}
        onDelete={async () => {}}
        onRefresh={async () => {}}
        onConvert={onConvert}
      />
    )

    expect(html).toContain('待转换')
    expect(html).toContain('重新转换')
  })
})
