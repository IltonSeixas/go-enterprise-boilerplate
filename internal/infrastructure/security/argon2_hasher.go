package security

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/application/port"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/apperror"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/valueobject"
)

// OWASP recommended Argon2id parameters (2024).
const (
	argon2Memory      = 64 * 1024
	argon2Iterations  = 3
	argon2Parallelism = 4
	argon2SaltLen     = 16
	argon2KeyLen      = 32
)

type Argon2Hasher struct{}

func NewArgon2Hasher() *Argon2Hasher { return &Argon2Hasher{} }

func (h *Argon2Hasher) Hash(password string) (valueobject.PasswordHash, error) {
	salt := make([]byte, argon2SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return valueobject.PasswordHash{}, apperror.ErrInternal
	}

	key := argon2.IDKey([]byte(password), salt, argon2Iterations, argon2Memory, argon2Parallelism, argon2KeyLen)

	phc := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		argon2Memory,
		argon2Iterations,
		argon2Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	)

	return valueobject.NewPasswordHashFromPHC(phc), nil
}

func (h *Argon2Hasher) Verify(password string, hash valueobject.PasswordHash) (bool, error) {
	parts := strings.Split(hash.PHC(), "$")
	if len(parts) != 6 {
		return false, nil
	}

	var memory, iterations uint32
	var parallelism uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism); err != nil {
		return false, nil
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, nil
	}
	storedKey, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, nil
	}

	candidate := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, uint32(len(storedKey)))
	return subtle.ConstantTimeCompare(candidate, storedKey) == 1, nil
}

var _ port.PasswordHasher = (*Argon2Hasher)(nil)
