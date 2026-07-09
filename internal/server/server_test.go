package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"distry/internal/problems"
)

type fakePinger struct {
	err error
}

func (f fakePinger) Ping(context.Context) error {
	return f.err
}

func TestHealthOK(t *testing.T) {
	rec := request(newTestServer(nil, nil), http.MethodGet, "/api/health")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	assertHealthResponse(t, rec, healthResponse{Status: "ok", DB: "ok"})
}

func TestHealthUnreachable(t *testing.T) {
	rec := request(newTestServer(errors.New("nope"), nil), http.MethodGet, "/api/health")

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", rec.Code)
	}
	assertHealthResponse(t, rec, healthResponse{Status: "error", DB: "unreachable"})
}

func TestListProblems(t *testing.T) {
	rec := request(newTestServer(nil, map[string]problems.Problem{
		"perfect-link": {
			Slug:       "perfect-link",
			Title:      "Perfect Point-to-Point Link",
			Difficulty: "easy",
			Tags:       []string{"links"},
			Order:      1,
		},
	}), http.MethodGet, "/api/problems")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	var got []problems.Summary
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Slug != "perfect-link" {
		t.Fatalf("unexpected summaries %+v", got)
	}
}

func TestGetProblem(t *testing.T) {
	rec := request(newTestServer(nil, map[string]problems.Problem{
		"perfect-link": {
			Slug:          "perfect-link",
			Title:         "Perfect Point-to-Point Link",
			Difficulty:    "easy",
			Language:      "go",
			Tags:          []string{"links"},
			Order:         1,
			Entrypoint:    "solution.go",
			DescriptionMD: "# Perfect Link\n",
			Templates:     map[string]string{"solution.go": "package solution\n"},
			RunConfig:     problems.RunConfig{Seeds: []int{1}, TimeoutSeconds: 30},
		},
	}), http.MethodGet, "/api/problems/perfect-link")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	var got problems.Problem
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.Slug != "perfect-link" || got.Templates["solution.go"] == "" {
		t.Fatalf("unexpected problem %+v", got)
	}
}

func TestGetProblemNotFound(t *testing.T) {
	rec := request(newTestServer(nil, nil), http.MethodGet, "/api/problems/missing")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rec.Code)
	}
}

func newTestServer(pingErr error, loaded map[string]problems.Problem) *Server {
	return New(
		fakePinger{err: pingErr},
		fakeProblemRepo{problems: loaded},
		http.NotFoundHandler(),
	)
}

func request(srv *Server, method, target string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, nil)
	rec := httptest.NewRecorder()

	srv.Routes().ServeHTTP(rec, req)
	return rec
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

type fakeProblemRepo struct {
	problems map[string]problems.Problem
}

func (f fakeProblemRepo) Upsert(context.Context, problems.Problem) error {
	return nil
}

func (f fakeProblemRepo) List(context.Context) ([]problems.Summary, error) {
	summaries := make([]problems.Summary, 0, len(f.problems))
	for _, problem := range f.problems {
		summaries = append(summaries, problems.Summary{
			Slug:       problem.Slug,
			Title:      problem.Title,
			Difficulty: problem.Difficulty,
			Tags:       problem.Tags,
			Order:      problem.Order,
		})
	}
	return summaries, nil
}

func (f fakeProblemRepo) Get(_ context.Context, slug string) (problems.Problem, error) {
	problem, ok := f.problems[slug]
	if !ok {
		return problems.Problem{}, problems.ErrNotFound
	}
	return problem, nil
}
