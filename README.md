# LedgerForge

A double-entry accounting ledger for fintech, split the way real payment
companies split it: a Go core that owns the money math, and a NestJS gateway
that owns the API surface.

Every transfer is a balanced journal entry. Debits always equal credits, per
currency. Entries are idempotent, balances are materialized, and money is
stored as integer minor units so rounding bugs can't happen.

## Architecture

```
                ┌─────────────────┐
                │   api-gateway   │  NestJS: JWT auth, validation,
                │   (NestJS)      │  rate limiting, Swagger, live
                └────────┬────────┘  WebSocket feed
                         │ HTTP
                ┌────────▼────────┐
                │  ledger-core    │  Go: double-entry validation,
                │     (Go)        │  idempotent posting, balances
                └────────┬────────┘
                         │ pgx
                ┌────────▼────────┐
                │    Postgres     │  journal, lines, materialized
                │                 │  balances; migrations on boot
                └─────────────────┘
```

The gateway never invents ledger semantics. It validates input, forwards to
the core, passes core errors through untouched, and broadcasts posted entries
over WebSockets. All accounting rules live in Go.

## What's inside

**ledger-core (Go)**
- Journal entries with strict double-entry validation: at least two lines,
  one-sided legs only, debits == credits per currency
- Idempotent posting via client-supplied keys; replays return the original
  entry and never double-post
- Materialized balances per account and currency, updated in the same
  transaction as the entry
- Money as int64 minor units (paise/cents); 21 supported currencies
- Postgres store (pgx, migrations embedded in the binary) and an in-memory
  store for tests and local dev
- `POST /v1/accounts`, `POST /v1/entries`, `GET /v1/accounts/{id}/balance`,
  `GET /v1/entries`, `GET /healthz`

**api-gateway (NestJS)**
- JWT auth (`POST /api/auth/login`; demo operator `admin` / `changeme`)
- `POST /api/transfers` builds a balanced two-line entry from debit account,
  credit account, currency, and amount; auto-generates an idempotency key when
  you don't supply one
- `POST /api/accounts`, `GET /api/accounts/{id}/balance`,
  `GET /api/journal`
- Request validation (class-validator), 100 req/min rate limiting,
  Swagger UI at `/docs`
- WebSocket feed on the `events` namespace: every posted entry broadcasts as
  `entry.posted` (replays are not rebroadcast)

## Run it

With Docker:

```bash
cd deploy
docker compose up --build
```

Without Docker (needs Go 1.24+ and Node 24+):

```bash
# terminal 1: ledger core (in-memory store)
cd ledger-core && go run ./cmd/ledger-core

# terminal 2: gateway
cd api-gateway && npm install && npm run start
```

Or run `./demo.sh` for a scripted end-to-end: login, create accounts, post a
transfer, replay it to show idempotency, and read the balance back.

Then open http://localhost:3000/docs for the interactive API.

## Verification

- Go: `cd ledger-core && go test ./...` (validation tables, service tests,
  HTTP handler tests) plus `go vet`
- Benchmark: `go test ./internal/ledger/ -bench=. -run=^$`
- Gateway: `cd api-gateway && npx vitest run` (service and auth unit tests)

## Configuration

| Variable                   | Default                 | Purpose                          |
| -------------------------- | ----------------------- | -------------------------------- |
| `LEDGER_DSN`               | (in-memory)             | Postgres connection string       |
| `LEDGER_ADDR`              | `:8081`                 | Core listen address              |
| `PORT`                     | `3000`                  | Gateway listen port              |
| `LEDGER_CORE_URL`          | `http://localhost:8081` | Where the gateway finds the core |
| `JWT_SECRET`               | `dev-secret-change-me`  | Token signing secret             |
| `LEDGER_ADMIN_USER`        | `admin`                 | Demo operator username           |
| `LEDGER_ADMIN_PASSWORD_HASH` | (hash of `changeme`)  | bcrypt hash for the demo login   |

## Honest limits

- The demo operator is a single env-configured account, not a users table.
- Rate limiting is per process; run more than one gateway and you need Redis.
- The WebSocket feed is fire-and-forget; clients that disconnect miss entries
  and should backfill from `/api/journal`.
- `LEDGER_DSN` unset means the in-memory store: fast, but everything is lost
  on restart. Don't run money through it.
- The Postgres store compiles and its SQL was reviewed, but it hasn't run
  against a live database in this environment. `docker compose up` is the
  way to exercise it before trusting it.
