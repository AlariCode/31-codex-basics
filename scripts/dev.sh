#!/usr/bin/env bash

set -euo pipefail

project_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
backend_dir="$project_root/backend"
frontend_dir="$project_root/frontend"
frontend_port="${FRONTEND_PORT:-3005}"

export DATABASE_URL="${DATABASE_URL:-postgres://uptime:uptime@localhost:5436/uptime?sslmode=disable}"
export JWT_SECRET="${JWT_SECRET:-local-development-jwt-secret-change-before-production}"
export CORS_ORIGIN="${CORS_ORIGIN:-http://localhost:${frontend_port}}"
export COOKIE_SECURE="${COOKIE_SECURE:-false}"
export NEXT_PUBLIC_API_URL="${NEXT_PUBLIC_API_URL:-http://localhost:8080}"

backend_pid=""
frontend_pid=""

stop_process_tree() {
  local pid="$1"
  local child_pid

  [[ -z "$pid" ]] && return
  while IFS= read -r child_pid; do
    stop_process_tree "$child_pid"
  done < <(pgrep -P "$pid" 2>/dev/null || true)
  kill "$pid" 2>/dev/null || true
}

cleanup() {
  trap - EXIT INT TERM
  stop_process_tree "$backend_pid"
  stop_process_tree "$frontend_pid"
  wait 2>/dev/null || true
}

trap cleanup EXIT INT TERM

echo "Starting PostgreSQL..."
(cd "$backend_dir" && docker compose up -d postgres)

for attempt in {1..15}; do
  if (cd "$backend_dir" && docker compose exec -T postgres pg_isready -U uptime -d uptime >/dev/null); then
    break
  fi
  if [[ "$attempt" == "15" ]]; then
    echo "PostgreSQL did not become ready in time." >&2
    exit 1
  fi
  sleep 1
done

echo "Applying database migrations..."
(cd "$backend_dir" && go run ./cmd/migrate)

echo "Starting backend at http://localhost:8080..."
pushd "$backend_dir" >/dev/null
go run ./cmd/server &
backend_pid="$!"
popd >/dev/null

echo "Starting frontend at http://localhost:${frontend_port}..."
pushd "$frontend_dir" >/dev/null
npm run dev -- --port "$frontend_port" &
frontend_pid="$!"
popd >/dev/null

echo "Both applications are running. Press Ctrl+C to stop them."
wait "$backend_pid" "$frontend_pid"
