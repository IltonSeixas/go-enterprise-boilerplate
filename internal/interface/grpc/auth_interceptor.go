package grpc

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/application/port"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/entity"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/repository"
)

type callerKey struct{}

// AuthenticatedCaller carries the identity extracted from the access token,
// mirroring the REST RequireAuth middleware (active-account check included).
type AuthenticatedCaller struct {
	ID   uuid.UUID
	Role entity.Role
}

// methodsRequiringAuth lists the full gRPC method names that must present a
// valid bearer token. AuthService methods are intentionally excluded since
// they are the entry points for obtaining tokens in the first place.
var methodsRequiringAuth = map[string]bool{
	"/boilerplate.v1.UserService/GetMe":          true,
	"/boilerplate.v1.UserService/UpdateProfile":  true,
	"/boilerplate.v1.UserService/ChangePassword": true,
}

// UnaryAuthInterceptor validates the `authorization: Bearer <token>` request
// metadata for protected RPCs and injects the resulting caller into the context.
func UnaryAuthInterceptor(tokens port.TokenService, users repository.UserRepository) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if !methodsRequiringAuth[info.FullMethod] {
			return handler(ctx, req)
		}

		caller, err := authenticate(ctx, tokens, users)
		if err != nil {
			return nil, err
		}

		return handler(context.WithValue(ctx, callerKey{}, caller), req)
	}
}

func authenticate(ctx context.Context, tokens port.TokenService, users repository.UserRepository) (AuthenticatedCaller, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return AuthenticatedCaller{}, status.Error(codes.Unauthenticated, "missing bearer token")
	}

	values := md.Get("authorization")
	if len(values) == 0 || !strings.HasPrefix(values[0], "Bearer ") {
		return AuthenticatedCaller{}, status.Error(codes.Unauthenticated, "missing bearer token")
	}

	token := strings.TrimPrefix(values[0], "Bearer ")
	claims, err := tokens.ValidateAccessToken(token)
	if err != nil {
		return AuthenticatedCaller{}, status.Error(codes.Unauthenticated, "invalid or expired token")
	}

	user, err := users.FindByID(ctx, claims.UserID)
	if err != nil || user == nil {
		return AuthenticatedCaller{}, status.Error(codes.Unauthenticated, "invalid or expired token")
	}

	if !user.IsActive() {
		return AuthenticatedCaller{}, status.Error(codes.PermissionDenied, "account is inactive")
	}

	return AuthenticatedCaller{ID: claims.UserID, Role: claims.Role}, nil
}

func callerFromContext(ctx context.Context) (AuthenticatedCaller, bool) {
	caller, ok := ctx.Value(callerKey{}).(AuthenticatedCaller)
	return caller, ok
}
