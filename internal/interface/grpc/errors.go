package grpc

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/apperror"
)

func toStatus(err error) error {
	if err == nil {
		return nil
	}

	code := codes.Internal
	switch {
	case errors.Is(err, apperror.ErrInvalidEmail),
		errors.Is(err, apperror.ErrInvalidPassword),
		errors.Is(err, apperror.ErrInvalidName),
		errors.Is(err, apperror.ErrInvalidRole):
		code = codes.InvalidArgument
	case errors.Is(err, apperror.ErrEmailAlreadyExists):
		code = codes.AlreadyExists
	case errors.Is(err, apperror.ErrUserNotFound):
		code = codes.NotFound
	case errors.Is(err, apperror.ErrInvalidCredentials), errors.Is(err, apperror.ErrTokenInvalid):
		code = codes.Unauthenticated
	case errors.Is(err, apperror.ErrAccountInactive), errors.Is(err, apperror.ErrInsufficientPerms):
		code = codes.PermissionDenied
	case errors.Is(err, apperror.ErrServiceUnavailable):
		code = codes.Unavailable
	}

	return status.Error(code, err.Error())
}
