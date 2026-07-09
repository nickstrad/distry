# 03 — Problem domain: manifest format, loader, storage, list/detail API

**Depends on:** 01. (No auth needed — problem list/detail can be public for MVP.)
**Enables:** 04, 05, 08, 10–12.

**Status:** Done.

Implemented the problem manifest model and loader, database read model and sync path,
list/detail repository and HTTP API, plus the initial `perfect-link` problem package with
description, template, and hidden harness directory. Loader and handler tests are green.

## Goal

Define how a problem is authored on disk, load problems into the DB at startup, and expose
`GET /api/problems` + `GET /api/problems/{slug}`.

## Problem package format (on disk, source of truth)

Problems live in `/problems/<slug>/`. This is the authoring contract used by plans 10–12
and by future book-to-problem conversions:

```
problems/perfect-link/
  manifest.yaml
  description.md            # full statement, invariants explained, examples
  template/                 # files shown to the user, keyed by manifest
    solution.go
  harness/                  # HIDDEN from users; used by the runner (plan 08)
    harness.go              # builds sim topology, wires user code, registers checkers
```

`manifest.yaml`:

```yaml
slug: perfect-link
title: Perfect Point-to-Point Link
difficulty: easy            # easy | medium | hard
language: go                # extensibility hook; only "go" for MVP
tags: [links, retransmission]
order: 1                    # display ordering
entrypoint: solution.go     # primary template file opened in the editor
templates:                  # user-editable files (relative to template/)
  - solution.go
runs:
  seeds: [1, 2, 3, 4, 5]    # default seed set for a Run
  timeout_seconds: 30
```

## Steps

1. **Types + loader** — `internal/problems`:
   `type Manifest struct{...}` matching the YAML (use `gopkg.in/yaml.v3`);
   `func LoadDir(fsys fs.FS) ([]Problem, error)` parses every problem, reads
   `description.md` and template file contents, and **validates**: unique slugs, templates
   exist, entrypoint listed in templates, difficulty enum, ≥1 seed. Takes `fs.FS` so tests
   use `fstest.MapFS`.
2. **Migration** (goose): `problems` table — `slug PK, title, difficulty, language, tags
   jsonb, order_idx, description_md, templates jsonb (name→content), run_config jsonb,
   updated_at`. DB is a read model; disk wins on conflict (upsert by slug at startup).
   Harness files are **not** stored in the DB and never leave the server.
3. **Repository** — interface first, pg implementation second:
   ```go
   type Repo interface {
       Upsert(ctx, Problem) error
       List(ctx) ([]Summary, error)     // slug, title, difficulty, tags, order
       Get(ctx, slug string) (Problem, error) // includes description + templates
   }
   ```
4. **Sync at startup**: server loads `/problems` (use `embed.FS` in prod, `os.DirFS` in
   dev) and upserts. Log count synced.
5. **HTTP** — `GET /api/problems` (list of summaries, ordered), `GET /api/problems/{slug}`
   (detail: description_md + template files). 404 JSON for unknown slug.
6. **Placeholder problem**: add one real-format problem `problems/perfect-link/` with a
   stub description and template (full version comes in plan 10) so the API returns data.

## Testing / DI notes

- Loader tests with `fstest.MapFS`: happy path + each validation failure.
- Handler tests with a fake `Repo` (in-memory map): list, get, 404.
- Repo pg implementation behind `integration` tag.

## Testable outcome

`go test ./...` green. With the server running, `curl /api/problems` returns
`perfect-link`, and `curl /api/problems/perfect-link` returns its description markdown and
template file contents; the harness directory is not exposed by any endpoint.
