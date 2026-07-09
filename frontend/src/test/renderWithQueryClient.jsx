import { QueryClientProvider } from '@tanstack/react-query'
import { render } from '@testing-library/react'
import { createQueryClient } from '../queryClient.js'

export function renderWithQueryClient(ui) {
  const queryClient = createQueryClient()
  return render(<QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>)
}
