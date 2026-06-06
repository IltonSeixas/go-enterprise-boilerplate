package config

import (
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Host             string        `mapstructure:"HOST"`
	Port             int           `mapstructure:"PORT"`
	Adapter          string        `mapstructure:"ADAPTER"`
	DatabaseURL      string        `mapstructure:"DATABASE_URL"`
	RedisURL         string        `mapstructure:"REDIS_URL"`
	JWTSecret        string        `mapstructure:"JWT_SECRET"`
	JWTAccessTTL     time.Duration `mapstructure:"JWT_ACCESS_TTL"`
	JWTRefreshTTL    time.Duration `mapstructure:"JWT_REFRESH_TTL"`
	OTLPEndpoint     string        `mapstructure:"OTLP_ENDPOINT"`
}

func Load() (*Config, error) {
	viper.AutomaticEnv()

	viper.SetDefault("HOST", "0.0.0.0")
	viper.SetDefault("PORT", 8080)
	viper.SetDefault("ADAPTER", "memory")
	viper.SetDefault("REDIS_URL", "redis://localhost:6379")
	viper.SetDefault("JWT_ACCESS_TTL", 15*time.Minute)
	viper.SetDefault("JWT_REFRESH_TTL", 7*24*time.Hour)
	viper.SetDefault("OTLP_ENDPOINT", "localhost:4317")

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
