# 00 — Overview & Architecture (read first)

Distry is a LeetCode-style platform for **distributed systems problems in Go**. A user
picks a problem, fills in Go template file(s) in a browser editor, presses **Run**, and the
backend compiles the user's code against a hidden harness and executes it inside a
**deterministic, seeded network simulator**. The simulator injects message delays, drops,
duplication, reordering, and node crashes; **invariant checkers** validate the algorithm.
Failures report the seed so the exact run is **replayable**.

## Non-negotiable engineering principles (apply to every plan)

1. **Dependency injection everywhere.** No package-level singletons. Every component
   (DB, clock, RNG, transport, repositories, runner) is an interface passed into
   constructors (`NewX(deps...)`). Tests use fakes/in-memory implementations.
2. **Unit tests are first-class.** Every plan defines its testable outcome. Pure logic
   (simulator, checkers) must be testable without a DB or network.
3. **Determinism.** The simulator must be a deterministic function of (code, seed, config).
   All randomness flows from one seeded `*rand.Rand`; all time is virtual.
4. **Language extensibility.** Go is the only language for the MVP, but the runner and
   problem manifest are keyed by language (`language: go`) so another runtime can be added
   by implementing one interface (`runner.LanguageRunner`).

## Stack (already scaffolded)

- Backend: Go 1.26 + chi (`main.go`), serves embedded React build in prod.
- Frontend: React 19 + Vite + TypeScript-ready JavaScript/TSX mix (`frontend/`), with
  Tailwind v4 and local shadcn/ui components. See `docs/guide-frontend.md`.
- DB: Neon Postgres, `DATABASE_URL` in `.env` (never commit or print it).
- Auth: in-house Go email/password auth — small `users` + `sessions` tables, bcrypt,
  HttpOnly session cookie. Deliberately minimal: it will be replaced by a cloud OAuth
  provider later, so everything depends only on the auth middleware/`UserFrom(ctx)` seam.
  See plan 02.

## Repository layout (target)

```
/cmd/server/          # Go API server main (move main.go here in plan 01)
/internal/config/     # env config loading
/internal/db/         # pgx pool, migrations
/internal/auth/       # users, sessions, password hashing, middleware
/internal/problems/   # problem domain: manifests, repo, API
/internal/solutions/  # user solution file persistence
/internal/runner/     # compile & execute user submissions
/internal/submissions/# submission lifecycle (queue, status, results)
/pkg/sim/             # deterministic network simulator (importable by harnesses)
/pkg/simtest/         # invariant checkers + harness helpers
/problems/            # problem content: manifest + description + templates + harness
/frontend/            # React app
/docs/                # these plans
```

## Plan order & dependency graph

| Plan | Status  | Slice                                                                        | Depends on                         |
| ---- | ------- | ---------------------------------------------------------------------------- | ---------------------------------- |
| 01   | Shipped | Foundations: layout, config, DB pool, migrations, health check               | —                                  |
| 02   | Shipped | Auth: Go email/password users + sessions + login UI                          | 01                                 |
| 03   | Shipped | Problem domain: manifest format, loader, DB, list/detail API                 | 01                                 |
| 04   | Shipped | Frontend problem browser: list + detail + read-only editor                   | 02, 03                             |
| 05   | Shipped | Solution persistence: save/load user files per problem                       | 02, 03 (UI part: 04)               |
| 06   | Shipped | Simulator core (`pkg/sim`): pure library, no deps on rest of app             | — (repo layout from 01)            |
| 07   | Shipped | Invariant checkers & harness contract (`pkg/simtest`)                        | 06                                 |
| 08   | Shipped | Runner & submission API: compile user code + harness, execute, store results | 03, 05, 06, 07                     |
| 09   | Shipped | Run/results UI: Run button, status polling, per-seed results, trace viewer   | 04, 05, 08                         |
| 10   | Shipped | Problem 1 (easy): Perfect Link (retransmit + dedup)                          | 07 (test locally), 08+09 (E2E)     |
| 11   | Shipped | Problem 2 (easy): LCR Leader Election on a ring                              | same as 10                         |
| 12   | Shipped | Problem 3 (medium): Uniform Reliable Broadcast with crash faults             | same as 10                         |
| 13   | Shipped | E2E hardening: replay endpoint, full-flow test, seed pinning, polish         | all                                |
| 14   | Shipped | Documentation: README, user guide, problem-author guide, godoc               | draftable after 07; final after 13 |
| 15   | Shipped | shadcn/ui + Tailwind v4 design system                                        | 04; best before 09                 |

Parallelizable tracks: **{01→02, 03→04→05}** (webapp track) and **{06→07}** (simulator
track) are independent until plan 08 joins them. Problems 10–12 are independent of each
other.

## MVP definition of done

A logged-in user can pick any of the 3 problems, edit the Go template(s) in the browser,
press Run, optionally pin custom seeds, and see pass/fail per seed with
invariant-violation details and an event trace for any failing seed. Replaying a failed
seed uses the historical submission snapshot and reproduces the result exactly.
