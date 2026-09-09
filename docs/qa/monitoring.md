# Monitoring verification — 2026-09-08

Implementation branch: `feat/opros-monitoringa`.

## Automated checks

- Backend: `go test ./...` and `go build ./...` passed.
- PostgreSQL integration and concurrency: `TEST_DATABASE_URL=<isolated migrated database> go test -race ./internal/monitor ./internal/publichttp` passed.
- Frontend: `npm test` — 25 tests passed; `npm run lint` — no errors, two pre-existing image warnings in auth components; `npm run build` passed.
- OpenAPI/Redoc regenerated with `.agents/skills/sync-backend-openapi/scripts/sync_openapi.sh`; verified stats authentication, period enum, response codes and latest-result fields.
- `git diff --check` passed.

The database tests cover concurrent increments, owner isolation, obsolete-result fencing,
URL edits preserving history, blocked destinations contributing no counters, atomic
rollback on a failed aggregate write, retention and cascading deletion. HTTP tests
cover status codes/redirects, timeout, TLS failure, public address validation, DNS
answer changes and closing response bodies without reading them. Scheduler tests
cover startup, concurrency, skipped starts, edits, deletion and graceful cancellation.

## Migrations

Used separate local databases `uptime_monitor_test` and `uptime_monitor_upgrade_test`.
The first received all migrations from an empty database. The second received migrations
1–5, a fixture user and a legacy localhost monitor, then migration 6. Existing interval
and URL remained intact; new fields initialized to version 1 / pending. Re-running the
migration succeeded without changes; schema version was 6 with dirty=false.
The main development database was not changed.

## Playwright MCP

Verified authenticated `/` at `http://localhost:3006`, connected to an isolated backend
at `http://localhost:8081`. The frontend copy contained the repository's implementation;
separate ports avoided restarting existing development services.

- Registered a QA account and rejected creation of a localhost monitor (400).
- Created a public URL with a 5-second interval; the first and subsequent results showed HTTP 200 / «Работает».
- Edited the same monitor to a URL returning 404; status changed to «Недоступен», while successful observations remained in history.
- Created a second point with a one-hour interval; its immediate first check succeeded.
- Seeded a legacy private URL in the isolated DB; it showed «Адрес запрещён» and no availability observations.
- Checked all four periods. Hourly detail showed minute counts; 7 days returned 169 clipped hourly buckets, 30 days 121 clipped six-hour buckets.
- Stopped the QA backend: the UI retained previously loaded data, reported refresh failure and marked the 5-second monitor stale.
- Restarted the backend: existing monitors were checked automatically, including the one-hour point; the UI recovered without manual recreation.
- Deleted the legacy point through its confirmation dialog; it disappeared from the list.
- Focused a graph bar: the visible tooltip contained times, percentage and success/failure counts.
- Inspected desktop (1440×1000) and mobile (390×844) screenshots. Cards and controls are legible, with no horizontal overflow or content collisions.
- On a fresh page load after recovery there were no console errors. Expected 400/401 responses and connection errors occurred only in validation/authentication and the deliberate outage scenario. Existing font preload warnings remain.

## Screenshots

- [Desktop](monitoring-desktop.png)
- [Mobile](monitoring-mobile.png)

## Local rollout

Apply `go run ./cmd/migrate` from `backend/` with the development `DATABASE_URL`, then
restart the backend with its existing authentication/environment configuration.
Run one backend instance; multiple replicas require distributed schedule ownership.
The existing service on port 8080 was not restarted by this verification.
