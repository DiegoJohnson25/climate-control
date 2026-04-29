import { createBrowserRouter, RouterProvider, Navigate } from 'react-router-dom'
import { SWRConfig } from 'swr'
import { AuthProvider } from './api/auth.jsx'
import { fetcher } from './api/fetcher'
import ProtectedRoute from './components/ProtectedRoute'
import Nav from './components/Nav'
import LoginPage from './pages/LoginPage'
import RegisterPage from './pages/RegisterPage'
import DashboardPage from './pages/DashboardPage'
import RoomDetailPage from './pages/RoomDetailPage'
import DevicesPage from './pages/DevicesPage'
import { useEffect, useState } from 'react'

// ---------------------------------------------------------------------------
// Dark mode
// Reads preference from localStorage on mount, syncs to <html> data-theme.
// ThemeProvider is rendered inside AuthProvider so all children can access
// both contexts.
// ---------------------------------------------------------------------------

function ThemeProvider({ children }) {
  const [theme, setTheme] = useState(
    () => localStorage.getItem('cc-theme') || 'light'
  )

  useEffect(() => {
    if (theme === 'dark') {
      document.documentElement.setAttribute('data-theme', 'dark')
    } else {
      document.documentElement.removeAttribute('data-theme')
    }
    localStorage.setItem('cc-theme', theme)
  }, [theme])

  function toggleTheme() {
    setTheme((t) => (t === 'dark' ? 'light' : 'dark'))
  }

  return children({ theme, toggleTheme })
}

// ---------------------------------------------------------------------------
// Layout — persistent Nav wrapping all protected pages
// ---------------------------------------------------------------------------

function AppLayout({ toggleTheme, theme }) {
  return (
    <>
      <Nav toggleTheme={toggleTheme} theme={theme} />
      <ProtectedRoute />
    </>
  )
}

// ---------------------------------------------------------------------------
// Router
// ---------------------------------------------------------------------------

function buildRouter(toggleTheme, theme) {
  return createBrowserRouter([
    {
      path: '/login',
      element: <LoginPage />,
    },
    {
      path: '/register',
      element: <RegisterPage />,
    },
    {
      path: '/',
      element: <Navigate to="/dashboard" replace />,
    },
    {
      path: '/',
      element: <AppLayout toggleTheme={toggleTheme} theme={theme} />,
      children: [
        { path: 'dashboard', element: <DashboardPage /> },
        { path: 'rooms/:id', element: <RoomDetailPage /> },
        { path: 'devices', element: <DevicesPage /> },
      ],
    },
  ])
}

// ---------------------------------------------------------------------------
// App root
// ---------------------------------------------------------------------------

export default function App() {
  return (
    <AuthProvider>
      <SWRConfig value={{ fetcher }}>
        <ThemeProvider>
          {({ theme, toggleTheme }) => (
            <RouterProvider router={buildRouter(toggleTheme, theme)} />
          )}
        </ThemeProvider>
      </SWRConfig>
    </AuthProvider>
  )
}
