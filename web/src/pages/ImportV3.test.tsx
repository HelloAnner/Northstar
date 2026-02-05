/**
 * 导入全屏页面测试
 *
 * @author Anner
 * Created on 2026/2/5
 */

import { describe, expect, it } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import { MemoryRouter } from 'react-router-dom'
import ImportV3 from './ImportV3'

describe('ImportV3', () => {
  it('renders import layout with empty state', () => {
    let html = ''
    expect(() => {
      html = renderToStaticMarkup(
        <MemoryRouter>
          <ImportV3 />
        </MemoryRouter>
      )
    }).not.toThrow()
    expect(html).toContain('data-testid="import-left-pane"')
    expect(html).toContain('data-testid="import-right-pane"')
    expect(html).toContain('暂无进度')
    expect(html).toContain('暂无日志')
    expect(html).toContain('选择文件')
  })
})
