# JWT-аутентификация с PostgreSQL и GORM

## Summary

Добавить регистрацию, вход и обновление access JWT через ротируемый refresh-токен. Пользователи и активные сессии хранятся в PostgreSQL, запускаемой Docker Compose. Для доступа к БД использовать GORM с PostgreSQL-драйвером; для контролируемых изменений схемы — `golang-migrate` и versioned SQL-миграции.

## Implementation Changes

- Добавить зависимости `gorm.io/gorm`, `gorm.io/driver/postgres`, `github.com/golang-migrate/migrate/v4`, JWT-библиотеку и bcrypt.
- Описать GORM-модели `User` и `RefreshSession`:
  - `User`: UUID, нормализованный уникальный email, name, хеш пароля, timestamps.
  - `RefreshSession`: UUID, user ID, хеш refresh-токена, срок действия, дата отзыва, timestamps.
- Создать versioned SQL-миграции для таблиц, уникального индекса email, внешнего ключа сессии на пользователя и индексов поиска refresh-сессии. Не выполнять `AutoMigrate` в runtime.
- Реализовать HTTP API:
  - `POST /api/v1/auth/register` — принимает `email`, `name`, `password`; создаёт пользователя, возвращает access JWT и устанавливает refresh cookie.
  - `POST /api/v1/auth/login` — проверяет email и пароль, возвращает новую пару токенов.
  - `POST /api/v1/auth/refresh` — читает refresh cookie, проверяет сессию, отзывает старый refresh-токен и выдаёт новую пару.
  - `POST /api/v1/auth/logout` — отзывает текущую refresh-сессию и очищает refresh cookie.
- Ответы содержат `access_token`, `token_type: "Bearer"`, `expires_in` и пользователя (`id`, `email`, `name`); refresh-токен в JSON не возвращается.
- Access-токен — подписанный JWT с user ID, временем выпуска, истечения и типом токена; срок действия — 1 день.
- Refresh-токен — криптографически случайное непрозрачное значение, которое хранится в БД только как хеш; срок действия — 30 дней.
- При невалидном/истёкшем refresh-токене вернуть `401` и удалить cookie; при повторной регистрации email — `409`; при неверных данных входа — `401`; при невалидном теле запроса — `400`.
- Настроить refresh-cookie: `HttpOnly`, `SameSite=Lax`, path `/api/v1/auth`; `Secure` задаётся конфигурацией.
- Добавить конфигурацию: HTTP-адрес, PostgreSQL DSN, JWT secret, TTL токенов, `CORS_ORIGIN`, cookie Secure. Credentialed CORS разрешать только для заданного origin.
- Добавить Docker Compose только для PostgreSQL с постоянным volume, команды применения миграций и переменные окружения в `backend/README.md`.

## Public Contract

- Фронтенд вызывает auth-методы с `credentials: "include"` и передаёт access JWT в `Authorization: Bearer <token>` для защищённых API.
- Refresh выполняется только cookie-механикой; успешный refresh делает прежний refresh-токен недействительным.
- Logout отзывает текущую refresh-сессию, удаляет cookie и возвращает `204`.
- В первой версии допускается несколько параллельных сессий одного пользователя.

## Test Plan

- Регистрация: успех, повторный email, невалидные поля, bcrypt-хеширование.
- Вход: корректные данные, неверный пароль, несуществующий email.
- JWT: подпись, claims, срок действия.
- Refresh: успешная ротация, запрет повторного использования старого токена, истечение/подмена, очистка cookie при `401`.
- Logout: отзыв refresh-сессии, очистка cookie и отказ refresh после выхода.
- Репозитории GORM и SQL-миграции на PostgreSQL.
- Проверки: `go test ./...` и `go build ./...`.

## Implementation Checklist

### Infrastructure and configuration

- [x] Add GORM, PostgreSQL driver, `golang-migrate`, JWT, bcrypt, and UUID dependencies to the Go module.
- [x] Add Docker Compose configuration for PostgreSQL with a persistent volume, database credentials, health check, and exposed local port.
- [x] Define application configuration for HTTP address, PostgreSQL DSN, JWT secret, token TTLs, `CORS_ORIGIN`, and cookie Secure mode.
- [x] Initialize the GORM PostgreSQL connection at service startup and close it gracefully on shutdown.
- [x] Add a migration command or executable that applies versioned migrations before the API is started.
- [x] Place each module's HTTP controller beside its domain code; auth routes are registered by `internal/auth`, while shared CORS remains in `internal/server`.

### Database schema and data access

- [x] Create SQL up/down migrations for the `users` table with UUID primary key and normalized unique email.
- [x] Create SQL up/down migrations for the `refresh_sessions` table with session ID, user foreign key, refresh-token hash, expiry, revocation timestamp, and supporting indexes.
- [x] Define the GORM `User` and `RefreshSession` models to match the database schema.
- [x] Implement the user repository: create user and find user by normalized email.
- [x] Implement the refresh-session repository: create session, find an active session by token hash, and atomically revoke the previous session during rotation.

### Authentication domain logic

- [x] Validate and normalize registration input: email, display name, and password.
- [x] Hash passwords with bcrypt and verify password hashes during login.
- [x] Generate signed access JWTs with user ID, issue time, expiry, and token type; set their lifetime to one day.
- [x] Generate cryptographically random opaque refresh tokens, store only their hash, and set their lifetime to 30 days.
- [x] Implement registration to persist a user and issue the initial token pair.
- [x] Implement login to verify credentials and issue a new token pair.
- [x] Implement refresh-token validation and rotation so a used refresh token cannot be reused.

### HTTP API and browser security

- [x] Add routing and JSON request/response handling for `POST /api/v1/auth/register`.
- [x] Add routing and JSON request/response handling for `POST /api/v1/auth/login`.
- [x] Add routing and cookie-based handling for `POST /api/v1/auth/refresh`.
- [x] Add `POST /api/v1/auth/logout` to revoke the active refresh session and clear its cookie.
- [x] Return the documented response fields and status codes, including `400`, `401`, and `409` cases.
- [x] Set and clear the refresh cookie with `HttpOnly`, `SameSite=Lax`, `/api/v1/auth` path, configured Secure flag, and appropriate expiry.
- [x] Add credentialed CORS middleware restricted to the configured `CORS_ORIGIN`.
- [x] Ensure errors never disclose password hashes, JWT secrets, or refresh-token values.

### Tests, documentation, and verification

- [x] Add unit tests for input validation, password hashing, and access-JWT claims and expiry.
- [x] Add authentication service tests for registration, successful and failed login, refresh rotation, and rejected reused/expired refresh tokens.
- [x] Add repository integration tests against PostgreSQL for user uniqueness and refresh-session rotation.
- [x] Add HTTP handler tests for successful responses, documented error statuses, CORS, and refresh-cookie attributes.
- [x] Verify logout rejects the revoked refresh token and returns the browser to `/login` from the dashboard profile menu.
- [x] Document local setup, environment variables, PostgreSQL Docker commands, migration commands, and API contract in `backend/README.md`.
- [x] Run `go test ./...` from `backend/` and resolve failures.
- [x] Run `go build ./...` from `backend/` and resolve failures.

## Assumptions

- Учётная запись состоит из email, отображаемого имени и пароля.
- Отзыв всех сессий, подтверждение email и восстановление пароля остаются за пределами первой версии.
