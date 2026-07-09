package auth

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type DBTX interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type PGUserRepo struct {
	db DBTX
}

func NewPGUserRepo(db DBTX) *PGUserRepo {
	return &PGUserRepo{db: db}
}

func (r *PGUserRepo) Create(ctx context.Context, username, email, passwordHash string) (User, error) {
	var user User
	err := r.db.QueryRow(ctx, `
		INSERT INTO users (username, email, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id::text, username, email
	`, username, email, passwordHash).Scan(&user.ID, &user.Username, &user.Email)
	if isUniqueViolation(err) {
		return User{}, ErrTaken
	}
	return user, err
}

func (r *PGUserRepo) ByEmail(ctx context.Context, email string) (User, string, error) {
	var user User
	var passwordHash string
	err := r.db.QueryRow(ctx, `
		SELECT id::text, username, email, password_hash
		FROM users
		WHERE email = $1
	`, email).Scan(&user.ID, &user.Username, &user.Email, &passwordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, "", ErrInvalidCredentials
	}
	return user, passwordHash, err
}

type PGSessionRepo struct {
	db DBTX
}

func NewPGSessionRepo(db DBTX) *PGSessionRepo {
	return &PGSessionRepo{db: db}
}

func (r *PGSessionRepo) Create(ctx context.Context, tokenHash []byte, userID string, expires time.Time) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO sessions (token_hash, user_id, expires_at)
		VALUES ($1, $2, $3)
	`, tokenHash, userID, expires)
	return err
}

func (r *PGSessionRepo) UserByTokenHash(ctx context.Context, tokenHash []byte) (User, error) {
	var user User
	err := r.db.QueryRow(ctx, `
		SELECT users.id::text, users.username, users.email
		FROM sessions
		JOIN users ON users.id = sessions.user_id
		WHERE sessions.token_hash = $1 AND sessions.expires_at > now()
	`, tokenHash).Scan(&user.ID, &user.Username, &user.Email)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrUnauthenticated
	}
	return user, err
}

func (r *PGSessionRepo) Delete(ctx context.Context, tokenHash []byte) error {
	_, err := r.db.Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, tokenHash)
	return err
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
