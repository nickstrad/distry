package db

import (
	"context"
	"embed"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrations embed.FS

func Up(ctx context.Context, pool *pgxpool.Pool) error {
	return Migrate(ctx, pool, "up")
}

func Migrate(ctx context.Context, pool *pgxpool.Pool, command string, args ...string) error {
	goose.SetBaseFS(migrations)
	defer goose.SetBaseFS(nil)

	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	conn := stdlib.OpenDBFromPool(pool)
	defer conn.Close()

	return goose.RunContext(ctx, command, conn, "migrations", args...)
}
