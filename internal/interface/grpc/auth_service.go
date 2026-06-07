package grpc

import (
	"context"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/application/dto"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/application/usecase"
	pb "github.com/IltonSeixas/go-enterprise-boilerplate/internal/interface/grpc/proto"
)

// AuthServer mirrors the REST endpoints under /api/v1/auth.
type AuthServer struct {
	pb.UnimplementedAuthServiceServer

	register *usecase.RegisterUser
	login    *usecase.LoginUser
	refresh  *usecase.RefreshToken
}

func NewAuthServer(register *usecase.RegisterUser, login *usecase.LoginUser, refresh *usecase.RefreshToken) *AuthServer {
	return &AuthServer{register: register, login: login, refresh: refresh}
}

func (s *AuthServer) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.AuthResponse, error) {
	out, err := s.register.Execute(ctx, dto.RegisterInput{
		Email:    req.GetEmail(),
		Password: req.GetPassword(),
		Name:     req.GetName(),
	})
	if err != nil {
		return nil, toStatus(err)
	}
	return toAuthResponse(out), nil
}

func (s *AuthServer) Login(ctx context.Context, req *pb.LoginRequest) (*pb.AuthResponse, error) {
	out, err := s.login.Execute(ctx, dto.LoginInput{
		Email:    req.GetEmail(),
		Password: req.GetPassword(),
	})
	if err != nil {
		return nil, toStatus(err)
	}
	return toAuthResponse(out), nil
}

func (s *AuthServer) RefreshToken(ctx context.Context, req *pb.RefreshTokenRequest) (*pb.AuthResponse, error) {
	out, err := s.refresh.Execute(ctx, dto.RefreshInput{
		RefreshToken: req.GetRefreshToken(),
	})
	if err != nil {
		return nil, toStatus(err)
	}
	return toAuthResponse(out), nil
}

func toAuthResponse(out dto.AuthOutput) *pb.AuthResponse {
	return &pb.AuthResponse{
		AccessToken:  out.AccessToken,
		RefreshToken: out.RefreshToken,
		User: &pb.UserSummary{
			Id:    out.User.ID.String(),
			Email: out.User.Email,
			Name:  out.User.Name,
			Role:  string(out.User.Role),
		},
	}
}
