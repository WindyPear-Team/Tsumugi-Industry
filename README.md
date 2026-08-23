# Tsumugi Industry

Go + Gin + GORM industrial control system foundation with an embedded Vite + React + shadcn/ui frontend.

## Features

- First-start system initialization flow for the first administrator.
- SQLite, MySQL, and PostgreSQL through GORM; choose with `DB_DRIVER` and `DB_DSN`.
- Environment variables control only `HOST`, `PORT`, and database connection information.
- Runtime settings, users, roles, and fine-grained permissions are stored in the database.
- JWT login and permission middleware foundation.
- Black-and-white shadcn/ui console with responsive sidebar, route-based pages, and light/dark theme switching.
- Production work orders with ordered process steps, operator workflow, PLC/gateway reports, idempotency, and audit events.

## Production reporting integration

Machine gateways can report a completed process step through the same state
machine used by the operator console. The request must include a unique
`Idempotency-Key`; retries return the existing result without counting it a
second time.

```http
POST /api/work-orders/{workOrderID}/steps/{stepID}/report
Authorization: Bearer <token>
Idempotency-Key: plc-line-a-20260823-000001
X-Production-Source: plc
Content-Type: application/json

{"passed_qty": 9, "failed_qty": 1, "reason": "外观缺陷", "payload": {"signal": "complete"}}
```

`X-Production-Source` accepts `gateway`, `plc`, or `operator`. The endpoint
validates that the work order and step are currently executable, records the
source and payload as a production event, and advances the next process step.

Copy `.env.example` to `.env` to configure a deployment.

## Development

```powershell
cd web
yarn install
yarn dev
```

## Build and run

```powershell
cd web
yarn build
cd ..
go run .
```

The Go server serves the embedded frontend and API at `http://localhost:8080`.

The first visit opens `/api/setup` through the initialization screen. After setup, sign in with the created administrator account.
