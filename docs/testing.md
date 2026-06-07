# Testing

## Philosophy

Tests are written before implementation (TDD). Domain entities, value objects, use cases, middleware and adapters are all tested in complete isolation — no real database, no real Redis, no network call.

The in-memory persistence adapter and the hand-written stubs in `internal/testutil` exist precisely to make the entire business logic testable without Docker, a database, or any external dependency.

---

## Running Tests

```bash
# All tests
go test ./...

# Specific package
go test ./internal/application/usecase/...

# With verbose output
go test ./... -v

# Coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

---

## Test Structure

```
internal/
├── domain/
│   ├── valueobject/
│   │   └── email_test.go              # value object tests
│   └── entity/
│       └── user_test.go               # entity invariant tests
│
├── application/
│   └── usecase/
│       ├── register_user_test.go      # use case tests with testutil stubs
│       ├── login_user_test.go
│       ├── get_user_test.go
│       ├── update_profile_test.go
│       └── change_password_test.go
│
├── infrastructure/
│   ├── persistence/memory/*_test.go   # in-memory adapter tests
│   └── security/*_test.go             # Argon2id hashing, JWT/token service tests
│
├── interface/
│   ├── grpc/*_test.go
│   └── http/middleware/*_test.go      # auth, rate limit, CORS, security headers
│
└── testutil/                          # hand-written stubs implementing port interfaces
    ├── stub_user_repo.go
    ├── stub_hasher.go
    └── stub_token_service.go
```

---

## Unit Tests

Tests live in `_test.go` files alongside the source. They cover:

- Value object construction (valid and invalid inputs)
- Entity invariant enforcement
- Use case business logic (success and failure paths)
- Middleware behavior (auth, rate limiting, CORS, security headers)

Repository and service port dependencies are replaced with hand-written stubs from `internal/testutil` that implement the same interfaces as the production adapters — there is no mocking framework and no generated code.

### Example — Value Object

```go
func TestNewEmail_ValidEmail_Succeeds(t *testing.T) {
    email, err := valueobject.NewEmail("user@example.com")
    assert.NoError(t, err)
    assert.Equal(t, "user@example.com", email.String())
}

func TestNewEmail_MissingAtSign_ReturnsError(t *testing.T) {
    _, err := valueobject.NewEmail("notanemail")
    assert.Error(t, err)
}
```

### Example — Use Case with Stub Repository

```go
func TestRegisterUser_DuplicateEmail(t *testing.T) {
    email, _ := valueobject.NewEmail("a@b.com")
    hash := valueobject.NewPasswordHashFromPHC("x")
    existing, _ := entity.NewUser(email, hash, "Existing", entity.RoleUser)

    repo := testutil.NewStubUserRepo()
    repo.SetFindByEmailResult(existing, nil)

    uc := usecase.NewRegisterUser(repo, testutil.NewStubHasher(), testutil.NewStubTokenService())
    _, err := uc.Execute(context.Background(), dto.RegisterInput{
        Email: "a@b.com", Password: "validpassword123", Name: "Test",
    })

    assert.ErrorIs(t, err, apperror.ErrEmailAlreadyExists)
}

func TestRegisterUser_FirstUserBecomesOwner(t *testing.T) {
    repo := testutil.NewStubUserRepo()
    repo.SetSaveFirstOwnerResult(true, nil)

    uc := usecase.NewRegisterUser(repo, testutil.NewStubHasher(), testutil.NewStubTokenService())
    out, err := uc.Execute(context.Background(), dto.RegisterInput{
        Email: "owner@b.com", Password: "validpassword123", Name: "Owner",
    })

    require.NoError(t, err)
    assert.Equal(t, entity.RoleOwner, out.User.Role)
}
```

---

## TDD Workflow

1. Write a failing test that describes the expected behavior
2. Run `go test ./...` — confirm it fails for the right reason
3. Write the minimum implementation to make it pass
4. Run `go test ./...` — confirm green
5. Refactor under green

Never write implementation code without a failing test first.

---

## Coverage Expectations

| Layer | Expectation |
|---|---|
| Domain (entities + value objects) | Every invariant covered, valid and invalid paths |
| Application (use cases) | Every success and failure path covered with `testutil` stubs |
| Infrastructure adapters | Behavior verified against the port contract |
| Interfaces (HTTP middleware, gRPC) | Request/response and authorization paths covered directly with `httptest` and in-process gRPC |

There is no enforced coverage threshold tool wired into the build — coverage is maintained through discipline, code review, and the TDD workflow above.
