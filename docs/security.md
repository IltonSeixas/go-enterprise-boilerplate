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

- Algorithm: HS256 (HMAC-SHA256) via `golang-jwt/jwt`
- TTL: 15 minutes (`JWT_ACCESS_TTL`)
- Claims: `sub` (user ID), `role`, `iat`, `exp`
- Transport: returned in the JSON response body (`access_token`); the client is responsible for storage and for sending it as `Authorization: Bearer <token>`
- Validation: signature + expiry checked on every authenticated request via Gin middleware

### Refresh Token

- Format: opaque UUID v4 via `uuid.New()`
- Storage: server-side in Redis with TTL 7 days (`JWT_REFRESH_TTL`)
- Transport: returned as a plain field (`refresh_token`) in the JSON response body — the client must store and resend it explicitly
- Rotation: a new refresh token is issued on every use; the old one is immediately invalidated
- Revocation: deleting the Redis key invalidates the session instantly

### Token Revocation

Access tokens cannot be revoked before expiry (stateless by design). The 15-minute TTL limits the exposure window. Refresh tokens, by contrast, are revocable instantly because they are stored server-side in Redis.

---

## Rate Limiting

Implemented as in-process Gin middleware (`internal/interface/http/middleware/rate_limit.go`) using a per-IP token bucket from `golang.org/x/time/rate` — no external store required.

```
Global default:    100 requests/sec, burst 20, per IP
/v1/auth/* default: 10 requests/sec, burst 5, per IP
```

The stricter limit on authentication endpoints mitigates credential stuffing and brute-force attacks. Limits are wired in `NewRouter` and can be tuned by changing the `rate.Limit`/burst arguments passed to `middleware.RateLimit`.

On limit exceeded, the server returns `429 Too Many Requests` with a `Retry-After` header.

---

## Security Headers

Applied globally via Gin middleware on every response:

| Header | Value |
|---|---|
| `Strict-Transport-Security` | `max-age=63072000; includeSubDomains` |
| `X-Content-Type-Options` | `nosniff` |
| `X-Frame-Options` | `DENY` |
| `X-XSS-Protection` | `0` |
| `Referrer-Policy` | `strict-origin-when-cross-origin` |

---

## CORS

CORS is implemented as a small, dependency-free Gin middleware (`internal/interface/http/middleware/cors.go`) backed by an explicit origin allow-list read from the `ALLOWED_ORIGINS` environment variable. The wildcard `*` is never honored — an `Origin` header that doesn't match the allow-list simply receives no CORS headers.

```go
func CORS(allowedOrigins []string) gin.HandlerFunc {
    allowed := make(map[string]struct{}, len(allowedOrigins))
    for _, origin := range allowedOrigins {
        allowed[origin] = struct{}{}
    }

    return func(c *gin.Context) {
        origin := c.GetHeader("Origin")
        if _, ok := allowed[origin]; ok {
            c.Header("Access-Control-Allow-Origin", origin)
            c.Header("Access-Control-Allow-Credentials", "true")
            c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
            c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
            c.Header("Vary", "Origin")
        }
        if c.Request.Method == http.MethodOptions {
            c.AbortWithStatus(http.StatusNoContent)
            return
        }
        c.Next()
    }
}
```

---

## Input Validation

All inputs are validated at the HTTP boundary before reaching any use case using `go-playground/validator`. Invalid input returns `400 Bad Request` with a structured error body — never a stack trace.

Domain-level invariants are re-enforced inside value object constructors regardless of what the HTTP layer does. The domain is the last line of defense.

---

## SQL Injection Prevention

All database queries use `pgx` with parameterized raw SQL (`$1`, `$2`, ... placeholders). String formatting into SQL is never used.

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
- The `zap` logger must never run at `Debug` level in production (would expose request bodies)

---

## Dependency Auditing

`govulncheck` runs on every CI push against the Go vulnerability database.

```bash
govulncheck ./...
```

Review `go.sum` before deploying. Every transitive dependency is a potential attack surface.
