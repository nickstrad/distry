# 08 — Runner & submissions: compile user code, execute per seed, store results

**Depends on:** 03 (problems + harness files), 05 (saved solutions), 06+07 (sim/harness).
**Enables:** 09.

**Status:** Done.

Implemented the submissions table/repository, async submission service, authenticated run
and polling APIs, Go runner temp-workspace compilation with local module replacement,
per-seed execution with timeouts, cleaned compile output, and focused service/server/runner
tests. The current `perfect-link` harness is still the placeholder from plan 03, so
end-to-end problem verdicts are unblocked for plan 10's real harness implementation.

## Goal

`POST /api/problems/{slug}/run` takes the user's **saved** solution, compiles it with the
problem's hidden harness, executes it once per seed in an isolated process with limits,
and stores a submission with per-seed `simtest.Report`s. Status is pollable.

## Data model (goose migration)

```sql
CREATE TABLE submissions (
  id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id      text NOT NULL,
  problem_slug text NOT NULL REFERENCES problems(slug),
  files        jsonb NOT NULL,       -- snapshot of what was run
  status       text NOT NULL,        -- queued|compiling|running|passed|failed|error
  compile_output text,
  reports      jsonb,                -- []simtest.Report (traces only for failed seeds)
  created_at   timestamptz DEFAULT now(),
  finished_at  timestamptz
);
```

## Runner design (`internal/runner`)

```go
// The language-extensibility seam. Only GoRunner for MVP.
type LanguageRunner interface {
    Compile(ctx context.Context, ws Workspace) (CompileResult, error)
    RunSeed(ctx context.Context, ws Workspace, seed int64) (simtest.Report, error)
}
```

**GoRunner per submission:**
1. Create temp workspace: `go.mod` (module `submission`, `replace distry/pkg/... =>` the
   server's vendored copy of `pkg/sim` + `pkg/simtest`), user files under `solution/`,
   problem `harness/` files, and a fixed `main.go` shim:
   `func main() { seed := flag.Int64(...); json.NewEncoder(os.Stdout).Encode(harness.Run(seed)) }`.
2. `go vet ./...` then `go build -o run ./` with `CGO_ENABLED=0`, network off
   (`GOFLAGS=-mod=mod` avoided — all deps local/vendored so builds need **no network**),
   compile timeout (e.g. 60s). Compile errors → status `error` with cleaned output
   (strip workspace paths).
3. For each manifest seed: exec `./run -seed N` with `context` timeout
   (manifest `timeout_seconds`), memory cap (start pragmatic: `GOMEMLIMIT` + a ulimit via
   `syscall.Setrlimit` in the shim; document that container-level sandboxing is a
   follow-up — acceptable for MVP because the operator is the only user), capture stdout
   JSON → `simtest.Report`. Non-zero exit / timeout / bad JSON ⇒ synthetic failed report.
4. Overall status: all seeds passed ⇒ `passed`, any violation ⇒ `failed`.

**Safety note (MVP posture):** running user Go code is arbitrary code execution. MVP is
single-operator (you), so process-level limits + no-network builds are acceptable, but
`LanguageRunner` must be constructed behind an interface so a container/gVisor-based
implementation can replace it without touching the submission service. Say this loudly in
code comments.

## Submission service (`internal/submissions`)

- `Service` with injected deps: solutions repo, problems repo (for harness files + run
  config), `LanguageRunner`, submissions repo, and a worker pool (channel-based, size 1–2)
  so runs are async.
- `POST /api/problems/{slug}/run` → snapshot saved solution → insert `queued` → enqueue →
  return `{submissionID}`. Reject if user has a submission already `queued|compiling|running`
  for this problem (409).
- `GET /api/submissions/{id}` → status + compile output + reports (owner-only).
- `GET /api/problems/{slug}/submissions` → recent list for the user.
- Optional now, required by plan 13: `POST /api/submissions/{id}/replay` with
  `{seed}` — reruns one seed with full trace. Fine to defer.

## Testing / DI notes

- Submission service tests with fake `LanguageRunner` + in-memory repos: lifecycle
  transitions, 409 on concurrent, owner scoping.
- GoRunner integration test (needs `go` toolchain, tag `integration`): compile+run a known
  toy problem with a correct and a buggy solution; assert reports.
- Determinism test: run same submission twice, byte-identical reports.

## Testable outcome

Via curl (auth cookie): save a solution (plan 05), `POST .../run`, poll
`GET /api/submissions/{id}` until `passed`/`failed`; a deliberately buggy solution yields
`failed` with a named violation and a trace for the failing seed; the same seed re-run
reproduces it.
