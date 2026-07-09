import { Link, useParams } from 'react-router-dom'
import Editor from '../components/Editor.jsx'
import Markdown from '../components/Markdown.jsx'
import { DifficultyBadge, TagList } from '../components/ProblemBadges.jsx'
import { useProblem } from '../hooks/useProblem.js'
import { useSolutionFiles } from '../hooks/useSolutionFiles.js'

export default function ProblemWorkspace() {
  const { slug } = useParams()
  const { problem, loading, error } = useProblem(slug)
  const solution = useSolutionFiles(problem)

  if (loading) return <section className="workspace">Loading problem...</section>
  if (error) return <section className="workspace error-panel">{error}</section>

  return (
    <section className="problem-workspace">
      <aside className="statement-pane">
        <Link className="back-link" to="/problems">
          Problems
        </Link>
        <div className="problem-meta">
          <h1>{problem.title}</h1>
          <div className="meta-row">
            <DifficultyBadge difficulty={problem.difficulty} />
            <TagList tags={problem.tags} />
          </div>
        </div>
        <Markdown>{problem.description_md}</Markdown>
      </aside>

      <section className="code-pane" aria-label="Solution editor">
        <CodeToolbar solution={solution} />
        <div className="editor-frame">
          <Editor
            fileName={solution.activeFile}
            value={solution.activeContent}
            onChange={solution.updateActiveFile}
          />
        </div>
      </section>
    </section>
  )
}

function CodeToolbar({ solution }) {
  return (
    <div className="code-toolbar">
      <FileTabs
        activeFile={solution.activeFile}
        fileNames={solution.fileNames}
        onSelect={solution.setActiveFile}
      />
      <div className="run-controls">
        <span>Changes are not saved yet</span>
        <button type="button" disabled>
          Run
        </button>
      </div>
    </div>
  )
}

function FileTabs({ activeFile, fileNames, onSelect }) {
  return (
    <div className="file-tabs" role="tablist" aria-label="Template files">
      {fileNames.map((fileName) => (
        <button
          className={fileName === activeFile ? 'file-tab active' : 'file-tab'}
          key={fileName}
          type="button"
          role="tab"
          aria-selected={fileName === activeFile}
          onClick={() => onSelect(fileName)}
        >
          {fileName}
        </button>
      ))}
    </div>
  )
}
