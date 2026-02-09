import { Link, useLocation, useNavigate } from 'react-router-dom'
import { Calculator, Info, LayoutDashboard, MessageCircle, Settings, Star, Target } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

type NavItem = {
  to: string
  label: string
  icon: React.ComponentType<{ className?: string }>
}

const navItems: NavItem[] = [
  { to: '/', label: '仪表盘', icon: LayoutDashboard },
  { to: '/indicators', label: '指标中心', icon: Target },
  { to: '/rules', label: '计算规则', icon: Calculator },
  { to: '/config', label: '全局配置', icon: Settings },
  { to: '/help', label: '帮助文档', icon: Info },
]

export default function SidebarNav() {
  const location = useLocation()
  const navigate = useNavigate()

  const isActive = (to: string) => {
    if (to === '/') {
      return location.pathname === '/'
    }
    return location.pathname.startsWith(to)
  }

  const openAiAssistant = () => {
    navigate('/?chat=1')
  }

  return (
    <aside className="flex h-full w-[220px] shrink-0 flex-col justify-between border-r border-[#E2E8F0] bg-white px-3 py-5">
      <div className="space-y-5">
        <div className="flex h-10 items-center gap-[10px] px-1">
          <div className="flex h-7 w-7 items-center justify-center rounded-md bg-[#3B82F6]">
            <Star className="h-4 w-4 fill-white text-white" />
          </div>
          <span className="text-base font-semibold text-[#0F172A]">Northstar</span>
        </div>

        <nav className="space-y-0.5">
          {navItems.map((item) => {
            const active = isActive(item.to)
            const Icon = item.icon
            return (
              <Link
                key={item.to}
                to={item.to}
                className={cn(
                  'flex h-10 items-center gap-[10px] rounded-lg px-3 text-sm transition-colors',
                  active
                    ? 'bg-[#EFF6FF] text-[#3B82F6]'
                    : 'text-[#64748B] hover:bg-[#F8FAFC] hover:text-[#334155]',
                )}
              >
                <Icon className="h-[18px] w-[18px]" />
                <span className={cn(active ? 'font-medium' : 'font-normal')}>{item.label}</span>
              </Link>
            )
          })}
        </nav>
      </div>

      <div className="pb-1">
        <Button
          onClick={openAiAssistant}
          className="h-11 w-full justify-center gap-2 rounded-lg bg-[#3B82F6] text-white hover:bg-[#2563EB]"
        >
          <MessageCircle className="h-[18px] w-[18px]" />
          AI 助手
        </Button>
      </div>
    </aside>
  )
}
