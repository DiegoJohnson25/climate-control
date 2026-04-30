import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { Eye, EyeOff } from 'lucide-react'
import { useAuth } from '../api/auth.jsx'

// ---------------------------------------------------------------------------
// RegisterPage
// Collects email + password, calls POST /auth/register, then immediately
// calls POST /auth/login with the same credentials to get an access token.
// Redirects to dashboard on success — no manual login step required.
// ---------------------------------------------------------------------------

export default function RegisterPage() {
  const { login } = useAuth()
  const navigate = useNavigate()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [showConfirm, setShowConfirm] = useState(false)
  const [error, setError] = useState(null)
  const [loading, setLoading] = useState(false)

  const passwordMismatch = confirm.length > 0 && password !== confirm
  const canSubmit = email.trim() && password.trim() && confirm.trim() && !passwordMismatch && !loading

  async function handleSubmit() {
    if (!canSubmit) return
    setError(null)
    setLoading(true)
    try {
      // Step 1 — register
      const regRes = await fetch('/api/v1/auth/register', {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password }),
      })

      if (!regRes.ok) {
        const body = await regRes.json().catch(() => ({}))
        if (regRes.status === 409) {
          setError('An account with that email already exists')
        } else {
          setError(body.error || 'Registration failed')
        }
        return
      }

      // Step 2 — auto-login with the same credentials
      const loginRes = await fetch('/api/v1/auth/login', {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password }),
      })

      if (!loginRes.ok) {
        setError('Account created but sign-in failed — please sign in manually')
        navigate('/login')
        return
      }

      const data = await loginRes.json()
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
              Create account
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
              <div style={{ position: 'relative', width: '100%' }}>
                <input
                  className="cc-input"
                  type={showPassword ? 'text' : 'password'}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  onKeyDown={handleKeyDown}
                  autoComplete="new-password"
                  style={{ paddingRight: 36 }}
                />
                <button
                  type="button"
                  className="cc-iconbtn"
                  onClick={() => setShowPassword((v) => !v)}
                  style={{
                    position: 'absolute',
                    right: 4,
                    top: '50%',
                    transform: 'translateY(-50%)',
                    color: 'var(--cc-fg-3)',
                  }}
                >
                  {showPassword ? <EyeOff size={15} /> : <Eye size={15} />}
                </button>
              </div>
              <span className="cc-meta">Minimum 4 characters</span>
            </label>

            <label style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
              <span className="cc-label">Confirm password</span>
              <div style={{ position: 'relative', width: '100%' }}>
                <input
                  className={`cc-input${passwordMismatch ? ' cc-input--error' : ''}`}
                  type={showConfirm ? 'text' : 'password'}
                  value={confirm}
                  onChange={(e) => setConfirm(e.target.value)}
                  onKeyDown={handleKeyDown}
                  autoComplete="new-password"
                  style={{ paddingRight: 36 }}
                />
                <button
                  type="button"
                  className="cc-iconbtn"
                  onClick={() => setShowConfirm((v) => !v)}
                  style={{
                    position: 'absolute',
                    right: 4,
                    top: '50%',
                    transform: 'translateY(-50%)',
                    color: 'var(--cc-fg-3)',
                  }}
                >
                  {showConfirm ? <EyeOff size={15} /> : <Eye size={15} />}
                </button>
              </div>
              {passwordMismatch && (
                <span style={{ fontSize: 'var(--cc-fs-xs)', color: 'var(--cc-danger-fg)' }}>
                  Passwords do not match
                </span>
              )}
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
              disabled={!canSubmit}
              style={{ marginTop: 6, height: 36 }}
            >
              {loading ? 'Creating Account…' : 'Create Account'}
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
          }}
        >
          Already have an account?{' '}
          <Link
            to="/login"
            style={{ color: 'var(--cc-fg-2)', textDecoration: 'underline' }}
          >
            Sign in
          </Link>
        </div>
      </div>
    </div>
  )
}
