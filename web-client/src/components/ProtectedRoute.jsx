import { useState, useEffect } from 'react'
import { Outlet, Navigate } from 'react-router-dom'
import { useAuth, doRefresh, getToken } from '../api/auth.jsx'

// ---------------------------------------------------------------------------
// ProtectedRoute
// Wraps all authenticated routes. On mount, if not already authenticated,
// attempts a silent refresh using the httpOnly refresh cookie. Renders
// nothing while the check is in flight to avoid a flash redirect to login.
//
// Three outcomes:
//   1. Already authenticated (token in memory) — render children immediately
//   2. Refresh succeeds — login() stores token, render children
//   3. Refresh fails — redirect to /login
// ---------------------------------------------------------------------------

export default function ProtectedRoute() {
  const { isAuthenticated, login } = useAuth()

  // If we already have a token in memory, no check needed.
  // Otherwise we need to attempt a silent refresh before rendering.
  const [checking, setChecking] = useState(() => !getToken())

  useEffect(() => {
    if (!checking) return

    doRefresh()
      .then((data) => {
        if (data?.access_token) login(data.access_token)
      })
      .catch(() => {
        // Refresh failed — isAuthenticated stays false, render redirects to /login
      })
      .finally(() => {
        setChecking(false)
      })
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  if (checking) return null
  if (!isAuthenticated) return <Navigate to="/login" replace />
  return <Outlet />
}
