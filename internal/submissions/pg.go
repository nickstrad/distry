package submissions

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"distry/pkg/simtest"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PGRepo struct {
	pool *pgxpool.Pool
}

func NewPGRepo(pool *pgxpool.Pool) *PGRepo {
	return &PGRepo{pool: pool}
}

func (r *PGRepo) HasActive(ctx context.Context, userID, slug string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM submissions
			WHERE user_id = $1
			  AND problem_slug = $2
			  AND status IN ('queued', 'compiling', 'running')
		)
	`, userID, slug).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check active submission: %w", err)
	}
	return exists, nil
}

func (r *PGRepo) Insert(ctx context.Context, submission Submission) (Submission, error) {
	files, err := json.Marshal(submission.Files)
	if err != nil {
		return Submission{}, fmt.Errorf("marshal submission files: %w", err)
	}
	inserted, err := scanSubmission(r.pool.QueryRow(ctx, `
		INSERT INTO submissions (user_id, problem_slug, files, status)
		VALUES ($1, $2, $3, $4)
		RETURNING id, user_id, problem_slug, files, status, compile_output, reports, created_at, finished_at
	`, submission.UserID, submission.ProblemSlug, string(files), submission.Status))
	if err != nil {
		return Submission{}, fmt.Errorf("insert submission: %w", err)
	}
	return inserted, nil
}

func (r *PGRepo) Get(ctx context.Context, userID, id string) (Submission, error) {
	where := "WHERE id = $1 AND user_id = $2"
	args := []any{id, userID}
	if userID == "" {
		where = "WHERE id = $1"
		args = []any{id}
	}
	row := r.pool.QueryRow(ctx, `
		SELECT id, user_id, problem_slug, files, status, compile_output, reports, created_at, finished_at
		FROM submissions
		`+where, args...)
	return scanSubmission(row)
}

func (r *PGRepo) ListForProblem(ctx context.Context, userID, slug string, limit int) ([]Submission, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, problem_slug, files, status, compile_output, reports, created_at, finished_at
		FROM submissions
		WHERE user_id = $1 AND problem_slug = $2
		ORDER BY created_at DESC
		LIMIT $3
	`, userID, slug, limit)
	if err != nil {
		return nil, fmt.Errorf("list submissions: %w", err)
	}
	defer rows.Close()

	var submissions []Submission
	for rows.Next() {
		submission, err := scanSubmission(rows)
		if err != nil {
			return nil, err
		}
		submissions = append(submissions, submission)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate submissions: %w", err)
	}
	return submissions, nil
}

func (r *PGRepo) UpdateStatus(ctx context.Context, id string, status Status) error {
	_, err := r.pool.Exec(ctx, `UPDATE submissions SET status = $2 WHERE id = $1`, id, status)
	if err != nil {
		return fmt.Errorf("update submission status: %w", err)
	}
	return nil
}

func (r *PGRepo) Finish(ctx context.Context, id string, status Status, compileOutput string, reports []simtest.Report) error {
	encodedReports, err := json.Marshal(reports)
	if err != nil {
		return fmt.Errorf("marshal submission reports: %w", err)
	}
	_, err = r.pool.Exec(ctx, `
		UPDATE submissions
		SET status = $2, compile_output = $3, reports = $4, finished_at = now()
		WHERE id = $1
	`, id, status, compileOutput, string(encodedReports))
	if err != nil {
		return fmt.Errorf("finish submission: %w", err)
	}
	return nil
}

type scanner interface {
	Scan(...any) error
}

func scanSubmission(row scanner) (Submission, error) {
	var submission Submission
	var files []byte
	var reports []byte
	var compileOutput sql.NullString
	err := row.Scan(
		&submission.ID,
		&submission.UserID,
		&submission.ProblemSlug,
		&files,
		&submission.Status,
		&compileOutput,
		&reports,
		&submission.CreatedAt,
		&submission.FinishedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Submission{}, ErrNotFound
	}
	if err != nil {
		return Submission{}, fmt.Errorf("scan submission: %w", err)
	}
	if err := json.Unmarshal(files, &submission.Files); err != nil {
		return Submission{}, fmt.Errorf("decode submission files: %w", err)
	}
	if compileOutput.Valid {
		submission.CompileOutput = compileOutput.String
	}
	if len(reports) > 0 {
		if err := json.Unmarshal(reports, &submission.Reports); err != nil {
			return Submission{}, fmt.Errorf("decode submission reports: %w", err)
		}
	}
	return submission, nil
}
