import { useState, useRef, useEffect } from 'react'
import { NavLink, useNavigate } from 'react-router-dom'
import { Thermometer, Sun, Moon } from 'lucide-react'
import { useAuth, getToken } from '../api/auth.jsx'
import { useUser } from '@/hooks/useUser'

// ---------------------------------------------------------------------------
// Nav
// Sticky 56px top bar. Brand mark, Dashboard/Devices nav links, user menu
// with theme toggle and logout. Email placeholder until useUser hook is
// wired up in 6b.
// ---------------------------------------------------------------------------

export default function Nav({ theme, toggleTheme }) {
  const { logout } = useAuth()
  const navigate = useNavigate()
  const { user } = useUser()
  const [menuOpen, setMenuOpen] = useState(false)
  const menuRef = useRef(null)

  // Close menu on outside click
  useEffect(() => {
    if (!menuOpen) return
    function onMouseDown(e) {
      if (menuRef.current && !menuRef.current.contains(e.target)) {
        setMenuOpen(false)
      }
    }
    document.addEventListener('mousedown', onMouseDown)
    return () => document.removeEventListener('mousedown', onMouseDown)
  }, [menuOpen])

  function handleLogout() {
    setMenuOpen(false)
    try {
      fetch('/api/v1/auth/logout', {
        method: 'POST',
        credentials: 'include',
        headers: { Authorization: `Bearer ${getToken()}` },
      })
    } catch {
      // Logout is best-effort — local state is always cleared regardless
      // of whether the server call succeeds. If the API call fails, the
      // refresh token remains valid on the server until it expires
      // naturally (JWT_REFRESH_TTL_DAYS). Acceptable trade-off for a
      // self-hosted single-user deployment.
    }
    logout()
    navigate('/login')
  }

  return (
    <nav
      style={{
        position: 'sticky',
        top: 0,
        zIndex: 30,
        height: 'var(--cc-nav-h)',
        borderBottom: '1px solid var(--cc-border)',
        background: 'var(--cc-surface)',
      }}
    >
      <div
        style={{
          maxWidth: 'var(--cc-max-width)',
          margin: '0 auto',
          height: '100%',
          padding: '0 var(--cc-page-pad-x)',
          display: 'flex',
          alignItems: 'center',
          gap: 28,
        }}
      >
        {/* Brand */}
        <NavLink
          to="/dashboard"
          style={{ display: 'flex', alignItems: 'center', gap: 8, textDecoration: 'none' }}
        >
          <div
            style={{
              width: 22,
              height: 22,
              borderRadius: 5,
              background: 'var(--cc-primary)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              color: 'var(--cc-primary-fg)',
              flexShrink: 0,
            }}
          >
            <Thermometer size={13} />
          </div>
          <span
            style={{
              fontSize: 14,
              fontWeight: 'var(--cc-fw-semibold)',
              letterSpacing: 'var(--cc-tracking-tight)',
              color: 'var(--cc-fg)',
            }}
          >
            Climate Control
          </span>
        </NavLink>

        {/* Nav links */}
        <div style={{ display: 'flex', gap: 24, marginLeft: 8 }}>
          <NavItem to="/dashboard">Dashboard</NavItem>
          <NavItem to="/devices">Devices</NavItem>
        </div>

        <div style={{ flex: 1 }} />

        {/* Theme toggle */}
        <button
          className="cc-iconbtn"
          onClick={toggleTheme}
          title={theme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'}
        >
          {theme === 'dark' ? <Sun size={15} /> : <Moon size={15} />}
        </button>

        {/* User menu */}
        <div ref={menuRef} style={{ position: 'relative' }}>
          <button
            onClick={() => setMenuOpen((o) => !o)}
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 8,
              background: menuOpen ? 'var(--cc-surface-2)' : 'transparent',
              border: `1px solid ${menuOpen ? 'var(--cc-border-strong)' : 'transparent'}`,
              padding: '4px 10px 4px 4px',
              borderRadius: 'var(--cc-radius-pill)',
              cursor: 'pointer',
              transition: 'background var(--cc-dur-fast) var(--cc-ease)',
            }}
          >
            <div
              style={{
                width: 26,
                height: 26,
                borderRadius: '50%',
                background: 'linear-gradient(135deg, #D97706 0%, #0891B2 100%)',
                color: '#fff',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                fontSize: 11,
                fontWeight: 'var(--cc-fw-semibold)',
                fontFamily: 'var(--cc-font-mono)',
                flexShrink: 0,
              }}
            >
              {user?.email?.[0]?.toUpperCase() ?? 'U'}
            </div>
            <span
              style={{
                fontFamily: 'var(--cc-font-mono)',
                fontSize: 12,
                color: 'var(--cc-fg-2)',
                maxWidth: 160,
                overflow: 'hidden',
                textOverflow: 'ellipsis',
                whiteSpace: 'nowrap',
              }}
            >
              {user?.email ?? 'Account'}
            </span>
          </button>

          {menuOpen && (
            <div
              className="cc-pop"
              style={{ position: 'absolute', top: 'calc(100% + 6px)', right: 0, width: 200 }}
            >
              <div style={{ padding: '8px 10px 10px' }}>
                <div className="cc-meta">{user?.email ?? 'Signed in'}</div>
              </div>
              <hr />
              <button
                onClick={() => {
                  setMenuOpen(false)
                  // TODO Phase 6g: open AccountSettingsModal
                }}
              >
                Account Settings
              </button>
              <hr />
              <button onClick={handleLogout}>Log Out</button>
            </div>
          )}
        </div>
      </div>
    </nav>
  )
}

// ---------------------------------------------------------------------------
// NavItem — active state underline matches the mockup tab style
// ---------------------------------------------------------------------------

function NavItem({ to, children }) {
  return (
    <NavLink
      to={to}
      style={({ isActive }) => ({
        position: 'relative',
        height: 'var(--cc-nav-h)',
        display: 'inline-flex',
        alignItems: 'center',
        fontSize: 13,
        fontWeight: 'var(--cc-fw-medium)',
        letterSpacing: 'var(--cc-tracking-tight)',
        color: isActive ? 'var(--cc-fg)' : 'var(--cc-fg-3)',
        textDecoration: 'none',
        transition: 'color var(--cc-dur-fast) var(--cc-ease)',
      })}
    >
      {({ isActive }) => (
        <>
          {children}
          {isActive && (
            <div
              style={{
                position: 'absolute',
                left: 0,
                right: 0,
                bottom: 0,
                height: 2,
                background: 'var(--cc-fg)',
              }}
            />
          )}
        </>
      )}
    </NavLink>
  )
}
