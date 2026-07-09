-- +goose Up
CREATE TABLE IF NOT EXISTS schema_bootstrap (
    id integer PRIMARY KEY DEFAULT 1,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT schema_bootstrap_singleton CHECK (id = 1)
);

INSERT INTO schema_bootstrap (id)
VALUES (1)
ON CONFLICT (id) DO NOTHING;

-- +goose Down
DROP TABLE IF EXISTS schema_bootstrap;
