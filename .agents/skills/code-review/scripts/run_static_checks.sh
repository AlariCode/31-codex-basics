#!/usr/bin/env bash
set -u

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
repository_root="$(cd -- "$script_dir/../../../.." && pwd)"
base_ref="${1:-main}"
status=0
changed_file_list="$(mktemp)"
trap 'rm -f "$changed_file_list"' EXIT

printf 'Review base: %s\n' "$base_ref"
printf 'Review HEAD: '
git -C "$repository_root" rev-parse --short HEAD

if git -C "$repository_root" rev-parse --verify --quiet "$base_ref" >/dev/null; then
  git -C "$repository_root" diff --name-only "$base_ref"...HEAD >>"$changed_file_list"
fi
git -C "$repository_root" diff --name-only >>"$changed_file_list"
git -C "$repository_root" diff --cached --name-only >>"$changed_file_list"
git -C "$repository_root" ls-files --others --exclude-standard >>"$changed_file_list"
sort -u "$changed_file_list" -o "$changed_file_list"

has_changed() { rg -q "^$1" "$changed_file_list"; }

run_check() {
  local label="$1"
  shift
  printf '\n[%s] %s\n' "$label" "$*"
  if "$@"; then
    printf '[PASS] %s\n' "$label"
  else
    printf '[FAIL] %s\n' "$label"
    status=1
  fi
}

if has_changed 'backend/'; then
  go_files="$(rg '^backend/.*\.go$' "$changed_file_list" | sed 's#^backend/##' || true)"
  while IFS= read -r file; do
    [[ -z "$file" ]] && continue
    if [[ -n "$(gofmt -l "$repository_root/backend/$file")" ]]; then
      printf '[FAIL] gofmt: %s\n' "$file"
      status=1
    fi
  done <<<"$go_files"
  run_check 'go vet' bash -lc "cd \"$repository_root/backend\" && go vet ./..."
  run_check 'go test' bash -lc "cd \"$repository_root/backend\" && go test ./..."
  run_check 'go build' bash -lc "cd \"$repository_root/backend\" && go build ./..."
else
  printf '\n[SKIP] Go checks: no changed backend files.\n'
fi

if has_changed 'frontend/'; then
  run_check 'frontend lint' bash -lc "cd \"$repository_root/frontend\" && npm run lint"
  if node -e "const p=require('./frontend/package.json'); process.exit(p.scripts?.test ? 0 : 1)" 2>/dev/null; then
    run_check 'frontend tests' bash -lc "cd \"$repository_root/frontend\" && npm run test"
  else
    printf '\n[SKIP] frontend tests: package.json has no test script.\n'
  fi
  run_check 'frontend build' bash -lc "cd \"$repository_root/frontend\" && npm run build"
else
  printf '\n[SKIP] Frontend checks: no changed frontend files.\n'
fi

printf '\n[Security] scanning changed text for credentials and private keys\n'
security_diff="$(mktemp)"
trap 'rm -f "$changed_file_list" "$security_diff"' EXIT
{
  if git -C "$repository_root" rev-parse --verify --quiet "$base_ref" >/dev/null; then
    git -C "$repository_root" diff --unified=0 "$base_ref"...HEAD -- . ':(exclude).codex/skills/code-review/**'
  fi
  git -C "$repository_root" diff --cached --unified=0 -- . ':(exclude).codex/skills/code-review/**'
  git -C "$repository_root" diff --unified=0 -- . ':(exclude).codex/skills/code-review/**'
  while IFS= read -r file; do
    [[ -z "$file" || ! -f "$repository_root/$file" ]] && continue
    [[ "$file" == .codex/skills/code-review/* ]] && continue
    git -C "$repository_root" diff --no-index --unified=0 /dev/null "$file" || true
  done < <(git -C "$repository_root" ls-files --others --exclude-standard)
} >"$security_diff"

if rg -n -i \
  -e 'BEGIN (RSA|OPENSSH|EC|DSA) PRIVATE KEY' \
  -e 'AKIA[0-9A-Z]{16}' \
  -e '(ghp_|github_pat_|xox[baprs]-|sk-[A-Za-z0-9])' \
  -e "(password|passwd|secret|api[_-]?key|access[_-]?token)[[:space:]]*[:=][[:space:]]*'[^']{16,}'" \
  -e '(password|passwd|secret|api[_-]?key|access[_-]?token)[[:space:]]*[:=][[:space:]]*"[^"]{16,}"' \
  -e '(password|passwd|secret|api[_-]?key|access[_-]?token)[[:space:]]*[:=][[:space:]]*[A-Za-z0-9+/=_-]{16,}' \
  "$security_diff" >/dev/null; then
  printf '[FAIL] possible credentials found in added lines (contents suppressed).\n'
  status=1
else
  printf '[PASS] no common credential patterns found in changed files.\n'
fi

printf '\nStatic check exit status: %s\n' "$status"
exit "$status"
