package submissions

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"distry/internal/problems"
	"distry/internal/solutions"
	"distry/pkg/sim"
	"distry/pkg/simtest"
)

func TestRunQueuesAndProcessesSubmission(t *testing.T) {
	repo := newMemoryRepo()
	problemRepo := fakeProblemRepo{problem: problems.Problem{
		Slug:      "perfect-link",
		Language:  "go",
		RunConfig: problems.RunConfig{Seeds: []int{1, 2}},
	}}
	runner := &fakeRunner{
		reports: map[int64]simtest.Report{
			1: {Version: simtest.ReportVersion, Seed: 1, Passed: true},
			2: {Version: simtest.ReportVersion, Seed: 2, Passed: false},
		},
	}
	svc := NewService(repo, fakeSolutionRepo{solution: solutions.Solution{
		UserID:      "user-a",
		ProblemSlug: "perfect-link",
		Files:       map[string]string{"solution.go": "package solution\n"},
	}}, problemRepo, map[string]LanguageRunner{"go": runner}, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc.Start(ctx)

	queued, err := svc.Run(ctx, "user-a", "perfect-link", nil)
	if err != nil {
		t.Fatal(err)
	}
	got := waitForStatus(t, repo, queued.ID, StatusFailed)
	if got.Status != StatusFailed || len(got.Reports) != 2 {
		t.Fatalf("unexpected processed submission %+v", got)
	}
	if runner.compiles != 1 || runner.runs != 2 {
		t.Fatalf("expected one compile and two seed runs, got compiles=%d runs=%d", runner.compiles, runner.runs)
	}
}

func TestRunRejectsConcurrentActiveSubmission(t *testing.T) {
	repo := newMemoryRepo()
	repo.submissions["active"] = Submission{
		ID:          "active",
		UserID:      "user-a",
		ProblemSlug: "perfect-link",
		Status:      StatusRunning,
		CreatedAt:   time.Now(),
	}
	svc := NewService(repo, fakeSolutionRepo{}, fakeProblemRepo{problem: problems.Problem{
		Slug:     "perfect-link",
		Language: "go",
	}}, map[string]LanguageRunner{"go": &fakeRunner{}}, 1)

	_, err := svc.Run(context.Background(), "user-a", "perfect-link", nil)
	if !errors.Is(err, ErrActiveRun) {
		t.Fatalf("expected ErrActiveRun, got %v", err)
	}
}

func TestRunUsesCustomSeeds(t *testing.T) {
	repo := newMemoryRepo()
	svc := NewService(repo, fakeSolutionRepo{solution: solutions.Solution{
		UserID:      "user-a",
		ProblemSlug: "perfect-link",
		Files:       map[string]string{"solution.go": "package solution\n"},
	}}, fakeProblemRepo{problem: problems.Problem{
		Slug:      "perfect-link",
		Language:  "go",
		RunConfig: problems.RunConfig{Seeds: []int{1, 2}},
	}}, map[string]LanguageRunner{"go": &fakeRunner{}}, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc.Start(ctx)

	queued, err := svc.Run(ctx, "user-a", "perfect-link", []int{9, 10})
	if err != nil {
		t.Fatal(err)
	}
	got := waitForStatus(t, repo, queued.ID, StatusPassed)
	if seeds := reportSeeds(got.Reports); !slices.Equal(seeds, []int{9, 10}) {
		t.Fatalf("expected custom seeds, got %+v", seeds)
	}
}

func TestReplayUsesSubmissionSnapshotAndFullTrace(t *testing.T) {
	repo := newMemoryRepo()
	repo.submissions["sub-a"] = Submission{
		ID:          "sub-a",
		UserID:      "user-a",
		ProblemSlug: "perfect-link",
		Files:       map[string]string{"solution.go": "snapshot"},
		Status:      StatusFailed,
		CreatedAt:   time.Now(),
	}
	runner := &fakeRunner{}
	svc := NewService(repo, fakeSolutionRepo{solution: solutions.Solution{
		UserID:      "user-a",
		ProblemSlug: "perfect-link",
		Files:       map[string]string{"solution.go": "draft"},
	}}, fakeProblemRepo{problem: problems.Problem{
		Slug:     "perfect-link",
		Language: "go",
	}}, map[string]LanguageRunner{"go": runner}, 1)

	report, err := svc.Replay(context.Background(), "user-a", "sub-a", 11)
	if err != nil {
		t.Fatal(err)
	}
	if report.Seed != 11 || !runner.fullTrace || runner.lastFiles["solution.go"] != "snapshot" {
		t.Fatalf("unexpected replay report=%+v fullTrace=%v files=%+v", report, runner.fullTrace, runner.lastFiles)
	}
}

func TestTraceIsCapped(t *testing.T) {
	report := simtest.Report{Trace: make([]sim.TraceEvent, maxStoredTraceEvents+1)}
	got := capTrace(report)
	if len(got.Trace) != maxStoredTraceEvents || !got.Truncated {
		t.Fatalf("expected capped truncated trace, got len=%d truncated=%v", len(got.Trace), got.Truncated)
	}
}

type fakeRunner struct {
	compiles  int
	runs      int
	reports   map[int64]simtest.Report
	fullTrace bool
	lastFiles map[string]string
}

func (f *fakeRunner) Compile(context.Context, Workspace) (CompileResult, error) {
	f.compiles++
	return CompileResult{Output: "ok"}, nil
}

func (f *fakeRunner) RunSeed(_ context.Context, ws Workspace, seed int64, opts RunSeedOptions) (simtest.Report, error) {
	f.runs++
	f.fullTrace = opts.FullTrace
	f.lastFiles = cloneFiles(ws.Files)
	if report, ok := f.reports[seed]; ok {
		return report, nil
	}
	return simtest.Report{Version: simtest.ReportVersion, Seed: seed, Passed: true}, nil
}

type fakeSolutionRepo struct {
	solution solutions.Solution
}

func (f fakeSolutionRepo) Get(context.Context, string, string) (solutions.Solution, error) {
	if f.solution.Files == nil {
		return solutions.Solution{}, solutions.ErrNotFound
	}
	return f.solution, nil
}

type fakeProblemRepo struct {
	problem problems.Problem
}

func (f fakeProblemRepo) Get(context.Context, string) (problems.Problem, error) {
	if f.problem.Slug == "" {
		return problems.Problem{}, problems.ErrNotFound
	}
	return f.problem, nil
}

type memoryRepo struct {
	mu          sync.Mutex
	next        int
	submissions map[string]Submission
}

func newMemoryRepo() *memoryRepo {
	return &memoryRepo{submissions: map[string]Submission{}}
}

func (r *memoryRepo) HasActive(_ context.Context, userID, slug string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, submission := range r.submissions {
		if submission.UserID == userID && submission.ProblemSlug == slug && slices.Contains([]Status{StatusQueued, StatusCompiling, StatusRunning}, submission.Status) {
			return true, nil
		}
	}
	return false, nil
}

func (r *memoryRepo) Insert(_ context.Context, submission Submission) (Submission, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.next++
	submission.ID = string(rune('a' + r.next - 1))
	submission.CreatedAt = time.Now()
	r.submissions[submission.ID] = submission
	return submission, nil
}

func (r *memoryRepo) Get(_ context.Context, userID, id string) (Submission, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	submission, ok := r.submissions[id]
	if !ok || (userID != "" && submission.UserID != userID) {
		return Submission{}, ErrNotFound
	}
	return submission, nil
}

func (r *memoryRepo) ListForProblem(_ context.Context, userID, slug string, _ int) ([]Submission, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var matches []Submission
	for _, submission := range r.submissions {
		if submission.UserID == userID && submission.ProblemSlug == slug {
			matches = append(matches, submission)
		}
	}
	return matches, nil
}

func (r *memoryRepo) UpdateStatus(_ context.Context, id string, status Status) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	submission := r.submissions[id]
	submission.Status = status
	r.submissions[id] = submission
	return nil
}

func (r *memoryRepo) Finish(_ context.Context, id string, status Status, compileOutput string, reports []simtest.Report) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	submission := r.submissions[id]
	now := time.Now()
	submission.Status = status
	submission.CompileOutput = compileOutput
	submission.Reports = reports
	submission.FinishedAt = &now
	r.submissions[id] = submission
	return nil
}

func waitForStatus(t *testing.T, repo *memoryRepo, id string, status Status) Submission {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		got, err := repo.Get(context.Background(), "", id)
		if err != nil {
			t.Fatal(err)
		}
		if got.Status == status {
			return got
		}
		time.Sleep(10 * time.Millisecond)
	}
	got, _ := repo.Get(context.Background(), "", id)
	t.Fatalf("timed out waiting for %s, got %s", status, got.Status)
	return Submission{}
}
