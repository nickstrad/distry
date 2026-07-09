package problems

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

func (r *PGRepo) Upsert(ctx context.Context, problem Problem) error {
	tags, err := jsonText(problem.Tags)
	if err != nil {
		return fmt.Errorf("marshal tags: %w", err)
	}
	templates, err := jsonText(problem.Templates)
	if err != nil {
		return fmt.Errorf("marshal templates: %w", err)
	}
	runConfig, err := jsonText(problem.RunConfig)
	if err != nil {
		return fmt.Errorf("marshal run config: %w", err)
	}

	_, err = r.pool.Exec(ctx, `
		INSERT INTO problems (
			slug, title, difficulty, language, tags, order_idx, entrypoint,
			description_md, templates, run_config, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, now())
		ON CONFLICT (slug) DO UPDATE SET
			title = EXCLUDED.title,
			difficulty = EXCLUDED.difficulty,
			language = EXCLUDED.language,
			tags = EXCLUDED.tags,
			order_idx = EXCLUDED.order_idx,
			entrypoint = EXCLUDED.entrypoint,
			description_md = EXCLUDED.description_md,
			templates = EXCLUDED.templates,
			run_config = EXCLUDED.run_config,
			updated_at = now()
	`, problem.Slug, problem.Title, problem.Difficulty, problem.Language, tags, problem.Order,
		problem.Entrypoint, problem.DescriptionMD, templates, runConfig)
	if err != nil {
		return fmt.Errorf("upsert problem %q: %w", problem.Slug, err)
	}
	return nil
}

func (r *PGRepo) List(ctx context.Context) ([]Summary, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT slug, title, difficulty, tags, order_idx
		FROM problems
		ORDER BY order_idx ASC, title ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list problems: %w", err)
	}
	defer rows.Close()

	var summaries []Summary
	for rows.Next() {
		var summary Summary
		var tags []byte
		if err := rows.Scan(&summary.Slug, &summary.Title, &summary.Difficulty, &tags, &summary.Order); err != nil {
			return nil, fmt.Errorf("scan problem summary: %w", err)
		}
		if err := json.Unmarshal(tags, &summary.Tags); err != nil {
			return nil, fmt.Errorf("decode problem tags: %w", err)
		}
		summaries = append(summaries, summary)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate problem summaries: %w", err)
	}
	return summaries, nil
}

func (r *PGRepo) ListSolved(ctx context.Context, userID string) (map[string]bool, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT problem_slug
		FROM submissions
		WHERE user_id = $1 AND status = 'passed'
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list solved problems: %w", err)
	}
	defer rows.Close()

	solved := map[string]bool{}
	for rows.Next() {
		var slug string
		if err := rows.Scan(&slug); err != nil {
			return nil, fmt.Errorf("scan solved problem: %w", err)
		}
		solved[slug] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate solved problems: %w", err)
	}
	return solved, nil
}

func (r *PGRepo) Get(ctx context.Context, slug string) (Problem, error) {
	var problem Problem
	var tags []byte
	var templates []byte
	var runConfig []byte

	err := r.pool.QueryRow(ctx, `
		SELECT slug, title, difficulty, language, tags, order_idx, entrypoint,
			description_md, templates, run_config
		FROM problems
		WHERE slug = $1
	`, slug).Scan(&problem.Slug, &problem.Title, &problem.Difficulty, &problem.Language,
		&tags, &problem.Order, &problem.Entrypoint, &problem.DescriptionMD, &templates, &runConfig)
	if errors.Is(err, pgx.ErrNoRows) {
		return Problem{}, ErrNotFound
	}
	if err != nil {
		return Problem{}, fmt.Errorf("get problem %q: %w", slug, err)
	}
	if err := json.Unmarshal(tags, &problem.Tags); err != nil {
		return Problem{}, fmt.Errorf("decode problem tags: %w", err)
	}
	if err := json.Unmarshal(templates, &problem.Templates); err != nil {
		return Problem{}, fmt.Errorf("decode problem templates: %w", err)
	}
	if err := json.Unmarshal(runConfig, &problem.RunConfig); err != nil {
		return Problem{}, fmt.Errorf("decode problem run config: %w", err)
	}
	return problem, nil
}

func Sync(ctx context.Context, repo Repo, loaded []Problem) error {
	for _, problem := range loaded {
		if err := repo.Upsert(ctx, problem); err != nil {
			return err
		}
	}
	return nil
}

func jsonText(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}
