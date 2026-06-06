# ADR-0002: Use Argon2id for Password Hashing

**Date:** 2024-01-01  
**Status:** Accepted

---

## Context

Passwords must be stored as hashes that are computationally expensive to reverse. The choice of algorithm determines resistance to offline brute-force attacks after a database breach.

## Decision

**Argon2id** via `golang.org/x/crypto/argon2` — the native Go implementation, no third-party crypto dependencies. Parameters: 64 MB memory, 3 iterations, 4 lanes (OWASP recommended).

## Consequences

**Positive:**
- Argon2id is the current OWASP recommendation for new systems.
- `golang.org/x/crypto` is maintained by the Go team — no supply chain risk beyond the standard ecosystem.
- Memory-hardness resists GPU-based attacks; time-hardness resists side-channel attacks.
- `subtle.ConstantTimeCompare` used for verification — no timing oracle.

**Negative:**
- Higher CPU/memory cost than bcrypt per login — acceptable at configured parameters (~80 ms on commodity hardware).

## Alternatives Considered

- **bcrypt** — 72-byte limit, no memory-hardness, not recommended for new systems by OWASP since 2019.
- **scrypt** — memory-hard but Argon2id is preferred by OWASP.
- **PBKDF2** — FIPS-compliant but not memory-hard; significantly weaker against GPU attacks.
