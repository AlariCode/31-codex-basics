#!/bin/sh

set -eu

payload=$(cat)
patch=$(printf '%s' "$payload" | jq -r '.tool_input.patch? // empty')
command=$(printf '%s' "$payload" | jq -r '.tool_input.command? // empty')
repo_root=$(git rev-parse --show-toplevel)

deny_migration_edit() {
  printf '%s\n' '{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny","permissionDecisionReason":"Committed migrations in backend/migrations are immutable. Create a new migration instead."}}'
  exit 0
}

# Shell commands have no structured list of changed files. Check each tracked
# migration against paths that are valid after changing into backend directories.
git -C "$repo_root" ls-files -- backend/migrations |
  while IFS= read -r path; do
    backend_path=${path#backend/}
    migration_name=${path#backend/migrations/}

    case "$patch" in
      *"$path"*) deny_migration_edit ;;
    esac

    case "$command" in
      *"$path"*|*"$backend_path"*) deny_migration_edit ;;
      *"backend/migrations"*"$migration_name"*) deny_migration_edit ;;
      *"cd backend"*"migrations"*"$migration_name"*) deny_migration_edit ;;
    esac
  done
