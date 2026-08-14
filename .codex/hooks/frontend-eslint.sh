#!/usr/bin/env bash
set -euo pipefail

# Consume the event payload so the hook protocol can complete cleanly.
cat >/dev/null

project_root=$(cd "$(dirname "$0")/../.." && pwd)
cd "$project_root"

files=()

collect_files() {
  while IFS= read -r -d '' file; do
    case "$file" in
      frontend/*.js|frontend/*.cjs|frontend/*.mjs|frontend/*.jsx|frontend/*.ts|frontend/*.cts|frontend/*.mts|frontend/*.tsx)
        files+=("${file#frontend/}")
        ;;
    esac
  done
}

collect_files < <(git diff --name-only -z --diff-filter=ACMR -- frontend)
collect_files < <(git diff --cached --name-only -z --diff-filter=ACMR -- frontend)
collect_files < <(git ls-files --others --exclude-standard -z -- frontend)

[ ${#files[@]} -gt 0 ] || { echo '{}'; exit 0; }

if ! (
  cd frontend
  ./node_modules/.bin/eslint "${files[@]}"
) >&2; then
  echo '{}'
  exit 1
fi

# PostToolUse accepts an empty JSON object when no feedback is required.
echo '{}'
