import { createContext, useContext, useState } from 'react'

// ---------------------------------------------------------------------------
// Module-level token store
// Used by the SWR fetcher and mutation functions — no hooks, no re-renders.
// ---------------------------------------------------------------------------

let accessToken = null

export function getToken() {
  return accessToken
}

function setToken(token) {
  accessToken = token
}

export function clearToken() {
  accessToken = null
}

// ---------------------------------------------------------------------------
// Refresh deduplication
// A single in-flight refresh promise is shared across all concurrent callers.
// If two SWR hooks 401 simultaneously, only one POST /auth/refresh fires.
// ---------------------------------------------------------------------------

let refreshPromise = null

export async function doRefresh() {
  if (!refreshPromise) {
    refreshPromise = fetch('/api/v1/auth/refresh', {
      method: 'POST',
      credentials: 'include',
    })
      .then((r) => {
        if (!r.ok) throw new Error('refresh failed')
        return r.json()
      })
      .then((data) => {
        setToken(data.access_token)
        return data
      })
      .finally(() => {
        refreshPromise = null
      })
  }
  return refreshPromise
}

// ---------------------------------------------------------------------------
// Auth context
// Provides isAuthenticated + login/logout to the component tree.
// Components that need to react to auth state changes (Nav, ProtectedRoute)
// read from here. The fetcher reads from the module-level token directly.
// ---------------------------------------------------------------------------

const AuthContext = createContext(null)

export function AuthProvider({ children }) {
  const [isAuthenticated, setIsAuthenticated] = useState(false)

  function login(token) {
    setToken(token)
    setIsAuthenticated(true)
  }

  function logout() {
    clearToken()
    setIsAuthenticated(false)
  }

  return (
    <AuthContext.Provider value={{ isAuthenticated, login, logout }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}
