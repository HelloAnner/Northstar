/**
 * 全局 Toast 容器（sonner）
 *
 * @author Anner
 * Created on 2026/2/4
 */

import { useEffect, useState } from 'react'
import { Toaster as Sonner } from 'sonner'
import { getInitialTheme } from '@/lib/theme'

export function Toaster() {
  const [theme, setTheme] = useState<'light' | 'dark'>(() => getInitialTheme())

  useEffect(() => {
    const root = document.documentElement
    const sync = () => {
      setTheme(root.classList.contains('dark') ? 'dark' : 'light')
    }
    sync()
    const observer = new MutationObserver(sync)
    observer.observe(root, { attributes: true, attributeFilter: ['class'] })
    return () => observer.disconnect()
  }, [])

  return (
    <Sonner
      theme={theme}
      position="top-center"
      duration={3000}
      toastOptions={{
        classNames: {
          toast:
            'bg-background text-foreground border-border shadow-lg rounded-xl px-4 py-3 max-w-[520px]',
          title: 'text-sm font-semibold',
          description: 'text-xs text-muted-foreground',
        },
      }}
    />
  )
}
