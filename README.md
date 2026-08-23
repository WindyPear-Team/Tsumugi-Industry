# Tsumugi Industry

Go + Gin + GORM industrial control system foundation with an embedded Vite + React + shadcn/ui frontend.

## Features

- First-start system initialization flow for the first administrator.
- SQLite, MySQL, and PostgreSQL through GORM; choose with `DB_DRIVER` and `DB_DSN`.
- Environment variables control only `HOST`, `PORT`, and database connection information.
- Runtime settings, users, roles, and fine-grained permissions are stored in the database.
- JWT login and permission middleware foundation.
- Black-and-white shadcn/ui console with responsive sidebar, route-based pages, and light/dark theme switching.

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
