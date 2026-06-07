# Configuration

All configuration is read from environment variables at startup (via Viper). The application fails fast with a clear error message if any required variable is missing or invalid.

A `.env.example` file in the repository root lists every variable. Copy it to `.env` for local development.

```bash
cp .env.example .env
```

---

## Reference

### Server

| Variable | Required | Default | Description |
|---|---|---|---|
| `HOST` | No | `0.0.0.0` | Bind address |
| `PORT` | No | `8080` | HTTP listen port |
| `GRPC_PORT` | No | `50051` | gRPC listen port |

### Persistence

| Variable | Required | Default | Description |
|---|---|---|---|
| `ADAPTER` | No | `memory` | Persistence adapter: `memory` or `postgres`. The `postgres` adapter is implemented but not yet wired into `main.go` — selecting it currently exits with a fatal error |
| `DATABASE_URL` | If `postgres` | — | PostgreSQL DSN (`postgres://user:pass@host/db`) |

### Cache

| Variable | Required | Default | Description |
|---|---|---|---|
| `REDIS_URL` | No | `redis://localhost:6379` | Redis connection string — used unconditionally for refresh-token storage |

### Authentication

| Variable | Required | Default | Description |
|---|---|---|---|
| `JWT_SECRET` | Yes | — | HS256 signing key — minimum 32 characters, use a random value |
| `JWT_ACCESS_TTL` | No | `15m` | Access token TTL (Go duration: `15m`, `1h`) |
| `JWT_REFRESH_TTL` | No | `168h` | Refresh token TTL (Go duration) |

### Security

| Variable | Required | Default | Description |
|---|---|---|---|
| `ALLOWED_ORIGINS` | No | `http://localhost:3000` | Comma-separated CORS allowed origins |

### Observability

| Variable | Required | Default | Description |
|---|---|---|---|
| `OTLP_ENDPOINT` | No | `localhost:4317` | OTLP gRPC endpoint for traces and metrics |

---

## Production Checklist

Before deploying to production:

- [ ] `JWT_SECRET` is a random value of at least 32 characters — never reuse development values
- [ ] `DATABASE_URL` includes TLS parameters (`sslmode=require`) — if you have wired up the postgres adapter
- [ ] `REDIS_URL` uses a password-protected Redis instance
- [ ] `ALLOWED_ORIGINS` lists only your actual frontend domains
- [ ] `OTLP_ENDPOINT` points to your observability backend
- [ ] All secrets are injected via a secrets manager — never committed to source control
