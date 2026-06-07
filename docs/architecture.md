# Architecture

## Overview

This project implements Clean Architecture (also known as Hexagonal Architecture or Ports & Adapters) combined with Domain-Driven Design tactical patterns. The goal is a codebase where the business rules can be read, tested, and reasoned about without any knowledge of Gin, pgx, or any other infrastructure package.

---

## Layer Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                        interface/                           │
│              (Gin HTTP handlers, gRPC services)             │
├─────────────────────────────────────────────────────────────┤
│                       application/                          │
│              (Use Cases, Input/Output Ports)                │
├─────────────────────────────────────────────────────────────┤
│                         domain/                             │
│          (Entities, Value Objects, Repository Interfaces)   │
├──────────────────────────┬──────────────────────────────────┤
│     infrastructure/      │         infrastructure/          │
│   (PostgreSQL adapter)   │      (In-Memory adapter)         │
└──────────────────────────┴──────────────────────────────────┘
```

**Dependency rule:** source code dependencies point inward only. The domain knows nothing about the layers outside it.

---

## Directory Structure

```
internal/
├── domain/
│   ├── entity/
│   │   └── user.go              # User aggregate root
│   ├── valueobject/
│   │   ├── email.go             # Email — validated on construction
│   │   ├── password_hash.go     # Opaque wrapper around hashed bytes
│   │   └── user_id.go           # UUID newtype
│   ├── repository/
│   │   └── user_repository.go   # Interface: the only contract infra must fulfill
│   └── apperror/
│       └── errors.go            # Domain error types (sentinel errors + types)
│
├── application/
│   ├── usecase/
│   │   ├── register_user.go
│   │   ├── login_user.go
│   │   ├── refresh_token.go
│   │   ├── get_user.go
│   │   ├── update_profile.go
│   │   └── change_password.go
│   ├── port/
│   │   ├── password_hasher.go   # Interface: Hash + Verify
│   │   └── token_service.go     # Interface: GeneratePair + Validate + Refresh
│   └── dto/
│       ├── auth.go              # RegisterInput, LoginInput, RefreshInput, AuthOutput, ...
│       └── user.go              # UpdateProfileInput, ChangePasswordInput, UserOutput, ...
│
├── infrastructure/
│   ├── persistence/
│   │   ├── memory/
│   │   │   └── user_repository.go
│   │   └── postgres/
│   │       └── user_repository.go   # pgx-based adapter (built but not wired by default)
│   ├── security/
│   │   ├── argon2_hasher.go
│   │   └── jwt_service.go           # Issues/validates JWTs; refresh tokens stored in Redis
│   └── telemetry/
│       └── setup.go                 # zap logger, OTLP tracing, Prometheus metrics
│
└── interface/
    ├── http/
    │   ├── router.go
    │   ├── middleware/
    │   │   ├── auth.go
    │   │   ├── cors.go
    │   │   ├── rate_limit.go
    │   │   └── security_headers.go
    │   └── handler/
    │       ├── auth_handler.go
    │       └── user_handler.go
    └── grpc/
        ├── auth_service.go
        ├── user_service.go
        ├── auth_interceptor.go
        └── errors.go

cmd/
└── server/
    └── main.go                  # Composition root
```

---

## Domain Layer

### Entities

`User` is the aggregate root. Construction goes through a `NewUser` function that returns `(*User, error)` — you cannot create a user in an invalid state. Fields are unexported; state is exposed only through methods.

### Value Objects

Value objects are immutable structs with unexported fields. `NewEmail("bad")` returns `(Email{}, ErrInvalidEmail)`. Once constructed, the value is always valid.

```go
type Email struct{ value string }

func NewEmail(v string) (Email, error) {
    // validation
    return Email{value: v}, nil
}

func (e Email) String() string { return e.value }
```

### Repository Interfaces

```go
type UserRepository interface {
    FindByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
    FindByEmail(ctx context.Context, email valueobject.Email) (*entity.User, error)
    Save(ctx context.Context, user *entity.User) error
    Delete(ctx context.Context, id uuid.UUID) error
    Count(ctx context.Context) (int64, error)
    SaveFirstOwner(ctx context.Context, user *entity.User) (bool, error)
}
```

The interface lives in `domain/repository/` — owned by the domain, not by the infrastructure that implements it.

---

## Application Layer

Each use case is a struct holding the interfaces it depends on. It exposes a single `Execute` method. No infrastructure package is imported here.

```go
type RegisterUser struct {
    users  repository.UserRepository
    hasher port.PasswordHasher
}

func (uc *RegisterUser) Execute(ctx context.Context, input dto.RegisterInput) error {
    // 1. validate input
    // 2. check uniqueness
    // 3. hash password
    // 4. construct entity
    // 5. persist
}
```

---

## Infrastructure Layer

Structs in `infrastructure/` implement the domain/application interfaces. They are the only place where `pgx`, `argon2`, `redis`, or any external package is imported.

The in-memory adapter uses a `sync.RWMutex`-protected map and is production-equivalent for the domain — it satisfies the same interface contract.

---

## Wiring (cmd/server/main.go)

`main.go` is the composition root. It reads config, builds adapters, injects them into use cases, and starts the server. It is the only file where concrete types are named.

```go
var userRepo repository.UserRepository
switch cfg.Adapter {
case "postgres":
    log.Fatal("postgres adapter: set DATABASE_URL and rebuild with postgres tag")
default:
    log.Info("using in-memory adapter")
    userRepo = memory.NewUserRepository()
}

registerUser := usecase.NewRegisterUser(userRepo, hasher)
```

The `postgres.UserRepository` adapter is fully implemented in `infrastructure/persistence/postgres/`, but `main.go` does not yet wire it up — selecting `ADAPTER=postgres` currently exits with a fatal error. Wiring it is a contribution opportunity; the adapter already satisfies the `UserRepository` port.
