import { Routes, Route, Navigate } from 'react-router-dom'
import { useAuth } from './contexts/AuthContext'
import Layout from './components/Layout'
import Login from './pages/Login'
import RecipientProfile from './pages/RecipientProfile'
import Contacts from './pages/Contacts'
import Preferences from './pages/Preferences'
import History from './pages/History'
import Templates from './pages/Templates'
import Groups from './pages/Groups'
import Campaigns from './pages/Campaigns'
import OperatorHistory from './pages/OperatorHistory'
import Analytics from './pages/Analytics'
import Unsubscribe from './pages/Unsubscribe'

interface ProtectedRouteProps {
  children: React.ReactNode
  allowedRoles?: string[]
}

function ProtectedRoute({ children, allowedRoles }: ProtectedRouteProps) {
  const { user, isLoading } = useAuth()

  if (isLoading) {
    return (
      <div className="loading-screen">
        <div className="spinner"></div>
        <p>Загрузка...</p>
      </div>
    )
  }

  if (!user) {
    return <Navigate to="/login" replace />
  }

  if (allowedRoles && !allowedRoles.includes(user.role)) {
    return <Navigate to="/app/profile" replace />
  }

  return <>{children}</>
}

function App() {
  return (
    <Routes>
      <Route path="/login" element={<Login />} />
      <Route path="/unsubscribe" element={<Unsubscribe />} />
      
      <Route path="/app" element={
        <ProtectedRoute allowedRoles={['recipient_user', 'source_operator', 'admin']}>
          <Layout />
        </ProtectedRoute>
      }>
        <Route index element={<Navigate to="/app/profile" replace />} />
        <Route path="profile" element={<RecipientProfile />} />
        <Route path="contacts" element={<Contacts />} />
        <Route path="preferences" element={<Preferences />} />
        <Route path="history" element={<History />} />
      </Route>

      <Route path="/operator" element={
        <ProtectedRoute allowedRoles={['source_operator', 'admin']}>
          <Layout isOperator />
        </ProtectedRoute>
      }>
        <Route index element={<Navigate to="/operator/templates" replace />} />
        <Route path="templates" element={<Templates />} />
        <Route path="groups" element={<Groups />} />
        <Route path="campaigns" element={<Campaigns />} />
        <Route path="history" element={<OperatorHistory />} />
        <Route path="analytics" element={<Analytics />} />
      </Route>

      <Route path="*" element={<Navigate to="/login" replace />} />
    </Routes>
  )
}

export default App