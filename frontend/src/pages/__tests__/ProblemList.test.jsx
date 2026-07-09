import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import ProblemList from '../ProblemList.jsx'

afterEach(() => {
  vi.restoreAllMocks()
})

describe('ProblemList', () => {
  it('renders problems returned by the API', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => [
        {
          slug: 'perfect-link',
          title: 'Perfect Point-to-Point Link',
          difficulty: 'easy',
          tags: ['links', 'retransmission'],
          order: 1,
        },
      ],
    })

    render(
      <MemoryRouter>
        <ProblemList />
      </MemoryRouter>,
    )

    expect(screen.getByText('Loading problems...')).toBeInTheDocument()
    expect(await screen.findByRole('link', { name: /perfect point-to-point link/i })).toHaveAttribute(
      'href',
      '/problems/perfect-link',
    )
    expect(screen.getByText('easy')).toBeInTheDocument()
    expect(screen.getByText('retransmission')).toBeInTheDocument()

    await waitFor(() => expect(fetch).toHaveBeenCalledWith('/api/problems', undefined))
  })
})
