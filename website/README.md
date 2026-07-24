# RxArgo — Web

Marketing + waitlist site for the RxArgo desktop app.

- **Framework:** Next.js 16 (App Router) + React 19, run with **Bun**
- **Database:** Turso (libSQL) via **Drizzle ORM**
- **Auth:** [RxLab OAuth](https://auth.rxlab.app) via `@rxtech-lab/authjs-rxlab` (Auth.js v5)
- **UI:** shadcn/ui (Base UI) + Tailwind v4 + **Framer Motion**

## How it works

1. **Waitlist** — visitors sign in and join the waitlist (one row per RxLab user).
2. **Auth** — sign-in goes through RxLab; the session carries the user id, email, and roles.
3. **Download gate** — while a user is `pending` they see "on the waitlist"; once an
   admin marks them `approved` (out of the waiting list), the page reveals the
   macOS download link. The link is resolved at request time from the **latest
   release** of `AUTOPILOT_RELEASE_REPO` via the Autopilot release API (falling
   back to `DOWNLOAD_URL` if the lookup fails).

Admins (RxLab `admin` role, or an email in `ADMIN_EMAILS`) approve entries at `/admin`.

## Setup

```bash
bun install
cp .env.example .env.local   # then fill in the values
```

### Environment

| Variable | Purpose |
| --- | --- |
| `AUTH_ISSUER` | RxLab issuer — `https://auth.rxlab.app` |
| `AUTH_CLIENT_ID` / `AUTH_CLIENT_SECRET` | RxLab OAuth client credentials |
| `AUTH_SECRET` | Auth.js JWT secret (`openssl rand -base64 32`) |
| `AUTH_URL` | Public base URL of the deployment |
| `TURSO_DATABASE_URL` / `TURSO_AUTH_TOKEN` | Turso libSQL connection |
| `AUTOPILOT_BASE_URL` | Autopilot API base (default `https://autopilot.rxlab.app`) |
| `AUTOPILOT_RELEASE_REPO` | Repo whose latest release is offered (default `rxtech-lab/argo-trading-macOS`) |
| `AUTOPILOT_TOKEN` | Bearer token (`dpat_...`, `release:file:read` scope) for a private release listing |
| `DOWNLOAD_URL` | Fallback macOS download link if the release lookup fails |
| `ADMIN_EMAILS` | Comma-separated admin emails (optional) |

> Register the RxLab OAuth client with redirect URI
> `{AUTH_URL}/api/auth/callback/rxlab` and scopes
> `openid email profile offline_access`.

For local development you can point Turso at a file: `TURSO_DATABASE_URL=file:./local.db`.

## Database

```bash
bun run db:generate   # generate a migration from lib/db/schema.ts
bun run db:migrate    # apply migrations (drizzle/*.sql)
bun run db:studio     # browse data
```

## Develop

```bash
bun run dev           # http://localhost:3000
bun run build         # production build
bun run start
```

## Layout

```
app/
  page.tsx                     # landing + auth/waitlist/download gate (dynamic)
  admin/page.tsx               # admin approval list
  actions.ts                   # server actions (sign in/out, join, approve)
  api/auth/[...nextauth]/      # Auth.js route handler
lib/
  auth.ts                      # createRxLabAuth (lazy init)
  session.ts                   # currentUser() + isAdmin()
  waitlist.ts                  # data access (join/approve/list)
  db/                          # drizzle client + schema
components/                    # hero backdrop, panels, shadcn ui
```
