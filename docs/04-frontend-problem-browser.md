# 04 — Frontend: problem list, problem detail, code editor (read-only slice)

**Status:** Done.
**Depends on:** 02 (auth shell + routing), 03 (problem API).
**Enables:** 05 (editing/saving), 09 (run UI).

## Goal

A signed-in user can browse problems and open one: description rendered as markdown on the
left, template file(s) in a Monaco editor on the right. Editing state is local-only in this
slice (persistence is plan 05).

## Steps

1. **Deps**: `react-router-dom` (if not added in 02), `@monaco-editor/react` (Go syntax
   highlighting built in), `react-markdown` (+ `remark-gfm`) for the description.
2. **Routes**:
   - `/problems` — table/cards: title, difficulty badge (easy/medium/hard colors), tags.
     Data from `GET /api/problems`.
   - `/problems/:slug` — workspace layout:
     - Left pane: title, difficulty, rendered `description.md`.
     - Right pane: file tabs (one per template file) above a Monaco editor
       (`language="go"`), initialized with template contents from
       `GET /api/problems/:slug`.
     - A disabled **Run** button placeholder (wired in plan 09) and a "changes are not
       saved yet" note (removed in plan 05).
3. **API layer**: small `src/api.js` fetch wrapper that throws on non-2xx and redirects to
   `/login` on 401 — every later plan reuses it. Keep components dumb; data fetching in a
   hook (`useProblem(slug)`) so it's testable.
4. **State shape** for the editor: `{ files: {name: content}, activeFile }` in a
   `useSolutionFiles` hook — plan 05 extends this same hook with save/load, so keep it
   isolated from the components.
5. **Structure**: `src/pages/ProblemList.jsx`, `src/pages/ProblemWorkspace.jsx`,
   `src/components/Editor.jsx`, `src/components/Markdown.jsx`.

## Testing notes

- Add `vitest` + `@testing-library/react` to the frontend (first frontend tests land
  here): render ProblemList from mocked fetch; workspace shows tabs + switches files.
- Manual: log in → browse → open perfect-link → see description + Go template with
  syntax highlighting; refresh keeps working (deep link).

## Testable outcome

`npm test` (vitest) passes. Manually: from a fresh login you can reach
`/problems/perfect-link`, read the statement, switch template file tabs, and type in the
editor (changes local-only, Run disabled).
