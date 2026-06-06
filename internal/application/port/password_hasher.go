package port

import "github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/valueobject"

type PasswordHasher interface {
	Hash(password string) (valueobject.PasswordHash, error)
	Verify(password string, hash valueobject.PasswordHash) (bool, error)
}
