---
name: code-review
description: Review the current Git diff in this mixed Go backend and Next.js frontend repository. Use when reviewing a branch, commit, staged changes, or working-tree changes; run the bundled static checks first, select backend/frontend/security checklist sections from the changed paths, and report actionable findings prioritized from P0 (critical) through P3 (low).
---

# Product Code Review

Review the actual change, its surrounding code, and its impact on the product. Treat automated checks as evidence, not as a replacement for manual reasoning. Do not report style preferences as findings unless they violate a repository rule or create a concrete risk.

## Workflow

### 1. Establish the review scope

- Read the root `AGENTS.md` and any `AGENTS.md` under each changed application.
- Identify the review target. Prefer the user-provided base or commit; otherwise use `main` when it exists. Record the current commit with `git rev-parse HEAD`.
- Inspect `git status --short`, staged and unstaged diffs, and `git diff --name-status <base>...HEAD`. Include uncommitted changes when present; do not silently review only the last commit.
- Summarize what changed before evaluating it. Follow changed symbols into callers, tests, route registration, API clients, migrations, and generated contracts as needed.
- Ignore build artifacts such as `frontend/.next/` and runtime uploads unless the diff explicitly changes their tracking or behavior.

### 2. Run automated checks before manual review

From the repository root run:

```bash
bash .codex/skills/code-review/scripts/run_static_checks.sh <base-ref>
```

Omit `<base-ref>` to use `main`. The script runs only relevant checks when Go or frontend files changed, and always scans changed text for common credentials and private keys. It reports every check and returns non-zero when a check fails. Continue the manual review after a failure, but include the failure as a finding when it affects the change.

Expected checks:

- Go: `gofmt -l`, `go vet ./...`, `go test ./...`, and `go build ./...` from `backend/`.
- Next.js: `npm run lint`, `npm run test` when available, and `npm run build` from `frontend/`.
- Security: scan the review diff for private keys, cloud access keys, passwords, tokens, and committed `.env` files without printing secret contents.

If a tool is unavailable or dependencies are not installed, mark the check as `BLOCKED` with the exact command and reason; never claim it passed.

### 3. Select the applicable checklist

Read `references/architecture.md` for the product's layer boundaries. Read `references/review-checklist.md` and apply:

- Always: scope, correctness, security/secrets, tests, naming/dead code, and maintainability.
- If any `backend/**/*.go`, migration, or backend API documentation changed: Go backend, API contract, persistence, and architecture sections.
- If any `frontend/**/*.{ts,tsx,css}` or frontend configuration changed: Next.js/frontend, API boundary, UI states, and architecture sections.
- If a route, request/response type, auth behavior, migration, environment variable, or generated OpenAPI file changed: API/security/data-compatibility sections even when the changed file is not under the obvious layer.
- If tests changed: check that they test behavior and failure paths rather than only implementation details.

For every checklist item, seek evidence in the diff and relevant surrounding code. Record `pass`, `fail`, or `not applicable`; do not turn every passing item into a verbose comment.

### 4. Report prioritized findings

Use this output shape:

1. **Scope and checks** — base, commit, changed areas, and the result of each automated check.
2. **Findings** — grouped by priority, highest first. Each finding must include:
   - `P0`–`P3` priority and a precise `path:line` location;
   - the concrete problem and why it matters to users, data, security, or maintainability;
   - a practical correction or verification step.
3. **Positive signals** — only notable evidence that the change follows the architecture or closes a meaningful risk.
4. **Residual risks / follow-ups** — blocked checks, assumptions, or out-of-scope concerns.

Priority rubric:

- `P0`: active security vulnerability or secret exposure, data loss/corruption, authentication bypass, or a critical-path outage/regression that blocks the product. Stop approval.
- `P1`: high-impact correctness, authorization, API/data incompatibility, migration failure, or broad production failure. Fix before merge.
- `P2`: normal functional defect, missing important tests/error handling, layer violation with concrete risk, or maintainability issue likely to cause bugs. Fix or explicitly accept.
- `P3`: low-risk cleanup, naming/documentation, minor duplication, or non-blocking test gap. Fix opportunistically.

Do not invent P0/P1 findings to make a review look thorough. If no actionable findings exist, say so explicitly. A failed lint/test/build command is evidence of a problem, but explain whether it is caused by the diff or is an environmental baseline failure.

## Resources

- `references/architecture.md` — fixed layer and dependency boundaries for this product.
- `references/review-checklist.md` — conditional review checklist with good and bad signals.
- `scripts/run_static_checks.sh` — changed-path-aware Go, Next.js, test, build, and secret checks.

Keep generated reports concise and actionable. The purpose of the skill is to help the team fix the highest-risk issues first, not to produce a score.
