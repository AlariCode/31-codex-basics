# Code review checklist

Apply the global section to every review, then apply only the conditional sections selected from the changed paths and behavior. For each item, look for evidence rather than assuming it is correct.

## Global correctness and scope

Good signals:

- The diff has one coherent purpose and changes only the files needed for it.
- Existing behavior and error paths are preserved unless the change explicitly updates the contract.
- Boundary cases are handled: empty input, malformed input, not found, unauthorized, duplicate, timeout, and dependency failure where relevant.
- The implementation is readable from names and structure without comments that merely restate code.

Bad signals:

- A fix changes a shared helper or API contract without checking all callers.
- New behavior is reachable only through an untested happy path.
- Unrelated formatting, generated output, or refactors obscure the behavioral change.
- Errors are ignored, swallowed, logged without context, or converted to a misleading success.

## P0/P1 safety gate

Flag immediately when present:

- Private keys, cloud access keys, database URLs with credentials, bearer/refresh tokens, passwords, or real personal data are committed.
- Authentication or authorization can be bypassed, credentials are returned/logged, or refresh tokens become script-readable.
- User-controlled input reaches SQL, shell, filesystem paths, HTML, redirects, or dynamic code without the required validation/escaping/parameterization.
- A migration can destroy or silently corrupt existing data, or an API change breaks all existing clients.
- A critical endpoint can panic, loop, exhaust memory, or fail closed/open for all users.

Good signals:

- Secrets come from environment/configuration and sensitive values are redacted from errors and logs.
- Authorization is checked at the backend boundary and object ownership is verified before reads/writes.
- Inputs have explicit size, type, format, and resource limits; SQL uses parameters.

## Go backend — conditional

Good signals:

- HTTP handlers stay thin: decode/validate, authenticate, call a service/use case, map result to status/JSON.
- Services express business rules independently of `net/http`; repositories isolate SQL and scan/close resources correctly.
- Exported identifiers have useful Go doc comments, packages are lowercase, and changed files are `gofmt` formatted.
- Context cancellation/timeouts are propagated to I/O; errors are wrapped with useful context and checked with `errors.Is/As` where appropriate.
- Tests cover success plus invalid input, auth failure, not-found/conflict, dependency failure, and regression cases.

Bad signals:

- SQL, environment loading, or business rules are embedded in a controller.
- A repository returns HTTP concepts, or a service writes directly to an `http.ResponseWriter`.
- Goroutines, response bodies, rows, files, or transactions are leaked; request context is discarded.
- `panic`, `log.Fatal`, unchecked type assertions, ignored errors, or broad `interface{}`/`any` hide failures in request paths.
- A route is registered without matching Swaggo annotations, security declarations, status codes, or regenerated docs.
- A migration has no safe rollback/forward reasoning, relies on destructive defaults, or is not compatible with deployed application order.

## Next.js/TypeScript frontend — conditional

Good signals:

- App Router pages compose domain features; components have explicit client boundaries and predictable state transitions.
- Strict TypeScript types match API payloads; validation is shared with the boundary and errors are normalized through the API client.
- Requests handle cancellation, loading, unauthorized, validation, empty, retry, and server-error states as appropriate.
- Effects have correct dependencies and cleanup; event handlers do not create duplicate requests or stale updates.
- Tests cover user-visible behavior, accessibility-relevant interactions, API failures, and important branches.

Bad signals:

- Direct `axios`/`fetch` setup is duplicated in feature modules, bypassing shared auth or error handling.
- `any`, unsafe casts, non-null assertions, or client-side-only checks conceal a contract mismatch.
- Secrets or privileged credentials are placed in `NEXT_PUBLIC_*`, bundled client code, or local storage.
- Server components import browser-only APIs/hooks, or client components are added merely to avoid understanding the boundary.
- Missing keys, unstable list keys, state updates after unmount, unbounded polling, or expensive work on every render cause concrete runtime risk.
- UI treats every response as success, loses field-level validation errors, or leaves users without feedback during async work.

## API and contract — conditional

Good signals:

- Method/path, auth, request fields, response fields, status codes, error shapes, and frontend usage agree.
- Backward-compatible changes are preferred; breaking changes include migration/versioning or an explicit rollout plan.
- OpenAPI/Redoc artifacts are regenerated rather than hand-edited.

Bad signals:

- Renamed fields, nullable changes, status-code changes, or auth changes are updated on only one side.
- CORS, cookie flags, content types, upload limits, or pagination/filter semantics are changed without checking browser behavior and docs.

## Tests, coverage, naming, and dead code — conditional or global

Good signals:

- Tests assert observable behavior and failure modes, not private implementation details.
- New branches have focused coverage; tests are deterministic and do not depend on shared state or real external services unless explicitly integration-scoped.
- Names reveal intent and units (`timeout`, `userID`, `contentType`); functions have one clear responsibility.
- Removed behavior removes its callers, flags, docs, tests, and dead imports too.

Bad signals:

- Coverage is added only for the happy path, or tests merely assert that a mock was called.
- Test-only shortcuts weaken production validation or security.
- Generic names (`data`, `handle`, `process`) hide domain meaning; oversized functions mix layers.
- Unreachable branches, stale compatibility aliases, unused config, commented-out code, and duplicate helpers remain after the change.

## Review conclusion

- Confirm every finding has a priority, location, impact, and correction.
- P0/P1 findings block approval unless the user explicitly asks for an informational review.
- If checks are blocked, state the exact command and environment limitation.
- If no findings remain, say `No actionable findings` and list meaningful checks that passed.
