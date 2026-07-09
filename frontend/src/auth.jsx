import { createContext, useCallback, useContext, useMemo } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from './api.js'

const AuthContext = createContext(null)
export const authUserQueryKey = ['auth', 'me']

export function AuthProvider({ children }) {
  const queryClient = useQueryClient()
  const { user, loading } = useAuthenticatedUser()

  const setUser = useCallback((nextUser) => {
    queryClient.setQueryData(authUserQueryKey, nextUser)
  }, [queryClient])

  const value = useMemo(() => ({ user, setUser, loading }), [user, setUser, loading])

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth() {
  return useContext(AuthContext)
}

function useAuthenticatedUser() {
  const { data = null, isPending } = useQuery({
    queryKey: authUserQueryKey,
    queryFn: () => api('/api/me'),
    throwOnError: false,
  })

  return { user: data, loading: isPending }
}
