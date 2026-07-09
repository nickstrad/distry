package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakePinger struct {
	err error
}

func (f fakePinger) Ping(context.Context) error {
	return f.err
}

func TestHealthOK(t *testing.T) {
	srv := New(fakePinger{}, http.NotFoundHandler())
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()

	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	assertHealthResponse(t, rec, healthResponse{Status: "ok", DB: "ok"})
}

func TestHealthUnreachable(t *testing.T) {
	srv := New(fakePinger{err: errors.New("nope")}, http.NotFoundHandler())
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()

	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", rec.Code)
	}
	assertHealthResponse(t, rec, healthResponse{Status: "error", DB: "unreachable"})
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
