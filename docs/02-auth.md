# 02 — Auth: in-house Go email/password auth + sessions + login UI

**Depends on:** 01.
**Enables:** 04, 05, 08 (any authenticated feature).

## Architecture

Auth lives entirely in the Go server — no separate service. A small `users` table with
username/email/password-hash, opaque session tokens in a `sessions` table, and an HttpOnly
cookie. This is a deliberate placeholder: it will later be **replaced by a cloud OAuth
provider**, so keep the seam clean — everything downstream depends only on
`auth.Middleware` + `auth.UserFrom(ctx)` (a stable `User{ID, Username, Email}`), never on
how the user proved who they are. Swapping in OAuth later means replacing the
signup/login handlers and how sessions get created; nothing else moves.

## Data model (goose migrations, plan 01 pipeline)

```sql
CREATE TABLE users (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  username      text NOT NULL UNIQUE,        -- citext or lower() unique index
  email         text NOT NULL UNIQUE,        -- ditto
  password_hash text NOT NULL,               -- nullable later when OAuth users exist
  created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE sessions (
  token_hash  bytea PRIMARY KEY,             -- sha256 of the opaque token
  user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  expires_at  timestamptz NOT NULL,
  created_at  timestamptz NOT NULL DEFAULT now()
);
```

Store only the token **hash**; the raw 32-byte random token (base64url) lives in the
cookie. A DB leak then can't forge sessions.

## Steps

1. **Password hashing** — `internal/auth`: use `golang.org/x/crypto/bcrypt`
   (cost ~12). `HashPassword` / `CheckPassword` wrappers so the algorithm is swappable.
2. **Repos (interfaces first)**:
   ```go
   type UserRepo interface {
       Create(ctx, username, email, passwordHash string) (User, error) // maps unique violations to ErrTaken
       ByEmail(ctx, email string) (User, string /*hash*/, error)
   }
   type SessionRepo interface {
       Create(ctx, tokenHash []byte, userID string, expires time.Time) error
       UserByTokenHash(ctx, tokenHash []byte) (User, error) // enforces expiry
       Delete(ctx, tokenHash []byte) error
   }
   ```
   pgx implementations in the same package (or `internal/auth/pg`).
3. **Service** — `auth.Service` (injected repos, injected `clock` and token source for
   testability): `SignUp`, `LogIn` (email+password → session token), `LogOut`,
   `Authenticate(token) (User, error)`. Validation: username 3–32 chars
   `[a-zA-Z0-9_-]`, email shape-checked + lowercased, password ≥ 8 chars. Session TTL 30
   days, sliding optional (skip for MVP). Identical error message for "no such email" and
   "wrong password".
4. **HTTP endpoints** (JSON, on the chi router):
   - `POST /api/auth/signup` `{username,email,password}` → creates user + session,
     sets cookie, returns `{id,username,email}`. 409 on taken username/email.
   - `POST /api/auth/login` `{email,password}` → session cookie + user JSON; 401 on bad
     credentials.
   - `POST /api/auth/logout` → deletes session, clears cookie.
   - Cookie: `distry_session`, HttpOnly, SameSite=Lax, Path=/, Secure in prod. Same
     origin as the API (plan 01 serves the frontend), so no CORS work needed; Vite dev
     proxy for `/api` covers dev.
5. **Middleware + probe route** —
   `auth.Middleware(svc)` reads the cookie, calls `Authenticate`, puts `User` in context,
   401 JSON otherwise; `auth.UserFrom(ctx)`. `GET /api/me` returns the current user —
   this is how the frontend knows it's logged in.
6. **Frontend**: introduce `react-router-dom`; `/login` and `/signup` pages (plain forms,
   inline error display), an auth context that loads `/api/me` on boot, a signed-in shell
   showing the username + Sign out. Unauthenticated users are redirected to `/login`.

## Testing / DI notes

- Service unit tests with fake repos + fixed clock/token source: signup happy path,
  duplicate email/username, bad password login, expired session rejected, logout kills
  session. No Postgres required.
- Middleware tests: valid cookie → user in context; missing/garbage/expired → 401.
- pgx repo tests behind the `integration` tag.
- Manual E2E: sign up → header shows username → refresh stays logged in → sign out →
  `/api/me` 401.

## Testable outcome

With only the Go server running: create an account and log in via the UI; the header
shows your username (from `/api/me`); sessions survive a page refresh; sign out returns
you to the login page; `/api/me` without a cookie returns 401. All unit tests pass with
no database running.
