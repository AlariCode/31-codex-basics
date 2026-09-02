# Backend Guidelines

## Structure

This is the `uptime-backend` Go module. The application entry point is `cmd/server/main.go`. Put packages that must not be imported externally under `internal/`; for example, application configuration belongs in `internal/config/`.

## Commands

Run these from `backend/`:

```bash
go build ./...   # compile every package
go test ./...    # run the Go test suite
gofmt -w <files> # format changed Go source files
```

## Development Data

Use the `mcp__postgres_uptime__execute_sql` MCP tool to inspect development data in the `uptime` PostgreSQL database. The tool accepts read-only SQL, so use `SELECT` queries only and limit results to the columns and rows needed for the task.

For example, to count registered users:

```sql
SELECT COUNT(*) AS user_count FROM public.users;
```

Do not expose user-level personal data, session tokens, password hashes, or database credentials in chat, logs, or committed files.

## Style and Naming

Use idiomatic Go and format all changed Go files with `gofmt`. Keep package names short and lowercase. Use PascalCase for exported identifiers and add Go doc comments to each exported type, function, and variable. Organize executables as `cmd/<service>` and application-only packages as `internal/<package>`.

## Testing

Place tests beside the code they cover in `*_test.go` files. Name tests descriptively, for example `TestLoadConfig_ReturnsDefaults`. Cover new behavior and failure cases, then run `go test ./...` before submitting.

## API Documentation

When adding or changing an HTTP route, update its swaggo annotations with the request parameters, security requirements, and response status codes. Regenerate the OpenAPI contract and Redoc HTML with `go generate ./cmd/server` from `backend/`, and include the updated files in `docs/` in the same change.

## Review Notes

Describe API, configuration, or operational changes in the pull request. When changing a contract consumed by the frontend, identify the affected route, payload, status code, or environment variable.
