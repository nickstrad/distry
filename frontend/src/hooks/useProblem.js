import { useApiResource } from './useApiResource.js'

export function useProblems() {
  const { data, loading, error } = useApiResource('/api/problems')
  return { problems: data || [], loading, error }
}

export function useProblem(slug) {
  const { data, loading, error } = useApiResource(`/api/problems/${slug}`)
  return { problem: data, loading, error }
}
