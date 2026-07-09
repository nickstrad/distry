# Distry

Distry is a LeetCode-style platform for learning distributed systems by writing Go
algorithms against deterministic, seeded simulations. Pick a problem, edit the starter
solution in the browser, press Run, and inspect per-seed invariant results and traces.

![Distry workspace preview](docs/assets/workspace-preview.svg)

## Architecture At A Glance

```mermaid
flowchart LR
  sim["pkg/sim<br/>seeded simulator"] --> simtest["pkg/simtest<br/>probe + checkers"]
  simtest --> platform["platform<br/>problems, runner, API, UI"]
```

`pkg/sim` is the deterministic discrete-event simulator. It owns virtual time, seeded
network behavior, timers, partitions, crashes, and the trace event stream. See
[`docs/06-simulator-core.md`](docs/06-simulator-core.md).

`pkg/simtest` turns simulator traces and probe observations into problem reports. Problem
harnesses use it for safety and liveness checkers such as duplicate delivery, no creation,
agreement, single leader, and bounded termination. See
[`docs/07-invariants-and-harness.md`](docs/07-invariants-and-harness.md).

The platform layer loads problem manifests, stores user solutions, compiles submissions,
runs configured seeds, and renders the workspace UI. Start with the user guide in
[`docs/guide-solving-problems.md`](docs/guide-solving-problems.md) and the author guide in
[`docs/guide-authoring-problems.md`](docs/guide-authoring-problems.md).

## Quickstart

Prerequisites:

- Go 1.26.4 or newer.
- Node.js compatible with the checked-in lockfiles.
- A Postgres database URL in `DATABASE_URL`.

From a clean clone:

```sh
go mod download
npm ci
npm --prefix frontend ci
```

Create `.env` in the repo root with variable names only:

```sh
DATABASE_URL=
PORT=8080
```

Run migrations explicitly:

```sh
make migrate
```

Start the app:

```sh
make dev
```

Open `http://localhost:8080`, sign up with an email and password, choose **Perfect
Point-to-Point Link**, edit `solution.go`, click **Save**, then click **Run**. The first
problem teaches retransmission and deduplication over a lossy network.

## Development

| Path | Purpose |
| --- | --- |
| `cmd/server/` | API and static frontend server entrypoint |
| `cmd/migrate/` | Goose migration command |
| `internal/config/` | `.env` and environment loading |
| `internal/db/` | pgx pool and embedded migrations |
| `internal/auth/` | email/password users, sessions, middleware |
| `internal/problems/` | manifests, loader, repository, API shape |
| `internal/solutions/` | saved user solution files |
| `internal/runner/` | compile/run workspaces for submissions |
| `internal/submissions/` | submission lifecycle and reports |
| `pkg/sim/` | public deterministic simulator API |
| `pkg/simtest/` | public probe, checkers, harness helpers |
| `problems/` | manifests, descriptions, templates, hidden harnesses |
| `frontend/` | React workspace UI |
| `docs/` | plans and platform guides |

Common checks:

```sh
make test
make test-integration
make examples
```

`make test` runs the regular Go suite, frontend typecheck, and frontend tests.
`make test-integration` runs tests guarded by the `integration` build tag. `make examples`
builds documented example packages.

## Project Status

See [`docs/backlog.md`](docs/backlog.md) for the current roadmap, including OAuth,
sandboxing, and additional languages/problems.
