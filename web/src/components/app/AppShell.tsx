import { Outlet } from 'react-router-dom'

export default function AppShell() {
  return (
    <div className="flex h-screen w-screen bg-gray-50">
      <div className="flex min-w-0 flex-1 flex-col">
        <Outlet />
      </div>
    </div>
  )
}

