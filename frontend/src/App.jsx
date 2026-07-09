import { createContext, useContext, useEffect, useMemo, useState } from 'react'
import { Link, Navigate, Route, Routes, useNavigate } from 'react-router-dom'
import './styles.css'

const AuthContext = createContext(null)

async function api(path, options) {
  const res = await fetch(path, options)
  const data = await res.json().catch(() => null)

  if (!res.ok) {
    throw new Error(data?.error || 'Something went wrong')
  }
  return data
}

function AuthProvider({ children }) {
  const [user, setUser] = useState(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let active = true
    api('/api/me')
      .then((me) => {
        if (active) setUser(me)
      })
      .catch(() => {
        if (active) setUser(null)
      })
      .finally(() => {
        if (active) setLoading(false)
      })
    return () => {
      active = false
    }
  }, [])

  const value = useMemo(() => ({ user, setUser, loading }), [user, loading])

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

function useAuth() {
  return useContext(AuthContext)
}

export default function App() {
  return (
    <AuthProvider>
      <Routes>
        <Route path="/login" element={<AuthPage mode="login" />} />
        <Route path="/signup" element={<AuthPage mode="signup" />} />
        <Route
          path="*"
          element={
            <RequireAuth>
              <Shell />
            </RequireAuth>
          }
        />
      </Routes>
    </AuthProvider>
  )
}

function RequireAuth({ children }) {
  const { user, loading } = useAuth()

  if (loading) {
    return <main className="centered">Loading...</main>
  }
  if (!user) {
    return <Navigate to="/login" replace />
  }
  return children
}

function AuthPage({ mode }) {
  const isSignup = mode === 'signup'
  const navigate = useNavigate()
  const { user, setUser, loading } = useAuth()
  const [form, setForm] = useState({ username: '', email: '', password: '' })
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)

  if (!loading && user) {
    return <Navigate to="/" replace />
  }

  function updateField(event) {
    setForm((current) => ({ ...current, [event.target.name]: event.target.value }))
  }

  async function submit(event) {
    event.preventDefault()
    setError('')
    setSubmitting(true)

    try {
      const data = await api(`/api/auth/${isSignup ? 'signup' : 'login'}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(authPayload(form, isSignup)),
      })
      setUser(data)
      navigate('/', { replace: true })
    } catch (err) {
      setError(err.message)
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <main className="auth-screen">
      <section className="auth-panel">
        <h1>{isSignup ? 'Create your account' : 'Welcome back'}</h1>
        <form onSubmit={submit}>
          {isSignup && (
            <label>
              Username
              <input
                name="username"
                value={form.username}
                onChange={updateField}
                autoComplete="username"
                required
              />
            </label>
          )}
          <label>
            Email
            <input
              name="email"
              type="email"
              value={form.email}
              onChange={updateField}
              autoComplete="email"
              required
            />
          </label>
          <label>
            Password
            <input
              name="password"
              type="password"
              value={form.password}
              onChange={updateField}
              autoComplete={isSignup ? 'new-password' : 'current-password'}
              minLength={8}
              required
            />
          </label>
          {error && <p className="error">{error}</p>}
          <button type="submit" disabled={submitting}>
            {submitting ? 'Working...' : isSignup ? 'Sign up' : 'Log in'}
          </button>
        </form>
        <p className="switch">
          {isSignup ? 'Already have an account?' : 'Need an account?'}{' '}
          <Link to={isSignup ? '/login' : '/signup'}>{isSignup ? 'Log in' : 'Sign up'}</Link>
        </p>
      </section>
    </main>
  )
}

function authPayload(form, isSignup) {
  if (isSignup) {
    return form
  }
  return { email: form.email, password: form.password }
}

function Shell() {
  const { user, setUser } = useAuth()
  const navigate = useNavigate()

  async function signOut() {
    await fetch('/api/auth/logout', { method: 'POST' })
    setUser(null)
    navigate('/login', { replace: true })
  }

  return (
    <main className="app-shell">
      <header>
        <h1>Distry</h1>
        <div className="account">
          <span>{user.username}</span>
          <button type="button" onClick={signOut}>
            Sign out
          </button>
        </div>
      </header>
      <section className="workspace">
        <h2>Ready</h2>
        <p>Backend health is available at <code>/api/health</code>.</p>
      </section>
    </main>
  )
}
