package config

import (
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Host           string        `mapstructure:"HOST"`
	Port           int           `mapstructure:"PORT"`
	GRPCPort       int           `mapstructure:"GRPC_PORT"`
	Adapter        string        `mapstructure:"ADAPTER"`
	DatabaseURL    string        `mapstructure:"DATABASE_URL"`
	RedisURL       string        `mapstructure:"REDIS_URL"`
	JWTSecret      string        `mapstructure:"JWT_SECRET"`
	JWTAccessTTL   time.Duration `mapstructure:"JWT_ACCESS_TTL"`
	JWTRefreshTTL  time.Duration `mapstructure:"JWT_REFRESH_TTL"`
	OTLPEndpoint   string        `mapstructure:"OTLP_ENDPOINT"`
	AllowedOrigins string        `mapstructure:"ALLOWED_ORIGINS"`
}

func Load() (*Config, error) {
	viper.AutomaticEnv()

	viper.SetDefault("HOST", "0.0.0.0")
	viper.SetDefault("PORT", 8080)
	viper.SetDefault("GRPC_PORT", 50051)
	viper.SetDefault("ADAPTER", "memory")
	viper.SetDefault("REDIS_URL", "redis://localhost:6379")
	viper.SetDefault("JWT_ACCESS_TTL", 15*time.Minute)
	viper.SetDefault("JWT_REFRESH_TTL", 7*24*time.Hour)
	viper.SetDefault("OTLP_ENDPOINT", "localhost:4317")
	viper.SetDefault("ALLOWED_ORIGINS", "http://localhost:3000")

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// AllowedOriginList splits the comma-separated ALLOWED_ORIGINS value into a
// trimmed slice, ready to be passed to the CORS middleware allow-list.
func (c *Config) AllowedOriginList() []string {
	parts := strings.Split(c.AllowedOrigins, ",")
	origins := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			origins = append(origins, trimmed)
		}
	}
	return origins
}
