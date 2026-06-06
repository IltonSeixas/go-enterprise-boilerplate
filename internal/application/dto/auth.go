package dto

import (
	"github.com/google/uuid"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/entity"
)

type RegisterInput struct {
	Email    string
	Password string
	Name     string
}

type LoginInput struct {
	Email    string
	Password string
}

type RefreshInput struct {
	RefreshToken string
}

type UserSummary struct {
	ID    uuid.UUID
	Email string
	Name  string
	Role  entity.Role
}

type AuthOutput struct {
	AccessToken  string
	RefreshToken string
	User         UserSummary
}
