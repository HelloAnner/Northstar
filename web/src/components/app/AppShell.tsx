import { Outlet } from 'react-router-dom'
import Sidebar from '@/components/app/Sidebar'

export default function AppShell() {
  return (
    <div className="flex h-screen w-screen bg-[#1A1A1A] text-white">
      <Sidebar />
      <div className="flex min-w-0 flex-1 flex-col">
        <Outlet />
      </div>
    </div>
  )
}

