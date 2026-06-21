package grpc_test

import (
	"context"
	"net"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/application/port"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/application/usecase"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/entity"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/valueobject"
	grpcinterface "github.com/IltonSeixas/go-enterprise-boilerplate/internal/interface/grpc"
	pb "github.com/IltonSeixas/go-enterprise-boilerplate/internal/interface/grpc/proto"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/testutil"
)

const bufSize = 1024 * 1024

type clients struct {
	auth pb.AuthServiceClient
	user pb.UserServiceClient
}

func startServer(t *testing.T) (clients, func()) {
	t.Helper()

	repo := testutil.NewStubUserRepo()
	hasher := testutil.NewStubHasher()

	e, err := valueobject.NewEmail("grpc-user@example.com")
	require.NoError(t, err)
	hash := valueobject.NewPasswordHashFromPHC("$argon2id$stub")
	user, err := entity.NewUser(e, hash, "Grpc User", entity.RoleUser)
	require.NoError(t, err)
	require.NoError(t, repo.Save(context.Background(), user))
	repo.SetFindByEmailResult(user, nil)

	tokens := testutil.NewStubTokenServiceWithClaims("valid-token", port.AccessTokenClaims{
		UserID: user.ID().UUID(),
		Role:   entity.RoleUser,
	})

	registerUser := usecase.NewRegisterUser(repo, hasher, tokens)
	loginUser := usecase.NewLoginUser(repo, hasher, tokens)
	refreshToken := usecase.NewRefreshToken(repo, tokens)
	getUser := usecase.NewGetUser(repo)
	listUsers := usecase.NewListUsers(repo)
	updateProfile := usecase.NewUpdateProfile(repo)
	changePassword := usecase.NewChangePassword(repo, hasher)
	changeRole := usecase.NewChangeUserRole(repo)

	lis := bufconn.Listen(bufSize)
	srv := grpc.NewServer(
		grpc.UnaryInterceptor(grpcinterface.UnaryAuthInterceptor(tokens, repo)),
	)
	pb.RegisterAuthServiceServer(srv, grpcinterface.NewAuthServer(registerUser, loginUser, refreshToken))
	pb.RegisterUserServiceServer(srv, grpcinterface.NewUserServer(getUser, listUsers, updateProfile, changePassword, changeRole))

	go func() {
		_ = srv.Serve(lis)
	}()

	dialer := func(context.Context, string) (net.Conn, error) { return lis.Dial() }
	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	cleanup := func() {
		_ = conn.Close()
		srv.Stop()
	}

	return clients{auth: pb.NewAuthServiceClient(conn), user: pb.NewUserServiceClient(conn)}, cleanup
}

func TestUserService_GetMe_RequiresBearerToken(t *testing.T) {
	c, cleanup := startServer(t)
	defer cleanup()

	_, err := c.user.GetMe(context.Background(), &pb.GetMeRequest{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing bearer token")
}

func TestUserService_GetMe_ReturnsProfileForAuthenticatedCaller(t *testing.T) {
	c, cleanup := startServer(t)
	defer cleanup()

	ctx := withBearerToken(context.Background(), "valid-token")
	resp, err := c.user.GetMe(ctx, &pb.GetMeRequest{})

	require.NoError(t, err)
	assert.Equal(t, "grpc-user@example.com", resp.GetEmail())
	assert.Equal(t, "Grpc User", resp.GetName())
	assert.True(t, resp.GetIsActive())
}

func TestUserService_UpdateProfile_PersistsNewName(t *testing.T) {
	c, cleanup := startServer(t)
	defer cleanup()

	ctx := withBearerToken(context.Background(), "valid-token")
	resp, err := c.user.UpdateProfile(ctx, &pb.UpdateProfileRequest{Name: "Updated Name"})

	require.NoError(t, err)
	assert.Equal(t, "Updated Name", resp.GetName())
}

func TestUserService_ChangeRole_RequiresBearerToken(t *testing.T) {
	c, cleanup := startServer(t)
	defer cleanup()

	_, err := c.user.ChangeRole(context.Background(), &pb.ChangeRoleRequest{
		UserId: uuid.NewString(),
		Role:   string(entity.RoleAdmin),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing bearer token")
}

func TestUserService_ChangeRole_OwnerCanPromoteAnotherUser(t *testing.T) {
	repo := testutil.NewStubUserRepo()
	hasher := testutil.NewStubHasher()

	ownerEmail, err := valueobject.NewEmail("owner@example.com")
	require.NoError(t, err)
	hash := valueobject.NewPasswordHashFromPHC("$argon2id$stub")
	owner, err := entity.NewUser(ownerEmail, hash, "Owner User", entity.RoleOwner)
	require.NoError(t, err)
	require.NoError(t, repo.Save(context.Background(), owner))

	targetEmail, err := valueobject.NewEmail("target@example.com")
	require.NoError(t, err)
	target, err := entity.NewUser(targetEmail, hash, "Target User", entity.RoleUser)
	require.NoError(t, err)
	require.NoError(t, repo.Save(context.Background(), target))

	tokens := testutil.NewStubTokenServiceWithClaims("owner-token", port.AccessTokenClaims{
		UserID: owner.ID().UUID(),
		Role:   entity.RoleOwner,
	})

	getUser := usecase.NewGetUser(repo)
	listUsers := usecase.NewListUsers(repo)
	updateProfile := usecase.NewUpdateProfile(repo)
	changePassword := usecase.NewChangePassword(repo, hasher)
	changeRole := usecase.NewChangeUserRole(repo)

	lis := bufconn.Listen(bufSize)
	srv := grpc.NewServer(
		grpc.UnaryInterceptor(grpcinterface.UnaryAuthInterceptor(tokens, repo)),
	)
	pb.RegisterUserServiceServer(srv, grpcinterface.NewUserServer(getUser, listUsers, updateProfile, changePassword, changeRole))

	go func() {
		_ = srv.Serve(lis)
	}()
	defer srv.Stop()

	dialer := func(context.Context, string) (net.Conn, error) { return lis.Dial() }
	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	defer conn.Close()

	client := pb.NewUserServiceClient(conn)

	ctx := withBearerToken(context.Background(), "owner-token")
	resp, err := client.ChangeRole(ctx, &pb.ChangeRoleRequest{
		UserId: target.ID().UUID().String(),
		Role:   string(entity.RoleAdmin),
	})

	require.NoError(t, err)
	assert.Equal(t, string(entity.RoleAdmin), resp.GetRole())
}

func TestAuthService_Register_RejectsDuplicateEmail(t *testing.T) {
	c, cleanup := startServer(t)
	defer cleanup()

	_, err := c.auth.Register(context.Background(), &pb.RegisterRequest{
		Email:    "grpc-user@example.com",
		Password: "another-strong-password",
		Name:     "Someone Else",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func withBearerToken(ctx context.Context, token string) context.Context {
	return metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)
}
