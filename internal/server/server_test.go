package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"distry/internal/auth"
)

type fakePinger struct {
	err error
}

func (f fakePinger) Ping(context.Context) error {
	return f.err
}

func TestHealthOK(t *testing.T) {
	srv := newTestServer(fakePinger{})
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()

	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	assertHealthResponse(t, rec, healthResponse{Status: "ok", DB: "ok"})
}

func TestHealthUnreachable(t *testing.T) {
	srv := newTestServer(fakePinger{err: errors.New("nope")})
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()

	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", rec.Code)
	}
	assertHealthResponse(t, rec, healthResponse{Status: "error", DB: "unreachable"})
}

func newTestServer(pinger Pinger) *Server {
	return New(pinger, auth.NewService(&fakeUserRepo{}, &fakeSessionRepo{}), http.NotFoundHandler())
}

type fakeUserRepo struct{}

func (f *fakeUserRepo) Create(context.Context, string, string, string) (auth.User, error) {
	return auth.User{}, nil
}

func (f *fakeUserRepo) ByEmail(context.Context, string) (auth.User, string, error) {
	return auth.User{}, "", auth.ErrInvalidCredentials
}

type fakeSessionRepo struct{}

func (f *fakeSessionRepo) Create(context.Context, []byte, string, time.Time) error {
	return nil
}

func (f *fakeSessionRepo) UserByTokenHash(context.Context, []byte) (auth.User, error) {
	return auth.User{}, auth.ErrUnauthenticated
}

func (f *fakeSessionRepo) Delete(context.Context, []byte) error {
	return nil
}

func assertHealthResponse(t *testing.T, rec *httptest.ResponseRecorder, want healthResponse) {
	t.Helper()

	var got healthResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("expected health response %+v, got %+v", want, got)
	}
}
