# Warehouse Server

A single-binary warehouse backend written in Go, with SQLite storage.
It tracks vendors, agency-global items, warehouses with flat storage
locations, and stock keyed by `(vendor_item_id, warehouse_id,
location_id)` — every change flowing through an audited movement log
(`receive` / `remove` / `transfer` / `adjust`) with a `planned →
executed | cancelled` lifecycle. Built for small sales agencies that
need stock truth without an ERP.

## Features

- Single binary, single SQLite file: no Postgres, no Redis, no ORM,
  no framework — stdlib `net/http` plus hand-written SQL
- Token auth with staggered permissions:
  `read` < `read-movements` < `write` < `admin`
- First-admin bootstrap: unauthenticated `POST /api/users` mints the
  initial admin key exactly while no active admin exists (and reopens
  if every admin gets disabled — no lockout)
- Atomic stock operations: transfers move both sides in one
  transaction; concurrent `remove`s can never oversell
- Planned movements: stage a future receipt or move with `planned_at`,
  `execute` it later or `cancel` it — planners never touch live stock
- Absolute-count `adjust` with a mandatory reason note, so recounts
  stay auditable
- Online backups via `VACUUM INTO` — a consistent snapshot with no
  downtime and no `cp`-of-a-live-DB footgun
- Small by design: ~4,300 lines of Go, one file per domain

Explicitly not an ERP: no invoicing, no payments, no purchasing, no
shipping carriers, no frontend. See `PLAN.md` for the full design.

## Prerequisites

This project uses [mise](https://mise.jdx.dev/) to pin the Go
toolchain (see `mise.toml`). The SQLite driver is pure Go
(`modernc.org/sqlite`), so there is no CGO and nothing else to
install — no compiler toolchain beyond Go, no system packages.

```sh
curl https://mise.run | sh
mise install
```

## Getting started

```sh
# Build the binary
mise run build

# Run it (migrates, then serves on :8080)
./dist/warehouse-server

# Or straight from source
mise run dev
```

Bootstrap the first admin (the plaintext key is returned exactly once —
save it) and take it for a spin:

```sh
KEY=$(curl -s -X POST localhost:8080/api/users \
  -d '{"username":"admin"}' | python3 -c "import json,sys;print(json.load(sys.stdin)['api_key'])")

AUTH="Authorization: Bearer $KEY"
curl -s -H "$AUTH" -X POST localhost:8080/api/vendors \
  -d '{"name":"Acme","code":"ACME"}'
curl -s -H "$AUTH" -X POST localhost:8080/api/items \
  -d '{"sku":"W-1","name":"Widget"}'
curl -s -H "$AUTH" -X POST localhost:8080/api/vendor-items \
  -d '{"vendor_id":1,"item_id":1,"vendor_sku":"BW-123"}'
curl -s -H "$AUTH" -X POST localhost:8080/api/warehouses \
  -d '{"name":"Berlin","code":"WH-BER"}'
curl -s -H "$AUTH" -X POST localhost:8080/api/warehouses/1/locations \
  -d '{"code":"A-01","name":"Shelf 1"}'

curl -s -H "$AUTH" -X POST localhost:8080/api/stock/receive \
  -d '{"vendor_item_id":1,"warehouse_id":1,"location_id":1,"quantity":100}'
curl -s -H "$AUTH" localhost:8080/api/stock
curl -s -H "$AUTH" localhost:8080/api/stock/movements
```

See `mise.toml` for the complete tool configuration and available tasks.

## Configuration

Everything is env vars with working defaults, so the bare binary just
runs. A set env var wins over the matching CLI flag.

| Variable | Flag | Default |
|---|---|---|
| `WAREHOUSE_LISTEN` | `-listen` | `:8080` |
| `WAREHOUSE_DATABASE` | `-db` | `./warehouse.db` |
| `WAREHOUSE_LOG_LEVEL` | – | `info` |

## CLI

```text
serve             migrate, then serve (default when omitted)
migrate           apply pending migrations and exit
version           print the build version and exit
backup <dest.db>  write a consistent snapshot and exit
```

```sh
./dist/warehouse-server backup /backups/warehouse-$(date +%F).db
```

## Verifying

```sh
mise run test-race   # full suite under the race detector
mise run vet         # go vet
```

The suite leans on a real (temporary) SQLite database — no mocks.
Highlights: a 10-goroutine concurrent-`remove` test proving no
oversell (exactly 5 of 10 succeed on 50 units), the planned →
execute/cancel lifecycle, the permission matrix per key level, and the
bootstrap-opens-once flow.

## Project layout

```text
cmd/warehouse-server/main.go  # CLI (serve|migrate|version|backup), flags, request logging
internal/
├── config/config.go          # Env-only config + defaults
├── db/db.go                  # SQLite open (WAL, FK, busy_timeout), migrations, backup
├── auth/auth.go               # Opaque API keys, staggered permissions, middleware identity
├── users/users.go             # Users + API keys CRUD, disable/revoke (never delete)
├── vendors/vendors.go         # Vendors, items, vendor-SKU catalog
├── warehouses/warehouses.go   # Warehouses + flat per-warehouse locations
├── inventory/
│   ├── inventory.go           # Stock reads with denormalized labels
│   └── movements.go           # receive/remove/adjust/transfer + planned lifecycle
├── httpapi/
│   ├── server.go              # Mux wiring, auth gate, JSON error envelope
│   ├── users.go               # User/key routes (incl. open bootstrap)
│   ├── masters.go             # Catalog + warehouse routes
│   ├── stock.go               # Stock + movement routes
│   └── helpers.go             # Shared handler utilities
├── core/core.go               # Shared kernel: error codes, Result[T], generic helpers
migrations/0001_init.sql       # Full v1 schema (forward-only from here)
PLAN.md                        # Design doc and phased implementation plan
mise.toml                      # Pinned toolchain + tasks
```

## License

MIT — do whatever you want with it, just don't blame me for your stock counts.
