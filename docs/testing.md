# Testing

## Philosophy

Tests are written before implementation (TDD). The test suite is organized in two strict tiers: unit tests that run in milliseconds with no external dependencies, and integration tests that run against real infrastructure.

The in-memory adapter exists precisely to make the entire business logic testable without Docker, a database, or any network call.

---

## Running Tests

```bash
# Unit tests only (fast, no external deps)
go test ./...

# Integration tests
go test ./... -tags=integration

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
│   │   ├── email_test.go        # value object tests
│   │   └── password_hash_test.go
│   └── entity/
│       └── user_test.go         # entity invariant tests
│
├── application/
│   └── usecase/
│       ├── register_user_test.go  # use case tests with mock repos
│       └── login_user_test.go
│
└── infrastructure/
    └── persistence/
        └── postgres/
            └── user_repository_integration_test.go  # +build integration
```

---

## Unit Tests

Unit tests live in `_test.go` files alongside the source. They cover:

- Value object construction (valid and invalid inputs)
- Entity invariant enforcement
- Use case business logic (success and failure paths)

Repository and port dependencies are replaced with `gomock`-generated mocks from the interface definitions.

### Example — Value Object

```go
func TestNewEmail_ValidEmail_Succeeds(t *testing.T) {
    email, err := valueobject.NewEmail("user@example.com")
    assert.NoError(t, err)
    assert.Equal(t, "user@example.com", email.String())
}

func TestNewEmail_MissingAtSign_ReturnsError(t *testing.T) {
    _, err := valueobject.NewEmail("notanemail")
    assert.ErrorIs(t, err, apperror.ErrInvalidEmail)
}
```

### Example — Use Case with Mock

```go
func TestRegisterUser_Success(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    mockRepo := mock.NewMockUserRepository(ctrl)
    mockHasher := mock.NewMockPasswordHasher(ctrl)

    mockRepo.EXPECT().
        FindByEmail(gomock.Any(), gomock.Any()).
        Return(nil, nil) // user does not exist

    mockRepo.EXPECT().
        Save(gomock.Any(), gomock.Any()).
        Return(nil)

    mockHasher.EXPECT().
        Hash(gomock.Any()).
        Return("$argon2id$...", nil)

    uc := usecase.NewRegisterUser(mockRepo, mockHasher)
    err := uc.Execute(context.Background(), validInput())

    assert.NoError(t, err)
}

func TestRegisterUser_DuplicateEmail_ReturnsError(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    mockRepo := mock.NewMockUserRepository(ctrl)
    mockRepo.EXPECT().
        FindByEmail(gomock.Any(), gomock.Any()).
        Return(existingUser(), nil)

    // Save must NOT be called — no EXPECT for it

    uc := usecase.NewRegisterUser(mockRepo, mock.NewMockPasswordHasher(ctrl))
    err := uc.Execute(context.Background(), validInput())

    assert.ErrorIs(t, err, apperror.ErrEmailAlreadyExists)
}
```

---

## Integration Tests

Integration tests use the `//go:build integration` tag and run against a real PostgreSQL instance.

```go
//go:build integration

func TestPostgresUserRepository_SaveAndFind(t *testing.T) {
    pool := testhelper.NewTestDB(t) // creates isolated schema, registers cleanup
    repo := postgres.NewUserRepository(pool)

    user := fixture.NewUser()
    err := repo.Save(context.Background(), user)
    require.NoError(t, err)

    found, err := repo.FindByEmail(context.Background(), user.Email())
    require.NoError(t, err)
    assert.Equal(t, user.ID(), found.ID())
}
```

Each integration test uses a unique schema per test run, dropped automatically via `t.Cleanup`.

---

## TDD Workflow

1. Write a failing test that describes the expected behavior
2. Run `go test ./...` — confirm it fails for the right reason
3. Write the minimum implementation to make it pass
4. Run `go test ./...` — confirm green
5. Refactor under green

Never write implementation code without a failing test first.

---

## Coverage Target

| Layer | Target |
|---|---|
| Domain (entities + value objects) | 100% |
| Application (use cases) | 100% |
| Infrastructure adapters | 80%+ |
| HTTP handlers | 70%+ (covered by integration tests) |
