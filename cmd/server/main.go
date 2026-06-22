package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/application/port"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/application/usecase"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/config"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/repository"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/infrastructure/audit"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/infrastructure/persistence/memory"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/infrastructure/persistence/postgres"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/infrastructure/security"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/infrastructure/telemetry"
	grpcinterface "github.com/IltonSeixas/go-enterprise-boilerplate/internal/interface/grpc"
	pb "github.com/IltonSeixas/go-enterprise-boilerplate/internal/interface/grpc/proto"
	httpinterface "github.com/IltonSeixas/go-enterprise-boilerplate/internal/interface/http"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/interface/http/handler"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	log, err := telemetry.InitLogger("go-enterprise-boilerplate")
	if err != nil {
		fmt.Fprintf(os.Stderr, "logger error: %v\n", err)
		os.Exit(1)
	}
	defer log.Sync() //nolint:errcheck

	ctx := context.Background()

	shutTrace, err := telemetry.InitTracing(ctx, "go-enterprise-boilerplate", cfg.OTLPEndpoint)
	if err != nil {
		log.Warn("tracing unavailable", zap.Error(err))
	}

	_, err = telemetry.InitPrometheus()
	if err != nil {
		log.Warn("prometheus unavailable", zap.Error(err))
	}

	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Fatal("invalid redis url", zap.Error(err))
	}
	opt.DialTimeout = cfg.RedisConnectTimeout
	opt.ReadTimeout = cfg.RedisCommandTimeout
	opt.WriteTimeout = cfg.RedisCommandTimeout
	redisClient := redis.NewClient(opt)
	defer redisClient.Close()

	var userRepo repository.UserRepository
	var dbPinger handler.Pinger
	var auditLog port.AuditPort
	switch cfg.Adapter {
	case "postgres":
		pgxCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
		if err != nil {
			log.Fatal("invalid postgres dsn", zap.Error(err))
		}
		pgxCfg.MaxConns = cfg.DBPoolMaxConns
		pgxCfg.MinConns = cfg.DBPoolMinConns
		pgxCfg.MaxConnIdleTime = cfg.DBPoolIdleTimeout
		pgxCfg.MaxConnLifetime = cfg.DBPoolMaxLifetime
		pgxCfg.ConnConfig.ConnectTimeout = cfg.DBPoolConnectTimeout

		pool, err := pgxpool.NewWithConfig(ctx, pgxCfg)
		if err != nil {
			log.Fatal("postgres connection error", zap.Error(err))
		}
		defer pool.Close()

		if err := postgres.Migrate(ctx, pool); err != nil {
			log.Fatal("postgres migration error", zap.Error(err))
		}

		log.Info("using postgres adapter")
		userRepo = postgres.NewUserRepository(pool)
		dbPinger = pool
		auditLog = audit.NewPostgresAuditLog(pool, log)
	default:
		log.Info("using in-memory adapter")
		userRepo = memory.NewUserRepository()
		auditLog = audit.NewMemoryAuditLog(log)
	}

	jwtPrivateKey, err := os.ReadFile(cfg.JWTPrivateKeyPath)
	if err != nil {
		log.Fatal("failed to read JWT_PRIVATE_KEY_PATH", zap.Error(err))
	}
	jwtPublicKey, err := os.ReadFile(cfg.JWTPublicKeyPath)
	if err != nil {
		log.Fatal("failed to read JWT_PUBLIC_KEY_PATH", zap.Error(err))
	}

	hasher := security.NewArgon2Hasher()
	tokenSvc, err := security.NewJWTService(
		jwtPrivateKey,
		jwtPublicKey,
		cfg.JWTAccessTTL,
		cfg.JWTRefreshTTL,
		redisClient,
		cfg.CircuitBreaker(),
		cfg.RetryPolicy(),
	)
	if err != nil {
		log.Fatal("failed to load Ed25519 JWT keys", zap.Error(err))
	}

	registerUser := usecase.NewRegisterUser(userRepo, hasher, tokenSvc, auditLog)
	loginUser := usecase.NewLoginUser(userRepo, hasher, tokenSvc, auditLog)
	refreshToken := usecase.NewRefreshToken(userRepo, tokenSvc, auditLog)
	getUser := usecase.NewGetUser(userRepo)
	listUsers := usecase.NewListUsers(userRepo)
	updateProfile := usecase.NewUpdateProfile(userRepo)
	changePassword := usecase.NewChangePassword(userRepo, hasher, auditLog)
	changeRole := usecase.NewChangeUserRole(userRepo, auditLog)

	authHandler := handler.NewAuthHandler(registerUser, loginUser, refreshToken)
	userHandler := handler.NewUserHandler(getUser, listUsers, updateProfile, changePassword, changeRole)
	healthHandler := handler.NewHealthHandler(handler.RedisPinger{Client: redisClient}, dbPinger)

	router := httpinterface.NewRouter(authHandler, userHandler, healthHandler, tokenSvc, userRepo, cfg.AllowedOriginList())

	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	grpcAddr := fmt.Sprintf("%s:%d", cfg.Host, cfg.GRPCPort)
	grpcListener, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatal("grpc listener error", zap.Error(err))
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(grpcinterface.UnaryAuthInterceptor(tokenSvc, userRepo)),
	)
	pb.RegisterAuthServiceServer(grpcServer, grpcinterface.NewAuthServer(registerUser, loginUser, refreshToken))
	pb.RegisterUserServiceServer(grpcServer, grpcinterface.NewUserServer(getUser, listUsers, updateProfile, changePassword, changeRole))
	reflection.Register(grpcServer)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Info("http server listening", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server error", zap.Error(err))
		}
	}()

	go func() {
		log.Info("grpc server listening", zap.String("addr", grpcAddr))
		if err := grpcServer.Serve(grpcListener); err != nil {
			log.Fatal("grpc server error", zap.Error(err))
		}
	}()

	<-quit
	log.Info("shutting down")

	shutCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutCtx); err != nil {
		log.Error("graceful shutdown failed", zap.Error(err))
	}

	grpcServer.GracefulStop()

	if shutTrace != nil {
		if err := shutTrace(shutCtx); err != nil {
			log.Warn("trace shutdown error", zap.Error(err))
		}
	}

	log.Info("server stopped")
}
