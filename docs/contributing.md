# Contributing

Contributions are welcome. Please read this document before opening a pull request.

---

## Prerequisites

- Go 1.25+
- `govulncheck`: `go install golang.org/x/vuln/cmd/govulncheck@latest`
- `staticcheck`: `go install honnef.co/go/tools/cmd/staticcheck@latest`

---

## Development Workflow

```bash
# Build
go build ./...

# Run all tests
go test ./...

# Run linter
staticcheck ./...

# Vet
go vet ./...

# Security audit
govulncheck ./...
```

All of the above run automatically in CI on every pull request. A PR will not be merged if any of these steps fail.

---

## Code Standards

### Architecture

- Never import infrastructure packages from `internal/domain/` or `internal/application/`
- Every new use case must have a corresponding `_test.go` file
- Every new value object must validate its invariants in the constructor and have tests for both valid and invalid inputs

### Style

- Follow `gofmt` formatting — enforced by CI
- Zero `go vet` and `staticcheck` warnings
- No `panic()` in non-test code — return errors explicitly
- Errors must be wrapped with context: `fmt.Errorf("register user: %w", err)`
- No comments that explain *what* the code does — only *why* when non-obvious

### Tests

- New behavior requires a test written first (TDD)
- Use the hand-written stubs in `internal/testutil` to satisfy repository and service ports — never wire real infrastructure into a unit test
- New stubs must implement the same port interface as the production adapter, keeping use case tests fully isolated

---

## Pull Request Guidelines

1. Fork the repository and create a branch from `main`
2. Branch naming: `feat/short-description`, `fix/short-description`, `docs/short-description`
3. Keep each PR focused on a single concern
4. Include tests for every behavior change
5. Update relevant documentation in `docs/` if the change affects it
6. Ensure CI passes before requesting review

---

## Commit Convention

```
feat: add password reset use case
fix: correct argon2 salt generation
docs: update security configuration reference
refactor: extract email validation into value object
test: add integration test for login flow
chore: update dependencies
```

---

## Reporting Security Vulnerabilities

Do **not** open a public GitHub issue for security vulnerabilities.

Send a private disclosure to [contact@iltonseixas.com](mailto:contact@iltonseixas.com) with:
- A description of the vulnerability
- Steps to reproduce
- Potential impact

You will receive a response within 72 hours.

---

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
