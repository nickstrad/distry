import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Link, Navigate, Route, Routes, useNavigate } from 'react-router-dom'
import { api } from './api.js'
import { AuthProvider, authUserQueryKey, useAuth } from './auth.jsx'
import { Button } from './components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from './components/ui/card'
import { Input } from './components/ui/input'
import { Label } from './components/ui/label'
import { TooltipProvider } from './components/ui/tooltip'
import ProblemList from './pages/ProblemList.jsx'
import ProblemWorkspace from './pages/ProblemWorkspace.jsx'
import './styles.css'

export default function App() {
  return (
    <TooltipProvider>
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
    </TooltipProvider>
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
  const authMutation = useAuthMutation({ isSignup, setError, setUser })

  if (!loading && user) {
    return <Navigate to="/problems" replace />
  }

  function updateField(event) {
    setForm((current) => ({ ...current, [event.target.name]: event.target.value }))
  }

  async function submit(event) {
    event.preventDefault()
    setError('')
    authMutation.mutate(authPayload(form, isSignup), {
      onSuccess: () => navigate('/problems', { replace: true }),
    })
  }

  return (
    <main className="auth-screen">
      <Card className="auth-panel">
        <CardHeader>
          <CardTitle>{isSignup ? 'Create your account' : 'Welcome back'}</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={submit}>
            {isSignup && (
              <Label className="form-label">
                Username
                <Input
                  name="username"
                  value={form.username}
                  onChange={updateField}
                  autoComplete="username"
                  required
                />
              </Label>
            )}
            <Label className="form-label">
              Email
              <Input
                name="email"
                type="email"
                value={form.email}
                onChange={updateField}
                autoComplete="email"
                required
              />
            </Label>
            <Label className="form-label">
              Password
              <Input
                name="password"
                type="password"
                value={form.password}
                onChange={updateField}
                autoComplete={isSignup ? 'new-password' : 'current-password'}
                minLength={8}
                required
              />
            </Label>
            {error && <p className="error">{error}</p>}
            <Button type="submit" disabled={authMutation.isPending} size="lg">
              {authMutation.isPending ? 'Working...' : isSignup ? 'Sign up' : 'Log in'}
            </Button>
          </form>
          <p className="switch">
            {isSignup ? 'Already have an account?' : 'Need an account?'}{' '}
            <Link to={isSignup ? '/login' : '/signup'}>{isSignup ? 'Log in' : 'Sign up'}</Link>
          </p>
        </CardContent>
      </Card>
    </main>
  )
}

function authPayload(form, isSignup) {
  return isSignup ? form : { email: form.email, password: form.password }
}

function useAuthMutation({ isSignup, setError, setUser }) {
  return useMutation({
    mutationFn: (payload) =>
      api(`/api/auth/${isSignup ? 'signup' : 'login'}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      }),
    onSuccess: setUser,
    onError: (err) => setError(err.message),
  })
}

function Shell() {
  const { user, setUser } = useAuth()
  const navigate = useNavigate()
  const logoutMutation = useLogoutMutation(setUser)

  function signOut() {
    logoutMutation.mutate(undefined, {
      onSettled: () => navigate('/login', { replace: true }),
    })
  }

  return (
    <main className="app-shell">
      <header className="topbar">
        <Link className="brand" to="/problems" aria-label="Distry problem list">
          <span className="brand-mark">D</span>
          <span>Distry</span>
        </Link>
        <div className="account">
          <span>{user.username}</span>
          <Button type="button" variant="outline" onClick={signOut}>
            Sign out
          </Button>
        </div>
      </header>
      <Routes>
        <Route path="/" element={<Navigate to="/problems" replace />} />
        <Route path="/problems" element={<ProblemList />} />
        <Route path="/problems/:slug" element={<ProblemWorkspace />} />
        <Route path="*" element={<Navigate to="/problems" replace />} />
      </Routes>
    </main>
  )
}

function useLogoutMutation(setUser) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: () => api('/api/auth/logout', { method: 'POST' }),
    onSettled: () => {
      queryClient.removeQueries({ queryKey: authUserQueryKey })
      setUser(null)
    },
  })
}
