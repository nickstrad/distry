#!/usr/bin/env sh
set -eu

staged_files="$(git diff --cached --name-only --diff-filter=ACMR)"

has_staged_file() {
  pattern="$1"
  printf '%s\n' "$staged_files" | grep -Eq "$pattern"
}

npm_script_exists() {
  package_dir="$1"
  script="$2"
  npm --prefix "$package_dir" pkg get "scripts.$script" | grep -qv '^{}$'
}

run_npm_script_if_exists() {
  package_dir="$1"
  script="$2"
  if npm_script_exists "$package_dir" "$script"; then
    npm --prefix "$package_dir" run "$script"
  fi
}

if has_staged_file '\.go$'; then
  echo "pre-commit: vetting Go packages"
  go vet ./...

  echo "pre-commit: testing Go packages"
  go test ./...
fi

if has_staged_file '^frontend/.*\.(js|jsx|ts|tsx|css|json|html|md)$'; then
  if npm_script_exists frontend lint; then
    echo "pre-commit: linting frontend"
    npm --prefix frontend run lint
  else
    echo "pre-commit: frontend lint script not configured; skipping"
  fi

  echo "pre-commit: typechecking frontend"
  run_npm_script_if_exists frontend typecheck

  echo "pre-commit: testing frontend"
  run_npm_script_if_exists frontend test
fi
