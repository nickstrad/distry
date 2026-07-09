# 01 — Foundations: layout, config, DB, migrations, health

**Depends on:** nothing.
**Enables:** 02, 03 (and establishes the repo layout used by 06).

**Status:** Done.

Implemented server layout, typed config loading, pgx pool creation, embedded goose
migrations with a manual migrate command, dependency-injected server routing, and the
database-backed `/api/health` endpoint. The old demo API route was removed.

## Goal

Restructure the scaffold into the target layout, add typed env config, a pgx connection
pool, SQL migrations, and a `/api/health` endpoint that verifies DB connectivity.

## Steps

1. **Move the server** to `cmd/server/main.go` (keep `static_dev.go`/`static_prod.go`
   working — move them alongside or into an `internal/web` package). Update `.air.toml`
   build target accordingly. `go run ./cmd/server` must still serve the React app.
2. **Config** — `internal/config`:
   ```go
   type Config struct {
       DatabaseURL string // from DATABASE_URL
       Port        string // default 8080
   }
   func Load() (Config, error) // reads env; loads .env in dev (github.com/joho/godotenv)
   ```
   Never log `DatabaseURL`.
3. **DB pool** — `internal/db`: `func NewPool(ctx, cfg) (*pgxpool.Pool, error)` using
   `github.com/jackc/pgx/v5/pgxpool`. Injected into the server; nothing global.
4. **Migrations** — use `goose` (github.com/pressly/goose/v3) with SQL files in
   `internal/db/migrations/`, embedded via `embed.FS` and run at startup (also expose
   `go run ./cmd/migrate` for manual control). First migration: a no-op marker table
   (`schema_bootstrap`) so the pipeline is provable. Later plans add their own migrations.
5. **Server wiring** — introduce a `Server` struct constructed with its dependencies:
   ```go
   type Server struct { pool *pgxpool.Pool; ... }
   func New(pool *pgxpool.Pool) *Server
   func (s *Server) Routes() chi.Router
   ```
   `main.go` only: load config → pool → migrate → New → ListenAndServe.
6. **Health endpoint** — `GET /api/health` returns `{"status":"ok","db":"ok"}` after
   `pool.Ping(ctx)`; 503 with `"db":"unreachable"` otherwise.
7. Remove the `/api/hello` demo route (update `App.jsx` to stop calling it, or leave the
   frontend untouched until plan 04 — either is fine; just don't leave a broken fetch).

## Testing / DI notes

- `Server` handlers take interfaces where they need data access; health check can accept a
  `Pinger interface { Ping(context.Context) error }` so it's unit-testable without pg.
- Unit tests: config loading (env set/unset), health handler with fake Pinger (ok + fail).
- Integration test (optional, build tag `integration`): run migrations against
  `DATABASE_URL` and ping.

## Testable outcome

- `go test ./...` passes.
- `go run ./cmd/server` starts, applies migrations to Neon, serves the frontend, and
  `curl localhost:8080/api/health` → `{"status":"ok","db":"ok"}`.
