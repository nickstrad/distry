# 05 — Solution persistence: save/load user files per problem

**Depends on:** 02 (auth), 03 (problems). UI portion depends on 04.
**Enables:** 08 (submissions run the *saved* files).

**Status:** Done.

Implemented per-user solution drafts with a `solutions` table, validation against problem
templates, authenticated save/load APIs, frontend draft loading with template fallback,
explicit Save/Cmd-S persistence, dirty-state warnings, reset-to-template, and user-scoped
handler tests.

## Goal

Each user has exactly one working draft per problem (a set of named files). Saving is
explicit (Save button + Cmd/Ctrl-S); loading a problem restores the draft, falling back to
the problem templates.

## Data model (goose migration)

```sql
CREATE TABLE solutions (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id       uuid NOT NULL REFERENCES users(id),
  problem_slug  text NOT NULL REFERENCES problems(slug),
  files         jsonb NOT NULL,          -- {"solution.go": "package solution\n..."}
  updated_at    timestamptz NOT NULL DEFAULT now(),
  UNIQUE (user_id, problem_slug)
);
```

## Steps

1. **Domain** — `internal/solutions`:
   ```go
   type Solution struct { UserID, ProblemSlug string; Files map[string]string; UpdatedAt time.Time }
   type Repo interface {
       Upsert(ctx, Solution) error
       Get(ctx, userID, slug string) (Solution, error) // ErrNotFound sentinel
   }
   ```
2. **Validation service** (unit-testable, no DB): file names must exactly match the
   problem manifest's `templates` list (no adding/renaming files in MVP), per-file size cap
   (e.g. 64 KiB), reject non-UTF-8. Inject the problems `Repo` to fetch the manifest.
3. **HTTP** (behind auth middleware):
   - `PUT /api/problems/{slug}/solution` — body `{"files": {...}}` → validate → upsert.
   - `GET /api/problems/{slug}/solution` — returns draft, or 404 (frontend falls back to
     templates).
4. **Frontend** (extends plan 04's `useSolutionFiles` hook):
   - On workspace load: try GET solution; 404 → use templates.
   - Save button + Cmd/Ctrl-S → PUT; show saved/dirty indicator ("Saved ✓" / "Unsaved
     changes"); warn on navigation with unsaved changes.
   - "Reset to template" action (client-side reset + save).

## Testing / DI notes

- Validation service: table-driven tests (unknown file, missing file, oversize, happy).
- Handlers with fake repos: PUT then GET round-trips; user A cannot read user B's draft
  (scoping is by the authenticated user ID from context — verify with two fake users).
- Manual: edit → save → hard-refresh → edits persist; second account sees pristine
  templates.

## Testable outcome

`go test ./...` green. Manually: type into the editor, Save, refresh the page and see your
code restored; a different user still gets the original template.
