# ADR-0004: Use Gin as the HTTP Framework

**Date:** 2024-01-01  
**Status:** Accepted

---

## Context

Go's standard `net/http` is capable but requires significant boilerplate for routing, middleware, and parameter binding in a production service.

## Decision

**Gin** (gin-gonic/gin).

## Consequences

**Positive:**
- Mature, widely adopted — extensive middleware ecosystem (`gin-contrib`).
- Radix tree router — fast and predictable routing behavior.
- Built-in parameter binding and validation hooks.
- Context-based middleware chain is idiomatic Go.

**Negative:**
- Gin's `Context` type is not the standard `context.Context` — requires explicit extraction when passing to domain/application layers (which must receive only `context.Context`).

## Alternatives Considered

- **`net/http` + `gorilla/mux`** — more idiomatic but more verbose; `gorilla/mux` is now in maintenance mode.
- **Echo** — similar to Gin; slightly less ecosystem breadth.
- **Chi** — lightweight and idiomatic, but fewer built-in features for parameter binding.
- **`net/http` only** — minimal dependencies but significant routing/middleware boilerplate.
