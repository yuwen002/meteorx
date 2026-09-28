package service

import (
	"context"
	"testing"
	"time"

	"meteorx/internal/config"
	"meteorx/internal/modules/auth/dto"
	"meteorx/internal/modules/auth/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockAPITokenRepo struct {
	mock.Mock
}

func (m *mockAPITokenRepo) Create(ctx context.Context, t *model.APIToken) error {
	args := m.Called(ctx, t)
	return args.Error(0)
}

func (m *mockAPITokenRepo) GetByTokenHash(ctx context.Context, hash string) (*model.APIToken, error) {
	args := m.Called(ctx, hash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.APIToken), args.Error(1)
}

func (m *mockAPITokenRepo) ListByUserID(ctx context.Context, userID string) ([]*model.APIToken, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.APIToken), args.Error(1)
}

func (m *mockAPITokenRepo) RevokeByID(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockAPITokenRepo) GetByID(ctx context.Context, id string) (*model.APIToken, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.APIToken), args.Error(1)
}

func (m *mockAPITokenRepo) UpdateLastUsedAt(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func newTestAPITokenSvc(repo *mockAPITokenRepo) *APITokenService {
	return NewAPITokenService(repo, nil, nil, nil, nil, config.AuthConfig{})
}

func TestCreateAPIToken_Success(t *testing.T) {
	repo := new(mockAPITokenRepo)
	svc := newTestAPITokenSvc(repo)

	repo.On("ListByUserID", mock.Anything, "user-1").Return([]*model.APIToken{}, nil)
	repo.On("Create", mock.Anything, mock.AnythingOfType("*model.APIToken")).Return(nil)

	resp, err := svc.Create(context.Background(), "user-1", "tenant-1", dto.CreateAPITokenReq{
		Name: "ci-token",
	})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "ci-token", resp.Name)
	assert.NotEmpty(t, resp.Token)
	assert.True(t, len(resp.Token) > 10)
	repo.AssertExpectations(t)
}

func TestCreateAPIToken_MaxExceeded(t *testing.T) {
	repo := new(mockAPITokenRepo)
	svc := newTestAPITokenSvc(repo)

	existing := make([]*model.APIToken, maxTokensPerUser)
	for i := range existing {
		existing[i] = &model.APIToken{ID: "t"}
	}
	repo.On("ListByUserID", mock.Anything, "user-1").Return(existing, nil)

	_, err := svc.Create(context.Background(), "user-1", "tenant-1", dto.CreateAPITokenReq{
		Name: "overflow",
	})
	assert.ErrorIs(t, err, ErrAPITokenMaxExceeded)
}

func TestCreateAPIToken_InvalidExpiresIn(t *testing.T) {
	repo := new(mockAPITokenRepo)
	svc := newTestAPITokenSvc(repo)

	repo.On("ListByUserID", mock.Anything, "user-1").Return([]*model.APIToken{}, nil)

	_, err := svc.Create(context.Background(), "user-1", "tenant-1", dto.CreateAPITokenReq{
		Name:      "bad-ttl",
		ExpiresIn: "not-a-duration",
	})
	assert.Error(t, err)
}

func TestListByUserID(t *testing.T) {
	repo := new(mockAPITokenRepo)
	svc := newTestAPITokenSvc(repo)

	now := time.Now()
	repo.On("ListByUserID", mock.Anything, "user-1").Return([]*model.APIToken{
		{ID: "t1", Name: "a", CreatedAt: now},
		{ID: "t2", Name: "b", CreatedAt: now},
	}, nil)

	tokens, err := svc.ListByUserID(context.Background(), "user-1")
	assert.NoError(t, err)
	assert.Len(t, tokens, 2)
}

func TestRevokeAPIToken_Success(t *testing.T) {
	repo := new(mockAPITokenRepo)
	svc := newTestAPITokenSvc(repo)

	now := time.Now()
	repo.On("GetByID", mock.Anything, "tok-1").Return(&model.APIToken{
		ID:     "tok-1",
		UserID: "user-1",
	}, nil)
	repo.On("RevokeByID", mock.Anything, "tok-1").Return(nil)

	err := svc.Revoke(context.Background(), "user-1", "tok-1")
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestRevokeAPIToken_NotFound(t *testing.T) {
	repo := new(mockAPITokenRepo)
	svc := newTestAPITokenSvc(repo)

	repo.On("GetByID", mock.Anything, "tok-99").Return(nil, ErrAPITokenNotFound)

	err := svc.Revoke(context.Background(), "user-1", "tok-99")
	assert.ErrorIs(t, err, ErrAPITokenNotFound)
}

func TestRevokeAPIToken_WrongUser(t *testing.T) {
	repo := new(mockAPITokenRepo)
	svc := newTestAPITokenSvc(repo)

	repo.On("GetByID", mock.Anything, "tok-1").Return(&model.APIToken{
		ID:     "tok-1",
		UserID: "user-2",
	}, nil)

	err := svc.Revoke(context.Background(), "user-1", "tok-1")
	assert.ErrorIs(t, err, ErrAPITokenNotFound)
}

func TestRevokeAPIToken_AlreadyRevoked(t *testing.T) {
	repo := new(mockAPITokenRepo)
	svc := newTestAPITokenSvc(repo)

	now := time.Now()
	repo.On("GetByID", mock.Anything, "tok-1").Return(&model.APIToken{
		ID:        "tok-1",
		UserID:    "user-1",
		RevokedAt: &now,
	}, nil)

	err := svc.Revoke(context.Background(), "user-1", "tok-1")
	assert.ErrorIs(t, err, ErrAPITokenRevoked)
}

func TestValidateAPIToken_InvalidPrefix(t *testing.T) {
	repo := new(mockAPITokenRepo)
	svc := newTestAPITokenSvc(repo)

	_, _, err := svc.Validate(context.Background(), "invalid-token")
	assert.ErrorIs(t, err, ErrAPITokenInvalid)
}

func TestValidateAPIToken_NotInDB(t *testing.T) {
	repo := new(mockAPITokenRepo)
	svc := newTestAPITokenSvc(repo)

	repo.On("GetByTokenHash", mock.Anything, mock.AnythingOfType("string")).Return(nil, ErrAPITokenNotFound)

	_, _, err := svc.Validate(context.Background(), "mxat_abcdef1234567890")
	assert.ErrorIs(t, err, ErrAPITokenInvalid)
}

func TestValidateAPIToken_Expired(t *testing.T) {
	repo := new(mockAPITokenRepo)
	svc := newTestAPITokenSvc(repo)

	past := time.Now().Add(-1 * time.Hour)
	repo.On("GetByTokenHash", mock.Anything, mock.AnythingOfType("string")).Return(&model.APIToken{
		ID:        "tok-1",
		UserID:    "user-1",
		TenantID:  "tenant-1",
		ExpiresAt: &past,
	}, nil)
	repo.On("UpdateLastUsedAt", mock.Anything, "tok-1").Return(nil)

	_, _, err := svc.Validate(context.Background(), "mxat_abcdef1234567890")
	assert.ErrorIs(t, err, ErrAPITokenExpired)
}

func TestGenerateAPIToken_Format(t *testing.T) {
	plain, hash, err := generateAPIToken()
	assert.NoError(t, err)
	assert.True(t, len(plain) > len(apiTokenPrefix))
	assert.Equal(t, apiTokenPrefix, plain[:len(apiTokenPrefix)])
	assert.NotEmpty(t, hash)
}

func TestHashAPIToken_Deterministic(t *testing.T) {
	token := "mxat_test123"
	h1 := hashAPIToken(token)
	h2 := hashAPIToken(token)
	assert.Equal(t, h1, h2)
}

func TestParseExpiresIn_Default(t *testing.T) {
	repo := new(mockAPITokenRepo)
	svc := newTestAPITokenSvc(repo)

	expiresAt, err := svc.parseExpiresIn("")
	assert.NoError(t, err)
	assert.NotNil(t, expiresAt)
	assert.True(t, expiresAt.After(time.Now()))
}

func TestParseExpiresIn_Custom(t *testing.T) {
	repo := new(mockAPITokenRepo)
	svc := newTestAPITokenSvc(repo)

	expiresAt, err := svc.parseExpiresIn("720h")
	assert.NoError(t, err)
	assert.NotNil(t, expiresAt)
}

func TestParseExpiresIn_ExceedsMax(t *testing.T) {
	repo := new(mockAPITokenRepo)
	svc := newTestAPITokenSvc(repo)

	_, err := svc.parseExpiresIn("87600h")
	assert.Error(t, err)
}