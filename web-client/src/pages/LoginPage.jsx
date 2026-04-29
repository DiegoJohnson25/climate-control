import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useAuth } from '../api/auth.jsx'

// ---------------------------------------------------------------------------
// LoginPage
// Centered 360px card. Calls POST /auth/login, stores token in context,
// redirects to dashboard on success. Dev hint footer retained for
// development builds.
// ---------------------------------------------------------------------------

export default function LoginPage() {
  const { login } = useAuth()
  const navigate = useNavigate()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState(null)
  const [loading, setLoading] = useState(false)

  async function handleSubmit() {
    if (!email.trim() || !password.trim()) return
    setError(null)
    setLoading(true)
    try {
      const res = await fetch('/api/v1/auth/login', {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password }),
      })
      if (!res.ok) {
        const body = await res.json().catch(() => ({}))
        setError(body.error || 'Invalid email or password')
        return
      }
      const data = await res.json()
      login(data.access_token)
      navigate('/dashboard')
    } catch {
      setError('Could not reach the server')
    } finally {
      setLoading(false)
    }
  }

  function handleKeyDown(e) {
    if (e.key === 'Enter') handleSubmit()
  }

  return (
    <div
      style={{
        minHeight: '100vh',
        background: 'var(--cc-bg)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        padding: 24,
      }}
    >
      <div style={{ width: 360 }}>
        <div className="cc-card" style={{ padding: 28 }}>
          <div style={{ marginBottom: 20 }}>
            <div
              style={{
                fontSize: 17,
                fontWeight: 'var(--cc-fw-semibold)',
                letterSpacing: 'var(--cc-tracking-tight)',
              }}
            >
              Sign in
            </div>
          </div>

          <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
            <label style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
              <span className="cc-label">Email</span>
              <input
                className="cc-input"
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                onKeyDown={handleKeyDown}
                autoComplete="email"
                autoFocus
              />
            </label>

            <label style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
              <span className="cc-label">Password</span>
              <input
                className="cc-input"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                onKeyDown={handleKeyDown}
                autoComplete="current-password"
              />
            </label>

            {error && (
              <div
                style={{
                  fontSize: 'var(--cc-fs-sm)',
                  color: 'var(--cc-danger-fg)',
                  background: 'var(--cc-danger-tint)',
                  border: '1px solid rgba(220,38,38,0.20)',
                  borderRadius: 'var(--cc-radius-sm)',
                  padding: '8px 10px',
                }}
              >
                {error}
              </div>
            )}

            <button
              className="cc-btn cc-btn--primary"
              onClick={handleSubmit}
              disabled={loading || !email.trim() || !password.trim()}
              style={{ marginTop: 6, height: 36 }}
            >
              {loading ? 'Signing in…' : 'Sign in'}
            </button>
          </div>
        </div>

        <div
          style={{
            marginTop: 16,
            padding: '10px 14px',
            fontFamily: 'var(--cc-font-mono)',
            fontSize: 11,
            color: 'var(--cc-fg-3)',
            textAlign: 'center',
            lineHeight: 1.5,
          }}
        >
          dev build · seed credentials active
        </div>

        <div
          style={{
            marginTop: 8,
            padding: '10px 14px',
            fontFamily: 'var(--cc-font-mono)',
            fontSize: 11,
            color: 'var(--cc-fg-3)',
            textAlign: 'center',
          }}
        >
          {"Don't have an account? "}
          <Link
            to="/register"
            style={{ color: 'var(--cc-fg-2)', textDecoration: 'underline' }}
          >
            Create one
          </Link>
        </div>
      </div>
    </div>
  )
}
