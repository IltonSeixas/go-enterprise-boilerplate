package config

import (
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Host              string        `mapstructure:"HOST"`
	Port              int           `mapstructure:"PORT"`
	GRPCPort          int           `mapstructure:"GRPC_PORT"`
	Adapter           string        `mapstructure:"ADAPTER"`
	DatabaseURL       string        `mapstructure:"DATABASE_URL"`
	RedisURL          string        `mapstructure:"REDIS_URL"`
	JWTPrivateKeyPath string        `mapstructure:"JWT_PRIVATE_KEY_PATH"`
	JWTPublicKeyPath  string        `mapstructure:"JWT_PUBLIC_KEY_PATH"`
	JWTAccessTTL      time.Duration `mapstructure:"JWT_ACCESS_TTL"`
	JWTRefreshTTL     time.Duration `mapstructure:"JWT_REFRESH_TTL"`
	OTLPEndpoint      string        `mapstructure:"OTLP_ENDPOINT"`
	AllowedOrigins    string        `mapstructure:"ALLOWED_ORIGINS"`
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

	// viper.AutomaticEnv only binds keys it already knows about, so keys
	// with no default must be bound explicitly to be read from the
	// environment.
	for _, key := range []string{"DATABASE_URL", "JWT_PRIVATE_KEY_PATH", "JWT_PUBLIC_KEY_PATH"} {
		if err := viper.BindEnv(key); err != nil {
			return nil, err
		}
	}

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
