package solutions

import (
	"context"
	"errors"
	"strings"
	"testing"

	"distry/internal/problems"
)

func TestServiceValidate(t *testing.T) {
	tests := []struct {
		name  string
		files map[string]string
		want  error
	}{
		{
			name:  "happy",
			files: map[string]string{"solution.go": "package solution\n"},
		},
		{
			name:  "unknown file",
			files: map[string]string{"solution.go": "package solution\n", "extra.go": "package solution\n"},
			want:  ErrValidation,
		},
		{
			name:  "missing file",
			files: map[string]string{},
			want:  ErrValidation,
		},
		{
			name:  "oversize",
			files: map[string]string{"solution.go": strings.Repeat("a", MaxFileBytes+1)},
			want:  ErrValidation,
		},
		{
			name:  "invalid utf8",
			files: map[string]string{"solution.go": string([]byte{0xff})},
			want:  ErrValidation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(&fakeRepo{}, fakeProblemRepo{
				problem: problems.Problem{
					Slug:      "perfect-link",
					Templates: map[string]string{"solution.go": "package solution\n"},
				},
			})
			err := svc.Validate(context.Background(), Solution{
				UserID:      "user-a",
				ProblemSlug: "perfect-link",
				Files:       tt.files,
			})
			if !errors.Is(err, tt.want) {
				t.Fatalf("expected error %v, got %v", tt.want, err)
			}
		})
	}
}

type fakeRepo struct{}

func (f *fakeRepo) Upsert(context.Context, Solution) error {
	return nil
}

func (f *fakeRepo) Get(context.Context, string, string) (Solution, error) {
	return Solution{}, ErrNotFound
}

type fakeProblemRepo struct {
	problem problems.Problem
}

func (f fakeProblemRepo) Get(context.Context, string) (problems.Problem, error) {
	return f.problem, nil
}
