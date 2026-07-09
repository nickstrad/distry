export function DifficultyBadge({ difficulty }) {
  return <span className={`difficulty ${difficulty}`}>{difficulty}</span>
}

export function TagList({ tags }) {
  return (
    <>
      {tags.map((tag) => (
        <span className="tag" key={tag}>
          {tag}
        </span>
      ))}
    </>
  )
}
