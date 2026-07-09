package solutions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PGRepo struct {
	pool *pgxpool.Pool
}

func NewPGRepo(pool *pgxpool.Pool) *PGRepo {
	return &PGRepo{pool: pool}
}

func (r *PGRepo) Upsert(ctx context.Context, solution Solution) error {
	files, err := json.Marshal(solution.Files)
	if err != nil {
		return fmt.Errorf("marshal solution files: %w", err)
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO solutions (user_id, problem_slug, files, updated_at)
		VALUES ($1, $2, $3, now())
		ON CONFLICT (user_id, problem_slug) DO UPDATE SET
			files = EXCLUDED.files,
			updated_at = now()
	`, solution.UserID, solution.ProblemSlug, string(files))
	if err != nil {
		return fmt.Errorf("upsert solution %q for user %q: %w", solution.ProblemSlug, solution.UserID, err)
	}
	return nil
}

func (r *PGRepo) Get(ctx context.Context, userID, slug string) (Solution, error) {
	var solution Solution
	var files []byte
	err := r.pool.QueryRow(ctx, `
		SELECT user_id, problem_slug, files, updated_at
		FROM solutions
		WHERE user_id = $1 AND problem_slug = $2
	`, userID, slug).Scan(&solution.UserID, &solution.ProblemSlug, &files, &solution.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Solution{}, ErrNotFound
	}
	if err != nil {
		return Solution{}, fmt.Errorf("get solution %q for user %q: %w", slug, userID, err)
	}
	if err := json.Unmarshal(files, &solution.Files); err != nil {
		return Solution{}, fmt.Errorf("decode solution files: %w", err)
	}
	return solution, nil
}
