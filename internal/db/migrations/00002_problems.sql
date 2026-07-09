-- +goose Up
CREATE TABLE problems (
    slug text PRIMARY KEY,
    title text NOT NULL,
    difficulty text NOT NULL,
    language text NOT NULL,
    tags jsonb NOT NULL DEFAULT '[]'::jsonb,
    order_idx integer NOT NULL,
    entrypoint text NOT NULL,
    description_md text NOT NULL,
    templates jsonb NOT NULL DEFAULT '{}'::jsonb,
    run_config jsonb NOT NULL DEFAULT '{}'::jsonb,
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS problems;
