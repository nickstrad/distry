package solutions

import (
	"context"
	"errors"
	"fmt"
	"time"
	"unicode/utf8"

	"distry/internal/problems"
)

const MaxFileBytes = 64 * 1024

var (
	ErrNotFound   = errors.New("solution not found")
	ErrValidation = errors.New("solution validation failed")
)

type Solution struct {
	UserID      string            `json:"-"`
	ProblemSlug string            `json:"problem_slug"`
	Files       map[string]string `json:"files"`
	UpdatedAt   time.Time         `json:"updated_at,omitempty"`
}

type Repo interface {
	Upsert(context.Context, Solution) error
	Get(ctx context.Context, userID, slug string) (Solution, error)
}

type ProblemRepo interface {
	Get(context.Context, string) (problems.Problem, error)
}

type Service struct {
	repo     Repo
	problems ProblemRepo
	maxBytes int
}

func NewService(repo Repo, problemRepo ProblemRepo) *Service {
	return &Service{repo: repo, problems: problemRepo, maxBytes: MaxFileBytes}
}

func (s *Service) Upsert(ctx context.Context, solution Solution) error {
	if err := s.Validate(ctx, solution); err != nil {
		return err
	}
	return s.repo.Upsert(ctx, solution)
}

func (s *Service) Get(ctx context.Context, userID, slug string) (Solution, error) {
	return s.repo.Get(ctx, userID, slug)
}

func (s *Service) Validate(ctx context.Context, solution Solution) error {
	problem, err := s.problems.Get(ctx, solution.ProblemSlug)
	if err != nil {
		return err
	}
	return validateFiles(solution.Files, problem.Templates, s.maxBytes)
}

func validateFiles(files, templates map[string]string, maxBytes int) error {
	if len(files) != len(templates) {
		return fmt.Errorf("%w: files must match problem templates", ErrValidation)
	}
	for name, contents := range files {
		if _, ok := templates[name]; !ok {
			return fmt.Errorf("%w: unknown file %q", ErrValidation, name)
		}
		if len(contents) > maxBytes {
			return fmt.Errorf("%w: file %q is too large", ErrValidation, name)
		}
		if !utf8.ValidString(contents) {
			return fmt.Errorf("%w: file %q is not UTF-8", ErrValidation, name)
		}
	}
	for name := range templates {
		if _, ok := files[name]; !ok {
			return fmt.Errorf("%w: missing file %q", ErrValidation, name)
		}
	}
	return nil
}
