import { ChevronsUpDown } from 'lucide-react'
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger } from '@/components/ui/dropdown-menu'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import { useProjectStore } from '@/store'
import { useLocation, useNavigate } from 'react-router-dom'

function StatusPill({ text }: { text: string }) {
  return (
    <div className="inline-flex h-6 items-center rounded-full border border-white/10 bg-[#00D4AA22] px-2.5 text-[11px] font-normal text-[#00D4AA]">
      {text}
    </div>
  )
}

export default function Topbar({ title, statusText }: { title: string; statusText: string }) {
  const navigate = useNavigate()
  const location = useLocation()
  const { current, index, refreshIndex, refreshCurrent, selectProject } = useProjectStore()

  const currentLabel = current?.project?.projectId
    ? `当前项目：${current.project.name || current.project.projectId}`
    : '当前项目：未选择'

  const items = index?.items ?? []

  return (
    <div className="flex h-14 items-center justify-between border-b border-white/10 px-5">
      <div className="flex items-center gap-2">
          <div className="text-[11px] text-[#A3A3A3]">{title}</div>
      </div>

      <div className="flex items-center gap-2.5">
        <DropdownMenu
          onOpenChange={(open) => {
            if (open) {
              refreshCurrent()
              refreshIndex()
            }
          }}
        >
          <DropdownMenuTrigger asChild>
            <Button
              variant="secondary"
              className={cn(
                'h-9 gap-2 rounded-full border border-white/10 bg-[#2D2D2D] px-3 text-[11px] font-normal text-white hover:bg-[#2D2D2D]'
              )}
            >
              <ChevronsUpDown className="h-3.5 w-3.5 text-white" />
              <span className="max-w-[220px] truncate">{currentLabel}</span>
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-[280px] border-white/10 bg-[#0D0D0D] text-white">
            <DropdownMenuItem
              className="text-[12px] text-[#A3A3A3]"
              onSelect={() => navigate('/')}
            >
              进入项目中心
            </DropdownMenuItem>
            <DropdownMenuSeparator className="bg-white/10" />
            {items.length === 0 ? (
              <DropdownMenuItem className="text-[12px] text-[#A3A3A3]">
                暂无项目
              </DropdownMenuItem>
            ) : (
              items.slice(0, 8).map((p) => {
                const active = current?.project?.projectId === p.projectId
                return (
                  <DropdownMenuItem
                    key={p.projectId}
                    className={cn(
                      'flex items-center justify-between text-[12px]',
                      active ? 'bg-white/5' : ''
                    )}
                    onSelect={async () => {
                      const selected = await selectProject(p.projectId)
                      const target = selected.hasData ? '/dashboard' : '/import'
                      if (location.pathname !== target) {
                        navigate(target)
                      }
                    }}
                  >
                    <span className="truncate text-white">{p.name || p.projectId}</span>
                    <span className="ml-2 text-[11px] text-[#A3A3A3]">{p.hasData ? '已就绪' : '需要导入'}</span>
                  </DropdownMenuItem>
                )
              })
            )}
          </DropdownMenuContent>
        </DropdownMenu>

        <StatusPill text={statusText} />
      </div>
    </div>
  )
}
