-- +goose Up
CREATE TABLE solutions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    problem_slug text NOT NULL REFERENCES problems(slug) ON DELETE CASCADE,
    files jsonb NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_id, problem_slug)
);

-- +goose Down
DROP TABLE IF EXISTS solutions;
