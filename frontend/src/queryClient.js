import { QueryClient } from '@tanstack/react-query'

export function createQueryClient({ defaultOptions, ...options } = {}) {
  return new QueryClient({
    defaultOptions: {
      ...defaultOptions,
      queries: { retry: false, ...defaultOptions?.queries },
      mutations: { retry: false, ...defaultOptions?.mutations },
    },
    ...options,
  })
}
