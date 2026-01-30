import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import AppShell from '@/components/app/AppShell'
import DashboardV3 from '@/pages/DashboardV3'
import HelpDocument from '@/pages/HelpDocument'

function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route element={<AppShell />}>
          <Route path="/" element={<DashboardV3 />} />
        </Route>
        <Route path="/help" element={<HelpDocument />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </BrowserRouter>
  )
}

export default App
