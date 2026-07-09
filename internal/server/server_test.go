package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"distry/internal/auth"
	"distry/internal/problems"
	"distry/internal/solutions"
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

func TestPutThenGetSolutionRoundTripsForAuthenticatedUser(t *testing.T) {
	srv := newTestServer(nil, map[string]problems.Problem{
		"perfect-link": {
			Slug:      "perfect-link",
			Templates: map[string]string{"solution.go": "package solution\n"},
		},
	})

	rec := requestWithBody(srv, http.MethodPut, "/api/problems/perfect-link/solution", `{"files":{"solution.go":"package solution\n// saved\n"}}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = request(srv, http.MethodGet, "/api/problems/perfect-link/solution")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var got solutions.Solution
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.Files["solution.go"] != "package solution\n// saved\n" {
		t.Fatalf("unexpected solution files %+v", got.Files)
	}
}

func TestSolutionIsScopedToAuthenticatedUser(t *testing.T) {
	store := newMemorySolutionRepo()
	problemRepo := fakeProblemRepo{problems: map[string]problems.Problem{
		"perfect-link": {
			Slug:      "perfect-link",
			Templates: map[string]string{"solution.go": "package solution\n"},
		},
	}}
	srvA := New(
		fakePinger{},
		auth.NewService(&fakeUserRepo{}, &fakeSessionRepo{user: auth.User{ID: "user-a", Username: "ada"}}),
		problemRepo,
		solutions.NewService(store, problemRepo),
		http.NotFoundHandler(),
	)
	srvB := New(
		fakePinger{},
		auth.NewService(&fakeUserRepo{}, &fakeSessionRepo{user: auth.User{ID: "user-b", Username: "grace"}}),
		problemRepo,
		solutions.NewService(store, problemRepo),
		http.NotFoundHandler(),
	)

	rec := requestWithBody(srvA, http.MethodPut, "/api/problems/perfect-link/solution", `{"files":{"solution.go":"package solution\n// user a\n"}}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = request(srvB, http.MethodGet, "/api/problems/perfect-link/solution")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected user B to get 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func newTestServer(pingErr error, loaded map[string]problems.Problem) *Server {
	problemRepo := fakeProblemRepo{problems: loaded}
	return New(
		fakePinger{err: pingErr},
		auth.NewService(&fakeUserRepo{}, &fakeSessionRepo{user: auth.User{ID: "user-a", Username: "ada"}}),
		problemRepo,
		solutions.NewService(newMemorySolutionRepo(), problemRepo),
		http.NotFoundHandler(),
	)
}

func request(srv *Server, method, target string) *httptest.ResponseRecorder {
	return requestWithBody(srv, method, target, "")
}

func requestWithBody(srv *Server, method, target, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: "token"})
	rec := httptest.NewRecorder()

	srv.Routes().ServeHTTP(rec, req)
	return rec
}

type fakeUserRepo struct{}

func (f *fakeUserRepo) Create(context.Context, string, string, string) (auth.User, error) {
	return auth.User{}, nil
}

func (f *fakeUserRepo) ByEmail(context.Context, string) (auth.User, string, error) {
	return auth.User{}, "", auth.ErrInvalidCredentials
}

type fakeSessionRepo struct {
	user auth.User
}

func (f *fakeSessionRepo) Create(context.Context, []byte, string, time.Time) error {
	return nil
}

func (f *fakeSessionRepo) UserByTokenHash(context.Context, []byte) (auth.User, error) {
	if f.user.ID == "" {
		return auth.User{}, auth.ErrUnauthenticated
	}
	return f.user, nil
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

type memorySolutionRepo struct {
	solutions map[string]solutions.Solution
}

func newMemorySolutionRepo() *memorySolutionRepo {
	return &memorySolutionRepo{solutions: map[string]solutions.Solution{}}
}

func (r *memorySolutionRepo) Upsert(_ context.Context, solution solutions.Solution) error {
	r.solutions[solution.UserID+"/"+solution.ProblemSlug] = solution
	return nil
}

func (r *memorySolutionRepo) Get(_ context.Context, userID, slug string) (solutions.Solution, error) {
	solution, ok := r.solutions[userID+"/"+slug]
	if !ok {
		return solutions.Solution{}, solutions.ErrNotFound
	}
	return solution, nil
}
