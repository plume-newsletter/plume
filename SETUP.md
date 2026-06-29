# Plume — Self-Host Setup

Plume ships as a single binary with the web UI embedded. Run it with Postgres.

## Run locally (one command)

```bash
docker compose up --build
```

App: http://localhost:8080  •  Health: http://localhost:8080/health

## Configuration (environment variables)

| Variable | Purpose |
|---|---|
| `PLUME_ADDR` | Listen address (default `:8080`) |
| `PLUME_DATABASE_URL` | PostgreSQL connection URL |
| `PLUME_ADMIN_EMAIL` | Bootstrap admin email (first run) |
| `PLUME_ADMIN_PASSWORD` | Bootstrap admin password (first run) |
| `PLUME_COOKIE_SECRET` | Secret for signing session cookies (**at least 32 bytes**) |
| `PLUME_SECRET_KEY` | Key encrypting SES credentials at rest (**exactly 32 bytes**) |

## Local development

Terminal 1 (API): `go run ./cmd/plume`
Terminal 2 (UI):  `cd web && npm install && npm run dev` (proxies /api to :8080)

Build the single binary: `make build` → `./plume`

Email sending defaults to a no-AWS log provider (added in a later phase).
