import { Navigate, Outlet } from 'react-router-dom'
import { useMe } from './useAuth'

export function ProtectedRoute() {
  const { isLoading, isError } = useMe()
  if (isLoading) return <div className="p-6">Loading…</div>
  if (isError) return <Navigate to="/login" replace />
  return <Outlet />
}
