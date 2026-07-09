import { createContext, useContext, useEffect, useMemo, useState } from 'react'
import { Link, Navigate, Route, Routes, useNavigate } from 'react-router-dom'
import './styles.css'

const AuthContext = createContext(null)

async function api(path, options = {}) {
  return requestJSON(path, options)
}

async function apiMaybe(path, options = {}) {
  return requestJSON(path, { ...options, allowNotFound: true })
}

async function requestJSON(path, { allowNotFound = false, ...options } = {}) {
  const res = await fetch(path, options)
  const data = await res.json().catch(() => null)

  if (allowNotFound && res.status === 404) {
    return null
  }
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
      <Workspace />
    </main>
  )
}

function Workspace() {
  const [problems, setProblems] = useState([])
  const [selectedSlug, setSelectedSlug] = useState('')
  const [problem, setProblem] = useState(null)
  const [error, setError] = useState('')
  const solution = useSolutionFiles(problem)

  useEffect(() => {
    let active = true
    api('/api/problems')
      .then((items) => {
        if (!active) return
        setProblems(items)
        setSelectedSlug((current) => current || items[0]?.slug || '')
      })
      .catch((err) => {
        if (active) setError(err.message)
      })
    return () => {
      active = false
    }
  }, [])

  useEffect(() => {
    if (!selectedSlug) return undefined
    let active = true
    setError('')
    api(`/api/problems/${selectedSlug}`)
      .then((data) => {
        if (active) setProblem(data)
      })
      .catch((err) => {
        if (active) setError(err.message)
      })
    return () => {
      active = false
    }
  }, [selectedSlug])

  function selectProblem(slug) {
    if (solution.dirty && !window.confirm('Discard unsaved changes?')) {
      return
    }
    setSelectedSlug(slug)
  }

  return (
    <section className="workspace">
      <aside className="problem-list" aria-label="Problems">
        {problems.map((item) => (
          <button
            className={item.slug === selectedSlug ? 'problem-link active' : 'problem-link'}
            key={item.slug}
            type="button"
            onClick={() => selectProblem(item.slug)}
          >
            <span>{item.title}</span>
            <small>{item.difficulty}</small>
          </button>
        ))}
      </aside>
      <section className="problem-panel">
        {error && <p className="error">{error}</p>}
        {!problem && !error && <p className="muted">Loading...</p>}
        {problem && <ProblemEditor problem={problem} solution={solution} />}
      </section>
    </section>
  )
}

function ProblemEditor({ problem, solution }) {
  return (
    <>
      <div className="problem-heading">
        <div>
          <h2>{problem.title}</h2>
          <p>
            {problem.language} / {problem.difficulty}
          </p>
        </div>
        <div className="actions">
          <span className={solution.dirty ? 'save-state dirty' : 'save-state'}>
            {saveLabel(solution)}
          </span>
          <button type="button" onClick={solution.reset} disabled={solution.busy}>
            Reset
          </button>
          <button type="button" onClick={solution.save} disabled={!solution.dirty || solution.busy}>
            Save
          </button>
        </div>
      </div>
      <div className="description">{problem.description_md}</div>
      <div className="editor-stack">
        {Object.entries(solution.files).map(([name, contents]) => (
          <label className="file-editor" key={name}>
            <span>{name}</span>
            <textarea
              spellCheck="false"
              value={contents}
              onChange={(event) => solution.updateFile(name, event.target.value)}
            />
          </label>
        ))}
      </div>
      {solution.error && <p className="error">{solution.error}</p>}
    </>
  )
}

function saveLabel(solution) {
  if (solution.saving) return 'Saving...'
  if (solution.dirty) return 'Unsaved changes'
  return 'Saved'
}

function useSolutionFiles(problem) {
  const [files, setFiles] = useState({})
  const [savedFiles, setSavedFiles] = useState({})
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    if (!problem) return undefined
    let active = true
    setLoading(true)
    setError('')
    apiMaybe(`/api/problems/${problem.slug}/solution`)
      .then((solution) => {
        if (!active) return
        const nextFiles = solution?.files || problem.templates
        setFiles(nextFiles)
        setSavedFiles(nextFiles)
      })
      .catch((err) => {
        if (active) setError(err.message)
      })
      .finally(() => {
        if (active) setLoading(false)
      })
    return () => {
      active = false
    }
  }, [problem])

  const dirty = useMemo(() => JSON.stringify(files) !== JSON.stringify(savedFiles), [files, savedFiles])

  useEffect(() => {
    function beforeUnload(event) {
      if (!dirty) return
      event.preventDefault()
      event.returnValue = ''
    }
    window.addEventListener('beforeunload', beforeUnload)
    return () => window.removeEventListener('beforeunload', beforeUnload)
  }, [dirty])

  useEffect(() => {
    function onKeyDown(event) {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 's') {
        event.preventDefault()
        if (dirty && !saving && problem) {
          save()
        }
      }
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [dirty, files, problem, saving])

  function updateFile(name, contents) {
    setFiles((current) => ({ ...current, [name]: contents }))
  }

  async function save() {
    await persist(files)
  }

  async function reset() {
    await persist(problem.templates, { updateEditorFirst: true })
  }

  async function persist(nextFiles, { updateEditorFirst = false } = {}) {
    if (updateEditorFirst) setFiles(nextFiles)
    setSaving(true)
    setError('')
    try {
      const solution = await api(`/api/problems/${problem.slug}/solution`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ files: nextFiles }),
      })
      setSavedFiles(solution.files)
      setFiles(solution.files)
    } catch (err) {
      setError(err.message)
    } finally {
      setSaving(false)
    }
  }

  return {
    busy: loading || saving,
    dirty,
    error,
    files,
    loading,
    reset,
    save,
    saving,
    updateFile,
  }
}
