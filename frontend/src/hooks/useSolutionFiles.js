import { useEffect, useMemo, useState } from 'react'

export function useSolutionFiles(problem) {
  const templateFiles = problem?.templates || {}
  const fileNames = useMemo(() => Object.keys(templateFiles).sort(), [templateFiles])
  const [files, setFiles] = useState({})
  const [activeFile, setActiveFile] = useState('')

  useEffect(() => {
    setFiles(templateFiles)
    setActiveFile(fileNames[0] || '')
  }, [problem?.slug])

  function updateActiveFile(content) {
    setFiles((current) => ({ ...current, [activeFile]: content }))
  }

  return {
    files,
    fileNames,
    activeFile,
    activeContent: files[activeFile] || '',
    setActiveFile,
    updateActiveFile,
  }
}
