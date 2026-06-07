# ADR-0003: Stateless JWT Access Tokens with Server-Side Refresh Tokens

**Date:** 2026-06-06  
**Status:** Accepted

---

## Context

Authentication requires a balance between statelessness (horizontal scalability) and revocability (security).

## Decision

A hybrid model: stateless JWT HS256 access token (TTL 15 min, returned in the response body) + opaque UUID refresh token stored server-side in Redis (TTL 7 days, rotated on use, returned as a plain JSON field).

## Consequences

**Positive:**
- Hot path requires no database lookup — cryptographic verification only.
- Sessions are revocable by deleting the Redis key.
- Refresh token rotation detects stolen tokens on next legitimate use.

**Negative:**
- Access tokens cannot be revoked within their 15-minute window without an additional blocklist.

## Alternatives Considered

- **Pure stateless JWT** — no revocation; unacceptable for a security boilerplate.
- **Server-side sessions only** — store lookup on every request; less scalable.
- **OAuth2/OIDC** — correct for multi-service auth; out of scope for a self-contained boilerplate.
