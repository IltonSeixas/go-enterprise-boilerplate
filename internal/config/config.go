package config

import (
	"strings"
	"time"

	"github.com/spf13/viper"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/infrastructure/resilience"
)

type Config struct {
	Host                   string        `mapstructure:"HOST"`
	Port                   int           `mapstructure:"PORT"`
	GRPCPort               int           `mapstructure:"GRPC_PORT"`
	Adapter                string        `mapstructure:"ADAPTER"`
	DatabaseURL            string        `mapstructure:"DATABASE_URL"`
	RedisURL               string        `mapstructure:"REDIS_URL"`
	JWTPrivateKeyPath      string        `mapstructure:"JWT_PRIVATE_KEY_PATH"`
	JWTPublicKeyPath       string        `mapstructure:"JWT_PUBLIC_KEY_PATH"`
	JWTAccessTTL           time.Duration `mapstructure:"JWT_ACCESS_TTL"`
	JWTRefreshTTL          time.Duration `mapstructure:"JWT_REFRESH_TTL"`
	OTLPEndpoint           string        `mapstructure:"OTLP_ENDPOINT"`
	AllowedOrigins         string        `mapstructure:"ALLOWED_ORIGINS"`
	TrustedProxies         string        `mapstructure:"TRUSTED_PROXIES"`
	CircuitFailThreshold   int           `mapstructure:"CIRCUIT_FAILURE_THRESHOLD"`
	CircuitResetTimeout    time.Duration `mapstructure:"CIRCUIT_RESET_TIMEOUT"`
	RetryMaxAttempts       int           `mapstructure:"RETRY_MAX_ATTEMPTS"`
	RetryInitialBackoff    time.Duration `mapstructure:"RETRY_INITIAL_BACKOFF"`
	RetryBackoffMultiplier int           `mapstructure:"RETRY_BACKOFF_MULTIPLIER"`
	DBPoolMaxConns         int32         `mapstructure:"DB_POOL_MAX_CONNS"`
	DBPoolMinConns         int32         `mapstructure:"DB_POOL_MIN_CONNS"`
	DBPoolConnectTimeout   time.Duration `mapstructure:"DB_POOL_CONNECT_TIMEOUT"`
	DBPoolIdleTimeout      time.Duration `mapstructure:"DB_POOL_IDLE_TIMEOUT"`
	DBPoolMaxLifetime      time.Duration `mapstructure:"DB_POOL_MAX_LIFETIME"`
	RedisConnectTimeout    time.Duration `mapstructure:"REDIS_CONNECT_TIMEOUT"`
	RedisCommandTimeout    time.Duration `mapstructure:"REDIS_COMMAND_TIMEOUT"`
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
	// Empty by default: Gin trusts no proxy and reads the client IP from the
	// raw socket address unless this is set to the load balancer/ingress
	// CIDRs in front of the service, which is required for rate limiting by
	// client IP to be meaningful behind a reverse proxy.
	viper.SetDefault("TRUSTED_PROXIES", "")
	viper.SetDefault("CIRCUIT_FAILURE_THRESHOLD", 5)
	viper.SetDefault("CIRCUIT_RESET_TIMEOUT", 30*time.Second)
	viper.SetDefault("RETRY_MAX_ATTEMPTS", 3)
	viper.SetDefault("RETRY_INITIAL_BACKOFF", 50*time.Millisecond)
	viper.SetDefault("RETRY_BACKOFF_MULTIPLIER", 2)
	viper.SetDefault("DB_POOL_MAX_CONNS", 10)
	viper.SetDefault("DB_POOL_MIN_CONNS", 2)
	viper.SetDefault("DB_POOL_CONNECT_TIMEOUT", 30*time.Second)
	viper.SetDefault("DB_POOL_IDLE_TIMEOUT", 10*time.Minute)
	viper.SetDefault("DB_POOL_MAX_LIFETIME", 30*time.Minute)
	viper.SetDefault("REDIS_CONNECT_TIMEOUT", 2*time.Second)
	viper.SetDefault("REDIS_COMMAND_TIMEOUT", 2*time.Second)

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
	return splitTrimmed(c.AllowedOrigins)
}

// TrustedProxyList splits the comma-separated TRUSTED_PROXIES value into a
// trimmed slice of IPs/CIDRs, ready to be passed to gin.Engine.SetTrustedProxies.
func (c *Config) TrustedProxyList() []string {
	return splitTrimmed(c.TrustedProxies)
}

func splitTrimmed(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// CircuitBreaker builds a CircuitBreaker from the configured threshold and
// reset timeout.
func (c *Config) CircuitBreaker() *resilience.CircuitBreaker {
	return resilience.NewCircuitBreaker(c.CircuitFailThreshold, c.CircuitResetTimeout)
}

// RetryPolicy builds a RetryPolicy from the configured retry settings.
func (c *Config) RetryPolicy() resilience.RetryPolicy {
	return resilience.NewRetryPolicy(c.RetryMaxAttempts, c.RetryInitialBackoff, c.RetryBackoffMultiplier)
}
