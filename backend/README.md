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
