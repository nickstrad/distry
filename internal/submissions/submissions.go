package submissions

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"distry/internal/problems"
	"distry/internal/solutions"
	"distry/pkg/simtest"
)

type Status string

const (
	StatusQueued    Status = "queued"
	StatusCompiling Status = "compiling"
	StatusRunning   Status = "running"
	StatusPassed    Status = "passed"
	StatusFailed    Status = "failed"
	StatusError     Status = "error"
)

var (
	ErrNotFound        = errors.New("submission not found")
	ErrActiveRun       = errors.New("active submission already exists")
	ErrUnsupported     = errors.New("unsupported submission language")
	ErrNoSavedSolution = errors.New("saved solution not found")
)

type Submission struct {
	ID            string            `json:"id"`
	UserID        string            `json:"-"`
	ProblemSlug   string            `json:"problem_slug"`
	Files         map[string]string `json:"files,omitempty"`
	Status        Status            `json:"status"`
	CompileOutput string            `json:"compile_output,omitempty"`
	Reports       []simtest.Report  `json:"reports,omitempty"`
	CreatedAt     time.Time         `json:"created_at"`
	FinishedAt    *time.Time        `json:"finished_at,omitempty"`
}

type Repo interface {
	HasActive(ctx context.Context, userID, slug string) (bool, error)
	Insert(ctx context.Context, submission Submission) (Submission, error)
	Get(ctx context.Context, userID, id string) (Submission, error)
	ListForProblem(ctx context.Context, userID, slug string, limit int) ([]Submission, error)
	UpdateStatus(ctx context.Context, id string, status Status) error
	Finish(ctx context.Context, id string, status Status, compileOutput string, reports []simtest.Report) error
}

type SolutionRepo interface {
	Get(ctx context.Context, userID, slug string) (solutions.Solution, error)
}

type ProblemRepo interface {
	Get(context.Context, string) (problems.Problem, error)
}

type LanguageRunner interface {
	Compile(ctx context.Context, ws Workspace) (CompileResult, error)
	RunSeed(ctx context.Context, ws Workspace, seed int64) (simtest.Report, error)
}

type Workspace struct {
	Problem problems.Problem
	Files   map[string]string
}

type CompileResult struct {
	Output string
}

type Service struct {
	repo      Repo
	solutions SolutionRepo
	problems  ProblemRepo
	runners   map[string]LanguageRunner
	jobs      chan string
	workers   int

	mu      sync.Mutex
	started bool
}

func NewService(repo Repo, solutionRepo SolutionRepo, problemRepo ProblemRepo, runners map[string]LanguageRunner, workers int) *Service {
	if workers <= 0 {
		workers = 1
	}
	return &Service{
		repo:      repo,
		solutions: solutionRepo,
		problems:  problemRepo,
		runners:   runners,
		jobs:      make(chan string, workers*4),
		workers:   workers,
	}
}

func (s *Service) Start(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		return
	}
	s.started = true
	for i := 0; i < s.workers; i++ {
		go s.worker(ctx)
	}
}

func (s *Service) Run(ctx context.Context, userID, slug string) (Submission, error) {
	active, err := s.repo.HasActive(ctx, userID, slug)
	if err != nil {
		return Submission{}, err
	}
	if active {
		return Submission{}, ErrActiveRun
	}
	problem, err := s.problems.Get(ctx, slug)
	if err != nil {
		return Submission{}, err
	}
	if _, ok := s.runners[problem.Language]; !ok {
		return Submission{}, fmt.Errorf("%w: %s", ErrUnsupported, problem.Language)
	}
	solution, err := s.solutions.Get(ctx, userID, slug)
	if errors.Is(err, solutions.ErrNotFound) {
		return Submission{}, ErrNoSavedSolution
	}
	if err != nil {
		return Submission{}, err
	}
	submission, err := s.repo.Insert(ctx, Submission{
		UserID:      userID,
		ProblemSlug: slug,
		Files:       cloneFiles(solution.Files),
		Status:      StatusQueued,
	})
	if err != nil {
		return Submission{}, err
	}
	select {
	case s.jobs <- submission.ID:
	default:
		go func() { s.jobs <- submission.ID }()
	}
	return submission, nil
}

func (s *Service) Get(ctx context.Context, userID, id string) (Submission, error) {
	return s.repo.Get(ctx, userID, id)
}

func (s *Service) ListForProblem(ctx context.Context, userID, slug string) ([]Submission, error) {
	return s.repo.ListForProblem(ctx, userID, slug, 20)
}

func (s *Service) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case id := <-s.jobs:
			s.process(ctx, id)
		}
	}
}

func (s *Service) process(ctx context.Context, id string) {
	submission, err := s.repo.Get(ctx, "", id)
	if err != nil {
		return
	}
	problem, err := s.problems.Get(ctx, submission.ProblemSlug)
	if err != nil {
		_ = s.repo.Finish(ctx, id, StatusError, "failed to load problem", nil)
		return
	}
	runner, ok := s.runners[problem.Language]
	if !ok {
		_ = s.repo.Finish(ctx, id, StatusError, "unsupported language", nil)
		return
	}
	ws := Workspace{Problem: problem, Files: cloneFiles(submission.Files)}
	if err := s.repo.UpdateStatus(ctx, id, StatusCompiling); err != nil {
		return
	}
	compile, err := runner.Compile(ctx, ws)
	if err != nil {
		_ = s.repo.Finish(ctx, id, StatusError, compile.Output, nil)
		return
	}
	if err := s.repo.UpdateStatus(ctx, id, StatusRunning); err != nil {
		return
	}
	reports, status := s.runSeeds(ctx, runner, ws, problem.RunConfig.Seeds)
	_ = s.repo.Finish(ctx, id, status, compile.Output, reports)
}

func (s *Service) runSeeds(ctx context.Context, runner LanguageRunner, ws Workspace, seeds []int) ([]simtest.Report, Status) {
	reports := make([]simtest.Report, 0, len(seeds))
	status := StatusPassed
	for _, seed := range seeds {
		report := runSeed(ctx, runner, ws, int64(seed))
		if !report.Passed {
			status = StatusFailed
		}
		reports = append(reports, report)
	}
	return reports, status
}

func runSeed(ctx context.Context, runner LanguageRunner, ws Workspace, seed int64) simtest.Report {
	report, err := runner.RunSeed(ctx, ws, seed)
	if err == nil {
		return report
	}
	return failedSeedReport(seed, err)
}

func failedSeedReport(seed int64, err error) simtest.Report {
	return simtest.Report{
		Version: simtest.ReportVersion,
		Seed:    seed,
		Passed:  false,
		Violations: []simtest.Violation{{
			Checker:  "runner.seed",
			Message:  err.Error(),
			EventSeq: -1,
		}},
	}
}

func cloneFiles(files map[string]string) map[string]string {
	cloned := make(map[string]string, len(files))
	for name, contents := range files {
		cloned[name] = contents
	}
	return cloned
}
