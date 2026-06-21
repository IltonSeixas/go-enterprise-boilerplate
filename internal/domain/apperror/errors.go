package apperror

import "errors"

var (
	ErrInvalidEmail       = errors.New("invalid email address")
	ErrInvalidPassword    = errors.New("password must be between 12 and 128 characters")
	ErrInvalidName        = errors.New("name must be between 1 and 100 characters")
	ErrUserNotFound       = errors.New("user not found")
	ErrEmailAlreadyExists = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrAccountInactive    = errors.New("account is inactive")
	ErrInsufficientPerms  = errors.New("insufficient permissions")
	ErrInternal           = errors.New("internal error")
	ErrTokenInvalid       = errors.New("token is invalid or expired")
	ErrInvalidRole        = errors.New("invalid role")
	ErrServiceUnavailable = errors.New("service temporarily unavailable")
)
