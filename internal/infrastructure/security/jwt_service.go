package security

import (
	"context"
	"crypto/ed25519"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/application/port"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/apperror"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/entity"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/infrastructure/resilience"
)

type jwtClaims struct {
	jwt.RegisteredClaims
	Role entity.Role `json:"role"`
}

type JWTService struct {
	privateKey   ed25519.PrivateKey
	publicKey    ed25519.PublicKey
	accessTTL    time.Duration
	refreshTTL   time.Duration
	redis        *redis.Client
	redisBreaker *resilience.CircuitBreaker
	retryPolicy  resilience.RetryPolicy
}

// reuseTombstoneTTL is how long a redeemed refresh token is remembered as
// "already used" so a replay of it can be detected as a reuse/theft signal.
// Shorter than the refresh token's own TTL — it only needs to outlast
// realistic client retry windows (network hiccups, double-submits), not the
// full session lifetime.
const reuseTombstoneTTL = 5 * time.Minute

// redeemScript atomically consumes a refresh token in a single Redis round
// trip, closing the check-then-act race that separate GET/DEL commands would
// otherwise leave open between concurrent replays of the same stolen token.
//
// KEYS[1] = used-refresh:<token> (tombstone)   KEYS[2] = refresh:<token>
// ARGV[1] = tombstone TTL (ms)
//
// Returns {"REUSED", userID} if a tombstone already exists (the token was
// already redeemed — a replay), {"OK", userID} if the token was live and is
// now atomically consumed, or {"INVALID"} if neither key exists.
var redeemScript = redis.NewScript(`
local usedVal = redis.call('GET', KEYS[1])
if usedVal then
  return {'REUSED', usedVal}
end
local userID = redis.call('GET', KEYS[2])
if not userID then
  return {'INVALID'}
end
redis.call('DEL', KEYS[2])
redis.call('SET', KEYS[1], userID, 'PX', ARGV[1])
return {'OK', userID}
`)

// privateKeyPEM and publicKeyPEM must be PKCS#8 PEM-encoded Ed25519 keys,
// e.g. generated via `openssl genpkey -algorithm ed25519`.
func NewJWTService(
	privateKeyPEM, publicKeyPEM []byte,
	accessTTL, refreshTTL time.Duration,
	redis *redis.Client,
	redisBreaker *resilience.CircuitBreaker,
	retryPolicy resilience.RetryPolicy,
) (*JWTService, error) {
	privateKey, err := jwt.ParseEdPrivateKeyFromPEM(privateKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("parse JWT private key: %w", err)
	}

	publicKey, err := jwt.ParseEdPublicKeyFromPEM(publicKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("parse JWT public key: %w", err)
	}

	return &JWTService{
		privateKey:   privateKey.(ed25519.PrivateKey),
		publicKey:    publicKey.(ed25519.PublicKey),
		accessTTL:    accessTTL,
		refreshTTL:   refreshTTL,
		redis:        redis,
		redisBreaker: redisBreaker,
		retryPolicy:  retryPolicy,
	}, nil
}

func isRedisRetryable(err error) bool {
	return err != nil && err != redis.Nil
}

func (s *JWTService) GeneratePair(ctx context.Context, userID uuid.UUID, role entity.Role) (port.TokenPair, error) {
	now := time.Now()
	claims := jwtClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.accessTTL)),
		},
		Role: role,
	}

	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims).SignedString(s.privateKey)
	if err != nil {
		return port.TokenPair{}, apperror.ErrInternal
	}

	refreshToken := uuid.New().String()

	_, err = resilience.CallWithResilience(ctx, s.redisBreaker, s.retryPolicy, isRedisRetryable,
		func(ctx context.Context) (struct{}, error) {
			pipe := s.redis.TxPipeline()
			pipe.Set(ctx, refreshKey(refreshToken), userID.String(), s.refreshTTL)
			pipe.Set(ctx, userRefreshKey(userID, refreshToken), refreshToken, s.refreshTTL)
			_, err := pipe.Exec(ctx)
			return struct{}{}, err
		})
	if err != nil {
		if errors.Is(err, apperror.ErrServiceUnavailable) {
			return port.TokenPair{}, err
		}
		return port.TokenPair{}, apperror.ErrInternal
	}

	return port.TokenPair{AccessToken: accessToken, RefreshToken: refreshToken}, nil
}

func (s *JWTService) ValidateAccessToken(token string) (port.AccessTokenClaims, error) {
	parsed, err := jwt.ParseWithClaims(token, &jwtClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodEd25519); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return s.publicKey, nil
	})
	if err != nil || !parsed.Valid {
		return port.AccessTokenClaims{}, apperror.ErrTokenInvalid
	}

	claims, ok := parsed.Claims.(*jwtClaims)
	if !ok {
		return port.AccessTokenClaims{}, apperror.ErrTokenInvalid
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return port.AccessTokenClaims{}, apperror.ErrTokenInvalid
	}

	return port.AccessTokenClaims{UserID: userID, Role: claims.Role}, nil
}

func (s *JWTService) FindUserIDByRefreshToken(ctx context.Context, token string) (uuid.UUID, bool, error) {
	stored, err := resilience.CallWithResilience(ctx, s.redisBreaker, s.retryPolicy, isRedisRetryable,
		func(ctx context.Context) (string, error) {
			return s.redis.Get(ctx, refreshKey(token)).Result()
		})
	if err == redis.Nil {
		return uuid.UUID{}, false, nil
	}
	if err != nil {
		if errors.Is(err, apperror.ErrServiceUnavailable) {
			return uuid.UUID{}, false, err
		}
		return uuid.UUID{}, false, apperror.ErrInternal
	}

	userID, err := uuid.Parse(stored)
	if err != nil {
		return uuid.UUID{}, false, apperror.ErrTokenInvalid
	}

	return userID, true, nil
}

// RedeemRefreshToken atomically consumes token via redeemScript. See
// port.TokenService.RedeemRefreshToken for the full contract.
func (s *JWTService) RedeemRefreshToken(ctx context.Context, token string) (port.RedemptionResult, error) {
	result, err := resilience.CallWithResilience(ctx, s.redisBreaker, s.retryPolicy, isRedisRetryable,
		func(ctx context.Context) ([]any, error) {
			raw, err := redeemScript.Run(ctx, s.redis,
				[]string{usedRefreshKey(token), refreshKey(token)},
				reuseTombstoneTTL.Milliseconds(),
			).Result()
			if err != nil {
				return nil, err
			}
			items, ok := raw.([]any)
			if !ok {
				return nil, fmt.Errorf("unexpected redeem script result type %T", raw)
			}
			return items, nil
		})
	if err != nil {
		if errors.Is(err, apperror.ErrServiceUnavailable) {
			return port.RedemptionResult{}, err
		}
		return port.RedemptionResult{}, apperror.ErrInternal
	}
	if len(result) == 0 {
		return port.RedemptionResult{}, apperror.ErrInternal
	}

	status, _ := result[0].(string)
	switch status {
	case "REUSED":
		userID, err := parseRedeemUserID(result)
		if err != nil {
			return port.RedemptionResult{}, err
		}
		return port.RedemptionResult{Outcome: port.RedemptionReused, UserID: userID}, nil
	case "OK":
		userID, err := parseRedeemUserID(result)
		if err != nil {
			return port.RedemptionResult{}, err
		}
		// The script already deleted refresh:<token> and wrote the tombstone;
		// the per-user index entry has no reuse-detection role, so it is
		// cleaned up here as a regular (non-atomic) follow-up delete.
		_, _ = resilience.CallWithResilience(ctx, s.redisBreaker, s.retryPolicy, isRedisRetryable,
			func(ctx context.Context) (struct{}, error) {
				return struct{}{}, s.redis.Del(ctx, userRefreshKey(userID, token)).Err()
			})
		return port.RedemptionResult{Outcome: port.RedemptionOK, UserID: userID}, nil
	default:
		return port.RedemptionResult{Outcome: port.RedemptionInvalid}, nil
	}
}

func parseRedeemUserID(result []any) (uuid.UUID, error) {
	if len(result) < 2 {
		return uuid.UUID{}, apperror.ErrInternal
	}
	raw, ok := result[1].(string)
	if !ok {
		return uuid.UUID{}, apperror.ErrInternal
	}
	userID, err := uuid.Parse(raw)
	if err != nil {
		return uuid.UUID{}, apperror.ErrTokenInvalid
	}
	return userID, nil
}

// RotateRefreshToken redeems oldToken and issues a fresh pair in one call.
// Kept as a convenience wrapper around RedeemRefreshToken + GeneratePair for
// callers that don't need to distinguish reuse from a plain invalid token.
func (s *JWTService) RotateRefreshToken(ctx context.Context, oldToken string, userID uuid.UUID, role entity.Role) (port.TokenPair, error) {
	result, err := s.RedeemRefreshToken(ctx, oldToken)
	if err != nil {
		return port.TokenPair{}, err
	}
	if result.Outcome != port.RedemptionOK || result.UserID != userID {
		return port.TokenPair{}, apperror.ErrTokenInvalid
	}

	return s.GeneratePair(ctx, userID, role)
}

func (s *JWTService) RevokeRefreshToken(ctx context.Context, token string) error {
	_, err := resilience.CallWithResilience(ctx, s.redisBreaker, s.retryPolicy, isRedisRetryable,
		func(ctx context.Context) (struct{}, error) {
			return struct{}{}, s.redis.Del(ctx, refreshKey(token)).Err()
		})
	return err
}

// RevokeAllRefreshTokens scans the per-user index and deletes every refresh
// token issued to userID, used when a reused (stolen) token is detected.
func (s *JWTService) RevokeAllRefreshTokens(ctx context.Context, userID uuid.UUID) error {
	_, err := resilience.CallWithResilience(ctx, s.redisBreaker, s.retryPolicy, isRedisRetryable,
		func(ctx context.Context) (struct{}, error) {
			pattern := userRefreshKeyPrefix(userID) + "*"
			var cursor uint64
			for {
				keys, next, err := s.redis.Scan(ctx, cursor, pattern, 100).Result()
				if err != nil {
					return struct{}{}, err
				}
				for _, indexKey := range keys {
					token := strings.TrimPrefix(indexKey, userRefreshKeyPrefix(userID))
					pipe := s.redis.TxPipeline()
					pipe.Del(ctx, refreshKey(token))
					pipe.Del(ctx, indexKey)
					if _, err := pipe.Exec(ctx); err != nil {
						return struct{}{}, err
					}
				}
				cursor = next
				if cursor == 0 {
					break
				}
			}
			return struct{}{}, nil
		})
	return err
}

var _ port.TokenService = (*JWTService)(nil)

func refreshKey(token string) string {
	return "refresh:" + token
}

func usedRefreshKey(token string) string {
	return "used-refresh:" + token
}

func userRefreshKeyPrefix(userID uuid.UUID) string {
	return "user-refresh:" + userID.String() + ":"
}

func userRefreshKey(userID uuid.UUID, token string) string {
	return userRefreshKeyPrefix(userID) + token
}
