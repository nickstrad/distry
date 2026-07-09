import { useCallback, useEffect, useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api, apiMaybe } from '../api.js'

export function useSolutionFiles(problem) {
  const queryClient = useQueryClient()
  const slug = problem?.slug
  const templateFiles = useMemo(() => problem?.templates || {}, [problem])
  const fileNames = useMemo(() => Object.keys(templateFiles).sort(), [templateFiles])
  const [files, setFiles] = useState({})
  const [savedFiles, setSavedFiles] = useState({})
  const [activeFile, setActiveFile] = useState('')
  const [loadedSlug, setLoadedSlug] = useState('')
  const [error, setError] = useState('')
  const { data: solution, error: loadError, isPending: loadingSolution } = useSolutionQuery(slug)
  const {
    mutateAsync: saveSolution,
    isPending: saving,
  } = useMutation({
    mutationFn: (nextFiles) =>
      api(solutionPath(slug), {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ files: nextFiles }),
      }),
    onSuccess: (solution) => {
      queryClient.setQueryData(solutionQueryKey(slug), solution)
      replaceFiles(setFiles, setSavedFiles, solution.files)
    },
    onError: (err) => setError(err.message),
  })

  const effectiveActiveFile = activeFile || fileNames[0] || ''
  const dirty = useMemo(() => !sameFiles(files, savedFiles), [files, savedFiles])
  const loading = loadingSolution || Boolean(slug && loadedSlug !== slug)
  const busy = loading || saving

  useEffect(() => {
    if (!slug) return
    setLoadedSlug('')
    setError('')
    setActiveFile(fileNames[0] || '')
  }, [slug, fileNames])

  useEffect(() => {
    if (!slug || loadingSolution) return
    if (loadError) {
      setError(loadError.message)
      return
    }
    replaceFiles(setFiles, setSavedFiles, solution?.files || templateFiles)
    setLoadedSlug(slug)
  }, [slug, solution, loadError, loadingSolution, templateFiles])

  const persist = useCallback(
    async (nextFiles, { updateEditorFirst = false } = {}) => {
      if (!slug) return
      if (updateEditorFirst) setFiles(nextFiles)
      setError('')
      await saveSolution(nextFiles).catch(() => {})
    },
    [saveSolution, slug],
  )

  const save = useCallback(() => persist(files), [files, persist])
  const reset = useCallback(
    () => persist(templateFiles, { updateEditorFirst: true }),
    [persist, templateFiles],
  )

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
  }, [dirty, busy, save])

  function updateActiveFile(content) {
    setFiles((current) => ({ ...current, [effectiveActiveFile]: content }))
  }

  return {
    busy,
    dirty,
    error,
    files,
    fileNames,
    activeFile: effectiveActiveFile,
    activeContent: files[effectiveActiveFile] || '',
    loading,
    reset,
    save,
    saving,
    setActiveFile,
    updateActiveFile,
  }
}

function useSolutionQuery(slug) {
  return useQuery({
    queryKey: solutionQueryKey(slug),
    queryFn: () => apiMaybe(solutionPath(slug)),
    enabled: Boolean(slug),
  })
}

function solutionPath(slug) {
  return `/api/problems/${slug}/solution`
}

function solutionQueryKey(slug) {
  return ['solution', slug]
}

function replaceFiles(setFiles, setSavedFiles, nextFiles) {
  setFiles(nextFiles)
  setSavedFiles(nextFiles)
}

function sameFiles(a, b) {
  const aKeys = Object.keys(a)
  const bKeys = Object.keys(b)
  return aKeys.length === bKeys.length && aKeys.every((key) => a[key] === b[key])
}
