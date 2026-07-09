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
	ErrInvalidSeeds    = errors.New("invalid seed list")
)

const maxStoredTraceEvents = 5000

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
	RunSeed(ctx context.Context, ws Workspace, seed int64, opts RunSeedOptions) (simtest.Report, error)
}

type Workspace struct {
	Problem problems.Problem
	Files   map[string]string
}

type CompileResult struct {
	Output string
}

type RunSeedOptions struct {
	FullTrace bool
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

func (s *Service) Run(ctx context.Context, userID, slug string, seeds []int) (Submission, error) {
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
	seeds, err = resolveSeeds(seeds, problem.RunConfig.Seeds)
	if err != nil {
		return Submission{}, err
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
		Reports:     seedPlaceholders(seeds),
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

func (s *Service) Replay(ctx context.Context, userID, id string, seed int64) (simtest.Report, error) {
	submission, err := s.repo.Get(ctx, userID, id)
	if err != nil {
		return simtest.Report{}, err
	}
	if err := ValidateSeeds([]int{int(seed)}); err != nil {
		return simtest.Report{}, err
	}
	problem, err := s.problems.Get(ctx, submission.ProblemSlug)
	if err != nil {
		return simtest.Report{}, err
	}
	runner, ok := s.runners[problem.Language]
	if !ok {
		return simtest.Report{}, fmt.Errorf("%w: %s", ErrUnsupported, problem.Language)
	}
	report := runSeed(ctx, runner, submission.Workspace(problem), seed, RunSeedOptions{FullTrace: true})
	return capTrace(report), nil
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
	ws := submission.Workspace(problem)
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
	seeds := seedsFromReports(submission.Reports, problem.RunConfig.Seeds)
	reports, status := s.runSeeds(ctx, runner, ws, seeds)
	_ = s.repo.Finish(ctx, id, status, compile.Output, reports)
}

func (s *Service) runSeeds(ctx context.Context, runner LanguageRunner, ws Workspace, seeds []int) ([]simtest.Report, Status) {
	reports := make([]simtest.Report, 0, len(seeds))
	status := StatusPassed
	for _, seed := range seeds {
		report := runSeed(ctx, runner, ws, int64(seed), RunSeedOptions{})
		if !report.Passed {
			status = StatusFailed
		}
		reports = append(reports, capTrace(report))
	}
	return reports, status
}

func runSeed(ctx context.Context, runner LanguageRunner, ws Workspace, seed int64, opts RunSeedOptions) simtest.Report {
	report, err := runner.RunSeed(ctx, ws, seed, opts)
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

func ValidateSeeds(seeds []int) error {
	if len(seeds) < 1 || len(seeds) > 20 {
		return fmt.Errorf("%w: provide 1-20 seeds", ErrInvalidSeeds)
	}
	for _, seed := range seeds {
		if seed < 0 {
			return fmt.Errorf("%w: seeds must be non-negative", ErrInvalidSeeds)
		}
	}
	return nil
}

func resolveSeeds(requested, defaults []int) ([]int, error) {
	if len(requested) == 0 {
		return defaults, nil
	}
	if err := ValidateSeeds(requested); err != nil {
		return nil, err
	}
	return requested, nil
}

func seedPlaceholders(seeds []int) []simtest.Report {
	reports := make([]simtest.Report, 0, len(seeds))
	for _, seed := range seeds {
		reports = append(reports, simtest.Report{Version: simtest.ReportVersion, Seed: int64(seed)})
	}
	return reports
}

func reportSeeds(reports []simtest.Report) []int {
	seeds := make([]int, 0, len(reports))
	for _, report := range reports {
		seeds = append(seeds, int(report.Seed))
	}
	return seeds
}

func seedsFromReports(reports []simtest.Report, defaults []int) []int {
	seeds := reportSeeds(reports)
	if len(seeds) == 0 {
		return defaults
	}
	return seeds
}

func capTrace(report simtest.Report) simtest.Report {
	if len(report.Trace) <= maxStoredTraceEvents {
		return report
	}
	report.Trace = report.Trace[:maxStoredTraceEvents]
	report.Truncated = true
	return report
}

func cloneFiles(files map[string]string) map[string]string {
	cloned := make(map[string]string, len(files))
	for name, contents := range files {
		cloned[name] = contents
	}
	return cloned
}

func (s Submission) Workspace(problem problems.Problem) Workspace {
	return Workspace{Problem: problem, Files: cloneFiles(s.Files)}
}
