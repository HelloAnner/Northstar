/**
 * AI 偏好表单测试
 *
 * @author Anner
 * Created on 2026/3/14
 */

import { describe, expect, it } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import { AIPreferenceFormView } from './AIPreferenceForm'

describe('AIPreferenceFormView', () => {
  it('renders content count and save button', () => {
    const html = renderToStaticMarkup(
      <AIPreferenceFormView
        value="回答尽量直接。"
        saving={false}
        onChange={() => {}}
        onSave={async () => {}}
      />
    )

    expect(html).toContain('AI 偏好')
    expect(html).toContain('7/500')
    expect(html).toContain('保存偏好')
  })
})
