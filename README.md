# Plume

**Open-source, self-hosted newsletter platform.** Send newsletters through your own
AWS SES (or other providers) for a fraction of typical SaaS pricing — manage lists,
compose campaigns, and track opens, clicks, bounces, and complaints.

A modern, self-hosted newsletter platform built as a single Go binary with an embedded
React UI: nothing to orchestrate, just run it next to Postgres.

> **Status: pre-release, under active development.** The backend (audience, sending,
> tracking, double opt-in, suppression) and the admin UI (auth, settings, brands,
> lists, subscribers, CSV import) are in place; campaign compose/send/report screens
> are in progress. Not yet recommended for production.

## Why Plume

- **Cheap sending, your infrastructure.** Connect your AWS SES account and pay AWS
  rates (~$0.10 per 1,000 emails) instead of per-subscriber SaaS pricing.
- **Truly self-hosted.** One static binary with the web UI embedded — plus Postgres.
  No external services required.
- **Yours to extend.** A first-class hook system (WordPress-style actions + filters)
  lets you react to and modify behavior without forking.
- **Open source (AGPL-3.0).** Free to self-host forever.

## Features

- **Audience** — brands (sender identities), lists, subscribers; manual add and CSV
  import; public subscribe with double opt-in confirmation.
- **Campaigns** — compose HTML email, send to a list through a background worker with
  rate limiting.
- **Deliverability** — open/click/bounce/complaint tracking; automatic suppression of
  bounced/complained/unsubscribed addresses, enforced at send time.
- **Compliance** — one-click unsubscribe on every email.
- **Reporting** — per-campaign opens (total + unique), clicks, bounces, complaints,
  unsubscribes.
- **Extensible** — built-in render filters (tracking pixel, unsubscribe link, click
  rewrite) are themselves implemented on the public hook system.

## Quick start (self-host)

Requires Docker.

```bash
git clone https://github.com/plume-newsletter/plume.git
cd plume
docker compose up --build
```

Open http://localhost:8080 and sign in with the bootstrap admin credentials from
`docker-compose.yml` (`PLUME_ADMIN_EMAIL` / `PLUME_ADMIN_PASSWORD`). Then go to
**Settings** to connect your AWS SES credentials.

See [SETUP.md](SETUP.md) for configuration (database URL, secrets, SES, SNS webhook).

## Local development

```bash
docker compose up db -d                 # Postgres only
go run ./cmd/plume                       # API on :8080 (log email provider by default — no AWS needed)
cd web && npm install && npm run dev     # Vite dev server, proxies /api to :8080
```

Build the single binary (embeds the web UI): `make build` → `./plume`.

## Tech stack

Go (net/http + chi), PostgreSQL (sqlc + goose), AWS SES (aws-sdk-go-v2); React +
TypeScript + Vite + Tailwind + shadcn/ui. Tests use real Postgres via Testcontainers.

## Contributing

Contributions are welcome — see [CONTRIBUTING.md](CONTRIBUTING.md) and our
[Code of Conduct](CODE_OF_CONDUCT.md). To report a security issue, see
[SECURITY.md](SECURITY.md).

## License

[GNU AGPL-3.0](LICENSE). Copyright (C) 2026 Weerayut Teja and contributors.

A separately-licensed managed/hosted offering may be provided in the future; the
self-hosted edition in this repository is and remains free and open source.
