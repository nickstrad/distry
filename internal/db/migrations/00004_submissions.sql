-- +goose Up
CREATE TABLE submissions (
  id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id        uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  problem_slug   text NOT NULL REFERENCES problems(slug),
  files          jsonb NOT NULL,
  status         text NOT NULL,
  compile_output text,
  reports        jsonb,
  created_at     timestamptz NOT NULL DEFAULT now(),
  finished_at    timestamptz
);

CREATE INDEX submissions_user_problem_created_idx
  ON submissions (user_id, problem_slug, created_at DESC);

CREATE INDEX submissions_active_idx
  ON submissions (user_id, problem_slug, status)
  WHERE status IN ('queued', 'compiling', 'running');

-- +goose Down
DROP TABLE submissions;
