# 13 — E2E hardening: replay, full-flow verification, polish

**Status:** Done.

**Depends on:** all previous plans (final integration slice).

## Goal

Close the loop on replayability, prove the whole platform works end-to-end with all three
problems, and fix the papercuts that block daily use.

## Steps

1. **Replay endpoint**: `POST /api/submissions/{id}/replay`
   `{seed}` reruns exactly one seed from the submission's **file snapshot** (not current
   draft) with full trace always included. UI: "Replay" button on any seed row of any
   historical submission. Determinism assertion: replay of a failed seed reproduces the
   identical violation + trace (make this an automated integration test, not just manual).
2. **Custom seed runs**: optional seed override on Run (input next to the Run button,
   `POST .../run {seeds:[...]}` validated 1–20 seeds) — essential for debugging a specific
   failure while iterating.
3. **Full-flow E2E test** (script or Playwright — pick pragmatically; a bash+curl script
   under `e2e/` is acceptable): fresh DB schema → start the server → sign up →
   for each of the 3 problems: save the known-correct solution (from
   `harness/testdata/correct`) → run → assert `passed`; save a buggy one → assert `failed`
   with expected checker name. This is the regression gate for everything.
4. **Papercuts** (timeboxed):
   - `make dev` / one command to start Vite + Go server together (via `mprocs`,
     configured in `mprocs.yaml`).
   - README: setup from clean clone (env vars, goose migrations, npm install).
   - Problem list shows per-user solved status (`passed` submission exists) — one query,
     big UX win.
   - Trace size guard: cap stored trace events per report (e.g. 5000) with an explicit
     `truncated: true` flag rendered in the UI.
5. **Backlog capture** (write `docs/backlog.md`, do not build): container sandboxing for
   the runner, second language, richer trace visualization (lamport/sequence diagrams),
   problem progression/unlocks, seed fuzzing (search for violating seeds), importing more
   book chapters.

## Testable outcome

`e2e/` suite passes from a clean database: all 3 problems pass with reference solutions
and fail with planted bugs; replaying any failed seed reproduces its violation exactly.
The README instructions stand up the platform from a fresh clone.
