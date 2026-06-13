package grpc

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/application/dto"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/application/usecase"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/entity"
	pb "github.com/IltonSeixas/go-enterprise-boilerplate/internal/interface/grpc/proto"
)

const timeLayout = "2006-01-02T15:04:05.999999999Z07:00"

// UserServer mirrors the REST endpoints under /api/v1/users.
// Every RPC requires an authenticated caller injected by UnaryAuthInterceptor.
type UserServer struct {
	pb.UnimplementedUserServiceServer

	getUser        *usecase.GetUser
	updateProfile  *usecase.UpdateProfile
	changePassword *usecase.ChangePassword
	changeRole     *usecase.ChangeUserRole
}

func NewUserServer(getUser *usecase.GetUser, updateProfile *usecase.UpdateProfile, changePassword *usecase.ChangePassword, changeRole *usecase.ChangeUserRole) *UserServer {
	return &UserServer{getUser: getUser, updateProfile: updateProfile, changePassword: changePassword, changeRole: changeRole}
}

func (s *UserServer) GetMe(ctx context.Context, _ *pb.GetMeRequest) (*pb.UserResponse, error) {
	caller, ok := callerFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing bearer token")
	}

	out, err := s.getUser.Execute(ctx, caller.ID, caller.Role, caller.ID)
	if err != nil {
		return nil, toStatus(err)
	}
	return toUserResponse(out), nil
}

func (s *UserServer) UpdateProfile(ctx context.Context, req *pb.UpdateProfileRequest) (*pb.UserResponse, error) {
	caller, ok := callerFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing bearer token")
	}

	out, err := s.updateProfile.Execute(ctx, caller.ID, dto.UpdateProfileInput{Name: req.GetName()})
	if err != nil {
		return nil, toStatus(err)
	}
	return toUserResponse(out), nil
}

func (s *UserServer) ChangePassword(ctx context.Context, req *pb.ChangePasswordRequest) (*pb.ChangePasswordResponse, error) {
	caller, ok := callerFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing bearer token")
	}

	err := s.changePassword.Execute(ctx, caller.ID, dto.ChangePasswordInput{
		CurrentPassword: req.GetCurrentPassword(),
		NewPassword:     req.GetNewPassword(),
	})
	if err != nil {
		return nil, toStatus(err)
	}
	return &pb.ChangePasswordResponse{}, nil
}

func (s *UserServer) ChangeRole(ctx context.Context, req *pb.ChangeRoleRequest) (*pb.UserResponse, error) {
	caller, ok := callerFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing bearer token")
	}

	if caller.Role != entity.RoleOwner {
		return nil, status.Error(codes.PermissionDenied, "insufficient permissions")
	}

	targetID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	role, err := entity.ParseRole(req.GetRole())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid role")
	}

	out, err := s.changeRole.Execute(ctx, caller.ID, targetID, dto.ChangeRoleInput{Role: role})
	if err != nil {
		return nil, toStatus(err)
	}
	return toUserResponse(out), nil
}

func toUserResponse(out dto.UserOutput) *pb.UserResponse {
	return &pb.UserResponse{
		Id:        out.ID.String(),
		Email:     out.Email,
		Name:      out.Name,
		Role:      string(out.Role),
		IsActive:  out.IsActive,
		CreatedAt: out.CreatedAt.Format(timeLayout),
		UpdatedAt: out.UpdatedAt.Format(timeLayout),
	}
}
