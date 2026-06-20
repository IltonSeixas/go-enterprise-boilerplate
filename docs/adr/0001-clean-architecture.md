# ADR-0001: Adopt Clean Architecture with Hexagonal Ports & Adapters

**Date:** 2026-06-06  
**Status:** Accepted

---

## Context

A backend boilerplate needs to remain useful as requirements evolve. The most common source of pain in long-lived systems is coupling between business logic and infrastructure concerns — when swapping a database, adding a transport protocol, or testing a use case requires touching unrelated code.

## Decision

The project adopts **Clean Architecture** in its Hexagonal / Ports & Adapters form, organized into four layers with a strict inward-only dependency rule:

1. **domain/** — entities, value objects, repository interfaces. No infrastructure packages (no `gin`, `pgx`, `go-redis`, `golang-jwt`, `opentelemetry`, `grpc`); plain data/utility packages such as `google/uuid` are fine.
2. **application/** — use cases, port interfaces. Imports domain only, plus the same data/utility packages — never the infrastructure packages listed above.
3. **infrastructure/** — adapters (PostgreSQL, Redis, Argon2). Implements application ports.
4. **interface/** — HTTP handlers, gRPC services. Calls application use cases.

Go's package system enforces inward-only imports naturally — circular imports are compile errors. It cannot, on its own, stop `domain/` from importing an infrastructure package, since that import isn't circular. That gap is closed by `internal/architecture`, an automated test — see [ADR-0006](0006-architecture-layering-test.md).

## Consequences

**Positive:**
- Domain and application layers are testable without infrastructure — unit tests use hand-written stubs from `internal/testutil` that implement the same port interfaces as the production adapters.
- Swapping infrastructure requires touching only the adapter.
- Go interfaces are implicit — any struct with the right methods satisfies the contract, with no annotation required.
- The dependency rule is a compiled, automatically-enforced test rather than a convention that erodes silently over time.

**Negative:**
- More initial structure than a flat `main.go`; requires discipline to maintain.
- Indirection can make call stacks longer to trace.

## Alternatives Considered

- **Flat structure** — simple for small services but becomes unmanageable at scale.
- **Standard Go project layout** only — does not enforce business logic isolation.
