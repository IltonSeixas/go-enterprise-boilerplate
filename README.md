# go-enterprise-boilerplate

[![CI](https://github.com/IltonSeixas/go-enterprise-boilerplate/actions/workflows/ci.yml/badge.svg)](https://github.com/IltonSeixas/go-enterprise-boilerplate/actions/workflows/ci.yml)
[![Docker](https://github.com/IltonSeixas/go-enterprise-boilerplate/actions/workflows/docker.yml/badge.svg)](https://github.com/IltonSeixas/go-enterprise-boilerplate/actions/workflows/docker.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](./LICENSE)

Production-ready enterprise backend boilerplate in **Go** — built on Clean Architecture, Domain-Driven Design, and Test-Driven Development. Runs immediately with an in-memory adapter; plug in PostgreSQL when ready for production.

---

## Philosophy

Go's simplicity is a feature, not a limitation. This boilerplate embraces idiomatic Go — explicit error handling, interfaces over inheritance, composition over abstraction — while enforcing the architectural discipline that large systems require. The domain package has zero external imports.

---

## Architecture

```
.
├── cmd/
│   └── server/           # main.go — wiring and startup
│
├── internal/
│   ├── domain/           # Enterprise business rules — no external deps
│   │   ├── entity/       # Aggregates and Entities
│   │   ├── valueobject/  # Immutable, self-validating values
│   │   ├── repository/   # Port interfaces
│   │   └── apperror/     # Domain error types
│   │
│   ├── application/      # Use cases — depends only on domain
│   │   ├── usecase/      # One struct per use case
│   │   ├── port/         # Input/output port interfaces
│   │   └── dto/          # Data transfer objects
│   │
│   ├── infrastructure/   # Adapters — implements domain interfaces
│   │   ├── persistence/
│   │   │   ├── memory/   # Default: zero-config, runs immediately
│   │   │   └── postgres/ # Production: pgx + squirrel
│   │   ├── security/     # Argon2id password hashing
│   │   ├── cache/        # Redis adapter
│   │   └── telemetry/    # OpenTelemetry setup
│   │
│   └── interface/        # Entry points
│       ├── http/          # Gin handlers, middleware, routes
│       └── grpc/          # gRPC server and service implementations
│
└── pkg/                  # Exported utilities (safe for external use)
    ├── pagination/
    └── validator/
```

### Dependency rule

```
interface/ → application/ → domain/
infrastructure/ → application/ → domain/
```

`domain/` and `application/` never import from `infrastructure/` or `interface/`.

---

## Stack

| Concern | Library |
|---|---|
| HTTP router | `gin-gonic/gin` |
| gRPC | `google.golang.org/grpc` + `protoc-gen-go` |
| Database (production) | `jackc/pgx` + `Masterminds/squirrel` |
| Password hashing | `golang.org/x/crypto/argon2` (native) |
| JWT | `golang-jwt/jwt/v5` |
| Validation | `go-playground/validator/v10` |
| Observability | `go.opentelemetry.io/otel` |
| Structured logging | `log/slog` (stdlib, Go 1.21+) |
| Config | `spf13/viper` |
| Testing | `testing` (stdlib) + `testify` + `gomock` |
| Migration | `golang-migrate/migrate` |

---

## Getting Started

### Prerequisites

- Go 1.22+
- Optional for production: PostgreSQL 15+, Redis 7+

### Run immediately (in-memory, zero config)

```bash
git clone https://github.com/your-org/go-enterprise-boilerplate
cd go-enterprise-boilerplate
go run ./cmd/server
```

The server starts on `http://localhost:3000`. No database required.

### Run with PostgreSQL

```bash
cp .env.example .env
# Edit .env: set DATABASE_URL, JWT_SECRET, etc.

go run ./cmd/server -adapter=postgres
```

---

## Security

### Password Hashing — Argon2id

Passwords are hashed with **Argon2id** via `golang.org/x/crypto/argon2` — the native Go implementation. No third-party crypto dependencies.

Parameters follow OWASP recommendations:
- Memory: 64 MB
- Iterations: 3
- Parallelism: 4

The `PasswordHasher` interface in `domain/repository/` abstracts the algorithm from all business logic.

### Authentication Flow

- **Access token**: JWT HS256, TTL 15 min, validated on every authenticated request
- **Refresh token**: opaque UUID, stored in Redis with TTL 7 days, rotated on every use
- **Revocation**: deleting the Redis entry immediately invalidates the session

### Security Middleware (applied globally via Gin)

- Rate limiting: sliding window per IP using Redis
- Security headers: `X-Content-Type-Options`, `X-Frame-Options`, `Strict-Transport-Security`, `Content-Security-Policy`
- CORS: explicit allow-list, never `*` in production
- Input validation: `validator` on all DTOs at the HTTP boundary
- Request ID: injected on every request, propagated through context

---

## API

### REST — `http://localhost:3000`

| Method | Path | Description |
|---|---|---|
| `POST` | `/api/v1/auth/register` | Register a new user |
| `POST` | `/api/v1/auth/login` | Authenticate, receive tokens |
| `POST` | `/api/v1/auth/refresh` | Rotate refresh token |
| `POST` | `/api/v1/auth/logout` | Revoke refresh token |
| `GET` | `/api/v1/users/me` | Get authenticated user profile |
| `GET` | `/health` | Health check |
| `GET` | `/metrics` | Prometheus metrics |

### gRPC — `localhost:50051`

Proto definitions in `proto/`. Regenerate with:

```bash
make proto
```

---

## Testing

```bash
go test ./...                          # all unit tests
go test ./... -tags=integration        # integration tests (requires Postgres)
go test ./internal/domain/... -v       # domain tests only
```

### Structure

- **Unit tests**: `_test.go` files co-located with source. Domain and use cases tested in complete isolation using `gomock`-generated mocks from repository port interfaces.
- **Integration tests**: `internal/infrastructure/**/*_integration_test.go`. Run against a real database using `testcontainers-go` or a local instance.

### TDD Approach

Write the use case test first, asserting against the port interface. The mock repository is generated from the interface definition — there's no coupling to any storage engine. Once the test is green, the use case works regardless of which adapter is wired at runtime.

---

## Observability

- **Traces**: `otelgin` middleware instruments every HTTP request automatically; use cases emit child spans via `context`-propagated tracer
- **Metrics**: Prometheus metrics at `/metrics` — request count, latency histograms, active connections
- **Logs**: structured JSON via `log/slog`, correlated with trace IDs via context

Export to any OTLP backend:

```env
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
```

---

## Configuration

All configuration via environment variables or `.env` file (Viper reads both).

| Variable | Default | Description |
|---|---|---|
| `HOST` | `0.0.0.0` | Bind address |
| `PORT` | `3000` | HTTP port |
| `GRPC_PORT` | `50051` | gRPC port |
| `DATABASE_URL` | — | PostgreSQL DSN |
| `REDIS_URL` | — | Redis URL |
| `JWT_SECRET` | — | HS256 signing key (min 32 chars) |
| `JWT_ACCESS_TTL` | `15m` | Access token TTL |
| `JWT_REFRESH_TTL` | `168h` | Refresh token TTL |
| `RATE_LIMIT_RPS` | `100` | Max requests/sec per IP |
| `LOG_LEVEL` | `info` | Log level |
| `ADAPTER` | `memory` | Persistence adapter: `memory` or `postgres` |

---

## Docker

```bash
# Multi-stage build — minimal final image (~10 MB)
docker build -t go-enterprise-boilerplate .

docker run -p 3000:3000 -p 50051:50051 --env-file .env go-enterprise-boilerplate
```

```bash
# Full stack: app + postgres + redis + jaeger
docker compose up
```

---

## CI/CD

GitHub Actions pipelines in `.github/workflows/`:

| Workflow | Trigger | Steps |
|---|---|---|
| `ci.yml` | push / PR | vet, staticcheck, test, govulncheck |
| `docker.yml` | push to `main` | build + push to GHCR |
| `release.yml` | tag `v*` | cross-compile binaries, create GitHub Release |

`govulncheck` scans for known vulnerabilities in dependencies on every push.

---

## Plugging in a Real Database

Implement the `UserRepository` interface from `internal/domain/repository/` and wire it in `cmd/server/main.go`. The in-memory adapter stays available for unit tests and local development — no test containers needed for the domain layer.

---

## Author

**Ilton Seixas** — [contact@iltonseixas.com](mailto:contact@iltonseixas.com)

---

## Disclaimer

This boilerplate is provided **as-is**, for educational and reference purposes only.

**No warranty.** The author makes no representations or warranties of any kind, express or implied, regarding the correctness, completeness, reliability, suitability, or availability of this software for any purpose. Your use of this code is entirely at your own risk.

**No liability.** To the fullest extent permitted by applicable law, the author shall not be held liable for any direct, indirect, incidental, special, consequential, or punitive damages arising from the use or misuse of this software — including but not limited to data breaches, security incidents, financial loss, service downtime, or regulatory non-compliance.

**Misuse.** The author is not responsible for any unlawful, harmful, or unethical use of this codebase by any party.

**Security.** Security patterns and cryptographic implementations in this project follow industry best practices at the time of writing. However, the threat landscape evolves. You are solely responsible for auditing, hardening, and maintaining any system you build on top of this code.

> **Never blindly trust third-party code — including this project.**
> The author strongly recommends that you read and understand every line before deploying to production. Security-sensitive components (authentication, password hashing, token management, input validation) deserve particular scrutiny. No code review by a stranger on the internet replaces your own.

---

## License

MIT — Copyright (c) Ilton Seixas
