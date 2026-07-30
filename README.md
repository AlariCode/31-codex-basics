# Uptime

## Local development

Start PostgreSQL, apply backend migrations, and run the Go API together with the Next.js frontend:

```bash
./scripts/dev.sh
```

The frontend is available at `http://localhost:3005` and the backend at `http://localhost:8080`.

The script accepts environment overrides when needed:

```bash
FRONTEND_PORT=3001 NEXT_PUBLIC_API_URL=http://localhost:8080 ./scripts/dev.sh
```

It leaves the PostgreSQL Docker container running after `Ctrl+C` so local data remains available. Stop it explicitly with:

```bash
cd backend && docker compose down
```
