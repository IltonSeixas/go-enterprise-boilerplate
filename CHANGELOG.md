# Changelog

All notable changes to this project will be documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/). This project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### Added
- Initial project structure: Clean Architecture + DDD layers (`/internal`, `/pkg`, `/cmd`)
- In-memory user repository adapter (zero-config default)
- Argon2id password hashing via `golang.org/x/crypto/argon2`
- JWT HS256 access token + opaque refresh token with Redis rotation
- Gin HTTP server with security middleware (rate limiting, CORS, security headers)
- gRPC server with user service
- OpenTelemetry tracing, Prometheus metrics, structured JSON logs via `slog`
- PostgreSQL adapter via `pgx` + `squirrel`
- Docker multi-stage image (`scratch` base) and docker-compose stack
- GitHub Actions CI (vet, staticcheck, test, govulncheck), Docker, and Release workflows
- Architecture documentation, ADRs, security policy

[Unreleased]: https://github.com/IltonSeixas/go-enterprise-boilerplate/compare/HEAD
