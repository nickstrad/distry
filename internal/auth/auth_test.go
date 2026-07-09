package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time {
	return c.now
}

type fixedTokens struct {
	token string
	err   error
}

func (t fixedTokens) Token() (string, error) {
	return t.token, t.err
}

type memoryUsers struct {
	byEmail map[string]storedUser
	err     error
}

type storedUser struct {
	user User
	hash string
}

func (m *memoryUsers) Create(ctx context.Context, username, email, passwordHash string) (User, error) {
	if m.err != nil {
		return User{}, m.err
	}
	if _, ok := m.byEmail[email]; ok {
		return User{}, ErrTaken
	}
	user := User{ID: "user-1", Username: username, Email: email}
	m.byEmail[email] = storedUser{user: user, hash: passwordHash}
	return user, nil
}

func (m *memoryUsers) ByEmail(ctx context.Context, email string) (User, string, error) {
	stored, ok := m.byEmail[email]
	if !ok {
		return User{}, "", ErrInvalidCredentials
	}
	return stored.user, stored.hash, nil
}

type memorySessions struct {
	now      time.Time
	sessions map[string]storedSession
	deleted  string
}

type storedSession struct {
	userID  string
	expires time.Time
}

func (m *memorySessions) Create(ctx context.Context, tokenHash []byte, userID string, expires time.Time) error {
	m.sessions[string(tokenHash)] = storedSession{userID: userID, expires: expires}
	return nil
}

func (m *memorySessions) UserByTokenHash(ctx context.Context, tokenHash []byte) (User, error) {
	session, ok := m.sessions[string(tokenHash)]
	if !ok || !session.expires.After(m.now) {
		return User{}, ErrUnauthenticated
	}
	return User{ID: session.userID, Username: "Ada", Email: "ada@example.com"}, nil
}

func (m *memorySessions) Delete(ctx context.Context, tokenHash []byte) error {
	m.deleted = string(tokenHash)
	delete(m.sessions, string(tokenHash))
	return nil
}

func TestSignUpCreatesUserAndSession(t *testing.T) {
	now := time.Date(2026, 7, 8, 12, 0, 0, 0, time.UTC)
	users := &memoryUsers{byEmail: map[string]storedUser{}}
	sessions := &memorySessions{now: now, sessions: map[string]storedSession{}}
	service := NewServiceWithDeps(users, sessions, fixedClock{now: now}, fixedTokens{token: "session-token"})

	user, token, err := service.SignUp(context.Background(), "Ada_1", " ADA@EXAMPLE.COM ", "password123")
	if err != nil {
		t.Fatal(err)
	}
	if token != "session-token" {
		t.Fatalf("expected fixed token, got %q", token)
	}
	if user.Email != "ada@example.com" {
		t.Fatalf("expected normalized email, got %q", user.Email)
	}
	session := sessions.sessions[string(tokenHash(token))]
	if session.userID != user.ID || !session.expires.Equal(now.Add(SessionTTL)) {
		t.Fatalf("unexpected session: %+v", session)
	}
}

func TestSignUpRejectsDuplicate(t *testing.T) {
	service := newTestService(&memoryUsers{
		byEmail: map[string]storedUser{"ada@example.com": {user: User{ID: "existing"}}},
	})

	_, _, err := service.SignUp(context.Background(), "Ada_1", "ada@example.com", "password123")
	if !errors.Is(err, ErrTaken) {
		t.Fatalf("expected ErrTaken, got %v", err)
	}
}

func TestLogInRejectsBadPassword(t *testing.T) {
	hash, err := HashPassword("password123")
	if err != nil {
		t.Fatal(err)
	}
	service := newTestService(&memoryUsers{
		byEmail: map[string]storedUser{"ada@example.com": {user: User{ID: "user-1"}, hash: hash}},
	})

	_, _, err = service.LogIn(context.Background(), "ada@example.com", "wrong-password")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthenticateRejectsExpiredSession(t *testing.T) {
	now := time.Date(2026, 7, 8, 12, 0, 0, 0, time.UTC)
	sessions := &memorySessions{
		now: now,
		sessions: map[string]storedSession{
			string(tokenHash("expired")): {userID: "user-1", expires: now.Add(-time.Minute)},
		},
	}
	service := NewServiceWithDeps(&memoryUsers{byEmail: map[string]storedUser{}}, sessions, fixedClock{now: now}, fixedTokens{token: "unused"})

	_, err := service.Authenticate(context.Background(), "expired")
	if !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("expected ErrUnauthenticated, got %v", err)
	}
}

func TestLogOutDeletesSession(t *testing.T) {
	sessions := &memorySessions{
		now: time.Now(),
		sessions: map[string]storedSession{
			string(tokenHash("session-token")): {userID: "user-1", expires: time.Now().Add(time.Hour)},
		},
	}
	service := NewServiceWithDeps(&memoryUsers{byEmail: map[string]storedUser{}}, sessions, fixedClock{now: time.Now()}, fixedTokens{token: "unused"})

	if err := service.LogOut(context.Background(), "session-token"); err != nil {
		t.Fatal(err)
	}
	if sessions.deleted != string(tokenHash("session-token")) {
		t.Fatal("expected logout to delete the session hash")
	}
}

func TestMiddlewareAcceptsValidCookie(t *testing.T) {
	now := time.Now()
	sessions := &memorySessions{
		now: now,
		sessions: map[string]storedSession{
			string(tokenHash("session-token")): {userID: "user-1", expires: now.Add(time.Hour)},
		},
	}
	service := NewServiceWithDeps(&memoryUsers{byEmail: map[string]storedUser{}}, sessions, fixedClock{now: now}, fixedTokens{token: "unused"})
	next := Middleware(service)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := UserFrom(r.Context())
		if !ok || user.ID != "user-1" {
			t.Fatalf("expected user in context, got %+v", user)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.AddCookie(&http.Cookie{Name: CookieName, Value: "session-token"})
	rec := httptest.NewRecorder()

	next.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", rec.Code)
	}
}

func TestMiddlewareRejectsMissingCookie(t *testing.T) {
	service := newTestService(&memoryUsers{byEmail: map[string]storedUser{}})
	next := Middleware(service)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not run")
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	rec := httptest.NewRecorder()

	next.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
}

func newTestService(users *memoryUsers) *Service {
	now := time.Date(2026, 7, 8, 12, 0, 0, 0, time.UTC)
	return NewServiceWithDeps(users, &memorySessions{now: now, sessions: map[string]storedSession{}}, fixedClock{now: now}, fixedTokens{token: "session-token"})
}
