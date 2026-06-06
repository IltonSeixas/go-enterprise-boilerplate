package valueobject_test

import (
	"strings"
	"testing"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/apperror"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/valueobject"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEmail(t *testing.T) {
	t.Run("valid email is accepted", func(t *testing.T) {
		email, err := valueobject.NewEmail("user@example.com")
		require.NoError(t, err)
		assert.Equal(t, "user@example.com", email.String())
	})

	t.Run("email is lowercased", func(t *testing.T) {
		email, err := valueobject.NewEmail("User@EXAMPLE.COM")
		require.NoError(t, err)
		assert.Equal(t, "user@example.com", email.String())
	})

	t.Run("missing at sign is rejected", func(t *testing.T) {
		_, err := valueobject.NewEmail("notanemail")
		assert.ErrorIs(t, err, apperror.ErrInvalidEmail)
	})

	t.Run("missing domain is rejected", func(t *testing.T) {
		_, err := valueobject.NewEmail("user@")
		assert.ErrorIs(t, err, apperror.ErrInvalidEmail)
	})

	t.Run("missing tld is rejected", func(t *testing.T) {
		_, err := valueobject.NewEmail("user@domain")
		assert.ErrorIs(t, err, apperror.ErrInvalidEmail)
	})

	t.Run("empty string is rejected", func(t *testing.T) {
		_, err := valueobject.NewEmail("")
		assert.ErrorIs(t, err, apperror.ErrInvalidEmail)
	})

	t.Run("too long is rejected", func(t *testing.T) {
		long := strings.Repeat("a", 250) + "@example.com"
		_, err := valueobject.NewEmail(long)
		assert.ErrorIs(t, err, apperror.ErrInvalidEmail)
	})
}
