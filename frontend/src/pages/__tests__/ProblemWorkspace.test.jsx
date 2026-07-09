import { fireEvent, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { renderWithQueryClient } from '../../test/renderWithQueryClient.jsx'
import ProblemWorkspace from '../ProblemWorkspace.jsx'

vi.mock('../../components/Editor.jsx', () => ({
  default: ({ fileName, value, onChange }) => (
    <label>
      Mock editor {fileName}
      <textarea aria-label={`editor-${fileName}`} value={value} onChange={(event) => onChange(event.target.value)} />
    </label>
  ),
}))

afterEach(() => {
  vi.restoreAllMocks()
})

describe('ProblemWorkspace', () => {
  it('renders markdown and switches editable template tabs', async () => {
    vi.spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce(jsonResponse({
        slug: 'perfect-link',
        title: 'Perfect Point-to-Point Link',
        difficulty: 'easy',
        language: 'go',
        tags: ['links'],
        order: 1,
        entrypoint: 'solution.go',
        description_md: '## Goal\n\nDeliver every message.',
        templates: {
          'helper.go': 'package solution\n\nfunc helper() {}\n',
          'solution.go': 'package solution\n\nfunc Solve() {}\n',
        },
        run_config: { seeds: [1], timeout_seconds: 30 },
      }))
      .mockResolvedValueOnce(jsonResponse({ error: 'solution not found' }, { ok: false, status: 404 }))

    renderWithQueryClient(
      <MemoryRouter initialEntries={['/problems/perfect-link']}>
        <Routes>
          <Route path="/problems/:slug" element={<ProblemWorkspace />} />
        </Routes>
      </MemoryRouter>,
    )

    await screen.findByRole('heading', { name: 'Perfect Point-to-Point Link' })
    expect(await screen.findByRole('heading', { name: 'Goal' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Run' })).toBeDisabled()
    expect(screen.getByText('Saved')).toBeInTheDocument()

    const helperEditor = screen.getByLabelText('editor-helper.go')
    expect(helperEditor).toHaveValue('package solution\n\nfunc helper() {}\n')
    fireEvent.change(helperEditor, { target: { value: 'package solution\n\nfunc changed() {}\n' } })
    expect(screen.getByText('Unsaved changes')).toBeInTheDocument()

    await userEvent.click(screen.getByRole('tab', { name: 'solution.go' }))
    expect(screen.getByLabelText('editor-solution.go')).toHaveValue('package solution\n\nfunc Solve() {}\n')

    await userEvent.click(screen.getByRole('tab', { name: 'helper.go' }))
    expect(screen.getByLabelText('editor-helper.go')).toHaveValue('package solution\n\nfunc changed() {}\n')
  })

  it('loads a saved solution and saves edits', async () => {
    const fetch = vi.spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce(jsonResponse({
        slug: 'perfect-link',
        title: 'Perfect Point-to-Point Link',
        difficulty: 'easy',
        language: 'go',
        tags: ['links'],
        order: 1,
        entrypoint: 'solution.go',
        description_md: 'Solve it.',
        templates: {
          'solution.go': 'package solution\n\nfunc Solve() {}\n',
        },
        run_config: { seeds: [1], timeout_seconds: 30 },
      }))
      .mockResolvedValueOnce(jsonResponse({
        problem_slug: 'perfect-link',
        files: {
          'solution.go': 'package solution\n\nfunc Saved() {}\n',
        },
      }))
      .mockResolvedValueOnce(jsonResponse({
        problem_slug: 'perfect-link',
        files: {
          'solution.go': 'package solution\n\nfunc Changed() {}\n',
        },
      }))

    renderWithQueryClient(
      <MemoryRouter initialEntries={['/problems/perfect-link']}>
        <Routes>
          <Route path="/problems/:slug" element={<ProblemWorkspace />} />
        </Routes>
      </MemoryRouter>,
    )

    const editor = await screen.findByLabelText('editor-solution.go')
    expect(editor).toHaveValue('package solution\n\nfunc Saved() {}\n')

    fireEvent.change(editor, { target: { value: 'package solution\n\nfunc Changed() {}\n' } })
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    expect(fetch).toHaveBeenLastCalledWith('/api/problems/perfect-link/solution', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ files: { 'solution.go': 'package solution\n\nfunc Changed() {}\n' } }),
    })
    expect(await screen.findByText('Saved')).toBeInTheDocument()
  })
})

function jsonResponse(body, { ok = true, status = 200 } = {}) {
  return {
    ok,
    status,
    json: async () => body,
  }
}
