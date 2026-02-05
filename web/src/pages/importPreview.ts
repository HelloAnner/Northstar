/**
 * 导入预览选择辅助
 *
 * @author Anner
 * Created on 2026/2/5
 */

export type PreviewSheetLite = {
  sheetName: string
}

export type PreviewSelection = {
  selectedSheet: string
  refreshToken: number
}

export function applyPreviewUpdate(
  prevSelected: string,
  prevToken: number,
  sheets: PreviewSheetLite[]
): PreviewSelection {
  if (sheets.length === 0) {
    return { selectedSheet: '', refreshToken: prevToken + 1 }
  }
  const names = new Set(sheets.map((s) => s.sheetName))
  if (!prevSelected || !names.has(prevSelected)) {
    return { selectedSheet: sheets[0].sheetName, refreshToken: prevToken + 1 }
  }
  return { selectedSheet: prevSelected, refreshToken: prevToken + 1 }
}
