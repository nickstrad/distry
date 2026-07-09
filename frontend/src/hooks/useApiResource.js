import { useQuery } from '@tanstack/react-query'
import { api } from '../api.js'

export function useApiResource(path) {
  const { data = null, error, isPending } = useQuery({
    queryKey: ['api', path],
    queryFn: () => api(path),
    enabled: Boolean(path),
  })

  return { data, loading: isPending, error: error?.message || '' }
}
