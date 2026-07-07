package security_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/application/port"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/entity"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/infrastructure/resilience"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/infrastructure/security"
)

// Test-only Ed25519 key pairs, generated via:
//
//	openssl genpkey -algorithm ed25519 -out priv.pem
//	openssl pkey -in priv.pem -pubout -out pub.pem
const (
	testPrivateKeyPEM = `-----BEGIN PRIVATE KEY-----
MC4CAQAwBQYDK2VwBCIEIOm10XqWlwNs5nf2k1BcPu1Fa9jQI4pE385WjIhPnBd8
-----END PRIVATE KEY-----
`
	testPublicKeyPEM = `-----BEGIN PUBLIC KEY-----
MCowBQYDK2VwAyEAXQYaNMR6DS321R2vbUGBA+LncpfxrGWvZjj6bA9Bu2Q=
-----END PUBLIC KEY-----
`
	otherPublicKeyPEM = `-----BEGIN PUBLIC KEY-----
MCowBQYDK2VwAyEA+Z70MUVwxaJ2l7QvoKSM9zgzI/k+heI1ycwuFhL18Ts=
-----END PUBLIC KEY-----
`
)

func newTestJWTService(t *testing.T, publicKeyPEM string) *security.JWTService {
	t.Helper()

	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	svc, err := security.NewJWTService(
		[]byte(testPrivateKeyPEM),
		[]byte(publicKeyPEM),
		15*time.Minute,
		7*24*time.Hour,
		client,
		resilience.NewCircuitBreaker(5, 30*time.Second),
		resilience.NewRetryPolicy(3, 50*time.Millisecond, 2),
	)
	require.NoError(t, err)
	return svc
}

func TestJWTService_GeneratePairThenValidate_SucceedsWithEdDSA(t *testing.T) {
	svc := newTestJWTService(t, testPublicKeyPEM)
	userID := uuid.New()

	pair, err := svc.GeneratePair(context.Background(), userID, entity.RoleUser)
	require.NoError(t, err)

	claims, err := svc.ValidateAccessToken(pair.AccessToken)
	require.NoError(t, err)
	require.Equal(t, userID, claims.UserID)
	require.Equal(t, entity.RoleUser, claims.Role)
}

func TestJWTService_ValidateAccessToken_RejectsTokenSignedWithDifferentKeyPair(t *testing.T) {
	signingSvc := newTestJWTService(t, testPublicKeyPEM)
	verifyingSvc := newTestJWTService(t, otherPublicKeyPEM)

	pair, err := signingSvc.GeneratePair(context.Background(), uuid.New(), entity.RoleUser)
	require.NoError(t, err)

	_, err = verifyingSvc.ValidateAccessToken(pair.AccessToken)
	require.Error(t, err)
}

func TestJWTService_RedeemRefreshToken_ReturnsOK_WhenTokenIsLive(t *testing.T) {
	svc := newTestJWTService(t, testPublicKeyPEM)
	userID := uuid.New()

	pair, err := svc.GeneratePair(context.Background(), userID, entity.RoleUser)
	require.NoError(t, err)

	result, err := svc.RedeemRefreshToken(context.Background(), pair.RefreshToken)
	require.NoError(t, err)
	require.Equal(t, port.RedemptionOK, result.Outcome)
	require.Equal(t, userID, result.UserID)
}

func TestJWTService_RedeemRefreshToken_ReturnsInvalid_WhenTokenNeverExisted(t *testing.T) {
	svc := newTestJWTService(t, testPublicKeyPEM)

	result, err := svc.RedeemRefreshToken(context.Background(), "never-issued")
	require.NoError(t, err)
	require.Equal(t, port.RedemptionInvalid, result.Outcome)
}

func TestJWTService_RedeemRefreshToken_ReturnsReused_OnSecondRedemption(t *testing.T) {
	svc := newTestJWTService(t, testPublicKeyPEM)
	userID := uuid.New()

	pair, err := svc.GeneratePair(context.Background(), userID, entity.RoleUser)
	require.NoError(t, err)

	first, err := svc.RedeemRefreshToken(context.Background(), pair.RefreshToken)
	require.NoError(t, err)
	require.Equal(t, port.RedemptionOK, first.Outcome)

	second, err := svc.RedeemRefreshToken(context.Background(), pair.RefreshToken)
	require.NoError(t, err)
	require.Equal(t, port.RedemptionReused, second.Outcome)
	require.Equal(t, userID, second.UserID)
}

func TestJWTService_RedeemRefreshToken_ConcurrentReplay_ExactlyOneWins(t *testing.T) {
	svc := newTestJWTService(t, testPublicKeyPEM)
	userID := uuid.New()

	pair, err := svc.GeneratePair(context.Background(), userID, entity.RoleUser)
	require.NoError(t, err)

	const attempts = 20
	results := make(chan port.RedemptionOutcome, attempts)
	var wg sync.WaitGroup
	wg.Add(attempts)
	for i := 0; i < attempts; i++ {
		go func() {
			defer wg.Done()
			result, err := svc.RedeemRefreshToken(context.Background(), pair.RefreshToken)
			require.NoError(t, err)
			results <- result.Outcome
		}()
	}
	wg.Wait()
	close(results)

	var okCount, reusedCount int
	for outcome := range results {
		switch outcome {
		case port.RedemptionOK:
			okCount++
		case port.RedemptionReused:
			reusedCount++
		}
	}

	require.Equal(t, 1, okCount, "exactly one concurrent redemption should succeed")
	require.Equal(t, attempts-1, reusedCount, "every other replay should be flagged as reused")
}

func TestJWTService_RevokeAllRefreshTokens_RevokesEveryTokenForUser(t *testing.T) {
	svc := newTestJWTService(t, testPublicKeyPEM)
	userID := uuid.New()

	pairA, err := svc.GeneratePair(context.Background(), userID, entity.RoleUser)
	require.NoError(t, err)
	pairB, err := svc.GeneratePair(context.Background(), userID, entity.RoleUser)
	require.NoError(t, err)

	require.NoError(t, svc.RevokeAllRefreshTokens(context.Background(), userID))

	resultA, err := svc.RedeemRefreshToken(context.Background(), pairA.RefreshToken)
	require.NoError(t, err)
	require.Equal(t, port.RedemptionInvalid, resultA.Outcome)

	resultB, err := svc.RedeemRefreshToken(context.Background(), pairB.RefreshToken)
	require.NoError(t, err)
	require.Equal(t, port.RedemptionInvalid, resultB.Outcome)
}
