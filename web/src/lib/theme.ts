/**
 * 主题工具（读取/切换/持久化）
 *
 * @author Anner
 * Created on 2026/2/3
 */

export type ThemeMode = 'light' | 'dark'

const STORAGE_KEY = 'northstar-theme'

const isThemeMode = (value: string | null): value is ThemeMode => value === 'light' || value === 'dark'

export const getStoredTheme = (): ThemeMode | null => {
  if (typeof window === 'undefined') return null
  const stored = window.localStorage.getItem(STORAGE_KEY)
  return isThemeMode(stored) ? stored : null
}

export const setStoredTheme = (theme: ThemeMode) => {
  if (typeof window === 'undefined') return
  window.localStorage.setItem(STORAGE_KEY, theme)
}

export const getInitialTheme = (): ThemeMode => {
  const stored = getStoredTheme()
  if (stored) return stored
  if (typeof window === 'undefined') return 'light'
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

export const applyTheme = (theme: ThemeMode) => {
  if (typeof document === 'undefined') return
  const root = document.documentElement
  root.classList.toggle('dark', theme === 'dark')
  root.style.colorScheme = theme
}
