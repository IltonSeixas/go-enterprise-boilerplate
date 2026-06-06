# Security

## Threat Model

This boilerplate is designed for multi-tenant web APIs exposed to the public internet. The primary threats addressed are:

- Credential stuffing and brute-force attacks
- Session hijacking and token theft
- Injection attacks (SQL, command)
- Information disclosure via error messages or logs
- Denial of service via resource exhaustion

---

## Password Hashing — Argon2id

All passwords are hashed using **Argon2id** via `golang.org/x/crypto/argon2` — the native Go implementation with no third-party crypto dependencies.

bcrypt and scrypt are not used.

### Parameters

```go
argon2.IDKey(
    password,
    salt,
    3,      // time cost: 3 iterations
    64*1024, // memory cost: 64 MB
    4,      // parallelism: 4 threads
    32,     // output key length: 32 bytes
)
```

These parameters meet the OWASP minimum recommendations. Adjust upward based on your hardware profile and acceptable latency budget.

### Salt

A cryptographically random 16-byte salt is generated per-hash via `crypto/rand`. The salt is encoded alongside the hash in a PHC-formatted string — never stored separately.

### Verification

Timing-safe comparison uses `subtle.ConstantTimeCompare` from the Go standard library. Never use `bytes.Equal` or `==` for hash comparison.

---

## Authentication

### Access Token (JWT HS256)

- Algorithm: HS256 (HMAC-SHA256)
- TTL: 15 minutes
- Claims: `sub` (user ID), `iat`, `exp`, `jti` (unique token ID)
- Storage: in-memory on the client — never in `localStorage` or cookies
- Validation: signature + expiry checked on every authenticated request via Gin middleware

### Refresh Token

- Format: opaque UUID v4 generated via `crypto/rand`
- Storage: server-side in Redis with TTL 7 days
- Transport: HttpOnly, Secure, SameSite=Strict cookie
- Rotation: a new refresh token is issued on every use; the old one is immediately invalidated
- Revocation: deleting the Redis key invalidates the session instantly

### Token Revocation

Access tokens cannot be revoked before expiry (stateless by design). The 15-minute TTL limits the exposure window. If immediate revocation is required, implement a short-lived Redis blocklist for `jti` values.

---

## Rate Limiting

Implemented as Gin middleware using a sliding window counter per IP address stored in Redis.

```
Default: 100 requests / 60 seconds per IP
Configurable via: RATE_LIMIT_RPS environment variable
```

Authentication endpoints have a stricter independent limit to mitigate credential stuffing.

On limit exceeded, the server returns `429 Too Many Requests` with a `Retry-After` header.

---

## Security Headers

Applied globally via Gin middleware on every response:

| Header | Value |
|---|---|
| `Strict-Transport-Security` | `max-age=63072000; includeSubDomains; preload` |
| `X-Content-Type-Options` | `nosniff` |
| `X-Frame-Options` | `DENY` |
| `Content-Security-Policy` | `default-src 'none'` (API — no HTML served) |
| `Referrer-Policy` | `no-referrer` |
| `Permissions-Policy` | `geolocation=(), camera=(), microphone=()` |

---

## CORS

CORS is configured with an explicit allow-list. The wildcard `*` is never permitted in production.

```go
config := cors.Config{
    AllowOrigins:     cfg.AllowedOrigins, // from environment variable
    AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
    AllowHeaders:     []string{"Authorization", "Content-Type"},
    AllowCredentials: true,
}
```

---

## Input Validation

All inputs are validated at the HTTP boundary before reaching any use case using `go-playground/validator`. Invalid input returns `400 Bad Request` with a structured error body — never a stack trace.

Domain-level invariants are re-enforced inside value object constructors regardless of what the HTTP layer does. The domain is the last line of defense.

---

## SQL Injection Prevention

All database queries use `pgx` with parameterized queries or `squirrel` query builder. String formatting into SQL is never used.

```go
rows, err := pool.Query(ctx,
    "SELECT id, email FROM users WHERE email = $1",
    email.String(),
)
```

---

## Sensitive Data

- Passwords are never logged, never returned in API responses, and never stored in plain text
- Tokens are never logged
- Error responses to clients contain a message and an error code — never internal details, stack traces, or database errors
- `slog` level must never be set to `Debug` in production (would expose request bodies)

---

## Dependency Auditing

`govulncheck` runs on every CI push against the Go vulnerability database.

```bash
govulncheck ./...
```

Review `go.sum` before deploying. Every transitive dependency is a potential attack surface.
