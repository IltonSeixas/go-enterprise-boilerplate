package config_test

import (
	"testing"

	"github.com/spf13/viper"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/config"
)

func TestLoad_ReadsKeysWithoutDefaults(t *testing.T) {
	t.Cleanup(viper.Reset)

	t.Setenv("DATABASE_URL", "postgres://test:test@localhost:5432/test")
	t.Setenv("JWT_SECRET", "test-secret-at-least-32-characters-long")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.DatabaseURL != "postgres://test:test@localhost:5432/test" {
		t.Errorf("DatabaseURL = %q, want value from DATABASE_URL env var", cfg.DatabaseURL)
	}
	if cfg.JWTSecret != "test-secret-at-least-32-characters-long" {
		t.Errorf("JWTSecret = %q, want value from JWT_SECRET env var", cfg.JWTSecret)
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
