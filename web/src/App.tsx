import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import AppShell from '@/components/app/AppShell'
import ProjectHub from '@/pages/ProjectHub'
import ProjectDetail from '@/pages/ProjectDetail'
import Dashboard from '@/pages/Dashboard'
import ImportWizard from '@/pages/ImportWizard'
import HelpDocument from '@/pages/HelpDocument'

function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route element={<AppShell />}>
          <Route path="/" element={<ProjectHub />} />
          <Route path="/projects/:id" element={<ProjectDetail />} />
          <Route path="/dashboard" element={<Dashboard />} />
          <Route path="/import" element={<ImportWizard />} />
        </Route>
        <Route path="/help" element={<HelpDocument />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </BrowserRouter>
  )
}

export default App
