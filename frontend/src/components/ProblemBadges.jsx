import { Badge } from './ui/badge'

export function DifficultyBadge({ difficulty }) {
  return (
    <Badge className="difficulty-badge" variant="secondary">
      {difficulty}
    </Badge>
  )
}

export function TagList({ tags }) {
  return (
    <>
      {tags.map((tag) => (
        <Badge className="tag-badge" key={tag} variant="outline">
          {tag}
        </Badge>
      ))}
    </>
  )
}
