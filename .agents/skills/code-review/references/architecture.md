# Uptime App architecture

Use this document as the architectural baseline during review. If a change intentionally introduces a new boundary, require the change to explain and test that boundary rather than treating the current layout as immutable.

## Repository boundary

- `backend/` and `frontend/` are independently runnable applications.
- The frontend is a browser client. The backend owns authentication, profiles, uploads, monitors, persistence, and HTTP policy.
- The frontend-to-backend request/response contract is an external boundary. Route registration, handler behavior, frontend API calls, and generated OpenAPI artifacts must agree.

## Go backend layers

```text
cmd/server or cmd/migrate  -> composition and process entrypoints
internal/server             -> HTTP-wide concerns such as CORS
internal/auth or monitor    -> controller -> service -> repository
internal/config             -> environment/configuration loading
internal/database           -> database connection and shared persistence setup
internal/models             -> persistence/domain data structures
migrations                  -> versioned schema changes
docs/                       -> generated OpenAPI/Redoc artifacts
```

Expected dependencies:

- Controllers parse/validate HTTP input, authenticate requests, select status codes, and translate service results into HTTP responses.
- Services contain use-case/business rules and should not depend on HTTP writers or request details.
- Repositories own SQL and persistence details; they should not decide HTTP status codes or render responses.
- `cmd/` wires dependencies and starts processes; it should not become a second business layer.
- `internal/` keeps application-only packages private.
- Route changes update Swaggo annotations and regenerate `backend/docs/` in the same change.
- Migrations are forward/backward pairs where the repository convention requires them; destructive schema changes need explicit safety reasoning.

## Next.js frontend layers

```text
src/app/                   -> App Router routes, layouts, page composition
src/features/<domain>/     -> domain UI, API functions, state, types, validation
src/features/api-client.ts -> shared HTTP transport and cross-cutting API behavior
src/features/api-error.ts  -> shared API error normalization
src/test/                  -> test setup
public/                    -> static assets
```

Expected dependencies:

- App Router pages compose feature components; domain behavior stays in the relevant `src/features/<domain>/` module.
- Feature API modules use the shared API client rather than creating incompatible HTTP clients or duplicating auth/error behavior.
- Server/client boundaries are explicit; browser-only hooks and providers do not leak into server components without the required client boundary.
- Types and validation describe the real backend contract; do not rely on unchecked `any` or duplicated, drifting payload shapes.
- Loading, empty, success, unauthorized, validation-error, server-error, and retry states are considered for user-facing flows.
- Tests are colocated with the behavior they cover and use the repository's Vitest/Testing Library setup.

## Cross-cutting invariants

- Authentication uses bearer access tokens for protected API routes and an HttpOnly refresh-token cookie where applicable; never expose refresh tokens to browser JavaScript or logs.
- CORS must allow only the configured frontend origin when credentials are enabled.
- Secrets and credentials belong in environment/configuration, not source, fixtures, generated artifacts, or committed `.env` files.
- Error responses must not leak stack traces, SQL, tokens, passwords, or sensitive personal data.
- Changes that affect one side of the API boundary require the other side and the generated contract to be checked.
