# Changelog

All notable changes to this project will be documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/). This project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### Added
- Initial project structure: Clean Architecture + DDD layers (`/internal`, `/cmd`)
- In-memory user repository adapter (zero-config default)
- Argon2id password hashing via `golang.org/x/crypto/argon2`
- JWT access token (returned in the response body) + opaque UUID refresh token with Redis-backed rotation (returned as a plain JSON field)
- Gin HTTP server with security middleware (per-IP rate limiting, CORS allow-list, security headers)
- gRPC server with auth and user services
- OpenTelemetry tracing, Prometheus metrics, structured logs via `zap`
- PostgreSQL adapter via `pgx` (implemented, not yet wired into the composition root)
- Docker multi-stage image (`scratch` base) and docker-compose stack
- GitHub Actions CI (vet, staticcheck, unit and integration tests, govulncheck), Docker, and Release workflows
- Architecture documentation, ADRs, security policy
- Test coverage for the HTTP router and per-IP rate limiting middleware
- `internal/architecture/layering_test.go` enforcing the Clean Architecture dependency rule from ADR-0001 at test time — see [ADR-0006](docs/adr/0006-architecture-layering-test.md)

### Changed
- **Breaking:** JWT access tokens are now signed with EdDSA (Ed25519) instead of HS256. `JWT_SECRET` is replaced by `JWT_PRIVATE_KEY_PATH`/`JWT_PUBLIC_KEY_PATH` — see [ADR-0005](docs/adr/0005-eddsa-jwt-signing.md). Tokens issued under the previous version are not valid under this one.

### Fixed
- gRPC `ChangeRole` RPC not enforcing authentication

[Unreleased]: https://github.com/IltonSeixas/go-enterprise-boilerplate/compare/HEAD
