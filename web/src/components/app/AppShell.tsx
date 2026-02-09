import { Outlet } from 'react-router-dom'
import SidebarNav from '@/components/app/SidebarNav'

export default function AppShell() {
  return (
    <div className="flex h-screen w-screen overflow-hidden bg-[#F8FAFC]">
      <SidebarNav />
      <div className="flex min-w-0 flex-1 flex-col overflow-auto">
        <Outlet />
      </div>
    </div>
  )
}
