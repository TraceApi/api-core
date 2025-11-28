package service_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/TraceApi/api-core/internal/core/domain"
	"github.com/TraceApi/api-core/internal/core/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- Mocks ---

type MockPassportRepository struct {
	mock.Mock
}

func (m *MockPassportRepository) Save(ctx context.Context, passport *domain.Passport) error {
	args := m.Called(ctx, passport)
	return args.Error(0)
}

func (m *MockPassportRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Passport, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Passport), args.Error(1)
}

func (m *MockPassportRepository) FindByCategory(ctx context.Context, category domain.ProductCategory, limit, offset int) ([]*domain.Passport, error) {
	args := m.Called(ctx, category, limit, offset)
	return args.Get(0).([]*domain.Passport), args.Error(1)
}

func (m *MockPassportRepository) Update(ctx context.Context, passport *domain.Passport) error {
	args := m.Called(ctx, passport)
	return args.Error(0)
}

type MockBlobStorage struct {
	mock.Mock
}

func (m *MockBlobStorage) UploadJSON(ctx context.Context, bucket string, key string, data []byte) (string, error) {
	args := m.Called(ctx, bucket, key, data)
	return args.String(0), args.Error(1)
}

type MockCacheRepository struct {
	mock.Mock
}

func (m *MockCacheRepository) GetIdempotency(ctx context.Context, hash string) (string, error) {
	args := m.Called(ctx, hash)
	return args.String(0), args.Error(1)
}

func (m *MockCacheRepository) SetIdempotency(ctx context.Context, hash string, passportID string) error {
	args := m.Called(ctx, hash, passportID)
	return args.Error(0)
}

func (m *MockCacheRepository) Get(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

func (m *MockCacheRepository) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	args := m.Called(ctx, key, value, ttl)
	return args.Error(0)
}

func (m *MockCacheRepository) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

// --- Tests ---

func TestCreatePassport_Success(t *testing.T) {
	// Setup
	mockRepo := new(MockPassportRepository)
	mockCache := new(MockCacheRepository)
	mockBlob := new(MockBlobStorage)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	svc, err := service.NewPassportService(mockRepo, mockCache, mockBlob, logger)
	assert.NoError(t, err)

	ctx := context.Background()
	manufacturerID := "test-manufacturer"
	category := domain.CategoryBattery

	// Valid Payload
	payload := map[string]interface{}{
		"batteryModel":  "Test Model",
		"chemistry":     "LITHIUM_ION",
		"ratedCapacity": 100,
		"carbonFootprint": map[string]interface{}{
			"totalCarbonFootprint": 50,
			"shareOfRenewables":    90,
		},
		"materialComposition": []interface{}{},
	}
	payloadBytes, _ := json.Marshal(payload)

	// Expectations
	mockCache.On("GetIdempotency", ctx, mock.Anything).Return("", errors.New("cache miss"))
	mockRepo.On("Save", ctx, mock.AnythingOfType("*domain.Passport")).Return(nil)
	mockCache.On("SetIdempotency", ctx, mock.Anything, mock.Anything).Return(nil)

	// Execute
	passport, err := svc.CreatePassport(ctx, manufacturerID, category, payloadBytes)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, passport)
	assert.Equal(t, manufacturerID, passport.ManufacturerID)
	assert.Equal(t, domain.StatusDraft, passport.Status)

	mockRepo.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestCreatePassport_InvalidSchema(t *testing.T) {
	// Setup
	mockRepo := new(MockPassportRepository)
	mockCache := new(MockCacheRepository)
	mockBlob := new(MockBlobStorage)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	svc, _ := service.NewPassportService(mockRepo, mockCache, mockBlob, logger)
	ctx := context.Background()

	// Invalid Payload (Missing required fields)
	payload := map[string]interface{}{
		"batteryModel": "Test Model",
	}
	payloadBytes, _ := json.Marshal(payload)

	// Expectations: Cache check happens, but Save should NOT be called
	mockCache.On("GetIdempotency", ctx, mock.Anything).Return("", errors.New("cache miss"))

	// Execute
	passport, err := svc.CreatePassport(ctx, "mfg-1", domain.CategoryBattery, payloadBytes)

	// Assertions
	assert.Error(t, err)
	assert.Nil(t, passport)
	assert.True(t, errors.Is(err, domain.ErrInvalidInput))
}

func TestCreatePassport_IdempotencyHit(t *testing.T) {
	// Setup
	mockRepo := new(MockPassportRepository)
	mockCache := new(MockCacheRepository)
	mockBlob := new(MockBlobStorage)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	svc, _ := service.NewPassportService(mockRepo, mockCache, mockBlob, logger)
	ctx := context.Background()

	existingID := uuid.New()
	existingPassport := &domain.Passport{ID: existingID}

	// Expectations
	// 1. Cache returns a hit (an existing UUID string)
	mockCache.On("GetIdempotency", ctx, mock.Anything).Return(existingID.String(), nil)
	// 2. Service fetches the full object from DB
	mockRepo.On("GetByID", ctx, existingID).Return(existingPassport, nil)

	// Execute
	passport, err := svc.CreatePassport(ctx, "mfg-1", domain.CategoryBattery, []byte("{}"))

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, existingID, passport.ID)
	mockRepo.AssertExpectations(t)
}

func TestPublishPassport_Success(t *testing.T) {
	// Setup
	mockRepo := new(MockPassportRepository)
	mockCache := new(MockCacheRepository)
	mockBlob := new(MockBlobStorage)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	svc, _ := service.NewPassportService(mockRepo, mockCache, mockBlob, logger)
	ctx := context.Background()

	id := uuid.New()
	passport := &domain.Passport{
		ID:         id,
		Status:     domain.StatusDraft,
		Attributes: json.RawMessage(`{"foo":"bar"}`),
	}

	// Expectations
	mockRepo.On("GetByID", ctx, id).Return(passport, nil)
	mockBlob.On("UploadJSON", ctx, "passports", mock.Anything, mock.Anything).Return("s3://bucket/key", nil)
	mockRepo.On("Update", ctx, mock.MatchedBy(func(p *domain.Passport) bool {
		return p.Status == domain.StatusPublished && p.StorageLocation == "s3://bucket/key" && p.ImmutabilityHash != ""
	})).Return(nil)

	// Execute
	published, err := svc.PublishPassport(ctx, id)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, domain.StatusPublished, published.Status)
	assert.Equal(t, "s3://bucket/key", published.StorageLocation)
	assert.NotEmpty(t, published.ImmutabilityHash)
	assert.NotNil(t, published.PublishedAt)

	mockRepo.AssertExpectations(t)
	mockBlob.AssertExpectations(t)
}
