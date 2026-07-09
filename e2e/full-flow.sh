#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PORT="${PORT:-18080}"
POSTGRES_IMAGE="${POSTGRES_IMAGE:-postgres:16-alpine}"
POSTGRES_PORT="${POSTGRES_PORT:-15432}"
POSTGRES_CONTAINER="${POSTGRES_CONTAINER:-distry-e2e-postgres-$$}"
POSTGRES_DB="${POSTGRES_DB:-distry_e2e}"
POSTGRES_USER="${POSTGRES_USER:-distry}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-distry}"
BASE_URL="http://localhost:${PORT}"
COOKIE_JAR="$(mktemp)"
SERVER_LOG="$(mktemp)"
SERVER_PID=""

cleanup() {
  if [[ -n "${SERVER_PID}" ]]; then
    kill "${SERVER_PID}" 2>/dev/null || true
    wait "${SERVER_PID}" 2>/dev/null || true
  fi
  docker rm -f "${POSTGRES_CONTAINER}" >/dev/null 2>&1 || true
  rm -f "${COOKIE_JAR}" "${SERVER_LOG}"
}
trap cleanup EXIT

api() {
  local method="$1"
  local path="$2"
  local body="${3:-}"
  if [[ -n "${body}" ]]; then
    curl -fsS -b "${COOKIE_JAR}" -c "${COOKIE_JAR}" -H "Content-Type: application/json" -X "${method}" --data "${body}" "${BASE_URL}${path}"
  else
    curl -fsS -b "${COOKIE_JAR}" -c "${COOKIE_JAR}" -X "${method}" "${BASE_URL}${path}"
  fi
}

json_file_payload() {
  local file="$1"
  node -e 'const fs=require("fs"); const file=process.argv[1]; const source=fs.readFileSync(file, "utf8").replace(/^package\s+\w+/m, "package solution"); console.log(JSON.stringify({files: {"solution.go": source}}));' "${file}"
}

json_get() {
  node -e 'let data=""; process.stdin.on("data", c => data += c); process.stdin.on("end", () => { const obj=JSON.parse(data); const path=process.argv[1].split("."); let value=obj; for (const key of path) value=value?.[key]; if (typeof value === "object") console.log(JSON.stringify(value)); else console.log(value ?? ""); });' "$1"
}

wait_for_server() {
  for _ in {1..60}; do
    if curl -fsS "${BASE_URL}/api/health" >/dev/null 2>&1; then
      return
    fi
    sleep 0.5
  done
  echo "server did not become healthy; log follows" >&2
  cat "${SERVER_LOG}" >&2
  exit 1
}

start_postgres() {
  docker rm -f "${POSTGRES_CONTAINER}" >/dev/null 2>&1 || true
  docker run -d \
    --name "${POSTGRES_CONTAINER}" \
    -e POSTGRES_DB="${POSTGRES_DB}" \
    -e POSTGRES_USER="${POSTGRES_USER}" \
    -e POSTGRES_PASSWORD="${POSTGRES_PASSWORD}" \
    -p "127.0.0.1:${POSTGRES_PORT}:5432" \
    "${POSTGRES_IMAGE}" >/dev/null

  for _ in {1..60}; do
    if docker exec "${POSTGRES_CONTAINER}" pg_isready -U "${POSTGRES_USER}" -d "${POSTGRES_DB}" >/dev/null 2>&1; then
      return
    fi
    sleep 0.5
  done
  echo "postgres container did not become ready; log follows" >&2
  docker logs "${POSTGRES_CONTAINER}" >&2 || true
  exit 1
}

wait_for_submission() {
  local id="$1"
  local response status
  for _ in {1..120}; do
    response="$(api GET "/api/submissions/${id}")"
    status="$(printf '%s' "${response}" | json_get status)"
    case "${status}" in
      passed|failed|error)
        printf '%s' "${response}"
        return
        ;;
    esac
    sleep 1
  done
  echo "submission ${id} did not finish" >&2
  exit 1
}

run_fixture() {
  local slug="$1"
  local fixture="$2"
  local want_status="$3"
  local want_checker="$4"
  local file="${ROOT}/problems/${slug}/harness/testdata/${fixture}/solution.go"
  local save_body started id finished status checker

  save_body="$(json_file_payload "${file}")"
  api PUT "/api/problems/${slug}/solution" "${save_body}" >/dev/null
  started="$(api POST "/api/problems/${slug}/run" "{}")"
  id="$(printf '%s' "${started}" | json_get submissionID)"
  finished="$(wait_for_submission "${id}")"
  status="$(printf '%s' "${finished}" | json_get status)"
  if [[ "${status}" != "${want_status}" ]]; then
    echo "${slug}/${fixture}: status ${status}, want ${want_status}" >&2
    printf '%s\n' "${finished}" >&2
    exit 1
  fi
  if [[ "${want_checker}" != "-" ]]; then
    checker="$(printf '%s' "${finished}" | json_get reports.0.violations.0.checker)"
    if [[ "${checker}" != "${want_checker}" ]]; then
      echo "${slug}/${fixture}: checker ${checker}, want ${want_checker}" >&2
      printf '%s\n' "${finished}" >&2
      exit 1
    fi
    replay_failed_seed "${id}" "${finished}"
  fi
}

replay_failed_seed() {
  local id="$1"
  local submission="$2"
  local seed original replay replay_checker original_checker replay_trace
  seed="$(printf '%s' "${submission}" | json_get reports.0.seed)"
  original_checker="$(printf '%s' "${submission}" | json_get reports.0.violations.0.checker)"
  original="$(printf '%s' "${submission}" | json_get reports.0.violations)"
  replay="$(api POST "/api/submissions/${id}/replay" "{\"seed\":${seed}}")"
  replay_checker="$(printf '%s' "${replay}" | json_get violations.0.checker)"
  replay_trace="$(printf '%s' "${replay}" | json_get trace.0.seq)"
  if [[ "${replay_checker}" != "${original_checker}" || -z "${replay_trace}" ]]; then
    echo "replay for ${id} did not reproduce checker with full trace" >&2
    printf 'original violations: %s\nreplay: %s\n' "${original}" "${replay}" >&2
    exit 1
  fi
}

cd "${ROOT}"

start_postgres
export DATABASE_URL="postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@127.0.0.1:${POSTGRES_PORT}/${POSTGRES_DB}?sslmode=disable"

go run ./cmd/migrate up
PORT="${PORT}" DISTRY_REPO_ROOT="${ROOT}" go run ./cmd/server >"${SERVER_LOG}" 2>&1 &
SERVER_PID="$!"
wait_for_server

suffix="$(date +%s)"
api POST /api/auth/signup "{\"username\":\"e2e${suffix}\",\"email\":\"e2e${suffix}@example.com\",\"password\":\"password123\"}" >/dev/null

run_fixture perfect-link correct passed -
run_fixture perfect-link bug-no-dedup failed NoDuplicateDelivery
run_fixture lcr-election correct passed -
run_fixture lcr-election bug-everyone-leader failed SingleLeader
run_fixture uniform-reliable-broadcast correct passed -
run_fixture uniform-reliable-broadcast bug-no-relay failed AllDelivered

echo "full-flow e2e passed"
