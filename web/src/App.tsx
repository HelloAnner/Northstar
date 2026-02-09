import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import AppShell from '@/components/app/AppShell'
import DashboardV3 from '@/pages/DashboardV3'
import ConfigPage from '@/pages/ConfigPage'
import HelpDocument from '@/pages/HelpDocument'
import IndicatorsPage from '@/pages/IndicatorsPage'
import RulesPage from '@/pages/RulesPage'
import { Toaster } from '@/components/ui/sonner'

function App() {
  return (
    <BrowserRouter>
      <Toaster />
      <Routes>
        <Route element={<AppShell />}>
          <Route path="/" element={<DashboardV3 />} />
          <Route path="/indicators" element={<IndicatorsPage />} />
          <Route path="/rules" element={<RulesPage />} />
          <Route path="/config" element={<ConfigPage />} />
          <Route path="/help" element={<HelpDocument />} />
          <Route path="/import" element={<Navigate to="/?import=1" replace />} />
        </Route>
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </BrowserRouter>
  )
}

export default App
