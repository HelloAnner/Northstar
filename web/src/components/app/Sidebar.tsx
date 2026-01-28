import { NavLink } from 'react-router-dom'
import { Clock, Folder } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { useProjectStore } from '@/store'
import { useNavigate } from 'react-router-dom'

export default function Sidebar() {
  const navigate = useNavigate()
  const { index, refreshIndex, refreshCurrent, selectProject } = useProjectStore()

  const lastEditedProjectId = index?.lastEditedProjectId || ''

  return (
    <aside className="flex h-full w-[260px] flex-col border-r border-white/10 bg-[#0D0D0D]">
      <div className="flex h-16 items-center gap-3 border-b border-white/10 px-4">
        <div className="h-7 w-7 rounded bg-[#FF6B35]" />
        <div className="text-base font-semibold tracking-tight">数据管理与模拟平台</div>
      </div>

      <nav className="flex flex-1 flex-col gap-1.5 px-3 py-3">
        <Button
          variant="ghost"
          className={cn(
            'h-10 justify-start gap-2 rounded-lg px-3 text-[13px] font-normal',
            'bg-transparent text-[#D4D4D4] hover:bg-white/5 hover:text-white'
          )}
          asChild
        >
          <NavLink
            to="/"
            end
            className={({ isActive }) =>
              cn(
                'relative flex w-full items-center justify-start gap-2 text-left',
                isActive && 'bg-[#2D2D2D] text-white hover:bg-[#2D2D2D]',
                isActive &&
                  "before:absolute before:left-0 before:top-0 before:h-full before:w-[3px] before:rounded-l-lg before:bg-[#FF6B35]"
              )
            }
          >
            <Folder className="h-4 w-4" />
            <span>项目中心</span>
          </NavLink>
        </Button>

        <Button
          variant="ghost"
          disabled={!lastEditedProjectId}
          className={cn(
            'h-10 justify-start gap-2 rounded-lg px-3 text-[13px] font-normal',
            'bg-transparent text-[#D4D4D4] hover:bg-white/5 hover:text-white',
            !lastEditedProjectId ? 'opacity-50' : ''
          )}
          onClick={async () => {
            // 避免首次进入时 index 为空
            await Promise.all([refreshCurrent(), refreshIndex()])
            const id = useProjectStore.getState().index?.lastEditedProjectId
            if (!id) return
            const selected = await selectProject(id)
            navigate(selected.hasData ? '/dashboard' : '/import')
          }}
        >
          <Clock className="h-4 w-4" />
          <span>上次修改项目</span>
        </Button>
      </nav>
    </aside>
  )
}
