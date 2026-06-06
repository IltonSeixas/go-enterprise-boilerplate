package security_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/infrastructure/security"
)

func TestArgon2Hasher_HashDiffersFromPlaintext(t *testing.T) {
	h := security.NewArgon2Hasher()
	hash, err := h.Hash("mysecretpassword")
	require.NoError(t, err)
	assert.NotEqual(t, "mysecretpassword", hash.PHC())
}

func TestArgon2Hasher_VerifyCorrectPassword(t *testing.T) {
	h := security.NewArgon2Hasher()
	hash, err := h.Hash("correctpassword1")
	require.NoError(t, err)

	ok, err := h.Verify("correctpassword1", hash)
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestArgon2Hasher_VerifyWrongPassword(t *testing.T) {
	h := security.NewArgon2Hasher()
	hash, err := h.Hash("correctpassword1")
	require.NoError(t, err)

	ok, err := h.Verify("wrongpassword123", hash)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestArgon2Hasher_TwoHashesDiffer(t *testing.T) {
	h := security.NewArgon2Hasher()
	h1, err := h.Hash("samepassword123")
	require.NoError(t, err)
	h2, err := h.Hash("samepassword123")
	require.NoError(t, err)
	assert.NotEqual(t, h1.PHC(), h2.PHC())
}
