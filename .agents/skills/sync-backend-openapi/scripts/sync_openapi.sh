#!/usr/bin/env bash

set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
repository_root="$(cd -- "$script_dir/../../../.." && pwd)"
backend_dir="${1:-$repository_root/backend}"

if [[ ! -d "$backend_dir" ]]; then
  printf 'Backend directory does not exist: %s\n' "$backend_dir" >&2
  exit 1
fi

if [[ ! -f "$backend_dir/go.mod" ]]; then
  printf 'Backend directory is not a Go module: %s\n' "$backend_dir" >&2
  exit 1
fi

cd -- "$backend_dir/cmd/server"

go run github.com/swaggo/swag/cmd/swag@v1.16.6 \
  init \
  -g cmd/server/main.go \
  -d ../.. \
  -o ../../docs \
  --parseInternal

npx --yes redoc-cli@0.13.21 \
  bundle ../../docs/swagger.yaml \
  -o ../../docs/swagger.html

printf 'OpenAPI and Redoc artifacts generated in %s/docs\n' "$backend_dir"
