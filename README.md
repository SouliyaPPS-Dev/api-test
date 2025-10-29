# Backend API

Golang REST API for the admin backoffice following a clean architecture layout. Each layer has a single responsibility: the domain defines entities and errors, use cases contain application logic, infrastructure houses PostgreSQL/JWT implementations, and the HTTP layer orchestrates requests.

## Features

- PostgreSQL persistence for users and product catalog.
- JWT-based authentication with configurable secret, issuer, and expiry.
- Clean architecture layering (domain → use case → infrastructure → interfaces).
- RESTful product CRUD endpoints protected by bearer auth.
- CORS middleware with configurable origin whitelist.

## Project Structure

```
backend/
├── cmd/server/main.go               # Application entrypoint & wiring
├── internal/
│   ├── app/                         # (reserved for future orchestration)
│   ├── config/                      # Environment configuration
│   ├── domain/                      # Core entities + domain errors
│   │   ├── auth/
│   │   └── product/
│   ├── httpserver/                  # HTTP handlers, middleware, routing
│   ├── infrastructure/
│   │   ├── postgres/                # PostgreSQL repositories + pool
│   │   └── token/                   # JWT token manager
│   └── usecase/                     # Application services (auth, product)
└── go.mod                           # Module definition + dependencies
```

## Configuration

The server reads configuration from environment variables:

| Variable                | Description                                  | Default       |
| ----------------------- | -------------------------------------------- | ------------- |
| `HTTP_PORT`             | HTTP bind address/port (`:8080` form ok)     | `8080`        |
| `DATABASE_URL`          | PostgreSQL DSN (`postgres://...`)            | **required**  |
| `JWT_SECRET`            | HMAC secret for JWT signing                  | **required**  |
| `JWT_ISSUER`            | JWT issuer claim                             | `backoffice`  |
| `JWT_EXPIRY`            | Token lifetime (Go duration string)          | `12h`         |
| `CORS_ALLOWED_ORIGINS`  | Comma separated list of allowed origins      | `*`           |

Optional HTTP timeouts can be adjusted with `HTTP_READ_TIMEOUT`, `HTTP_WRITE_TIMEOUT`, `HTTP_IDLE_TIMEOUT` (seconds).

Environment variables can also be stored in a `.env` file in this directory. The application will read it automatically on startup if present.

### Database Schema

Create the following tables (example migration):

```sql
CREATE TABLE users (
  id            TEXT PRIMARY KEY,
  email         TEXT NOT NULL UNIQUE,
  name          TEXT,
  password_hash TEXT NOT NULL,
  created_at    TIMESTAMP WITH TIME ZONE NOT NULL,
  updated_at    TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE TABLE products (
  id          TEXT PRIMARY KEY,
  name        TEXT NOT NULL,
  description TEXT,
  sku         TEXT NOT NULL UNIQUE,
  price       NUMERIC(12,2) NOT NULL DEFAULT 0,
  quantity    INTEGER NOT NULL DEFAULT 0,
  created_at  TIMESTAMP WITH TIME ZONE NOT NULL,
  updated_at  TIMESTAMP WITH TIME ZONE NOT NULL
);
```

## Running the Server

```bash
cd backend
cp .env.example .env   # adjust values as needed
go run ./cmd/server
```

The server listens on `http://localhost:8080` by default.

### Hot Reload with Air

This project ships with an `.air.toml` configuration for hot reloading during development.

1. Install Air once (requires Go 1.20+):
   ```bash
   go install github.com/air-verse/air@latest
   ```
2. From the project root, run:
   ```bash
   air
   ```

Air watches the `cmd` and `internal` directories, rebuilds the binary into `./tmp/bin`, and restarts the server whenever Go files or configuration assets change.

## API Reference

### Authentication

- `POST /auth/register`  
  `{"email":"user@example.com","password":"secret","name":"Admin"}`

- `POST /auth/login`  
  Returns `{"token":"...","user":{...}}`

### Products (Bearer token required)

- `GET /products`
- `POST /products`
- `GET /products/{id}`
- `PUT /products/{id}`
- `PATCH /products/{id}`
- `DELETE /products/{id}`

Send the JWT via `Authorization: Bearer <token>`. All endpoints use JSON.

## Testing

```bash
cd backend
export GOMODCACHE=$(pwd)/.gomodcache
export GOCACHE=$(pwd)/.gocache
go test ./...
```

## Docker

Build the production image locally:

```bash
docker build -t backoffice-backend .
```

Run the container (configure environment variables to match your Railway deployment):

```bash
docker run --rm -p 8080:8080 \
  -e DATABASE_URL=postgres://... \
  -e JWT_SECRET=super-secret \
  -e JWT_ISSUER=backoffice \
  backoffice-backend
```

### Docker Compose (API + Postgres)

For local development with PostgreSQL:

```bash
docker compose up --build
```

This starts:

- `api`: the Go application running on `http://localhost:8080`
- `postgres`: PostgreSQL 16 on `localhost:5432` with database `test`

The API uses the default `DATABASE_URL=postgres://postgres:postgres@postgres:5432/test?sslmode=disable`. Override `POSTGRES_DB`, `POSTGRES_USER`, or `POSTGRES_PASSWORD` with shell environment variables when launching `docker compose`.

To override application secrets, set them in your shell before starting Compose (e.g. `export JWT_SECRET=my-secret`) or create a `.env` file in this directory—Compose will substitute them automatically.

To shut everything down and remove the database volume:

```bash
docker compose down -v
```

## CI/CD (Railway)

Automated testing and deployment to Railway run from `.github/workflows/railway-deploy.yml`. The workflow:

- Runs on pushes to the `main` branch (and can be triggered manually).
- Executes `go test ./...`.
- Installs the Railway CLI and calls `railway up`, which builds from the repository `Dockerfile`.

Configure these GitHub secrets before enabling the workflow:

- `RAILWAY_TOKEN` – Railway project token with deploy permissions.
- `RAILWAY_SERVICE_ID` – Target service identifier in Railway.
- `RAILWAY_ENVIRONMENT_ID` *(optional)* – Deploy to a non-default environment if required.

### Railway Postgres Setup

1. In your Railway project add the **Postgres** plugin (Create → Database → Postgres). Railway provisions the database and exposes credentials as environment variables (`DATABASE_URL`, `PGHOST`, `PGUSER`, etc.).
2. Link the Postgres plugin to the API service (Service → Variables → Link Service). When linked, the next deploy automatically injects the credentials; the API now accepts either the single `DATABASE_URL` value or the individual `PG*` variables that Railway provides.
3. Define the remaining application secrets in Railway → Variables, e.g. `JWT_SECRET=my-strong-secret` (set `JWT_ISSUER` or other options if you need overrides).
4. Redeploy (`railway up` locally or via the GitHub Actions workflow). If you still see `database configuration missing: provide DATABASE_URL or PG* env vars` in the logs, re-check that the Postgres plugin is attached to the service or manually set `DATABASE_URL` under Variables. Connection strings starting with either `postgres://` or `postgresql://` work—the server normalises them automatically.
