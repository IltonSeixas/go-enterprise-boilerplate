package testutil

import (
	"context"
	"sort"
	"sync"

	"github.com/google/uuid"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/application/port"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/apperror"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/entity"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/repository"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/valueobject"
)

// StubUserRepo is a minimal in-memory stub for unit tests.
type StubUserRepo struct {
	mu sync.RWMutex

	findByEmailUser *entity.User
	findByEmailErr  error

	saveFirstOwnerClaimed bool
	saveFirstOwnerErr     error

	store map[uuid.UUID]*entity.User
}

func NewStubUserRepo() *StubUserRepo {
	return &StubUserRepo{
		store:                 make(map[uuid.UUID]*entity.User),
		saveFirstOwnerClaimed: true,
	}
}

func (s *StubUserRepo) SetFindByEmailResult(u *entity.User, err error) {
	s.findByEmailUser = u
	s.findByEmailErr = err
}

func (s *StubUserRepo) SetSaveFirstOwnerResult(claimed bool, err error) {
	s.saveFirstOwnerClaimed = claimed
	s.saveFirstOwnerErr = err
}

func (s *StubUserRepo) FindByID(_ context.Context, id uuid.UUID) (*entity.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.store[id]
	if !ok {
		return nil, apperror.ErrUserNotFound
	}
	return u, nil
}

func (s *StubUserRepo) FindByEmail(_ context.Context, _ valueobject.Email) (*entity.User, error) {
	return s.findByEmailUser, s.findByEmailErr
}

func (s *StubUserRepo) Save(_ context.Context, u *entity.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.store[u.ID().UUID()] = u
	return nil
}

func (s *StubUserRepo) Delete(_ context.Context, id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.store, id)
	return nil
}

func (s *StubUserRepo) Count(_ context.Context) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return int64(len(s.store)), nil
}

func (s *StubUserRepo) SaveFirstOwner(_ context.Context, u *entity.User) (bool, error) {
	if s.saveFirstOwnerErr != nil {
		return false, s.saveFirstOwnerErr
	}
	if s.saveFirstOwnerClaimed {
		s.mu.Lock()
		s.store[u.ID().UUID()] = u
		s.mu.Unlock()
	}
	return s.saveFirstOwnerClaimed, nil
}

func (s *StubUserRepo) FindPaginated(_ context.Context, offset, limit int64) ([]*entity.User, int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	users := make([]*entity.User, 0, len(s.store))
	for _, u := range s.store {
		users = append(users, u)
	}
	sort.Slice(users, func(i, j int) bool {
		if users[i].CreatedAt().Equal(users[j].CreatedAt()) {
			return users[i].ID().UUID().String() < users[j].ID().UUID().String()
		}
		return users[i].CreatedAt().Before(users[j].CreatedAt())
	})

	total := int64(len(users))
	start := offset
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}

	page := make([]*entity.User, end-start)
	copy(page, users[start:end])
	return page, total, nil
}

var _ repository.UserRepository = (*StubUserRepo)(nil)

// StubHasher always succeeds with a fixed PHC string.
type StubHasher struct{}

func NewStubHasher() *StubHasher { return &StubHasher{} }

func (h *StubHasher) Hash(_ string) (valueobject.PasswordHash, error) {
	return valueobject.NewPasswordHashFromPHC("$argon2id$stub"), nil
}

func (h *StubHasher) Verify(_ string, _ valueobject.PasswordHash) (bool, error) {
	return true, nil
}

var _ port.PasswordHasher = (*StubHasher)(nil)

// StubHasherRejectAll rejects all verification attempts.
type StubHasherRejectAll struct{}

func NewStubHasherRejectAll() *StubHasherRejectAll { return &StubHasherRejectAll{} }

func (h *StubHasherRejectAll) Hash(_ string) (valueobject.PasswordHash, error) {
	return valueobject.NewPasswordHashFromPHC("$argon2id$stub"), nil
}

func (h *StubHasherRejectAll) Verify(_ string, _ valueobject.PasswordHash) (bool, error) {
	return false, nil
}

var _ port.PasswordHasher = (*StubHasherRejectAll)(nil)

// StubTokenServiceRejectAll rejects all token validation attempts.
type StubTokenServiceRejectAll struct{}

func NewStubTokenServiceRejectAll() *StubTokenServiceRejectAll {
	return &StubTokenServiceRejectAll{}
}

func (s *StubTokenServiceRejectAll) GeneratePair(_ context.Context, _ uuid.UUID, _ entity.Role) (port.TokenPair, error) {
	return port.TokenPair{}, apperror.ErrTokenInvalid
}

func (s *StubTokenServiceRejectAll) ValidateAccessToken(_ string) (port.AccessTokenClaims, error) {
	return port.AccessTokenClaims{}, apperror.ErrTokenInvalid
}

func (s *StubTokenServiceRejectAll) FindUserIDByRefreshToken(_ context.Context, _ string) (uuid.UUID, bool, error) {
	return uuid.UUID{}, false, nil
}

func (s *StubTokenServiceRejectAll) RedeemRefreshToken(_ context.Context, _ string) (port.RedemptionResult, error) {
	return port.RedemptionResult{Outcome: port.RedemptionInvalid}, nil
}

func (s *StubTokenServiceRejectAll) RotateRefreshToken(_ context.Context, _ string, _ uuid.UUID, _ entity.Role) (port.TokenPair, error) {
	return port.TokenPair{}, apperror.ErrTokenInvalid
}

func (s *StubTokenServiceRejectAll) RevokeRefreshToken(_ context.Context, _ string) error { return nil }

func (s *StubTokenServiceRejectAll) RevokeAllRefreshTokens(_ context.Context, _ uuid.UUID) error {
	return nil
}

var _ port.TokenService = (*StubTokenServiceRejectAll)(nil)

// StubTokenServiceWithClaims validates a fixed token and returns preset claims.
type StubTokenServiceWithClaims struct {
	ValidToken        string
	ValidRefreshToken string
	Claims            port.AccessTokenClaims
}

func NewStubTokenServiceWithClaims(token string, claims port.AccessTokenClaims) *StubTokenServiceWithClaims {
	return &StubTokenServiceWithClaims{ValidToken: token, ValidRefreshToken: "refresh-stub", Claims: claims}
}

func (s *StubTokenServiceWithClaims) GeneratePair(_ context.Context, id uuid.UUID, role entity.Role) (port.TokenPair, error) {
	return port.TokenPair{AccessToken: s.ValidToken, RefreshToken: "refresh-stub"}, nil
}

func (s *StubTokenServiceWithClaims) ValidateAccessToken(token string) (port.AccessTokenClaims, error) {
	if token != s.ValidToken {
		return port.AccessTokenClaims{}, apperror.ErrTokenInvalid
	}
	return s.Claims, nil
}

func (s *StubTokenServiceWithClaims) FindUserIDByRefreshToken(_ context.Context, token string) (uuid.UUID, bool, error) {
	if token != s.ValidRefreshToken {
		return uuid.UUID{}, false, nil
	}
	return s.Claims.UserID, true, nil
}

func (s *StubTokenServiceWithClaims) RedeemRefreshToken(_ context.Context, token string) (port.RedemptionResult, error) {
	if token != s.ValidRefreshToken {
		return port.RedemptionResult{Outcome: port.RedemptionInvalid}, nil
	}
	return port.RedemptionResult{Outcome: port.RedemptionOK, UserID: s.Claims.UserID}, nil
}

func (s *StubTokenServiceWithClaims) RotateRefreshToken(_ context.Context, _ string, id uuid.UUID, role entity.Role) (port.TokenPair, error) {
	return port.TokenPair{AccessToken: s.ValidToken, RefreshToken: "refresh-new"}, nil
}

func (s *StubTokenServiceWithClaims) RevokeRefreshToken(_ context.Context, _ string) error {
	return nil
}

func (s *StubTokenServiceWithClaims) RevokeAllRefreshTokens(_ context.Context, _ uuid.UUID) error {
	return nil
}

var _ port.TokenService = (*StubTokenServiceWithClaims)(nil)

// StubTokenService returns fixed tokens without I/O.
type StubTokenService struct{}

func NewStubTokenService() *StubTokenService { return &StubTokenService{} }

func (s *StubTokenService) GeneratePair(_ context.Context, _ uuid.UUID, _ entity.Role) (port.TokenPair, error) {
	return port.TokenPair{AccessToken: "access-stub", RefreshToken: "refresh-stub"}, nil
}

func (s *StubTokenService) ValidateAccessToken(_ string) (port.AccessTokenClaims, error) {
	return port.AccessTokenClaims{UserID: uuid.New(), Role: entity.RoleUser}, nil
}

func (s *StubTokenService) FindUserIDByRefreshToken(_ context.Context, _ string) (uuid.UUID, bool, error) {
	return uuid.New(), true, nil
}

func (s *StubTokenService) RedeemRefreshToken(_ context.Context, _ string) (port.RedemptionResult, error) {
	return port.RedemptionResult{Outcome: port.RedemptionOK, UserID: uuid.New()}, nil
}

func (s *StubTokenService) RotateRefreshToken(_ context.Context, _ string, id uuid.UUID, role entity.Role) (port.TokenPair, error) {
	return port.TokenPair{AccessToken: "access-new", RefreshToken: "refresh-new"}, nil
}

func (s *StubTokenService) RevokeRefreshToken(_ context.Context, _ string) error { return nil }

func (s *StubTokenService) RevokeAllRefreshTokens(_ context.Context, _ uuid.UUID) error { return nil }

var _ port.TokenService = (*StubTokenService)(nil)

// StubTokenServiceReusedToken simulates RedeemRefreshToken reporting a reuse
// signal for every token, and records whether RevokeAllRefreshTokens was called.
type StubTokenServiceReusedToken struct {
	UserID     uuid.UUID
	RevokedAll bool
}

func (s *StubTokenServiceReusedToken) GeneratePair(_ context.Context, _ uuid.UUID, _ entity.Role) (port.TokenPair, error) {
	return port.TokenPair{}, apperror.ErrInternal
}

func (s *StubTokenServiceReusedToken) ValidateAccessToken(_ string) (port.AccessTokenClaims, error) {
	return port.AccessTokenClaims{}, apperror.ErrTokenInvalid
}

func (s *StubTokenServiceReusedToken) FindUserIDByRefreshToken(_ context.Context, _ string) (uuid.UUID, bool, error) {
	return uuid.UUID{}, false, nil
}

func (s *StubTokenServiceReusedToken) RedeemRefreshToken(_ context.Context, _ string) (port.RedemptionResult, error) {
	return port.RedemptionResult{Outcome: port.RedemptionReused, UserID: s.UserID}, nil
}

func (s *StubTokenServiceReusedToken) RotateRefreshToken(_ context.Context, _ string, _ uuid.UUID, _ entity.Role) (port.TokenPair, error) {
	return port.TokenPair{}, apperror.ErrTokenInvalid
}

func (s *StubTokenServiceReusedToken) RevokeRefreshToken(_ context.Context, _ string) error {
	return nil
}

func (s *StubTokenServiceReusedToken) RevokeAllRefreshTokens(_ context.Context, userID uuid.UUID) error {
	s.RevokedAll = true
	return nil
}

var _ port.TokenService = (*StubTokenServiceReusedToken)(nil)

// StubPinger returns a fixed error without I/O.
type StubPinger struct {
	Err error
}

func NewStubPinger(err error) *StubPinger { return &StubPinger{Err: err} }

func (p *StubPinger) Ping(_ context.Context) error { return p.Err }

// StubAuditPort records every event it receives for later assertions.
type StubAuditPort struct {
	mu     sync.Mutex
	Events []entity.AuditEvent
}

func NewStubAuditPort() *StubAuditPort { return &StubAuditPort{} }

func (s *StubAuditPort) Record(_ context.Context, event entity.AuditEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Events = append(s.Events, event)
}

var _ port.AuditPort = (*StubAuditPort)(nil)
