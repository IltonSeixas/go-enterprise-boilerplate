package port

import (
	"context"

	"github.com/google/uuid"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/entity"
)

type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

type AccessTokenClaims struct {
	UserID uuid.UUID
	Role   entity.Role
}

// RedemptionOutcome is the result of atomically redeeming a refresh token.
type RedemptionOutcome int

const (
	// RedemptionInvalid means the token was never issued or has already expired.
	RedemptionInvalid RedemptionOutcome = iota
	// RedemptionOK means the token was live and has now been atomically consumed.
	RedemptionOK
	// RedemptionReused means the token was already redeemed within the reuse-detection
	// window — a replay signal. UserID is still populated so the caller can revoke the
	// rest of that user's sessions.
	RedemptionReused
)

type RedemptionResult struct {
	Outcome RedemptionOutcome
	UserID  uuid.UUID
}

type TokenService interface {
	GeneratePair(ctx context.Context, userID uuid.UUID, role entity.Role) (TokenPair, error)
	ValidateAccessToken(token string) (AccessTokenClaims, error)
	FindUserIDByRefreshToken(ctx context.Context, token string) (uuid.UUID, bool, error)
	// RedeemRefreshToken atomically consumes token: if it is live, it is marked used
	// and RedemptionOK is returned with the owning user id; if it was already
	// redeemed within the reuse-detection window, RedemptionReused is returned
	// (without further side effects) so the caller can treat this as a theft signal;
	// otherwise RedemptionInvalid is returned. This single atomic operation replaces
	// what would otherwise be a check-then-act sequence across separate Redis calls —
	// a sequence that leaves a race window where two concurrent replays of the same
	// stolen token could both read it as still valid before either finishes revoking it.
	RedeemRefreshToken(ctx context.Context, token string) (RedemptionResult, error)
	RotateRefreshToken(ctx context.Context, oldToken string, userID uuid.UUID, role entity.Role) (TokenPair, error)
	RevokeRefreshToken(ctx context.Context, token string) error
	// RevokeAllRefreshTokens revokes every refresh token issued to userID, used when
	// reuse of an already-redeemed token is detected.
	RevokeAllRefreshTokens(ctx context.Context, userID uuid.UUID) error
}
