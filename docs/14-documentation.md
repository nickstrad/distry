# 14 — Documentation: platform guides + top-level README

**Depends on:** conceptually all plans (documents what they built). Can be drafted any
time after 07 (the contracts are stable then); finalized after 13. No plan depends on it.

## Goal

Make the platform usable and extensible without reading the source: a README that gets a
fresh clone running, a user guide for solving problems, and an author guide for growing
the problem set (expanding plan 07's `problem-authoring.md` stub into the real thing).

## Deliverables

### 1. `README.md` (repo root)

- **What Distry is** — 2–3 sentences + a screenshot of the workspace (problem, editor,
  results panel) once plan 09 exists.
- **Architecture at a glance** — a short diagram/description of the three layers:
  `pkg/sim` (deterministic seeded simulator) → `pkg/simtest` (probe, checkers, harness
  contract) → platform (problems/solutions/runner/UI). One paragraph each, linking to
  `docs/` plans and the guides below for depth.
- **Quickstart** — from clean clone to running app, verified by actually following it:
  prerequisites (Go, Node, a Postgres `DATABASE_URL`), `.env` setup (document variable
  names only, never values), migrations, `make dev` (from plan 13), first signup, solving
  problem 1.
- **Development** — repo layout table (mirroring `docs/00-overview.md`), how to run unit
  vs `integration`-tagged tests, the e2e suite (plan 13).
- **Project status / roadmap** — link `docs/backlog.md` (OAuth swap, sandboxing, more
  languages/problems).

### 2. `docs/guide-solving-problems.md` (user guide)

- The workflow: pick a problem → read invariants → edit templates → Save → Run → read
  per-seed results → use the trace viewer → replay a failing seed.
- **How to read a failure**: violation card anatomy (checker name, message, pinned event
  seq), walking the trace backwards from the violating event, filtering by node/kind.
- **How to unit-test your solution locally** before ever pressing Run: a complete worked
  example driving a solution with a fake `sim.Context` — this is where the DI design pays
  off for users, so show real runnable code (keep it in `examples/` and compile it in CI
  so it never rots).
- Determinism and seeds: what a seed controls, why the same seed always reproduces, when
  to use custom-seed runs.

### 3. `docs/guide-authoring-problems.md` (author guide — grows plan 07's stub)

The book-chapter-to-problem recipe, written as a checklist with the perfect-link problem
as the running example:

1. Extract the properties from the book; classify each as safety or liveness.
2. Map to existing `simtest` checkers; write a new checker only if genuinely novel (with
   its own unit tests — it joins the shared library).
3. Design the environment: topology, `NetworkConfig`, seed-derived variation, and — the
   part that needs the most craft — fault schedules that actually discriminate (cite the
   URB eager-deliver case from plan 12 as the canonical example).
4. Write manifest → description → template (skeleton + Deps + local-testing comment) →
   harness.
5. Prove it: `harness/testdata/` with a correct solution (pass 100+ seeds) and one
   planted bug per property, each caught deterministically.
6. Reference tables: manifest field reference, `sim`/`simtest` public API summary, report
   JSON schema (v1), trace event kinds.

### 4. Godoc pass

`pkg/sim` and `pkg/simtest` get proper package docs (`doc.go` with runnable examples via
`Example*` test functions) — these packages are user-facing API surface, hold them to
library documentation standards.

## Approach notes

- Write docs against reality: every command in the README and every code snippet in the
  guides must be executed/compiled as part of writing this plan (snippets live in
  `examples/` and build in CI; the quickstart is verified on a clean checkout).
- Keep `docs/00-overview.md`'s dependency table updated to mark shipped plans, so docs/
  remains truthful as a project index.

## Testable outcome

- A newcomer can go from `git clone` to solving problem 1 using only `README.md`.
- `go build ./examples/...` and `go test ./...` (including `Example` funcs) pass, proving
  all documented snippets compile.
- A problem author can add a trivial fourth problem end-to-end using only
  `guide-authoring-problems.md`, without reading platform source.
