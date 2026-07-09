import MonacoEditor from '@monaco-editor/react'

export default function Editor({ fileName, value, onChange }) {
  return (
    <MonacoEditor
      key={fileName}
      language="go"
      path={fileName}
      theme="vs-dark"
      value={value}
      onChange={(next) => onChange(next || '')}
      options={{
        fontFamily: '"JetBrains Mono", "SFMono-Regular", Consolas, monospace',
        fontSize: 14,
        lineNumbersMinChars: 3,
        minimap: { enabled: false },
        padding: { top: 14, bottom: 14 },
        scrollBeyondLastLine: false,
        smoothScrolling: true,
        tabSize: 2,
      }}
    />
  )
}
