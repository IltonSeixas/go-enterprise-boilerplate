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
│   │   │   └── postgres/ # Production: pgx with parameterized raw SQL
│   │   ├── security/     # Argon2id hashing, JWT issuance, Redis-backed refresh tokens
│   │   └── telemetry/    # OpenTelemetry + zap setup
│   │
│   └── interface/        # Entry points
│       ├── http/          # Gin handlers, middleware, routes
│       └── grpc/          # gRPC server and service implementations
│
└── proto/                # Protocol Buffers definitions and generated code
```

### Dependency rule

```
interface/ → application/ → domain/
infrastructure/ → application/ → domain/
```

`domain/` and `application/` never import from `infrastructure/` or `interface/`. Enforced automatically by `internal/architecture/layering_test.go` (see [ADR-0006](docs/adr/0006-architecture-layering-test.md)) as part of the regular `go test ./...` run.

---

## Stack

| Concern | Library |
|---|---|
| HTTP router | `gin-gonic/gin` |
| gRPC | `google.golang.org/grpc` + `protoc-gen-go` |
| Database (production) | `jackc/pgx` with parameterized raw SQL |
| Password hashing | `golang.org/x/crypto/argon2` (native) |
| JWT | `golang-jwt/jwt/v5` |
| Validation | Gin binding tags backed by `go-playground/validator/v10` |
| Observability | `go.opentelemetry.io/otel` |
| Structured logging | `go.uber.org/zap` |
| Config | `spf13/viper` |
| Testing | `testing` (stdlib) + `testify` + hand-written stubs (`internal/testutil`) |
| Rate limiting | `golang.org/x/time/rate` (in-memory, per-IP token bucket) |

---

## Getting Started

### Prerequisites

- Go 1.25+ (Docker builds use 1.26.4)
- Optional for production: PostgreSQL 15+, Redis 7+

### Run immediately (in-memory, zero database)

```bash
git clone https://github.com/your-org/go-enterprise-boilerplate
cd go-enterprise-boilerplate
cp .env.example .env
openssl genpkey -algorithm ed25519 -out jwt_private.pem
openssl pkey -in jwt_private.pem -pubout -out jwt_public.pem
go run ./cmd/server
```

The server starts on `http://localhost:8080`. No database required.

### Persistence adapter

The adapter is selected at runtime via the `ADAPTER` environment variable (`memory`, the default, or `postgres`):

```bash
cp .env.example .env
# Edit .env: set ADAPTER=postgres, DATABASE_URL, JWT_PRIVATE_KEY_PATH, JWT_PUBLIC_KEY_PATH, etc.

go run ./cmd/server
```

> **Note**: when `ADAPTER=postgres`, the server connects using `DATABASE_URL` and applies the embedded schema migrations automatically on startup.

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

- **Access token**: JWT EdDSA (Ed25519), TTL 15 min, validated on every authenticated request
- **Refresh token**: opaque UUID, stored in Redis with TTL 7 days, rotated on every use
- **Revocation**: deleting the Redis entry immediately invalidates the session

### Security Middleware (applied globally via Gin)

- Rate limiting: in-memory per-IP token bucket via `golang.org/x/time/rate` (stricter limit on `/v1/auth/*`)
- Security headers: `X-Content-Type-Options`, `X-Frame-Options`, `Strict-Transport-Security`, `Referrer-Policy`
- CORS: explicit allow-list, never `*` in production
- Input validation: Gin binding tags (backed by `go-playground/validator`) on all request DTOs

### Audit Logging

Every identity- and access-sensitive use case (registration, login success/failure, password change, role change, token refresh) records an immutable `AuditEvent` through the `AuditPort` interface in `internal/application/port/`. The in-memory adapter is the zero-config default; the PostgreSQL adapter persists to a dedicated `audit_log` table and never fails the use case it observes, degrading gracefully if the audit store itself is unavailable.

---

## API

### REST — `http://localhost:8080`

| Method | Path | Description |
|---|---|---|
| `POST` | `/v1/auth/register` | Register a new user |
| `POST` | `/v1/auth/login` | Authenticate, receive tokens |
| `POST` | `/v1/auth/refresh` | Rotate refresh token |
| `GET` | `/v1/users/me` | Get authenticated user profile |
| `PUT` | `/v1/users/me` | Update authenticated user profile |
| `PUT` | `/v1/users/me/password` | Change authenticated user password |
| `GET` | `/v1/users/:id` | Get user by ID |
| `PUT` | `/v1/users/:id/role` | Change a user's role (Owner only, cannot change own role) |
| `GET` | `/health` | Liveness check |
| `GET` | `/ready` | Readiness check |
| `GET` | `/metrics` | Prometheus metrics |

### gRPC — `localhost:50051`

Proto definitions live in `proto/boilerplate/v1/boilerplate.proto`. Generated stubs are committed under `internal/interface/grpc/proto/`; regenerate them with:

```bash
make proto  # requires protoc, protoc-gen-go and protoc-gen-go-grpc on PATH
```

| Service | RPC | Mirrors |
|---|---|---|
| `AuthService` | `Register`, `Login`, `RefreshToken` | `/v1/auth/*` |
| `UserService` | `GetMe`, `UpdateProfile`, `ChangePassword`, `ChangeRole` | `/v1/users/*` |

`UserService` RPCs require an `authorization: Bearer <access_token>` request metadata entry, validated by a unary interceptor that mirrors the REST `RequireAuth` middleware (active-account check included). Server reflection is enabled for easy inspection with tools like `grpcurl`.

---

## Testing

```bash
go test ./...                          # all tests
go test ./internal/domain/... -v       # domain tests only
```

### Structure

- **Unit tests**: `_test.go` files co-located with source. Domain entities, value objects and use cases are tested in complete isolation using hand-written stubs from `internal/testutil` that satisfy the repository and service port interfaces — no Spring-style mocking framework, no real infrastructure.
- **Architecture tests**: `internal/architecture/layering_test.go` enforces the Clean Architecture dependency rule from [ADR-0001](docs/adr/0001-clean-architecture.md) at test time — see [ADR-0006](docs/adr/0006-architecture-layering-test.md). Runs as part of the regular `go test ./...` step.

### TDD Approach

Write the use case test first, asserting against the port interface and a stub from `internal/testutil`. There's no coupling to any storage engine. Once the test is green, the use case works regardless of which adapter is wired at runtime.

---

## Observability

- **Traces**: an OTLP gRPC exporter and tracer provider are configured at startup; use cases and adapters emit spans via `context`-propagated tracers
- **Metrics**: Prometheus metrics at `/metrics` — exported through the OpenTelemetry Prometheus bridge
- **Logs**: structured JSON via `go.uber.org/zap`, correlated with trace IDs via context

Export to any OTLP backend:

```env
OTLP_ENDPOINT=localhost:4317
```

### Resilience

Redis calls made by the JWT service (refresh token issuance, validation, rotation and revocation) are wrapped in a `Closed → Open → Half-Open` circuit breaker (`internal/infrastructure/resilience`) combined with a retry policy. A transient Redis failure that succeeds on retry counts as a single success against the breaker's failure rate, rather than inflating it.

---

## Configuration

All configuration via environment variables or `.env` file (Viper reads both).

| Variable | Default | Description |
|---|---|---|
| `HOST` | `0.0.0.0` | Bind address |
| `PORT` | `8080` | HTTP port |
| `GRPC_PORT` | `50051` | gRPC port |
| `ADAPTER` | `memory` | Persistence adapter: `memory` or `postgres` |
| `DATABASE_URL` | — | PostgreSQL DSN (required when `ADAPTER=postgres`) |
| `REDIS_URL` | `redis://localhost:6379` | Redis connection URL (refresh token storage) |
| `DB_POOL_MAX_CONNS` | `10` | Maximum number of pooled Postgres connections (used when `ADAPTER=postgres`) |
| `DB_POOL_MIN_CONNS` | `2` | Minimum number of idle Postgres connections kept open |
| `DB_POOL_CONNECT_TIMEOUT` | `30s` | Max time to wait for a free connection from the pool (Go duration string) |
| `DB_POOL_IDLE_TIMEOUT` | `10m` | Time before an idle connection above the minimum is closed |
| `DB_POOL_MAX_LIFETIME` | `30m` | Max lifetime of a pooled connection before it is recycled |
| `REDIS_CONNECT_TIMEOUT` | `2s` | Max time to establish the Redis TCP connection |
| `REDIS_COMMAND_TIMEOUT` | `2s` | Max time to wait for a Redis command response (read and write) |
| `JWT_PRIVATE_KEY_PATH` | — | Path to the Ed25519 PEM private key used to sign access tokens |
| `JWT_PUBLIC_KEY_PATH` | — | Path to the Ed25519 PEM public key used to verify access tokens |
| `JWT_ACCESS_TTL` | `15m` | Access token TTL (Go duration string) |
| `JWT_REFRESH_TTL` | `168h` | Refresh token TTL (Go duration string) |
| `ALLOWED_ORIGINS` | `http://localhost:3000` | Comma-separated CORS allow-list |
| `OTLP_ENDPOINT` | `localhost:4317` | OTLP gRPC traces/metrics endpoint |

---

## Docker

```bash
# Multi-stage build — minimal final image (FROM scratch)
docker build -t go-enterprise-boilerplate .

docker run -p 8080:8080 -p 50051:50051 --env-file .env \
  -v "$(pwd)/jwt_private.pem:/app/jwt_private.pem:ro" \
  -v "$(pwd)/jwt_public.pem:/app/jwt_public.pem:ro" \
  go-enterprise-boilerplate
```

```bash
# Full stack: app + postgres + redis + jaeger
# Requires jwt_private.pem/jwt_public.pem in the repo root — see Configuration above
openssl genpkey -algorithm ed25519 -out jwt_private.pem
openssl pkey -in jwt_private.pem -pubout -out jwt_public.pem
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
