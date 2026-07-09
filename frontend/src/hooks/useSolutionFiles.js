import { useEffect, useMemo, useState } from 'react'
import { api, apiMaybe } from '../api.js'

export function useSolutionFiles(problem) {
  const templateFiles = problem?.templates || {}
  const fileNames = useMemo(() => Object.keys(templateFiles).sort(), [templateFiles])
  const [files, setFiles] = useState({})
  const [savedFiles, setSavedFiles] = useState({})
  const [activeFile, setActiveFile] = useState('')
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    if (!problem) return undefined
    let active = true
    setLoading(true)
    setError('')
    setActiveFile(fileNames[0] || '')

    apiMaybe(`/api/problems/${problem.slug}/solution`)
      .then((solution) => {
        if (!active) return
        const nextFiles = solution?.files || templateFiles
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
  }, [problem?.slug])

  const dirty = useMemo(() => JSON.stringify(files) !== JSON.stringify(savedFiles), [files, savedFiles])
  const busy = loading || saving

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
        if (dirty && !busy) {
          save()
        }
      }
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [dirty, busy, files, problem?.slug])

  function updateActiveFile(content) {
    setFiles((current) => ({ ...current, [activeFile]: content }))
  }

  async function save() {
    await persist(files)
  }

  async function reset() {
    await persist(templateFiles, { updateEditorFirst: true })
  }

  async function persist(nextFiles, { updateEditorFirst = false } = {}) {
    if (!problem) return
    if (updateEditorFirst) setFiles(nextFiles)
    setSaving(true)
    setError('')
    try {
      const solution = await api(`/api/problems/${problem.slug}/solution`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ files: nextFiles }),
      })
      setFiles(solution.files)
      setSavedFiles(solution.files)
    } catch (err) {
      setError(err.message)
    } finally {
      setSaving(false)
    }
  }

  return {
    busy,
    dirty,
    error,
    files,
    fileNames,
    activeFile,
    activeContent: files[activeFile] || '',
    loading,
    reset,
    save,
    saving,
    setActiveFile,
    updateActiveFile,
  }
}
