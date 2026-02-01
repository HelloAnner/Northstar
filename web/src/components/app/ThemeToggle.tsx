/**
 * 主题切换按钮
 *
 * @author Anner
 * Created on 2026/2/1
 */

import { useEffect, useState } from 'react'
import { Moon, Sun } from 'lucide-react'
import { Button, type ButtonProps } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import { applyTheme, getInitialTheme, setStoredTheme, type ThemeMode } from '@/lib/theme'

interface ThemeToggleProps {
  className?: string
  variant?: ButtonProps['variant']
  size?: ButtonProps['size']
}

export default function ThemeToggle({ className, variant = 'outline', size = 'icon' }: ThemeToggleProps) {
  const [theme, setTheme] = useState<ThemeMode>(() => getInitialTheme())

  useEffect(() => {
    applyTheme(theme)
    setStoredTheme(theme)
  }, [theme])

  const isDark = theme === 'dark'
  const label = isDark ? '切换到明亮模式' : '切换到暗色模式'

  return (
    <Button
      aria-label={label}
      className={cn('rounded-full', className)}
      size={size}
      variant={variant}
      onClick={() => setTheme(isDark ? 'light' : 'dark')}
    >
      {isDark ? <Sun className="h-4 w-4" /> : <Moon className="h-4 w-4" />}
    </Button>
  )
}
