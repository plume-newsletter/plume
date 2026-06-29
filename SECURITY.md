# Security Policy

Plume sends email on your behalf and stores provider credentials, so we take security
seriously. Thank you for helping keep it and its users safe.

## Reporting a vulnerability

**Please do not open a public issue for security vulnerabilities.**

Report privately via either:

- **GitHub Security Advisories** — the preferred channel: open a report at
  **Security → Report a vulnerability** on the repository
  (https://github.com/plume-newsletter/plume/security/advisories/new), or
- **Email** — teja.weerayut@gmail.com.

Please include: a description of the issue, steps to reproduce (a proof of concept if
possible), the affected version/commit, and the potential impact.

## What to expect

- We aim to acknowledge a report within **3 business days**.
- We will investigate, keep you updated on progress, and coordinate a fix and
  disclosure timeline with you.
- With your permission, we will credit you when the fix is published.

## Scope

Issues of particular interest: authentication/session handling, credential storage and
encryption, owner-scoping/access-control gaps, the public tracking and
subscribe/unsubscribe endpoints, and SSRF/injection in any user-supplied content.

## Supported versions

Plume is pre-release; security fixes are applied to the latest `main`. A supported-
versions policy will accompany the first tagged release.
