# ADR-0006: Enforce the Layering Rule with an Automated Test

**Date:** 2026-06-19
**Status:** Accepted

---

## Context

[ADR-0001](0001-clean-architecture.md) defines a strict inward-only dependency rule between `domain/`, `application/`, `infrastructure/`, and `interface/`. Go's compiler rejects circular imports, but that says nothing about *direction* — nothing stops `internal/domain` from importing `github.com/gin-gonic/gin` directly; that import is perfectly acyclic and would compile cleanly. Until now this rule was enforced only by code review and contributor discipline.

Auditing the existing code while writing this rule found that `domain/` and `application/` already depend on `github.com/google/uuid`, a plain data/utility package with no infrastructure coupling. ADR-0001's original text ("zero external imports") did not match this reality and was stricter than necessary.

## Decision

Add `internal/architecture/layering_test.go`, a Go test using `golang.org/x/tools/go/packages` to load the real import graph of `internal/domain/...` and `internal/application/...` and assert it contains none of a list of forbidden infrastructure packages. It runs as part of the existing `go test ./...` step — no new CI stage.

The rule distinguishes between two different concerns that the original ADR conflated:

1. **Data/utility packages** (`github.com/google/uuid`) — carry no infrastructure coupling. Allowed in both `domain/` and `application/`.
2. **Infrastructure packages** (`gin-gonic/gin`, `jackc/pgx`, `redis/go-redis`, `golang-jwt/jwt`, `opentelemetry`, `prometheus/client_golang`, `go.uber.org/zap`, `spf13/viper`, `golang.org/x/crypto`, `google.golang.org/grpc`/`protobuf`, and the project's own `internal/infrastructure`/`internal/interface` packages) — couple business logic to a specific runtime, transport, or persistence choice. Forbidden in both `domain/` and `application/`.

Two checks are encoded:
- `internal/domain/...` must not import `internal/application` or any infrastructure package.
- `internal/application/...` must not import any infrastructure package.

Using `go/packages` instead of text scanning means the test follows real import resolution (including transitive package paths like `go.opentelemetry.io/otel/sdk`), not a regex over source text.

## Consequences

**Positive:**
- A pull request that violates the layering rule now fails `go test ./...` instead of relying on a reviewer noticing an import.
- The rule's text is the rule — no drift between what ADR-0001 says and what the codebase actually does.
- `golang.org/x/tools` was already an indirect dependency (pulled in by `staticcheck`'s toolchain); promoting it to direct added no new supply-chain surface.

**Negative:**
- `packages.Load` does a real type-check pass, so the test takes longer than a plain text scan (still well under a second for this codebase).
- The infrastructure-package list must be kept in sync by hand as `go.mod` changes; a newly added infrastructure dependency is invisible to the test until added to the list.

## Alternatives Considered

- **Keep enforcing via code review only** — what ADR-0001 originally specified; demonstrated to drift from reality once domain/application adopted `google/uuid`, a dependency not mentioned in the ADR's stricter wording.
- **`depguard` (golangci-lint plugin)** — can forbid import paths per package via YAML config, but this repo has no `golangci-lint` installed or configured; adopting it just for this rule would mean introducing an entire new linter and CI step for a single check.
- **`go-arch-lint`** — a dedicated architecture linter, but it is a separate binary with its own YAML config; `golang.org/x/tools/go/packages` achieves the same result as a plain Go test using a dependency already present in the module graph.
