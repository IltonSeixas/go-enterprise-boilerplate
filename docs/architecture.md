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
│   │   └── logout_user.go
│   ├── port/
│   │   ├── password_hasher.go   # Interface: Hash + Verify
│   │   └── token_issuer.go      # Interface: Issue + Validate JWT
│   └── dto/
│       ├── register_input.go
│       └── auth_output.go
│
├── infrastructure/
│   ├── persistence/
│   │   ├── memory/
│   │   │   └── user_repository.go
│   │   └── postgres/
│   │       └── user_repository.go
│   ├── security/
│   │   ├── argon2_hasher.go
│   │   └── jwt_service.go
│   ├── cache/
│   │   └── redis_store.go
│   └── telemetry/
│       └── setup.go
│
└── interface/
    ├── http/
    │   ├── router.go
    │   ├── middleware/
    │   │   ├── auth.go
    │   │   ├── rate_limit.go
    │   │   └── security_headers.go
    │   └── handler/
    │       ├── auth_handler.go
    │       └── user_handler.go
    └── grpc/
        └── user_service.go

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
    FindByEmail(ctx context.Context, email valueobject.Email) (*entity.User, error)
    Save(ctx context.Context, user *entity.User) error
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
    userRepo = postgres.NewUserRepository(pool)
default:
    userRepo = memory.NewUserRepository()
}

registerUser := usecase.NewRegisterUser(userRepo, hasher)
```
