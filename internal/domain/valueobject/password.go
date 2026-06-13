package valueobject

import "github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/apperror"

const (
	minPasswordLength = 12
	maxPasswordLength = 128
)

// ValidatePassword enforces the password length policy shared by registration
// and password-change flows.
func ValidatePassword(plain string) error {
	if len(plain) < minPasswordLength || len(plain) > maxPasswordLength {
		return apperror.ErrInvalidPassword
	}
	return nil
}
