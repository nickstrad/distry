import { useEffect, useMemo, useState } from 'react'

export function useSolutionFiles(problem) {
  const templateFiles = useMemo(() => problem?.templates || {}, [problem])
  const fileNames = useMemo(() => Object.keys(templateFiles).sort(), [templateFiles])
  const [files, setFiles] = useState({})
  const [activeFile, setActiveFile] = useState('')
  const effectiveActiveFile = activeFile || fileNames[0] || ''

  useEffect(() => {
    setFiles(templateFiles)
    setActiveFile(fileNames[0] || '')
  }, [fileNames, templateFiles])

  function updateActiveFile(content) {
    setFiles((current) => ({ ...current, [effectiveActiveFile]: content }))
  }

  return {
    files,
    fileNames,
    activeFile: effectiveActiveFile,
    activeContent: files[effectiveActiveFile] || '',
    setActiveFile,
    updateActiveFile,
  }
}
