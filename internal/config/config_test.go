package config_test

import (
	"testing"
	"time"

	"github.com/spf13/viper"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/config"
)

func TestLoad_ReadsKeysWithoutDefaults(t *testing.T) {
	t.Cleanup(viper.Reset)

	t.Setenv("DATABASE_URL", "postgres://test:test@localhost:5432/test")
	t.Setenv("JWT_PRIVATE_KEY_PATH", "/etc/jwt/jwt_private.pem")
	t.Setenv("JWT_PUBLIC_KEY_PATH", "/etc/jwt/jwt_public.pem")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.DatabaseURL != "postgres://test:test@localhost:5432/test" {
		t.Errorf("DatabaseURL = %q, want value from DATABASE_URL env var", cfg.DatabaseURL)
	}
	if cfg.JWTPrivateKeyPath != "/etc/jwt/jwt_private.pem" {
		t.Errorf("JWTPrivateKeyPath = %q, want value from JWT_PRIVATE_KEY_PATH env var", cfg.JWTPrivateKeyPath)
	}
	if cfg.JWTPublicKeyPath != "/etc/jwt/jwt_public.pem" {
		t.Errorf("JWTPublicKeyPath = %q, want value from JWT_PUBLIC_KEY_PATH env var", cfg.JWTPublicKeyPath)
	}
}

func TestLoad_Defaults(t *testing.T) {
	t.Cleanup(viper.Reset)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Adapter != "memory" {
		t.Errorf("Adapter = %q, want %q", cfg.Adapter, "memory")
	}
	if cfg.DatabaseURL != "" {
		t.Errorf("DatabaseURL = %q, want empty when unset", cfg.DatabaseURL)
	}
}

func TestLoad_PoolAndRedisTimeoutDefaults(t *testing.T) {
	t.Cleanup(viper.Reset)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.DBPoolMaxConns != 10 {
		t.Errorf("DBPoolMaxConns = %d, want 10", cfg.DBPoolMaxConns)
	}
	if cfg.DBPoolMinConns != 2 {
		t.Errorf("DBPoolMinConns = %d, want 2", cfg.DBPoolMinConns)
	}
	if cfg.DBPoolConnectTimeout != 30*time.Second {
		t.Errorf("DBPoolConnectTimeout = %v, want 30s", cfg.DBPoolConnectTimeout)
	}
	if cfg.DBPoolIdleTimeout != 10*time.Minute {
		t.Errorf("DBPoolIdleTimeout = %v, want 10m", cfg.DBPoolIdleTimeout)
	}
	if cfg.DBPoolMaxLifetime != 30*time.Minute {
		t.Errorf("DBPoolMaxLifetime = %v, want 30m", cfg.DBPoolMaxLifetime)
	}
	if cfg.RedisConnectTimeout != 2*time.Second {
		t.Errorf("RedisConnectTimeout = %v, want 2s", cfg.RedisConnectTimeout)
	}
	if cfg.RedisCommandTimeout != 2*time.Second {
		t.Errorf("RedisCommandTimeout = %v, want 2s", cfg.RedisCommandTimeout)
	}
}

func TestLoad_PoolAndRedisTimeoutOverrides(t *testing.T) {
	t.Cleanup(viper.Reset)

	t.Setenv("DB_POOL_MAX_CONNS", "25")
	t.Setenv("DB_POOL_MIN_CONNS", "5")
	t.Setenv("DB_POOL_CONNECT_TIMEOUT", "15s")
	t.Setenv("DB_POOL_IDLE_TIMEOUT", "5m")
	t.Setenv("DB_POOL_MAX_LIFETIME", "15m")
	t.Setenv("REDIS_CONNECT_TIMEOUT", "1500ms")
	t.Setenv("REDIS_COMMAND_TIMEOUT", "1500ms")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.DBPoolMaxConns != 25 {
		t.Errorf("DBPoolMaxConns = %d, want 25", cfg.DBPoolMaxConns)
	}
	if cfg.DBPoolConnectTimeout != 15*time.Second {
		t.Errorf("DBPoolConnectTimeout = %v, want 15s", cfg.DBPoolConnectTimeout)
	}
	if cfg.RedisCommandTimeout != 1500*time.Millisecond {
		t.Errorf("RedisCommandTimeout = %v, want 1500ms", cfg.RedisCommandTimeout)
	}
}
