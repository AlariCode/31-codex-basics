---
name: sync-backend-openapi
description: Keep a Go backend's HTTP routes and Swaggo OpenAPI contract synchronized. Use when an API route is created, edited, renamed, or deleted, when request or response behavior changes, or when backend Swagger/OpenAPI documentation must be regenerated and verified.
---

# Sync Backend OpenAPI

Update the backend's Swaggo annotations and generated OpenAPI artifacts whenever an API route changes.

## Workflow

1. Read the repository and backend `AGENTS.md` instructions before editing.
2. Inspect the route registration and handler implementation. Treat the registered method/path as the source of truth for whether an endpoint exists; include file-serving routes only when they are intended as public API operations.
3. Find the handler's existing Swaggo block. Add or update annotations next to the handler, keeping the block accurate for:
   - `@Summary`, `@Description`, and `@Tags`;
   - `@Accept` and `@Produce`;
   - every path, query, header, cookie, body, or multipart parameter;
   - `@Security` for protected endpoints;
   - all meaningful success and failure status codes with response schemas;
   - an exact `@Router /path [method]` matching the registered route.
4. Update request/response example types or exported documentation models when the JSON or multipart contract changed. Use the actual Go JSON tags and handler behavior; do not document fields that are not returned or accepted.
5. For a deleted route, remove its operation from annotations and confirm the generated contract no longer contains its path. Do not leave stale generated operations behind.
6. Run the bundled synchronization script. With the default project layout, invoke it from the repository root:

   ```bash
   bash .codex/skills/sync-backend-openapi/scripts/sync_openapi.sh
   ```

   Pass a backend directory as the first argument when working from a different checkout. The script regenerates `backend/docs/docs.go`, `backend/docs/swagger.json`, and `backend/docs/swagger.yaml` with the pinned Swaggo version, then explicitly builds `backend/docs/swagger.html` with the pinned Redoc CLI version.
7. Inspect the generated JSON/YAML for the changed path, method, parameters, security, response codes, and schemas. Check that the deleted path is absent.
8. Run targeted tests when available, then `go test ./...` and `go build ./...` from `backend/`. If generation or tests cannot run because a dependency or external service is unavailable, report the exact command and blocker.
9. Review the diff. Keep generated documentation changes with the route change, avoid unrelated formatting churn, and do not manually edit generated files unless generation itself is unavailable and the user explicitly accepts that fallback.

## Project-specific conventions

- The backend module is `backend/`; its OpenAPI output is in `backend/docs/`.
- The specification entry point and generation directives are in `backend/cmd/server/main.go`.
- Routes are registered with Go 1.22+ `http.ServeMux` patterns such as `POST /api/v1/auth/login`.
- Authentication uses `Authorization: Bearer <access_token>` for protected API routes; refresh-token behavior uses the `refresh_token` cookie and should be described in `@Description` when relevant.
- JSON request and response types may be unexported. Swaggo can document them; preserve the repository's existing naming and style.
- Use Swaggo syntax compatible with the pinned `github.com/swaggo/swag` version in `backend/go.mod`.
- Prefer `scripts/sync_openapi.sh` for generation so the JSON/YAML contract and Redoc HTML are produced together.

## Annotation checklist

Before finishing, verify:

- Every intended API route has exactly one matching operation in the generated contract.
- Every `@Router` method is lowercase and matches the registered HTTP method.
- Protected operations declare `@Security BearerAuth`.
- Multipart file parameters use `@Accept multipart/form-data` and `@Param <name> formData file true "..."`.
- Body schemas, response schemas, status codes, and error payloads reflect handler behavior.
- Generated docs are included in the diff and no stale operation remains after route deletion.
