package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/application/usecase"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/config"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/repository"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/infrastructure/persistence/memory"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/infrastructure/security"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/infrastructure/telemetry"
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
	redisClient := redis.NewClient(opt)
	defer redisClient.Close()

	var userRepo repository.UserRepository
	switch cfg.Adapter {
	case "postgres":
		log.Fatal("postgres adapter: set DATABASE_URL and rebuild with postgres tag")
	default:
		log.Info("using in-memory adapter")
		userRepo = memory.NewUserRepository()
	}

	hasher := security.NewArgon2Hasher()
	tokenSvc := security.NewJWTService(
		cfg.JWTSecret,
		cfg.JWTAccessTTL,
		cfg.JWTRefreshTTL,
		redisClient,
	)

	authHandler := handler.NewAuthHandler(
		usecase.NewRegisterUser(userRepo, hasher, tokenSvc),
		usecase.NewLoginUser(userRepo, hasher, tokenSvc),
		usecase.NewRefreshToken(userRepo, tokenSvc),
	)

	userHandler := handler.NewUserHandler(
		usecase.NewGetUser(userRepo),
		usecase.NewUpdateProfile(userRepo),
		usecase.NewChangePassword(userRepo, hasher),
	)

	router := httpinterface.NewRouter(authHandler, userHandler, tokenSvc, userRepo)

	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Info("server listening", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server error", zap.Error(err))
		}
	}()

	<-quit
	log.Info("shutting down")

	shutCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutCtx); err != nil {
		log.Error("graceful shutdown failed", zap.Error(err))
	}

	if shutTrace != nil {
		if err := shutTrace(shutCtx); err != nil {
			log.Warn("trace shutdown error", zap.Error(err))
		}
	}

	log.Info("server stopped")
}
