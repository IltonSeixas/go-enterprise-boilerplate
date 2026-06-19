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
| `ADAPTER` | No | `memory` | Persistence adapter: `memory` or `postgres`. When `postgres` is selected, schema migrations are applied automatically at startup |
| `DATABASE_URL` | If `postgres` | — | PostgreSQL DSN (`postgres://user:pass@host/db`) |

### Cache

| Variable | Required | Default | Description |
|---|---|---|---|
| `REDIS_URL` | No | `redis://localhost:6379` | Redis connection string — used unconditionally for refresh-token storage |

### Authentication

| Variable | Required | Default | Description |
|---|---|---|---|
| `JWT_PRIVATE_KEY_PATH` | Yes | — | Path to the Ed25519 PEM private key used to sign access tokens |
| `JWT_PUBLIC_KEY_PATH` | Yes | — | Path to the Ed25519 PEM public key used to verify access tokens |
| `JWT_ACCESS_TTL` | No | `15m` | Access token TTL (Go duration: `15m`, `1h`) |
| `JWT_REFRESH_TTL` | No | `168h` | Refresh token TTL (Go duration) |

Generate a key pair with:

```bash
openssl genpkey -algorithm ed25519 -out jwt_private.pem
openssl pkey -in jwt_private.pem -pubout -out jwt_public.pem
```

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

- [ ] `JWT_PRIVATE_KEY_PATH`/`JWT_PUBLIC_KEY_PATH` point to a key pair generated specifically for this environment — never reuse development keys
- [ ] `DATABASE_URL` includes TLS parameters (`sslmode=require`) when `ADAPTER=postgres`
- [ ] `REDIS_URL` uses a password-protected Redis instance
- [ ] `ALLOWED_ORIGINS` lists only your actual frontend domains
- [ ] `OTLP_ENDPOINT` points to your observability backend
- [ ] All secrets are injected via a secrets manager — never committed to source control
