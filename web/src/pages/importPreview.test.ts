/**
 * 导入预览选择测试
 *
 * @author Anner
 * Created on 2026/2/5
 */

import { describe, expect, it } from 'vitest'
import { applyPreviewUpdate } from './importPreview'

describe('applyPreviewUpdate', () => {
  it('keeps selection and bumps token when sheet exists', () => {
    const result = applyPreviewUpdate('批发', 1, [{ sheetName: '批发' }, { sheetName: '零售' }])
    expect(result.selectedSheet).toBe('批发')
    expect(result.refreshToken).toBe(2)
  })

  it('selects first sheet when selection missing', () => {
    const result = applyPreviewUpdate('', 3, [{ sheetName: '住宿' }, { sheetName: '餐饮' }])
    expect(result.selectedSheet).toBe('住宿')
    expect(result.refreshToken).toBe(4)
  })
})
