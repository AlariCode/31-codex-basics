#!/usr/bin/env bash
set -euo pipefail

# Consume the event payload so the hook protocol can complete cleanly.
cat >/dev/null

# gofmt должен быть установлен
command -v gofmt >/dev/null 2>&1 || { echo '{}'; exit 0; }

# Используем цикл вместо mapfile: системный Bash в macOS не поддерживает mapfile.
while IFS= read -r file; do
  [ -f "$file" ] && gofmt -w "$file"
done < <(git diff --name-only --diff-filter=ACM 2>/dev/null | grep '\.go$' || true)

# PostToolUse accepts an empty JSON object when no feedback is required.
echo '{}'