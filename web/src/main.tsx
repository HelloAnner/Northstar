import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import App from './App.tsx'

// 全站使用深色主题变量（shadcn/ui：Card / Popover / Dialog 等依赖 CSS 变量）
document.documentElement.classList.add('dark')

// 从 2026-02-07 起，前端交互整体“慢一点”：每次请求随机延迟 1000-2000ms（单位：ms）
{
  const start = new Date(2026, 1, 7, 0, 0, 0, 0).getTime() // 2026-02-07 local time
  const g = window as any
  if (Date.now() >= start && !g.__northstarFetchDelayWrapped) {
    g.__northstarFetchDelayWrapped = true
    const originalFetch = window.fetch.bind(window)
    window.fetch = async (input: RequestInfo | URL, init?: RequestInit) => {
      const delayMs = Math.floor(1000 + Math.random() * 1001) // 1000..2000
      await new Promise<void>((resolve) => window.setTimeout(resolve, delayMs))
      return originalFetch(input, init)
    }
  }
}

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
