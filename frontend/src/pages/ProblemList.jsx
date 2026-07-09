import { Link } from 'react-router-dom'
import { DifficultyBadge, TagList } from '../components/ProblemBadges.jsx'
import { useProblems } from '../hooks/useProblem.js'

export default function ProblemList() {
  const { problems, loading, error } = useProblems()

  if (loading) return <section className="workspace">Loading problems...</section>
  if (error) return <section className="workspace error-panel">{error}</section>

  return (
    <section className="problem-list">
      <div className="section-heading">
        <div>
          <p className="eyebrow">Problem set</p>
          <h1>Distributed systems drills</h1>
        </div>
        <span className="count">{problems.length} available</span>
      </div>
      <div className="problem-table">
        {problems.map((problem) => (
          <Link
            className="problem-row"
            key={problem.slug}
            to={`/problems/${problem.slug}`}
          >
            <span className="problem-order">{String(problem.order).padStart(2, '0')}</span>
            <span className="problem-main">
              <span className="problem-title">{problem.title}</span>
              <span className="tag-list">
                <TagList tags={problem.tags} />
              </span>
            </span>
            <DifficultyBadge difficulty={problem.difficulty} />
          </Link>
        ))}
      </div>
    </section>
  )
}
