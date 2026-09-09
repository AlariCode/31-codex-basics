# Uptime backend

Go API with registration, password login, access JWTs, and rotatable refresh-token sessions.

## Local setup

Run PostgreSQL from this directory:

```bash
docker compose up -d
```

Copy these values into your shell (replace `JWT_SECRET` with a random value of at least 32 characters outside local development):

```bash
export DATABASE_URL='postgres://uptime:uptime@localhost:5436/uptime?sslmode=disable'
export JWT_SECRET='change-this-to-a-long-random-local-development-secret'
export CORS_ORIGIN='http://localhost:3005'
export COOKIE_SECURE=false
```

Apply schema migrations, then start the API:

```bash
go run ./cmd/migrate
go run ./cmd/server
```

The service listens on `:8080` by default. Run `go test ./...` and `go build ./...` before submitting changes. To run PostgreSQL repository integration tests after migrations, set `TEST_DATABASE_URL` to the same local connection string.

## API documentation

Regenerate the Swagger contract and Redoc HTML from the backend directory:

```bash
go generate ./cmd/server
```

The generated files are `docs/swagger.json`, `docs/swagger.yaml`, and `docs/swagger.html`.

## Configuration

| Variable | Required | Default | Description |
| --- | --- | --- | --- |
| `DATABASE_URL` | Yes | — | PostgreSQL connection URL. The bundled Compose service uses port `5436` by default. |
| `JWT_SECRET` | Yes | — | HMAC signing secret; at least 32 characters. |
| `HTTP_ADDR` | No | `:8080` | Address for the API server. |
| `ACCESS_TOKEN_TTL` | No | `24h` | Access JWT lifetime. |
| `REFRESH_TOKEN_TTL` | No | `720h` | Refresh-session and cookie lifetime. |
| `CORS_ORIGIN` | No | `http://localhost:3005` | Allowed browser origin. |
| `COOKIE_SECURE` | No | `false` | Set to `true` behind HTTPS in production. |
| `AVATAR_DIR` | No | `uploads/avatars` | Local directory used to store uploaded files; avatars and generic files use subdirectories. |
| `FAVICON_DIR` | No | `uploads/favicons` | Local directory used to cache downloaded PNG favicons for monitored sites. |
| `POSTGRES_PORT` | No | `5436` | Host port mapped by `docker compose` to the bundled PostgreSQL container. |

## Authentication API

All endpoints use JSON and are prefixed with `/api/v1/auth`.

- `POST /register` accepts `{ "email", "name", "password" }`, creates an account, returns `201`, and sets the refresh cookie.
- `POST /login` accepts `{ "email", "password" }`, returns `200`, and sets a new refresh cookie.
- `POST /refresh` requires the `refresh_token` cookie, rotates it, and returns `200` with a new access JWT.
- `POST /logout` revokes the current refresh session, clears its cookie, and returns `204`.

Profile routes require `Authorization: Bearer <access_token>`:

- `GET /api/v1/profile` returns the current user's `id`, `email`, and `name`.
- `PATCH /api/v1/profile` accepts `{ "name": "..." }`, updates the display name, and returns the updated profile.
- `POST /api/v1/profile/avatar` accepts a multipart field named `avatar` containing a JPEG or PNG up to 5 MiB, stores it locally, and returns the updated profile.
- `POST /api/v1/upload` (also `/api/v1/uploads`) accepts a multipart field named `file` containing any file up to 20 MiB and returns its URL and metadata.

Profile responses include `avatar_url` (or `null`). The `/uploads/avatars/` and `/uploads/files/` paths serve stored files.

Successful responses contain `access_token`, `token_type` (`Bearer`), `expires_in`, and `user`. Browser callers must use `credentials: "include"` so the HttpOnly refresh cookie is sent. Use `Authorization: Bearer <access_token>` for future protected API endpoints.

Registration validation errors return `400`, duplicate email returns `409`, and invalid credentials or refresh tokens return `401`. A failed refresh clears the refresh cookie.

## URL monitoring

Run exactly **one backend instance**: the in-process scheduler owns all schedules.
It restores saved monitors at startup and checks immediately after create/update,
independently of browser sessions. It performs GET requests with a 10-second timeout,
a shared client and at most 20 concurrent checks. Only an immediate HTTP 200 succeeds;
redirects are not followed. The response body is closed without storing it.

The existing interval range (1 second through 7 days) is unchanged. Intervals are
measured between starts; slow checks skip elapsed starts, and a saturated worker pool
may delay checks. A monitor never has overlapping checks or a backlog of missed runs.
Edits cancel obsolete checks; configuration versions fence their database writes.
Shutdown cancellation is not a failed site check.

Only public HTTP/HTTPS destinations are allowed. Explicit local addresses are rejected
by create/update. DNS answers are validated at connection time, and connections use
the validated IP without a second lookup. The same restriction applies to favicon
requests and their redirects; environment proxies are disabled. Existing private URLs
remain visible with `last_status=blocked` and contribute no availability observations.
A DNS failure, timeout or TLS error for an otherwise allowed URL counts as a failure.

Migration `000006_add_monitor_results` adds current observation fields and
`monitor_minutes`. Apply it using `go run ./cmd/migrate` **before** starting the new
backend. No existing monitor configuration or history is removed on upgrade.
Each completed result updates the last observation and one UTC minute's counters in
a single transaction. No individual request history is retained. Changing a URL
preserves the point's history but resets its current status; changing the interval
preserves both. Deleting a monitor cascades to its history.

History reads are limited to 30 days. Cleanup runs on startup and hourly in batches
of 5,000 rows; physical retention can lag by an hour plus the current minute.
Ratios represent successful checks, not measured time without outages. Missing data
is null, not 0% or 100%. A failed database write is logged without blind retries and
leaves a gap rather than a fabricated observation.

### Monitoring API

Both endpoints require the existing Bearer authentication and return only the user's data.

- `GET /api/v1/monitors` now includes `last_checked_at`, `last_status`
  (`pending`, `up`, `down`, `blocked`), nullable `last_http_status`, and `last_error`
  (`timeout`, `dns`, `tls`, `network`, `forbidden_address`, or empty).
  Listing monitors never fetches favicons. Creation/update still refreshes favicons.
- `GET /api/v1/monitors/stats?period=24h` returns `from`, `to`, `bucket_seconds` and
  `monitors`. Each monitor has its ID, success/failure totals, nullable `uptime_percent`
  and `points` with clipped UTC `start`/`end`, counters and nullable percentage.
  Supported periods/buckets: `1h`/1 minute, `24h`/15 minutes, `7d`/1 hour, `30d`/6 hours.
  The default is `24h`; unsupported periods return 400. The partially expired oldest
  minute is excluded because individual observation timestamps are not stored.

Integration tests must use an **isolated**, migrated database:

```bash
TEST_DATABASE_URL='<isolated test database URL>' go test -race ./internal/monitor ./internal/publichttp
```

For higher scale or multiple backend replicas, first replace in-process schedule
ownership with durable job claims/leases. This version introduces no queue service
or monitoring configuration environment variables.
