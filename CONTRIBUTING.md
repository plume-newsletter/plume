# Contributing to Plume

Thanks for your interest in improving Plume! This guide covers how to get set up,
the conventions we follow, and how to submit changes.

By participating you agree to our [Code of Conduct](CODE_OF_CONDUCT.md).

## Getting set up

Requirements: Go 1.26+, Node 20+ (24 recommended), Docker (for Postgres and for the
integration tests).

```bash
git clone https://github.com/plume-newsletter/plume.git
cd plume
docker compose up db -d                  # Postgres
go run ./cmd/plume                        # API on :8080 (log email provider by default)
cd web && npm install && npm run dev      # web UI dev server (proxies /api)
```

See [SETUP.md](SETUP.md) for full configuration.

## Project layout

```
cmd/plume/        binary entrypoint + wiring
internal/         backend: domain services, store (sqlc + goose), email providers, hooks, httpapi
web/              React + TypeScript admin UI (embedded into the binary at build)
```

## Running the checks (do this before every PR)

Backend:

```bash
gofmt -l .        # must print nothing
go vet ./...
go build ./...
go test ./...      # uses real Postgres via Testcontainers — Docker must be running
```

Frontend:

```bash
cd web
npm run build
npm test
npx oxlint
```

## Conventions

- **Database:** SQL lives in `internal/store/queries/*.sql`; run `sqlc generate` after
  editing. Never hand-edit generated code in `internal/store/gen/`. Migrations are
  additive goose files in `internal/store/migrations/`.
- **Tests:** prefer real dependencies (Testcontainers Postgres) over mocks; assert
  behavior, not implementation. Frontend uses Vitest + Testing Library + MSW.
- **API shape:** owner-scoped at the SQL layer; the owner comes from the session,
  never the request body.
- **Commits:** conventional-style messages (`feat:`, `fix:`, `docs:`, `chore:`…).

## Developer Certificate of Origin (DCO)

We use the [DCO](https://developercertificate.org/): sign off your commits to certify
you wrote the code (or have the right to submit it) and license it under the project's
AGPL-3.0.

```bash
git commit -s -m "feat: ..."
```

This adds a `Signed-off-by: Your Name <you@example.com>` line. Configure your name and
email with `git config user.name` / `git config user.email`.

## Submitting a pull request

1. Fork and create a topic branch off `main`.
2. Make focused changes with tests; keep the checks above green.
3. Open a PR describing **what** and **why**. Link any related issue.
4. A maintainer will review. Be ready for a round or two of feedback.

For anything large or design-changing, please open an issue to discuss first.
