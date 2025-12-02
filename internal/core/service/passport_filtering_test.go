package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/TraceApi/api-core/internal/core/domain"
	"github.com/TraceApi/api-core/internal/core/ports"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock dependencies
type MockRepo struct{ mock.Mock }

func (m *MockRepo) Save(ctx context.Context, p *domain.Passport) error {
	args := m.Called(ctx, p)
	return args.Error(0)
}
func (m *MockRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Passport, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Passport), args.Error(1)
}
func (m *MockRepo) Update(ctx context.Context, p *domain.Passport) error {
	args := m.Called(ctx, p)
	return args.Error(0)
}

func (m *MockRepo) FindByCategory(ctx context.Context, category domain.ProductCategory, limit, offset int) ([]*domain.Passport, error) {
	args := m.Called(ctx, category, limit, offset)
	return args.Get(0).([]*domain.Passport), args.Error(1)
}

func (m *MockRepo) FindByManufacturer(ctx context.Context, manufacturerID string) ([]*domain.Passport, error) {
	args := m.Called(ctx, manufacturerID)
	return args.Get(0).([]*domain.Passport), args.Error(1)
}

type MockCache struct{ mock.Mock }

func (m *MockCache) Get(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}
func (m *MockCache) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	args := m.Called(ctx, key, value, ttl)
	return args.Error(0)
}
func (m *MockCache) GetIdempotency(ctx context.Context, hash string) (string, error) {
	args := m.Called(ctx, hash)
	return args.String(0), args.Error(1)
}
func (m *MockCache) SetIdempotency(ctx context.Context, hash string, id string) error {
	args := m.Called(ctx, hash, id)
	return args.Error(0)
}
func (m *MockCache) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

// Ensure interface compliance
var _ ports.PassportRepository = (*MockRepo)(nil)
var _ ports.CacheRepository = (*MockCache)(nil)

func TestGetPassport_Filtering(t *testing.T) {
	// Setup Service
	repo := new(MockRepo)
	cache := new(MockCache)
	// We don't need real BlobStore or EventBus for this test
	svc, err := NewPassportService(repo, cache, nil, nil, nil)
	assert.NoError(t, err)

	// Create a passport with restricted data
	fullAttributes := `{"batteryModel": "Test", "disassemblyInstructions": {"secret": "data"}}`
	id := uuid.New()
	passport := &domain.Passport{
		ID:              id,
		ProductCategory: domain.CategoryBattery,
		Attributes:      json.RawMessage(fullAttributes),
	}

	// Mock Repo to return the passport
	repo.On("GetByID", mock.Anything, id).Return(passport, nil).Once()
	// Mock Cache Miss (to force DB hit)
	cache.On("Get", mock.Anything, mock.Anything).Return("", assert.AnError)
	// Mock Cache Set (ignored)
	cache.On("Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Test 1: Public Context (Should Filter)
	ctxPublic := context.Background() // No context value = Public
	pPublic, err := svc.GetPassport(ctxPublic, id)
	assert.NoError(t, err)

	var attrsPublic map[string]interface{}
	json.Unmarshal(pPublic.Attributes, &attrsPublic)
	assert.Equal(t, "Test", attrsPublic["batteryModel"])
	assert.Nil(t, attrsPublic["disassemblyInstructions"], "Restricted field should be removed")

	// Test 2: Restricted Context (Should NOT Filter)
	ctxRestricted := context.WithValue(context.Background(), domain.ViewContextKey, domain.ViewContextRestricted)

	// Reset passport data because previous call mutated it
	passportRestricted := &domain.Passport{
		ID:              id,
		ProductCategory: domain.CategoryBattery,
		Attributes:      json.RawMessage(fullAttributes),
	}

	// Reset mocks for second call
	repo.On("GetByID", mock.Anything, id).Return(passportRestricted, nil).Once()

	pRestricted, err := svc.GetPassport(ctxRestricted, id)
	assert.NoError(t, err)

	var attrsRestricted map[string]interface{}
	json.Unmarshal(pRestricted.Attributes, &attrsRestricted)
	assert.Equal(t, "Test", attrsRestricted["batteryModel"])
	assert.NotNil(t, attrsRestricted["disassemblyInstructions"], "Restricted field should be present")
}

func TestGetPassport_Filtering_Textile(t *testing.T) {
	// Setup Service
	repo := new(MockRepo)
	cache := new(MockCache)
	// We don't need real BlobStore or EventBus for this test
	// NewPassportService will load the embedded textile.json which SHOULD have supplyChainDetails restricted
	svc, err := NewPassportService(repo, cache, nil, nil, nil)
	assert.NoError(t, err)

	// Create a passport with restricted data
	fullAttributes := `{"garmentType": "T-Shirt", "supplyChainDetails": {"spinningFactory": "Secret Factory"}}`
	id := uuid.New()
	passport := &domain.Passport{
		ID:              id,
		ProductCategory: domain.CategoryTextile,
		Attributes:      json.RawMessage(fullAttributes),
	}

	// Mock Repo to return the passport
	repo.On("GetByID", mock.Anything, id).Return(passport, nil).Once()
	// Mock Cache Miss (to force DB hit)
	cache.On("Get", mock.Anything, mock.Anything).Return("", assert.AnError)
	// Mock Cache Set (ignored)
	cache.On("Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Test 1: Public Context (Should Filter)
	ctxPublic := context.Background() // No context value = Public
	pPublic, err := svc.GetPassport(ctxPublic, id)
	assert.NoError(t, err)

	var attrsPublic map[string]interface{}
	json.Unmarshal(pPublic.Attributes, &attrsPublic)
	assert.Equal(t, "T-Shirt", attrsPublic["garmentType"])
	assert.Nil(t, attrsPublic["supplyChainDetails"], "Restricted field should be removed")
}
