package crm

import (
	"context"
	"testing"
	"time"

	"github.com/jhoicas/Inventario-api/internal/domain"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type loyaltyProfileRepoFake struct {
	profile *entity.CRMCustomerProfile
}

func (f *loyaltyProfileRepoFake) GetByCustomerID(customerID string) (*entity.CRMCustomerProfile, error) {
	if f.profile != nil && f.profile.CustomerID == customerID {
		return f.profile, nil
	}
	return nil, nil
}

func (f *loyaltyProfileRepoFake) GetProfile360(ctx context.Context, companyID, customerID string) (*entity.Profile360, error) {
	return nil, nil
}

func (f *loyaltyProfileRepoFake) Upsert(profile *entity.CRMCustomerProfile) error { return nil }

func (f *loyaltyProfileRepoFake) ListByCompany(companyID string, limit, offset int) ([]*entity.CRMCustomerProfile, error) {
	return nil, nil
}

type loyaltyCustomerRepoFake struct {
	customer *entity.Customer
}

func (f *loyaltyCustomerRepoFake) Create(customer *entity.Customer) error { return nil }

func (f *loyaltyCustomerRepoFake) GetByID(id string) (*entity.Customer, error) {
	if f.customer != nil && f.customer.ID == id {
		return f.customer, nil
	}
	return nil, nil
}

func (f *loyaltyCustomerRepoFake) GetByCompanyAndTaxID(companyID, taxID string) (*entity.Customer, error) {
	return nil, nil
}

func (f *loyaltyCustomerRepoFake) ListByCompany(companyID string, search string, limit, offset int) ([]*entity.Customer, error) {
	return nil, nil
}

func (f *loyaltyCustomerRepoFake) Update(customer *entity.Customer) error { return nil }

func (f *loyaltyCustomerRepoFake) Delete(id string) error { return nil }

type loyaltyCategoryRepoFake struct{}

func (f *loyaltyCategoryRepoFake) Create(category *entity.CRMCategory) error { return nil }

func (f *loyaltyCategoryRepoFake) GetByID(id string) (*entity.CRMCategory, error) { return nil, nil }

func (f *loyaltyCategoryRepoFake) ListByCompany(companyID string, limit, offset int) ([]*entity.CRMCategory, error) {
	return []*entity.CRMCategory{}, nil
}

func (f *loyaltyCategoryRepoFake) Update(category *entity.CRMCategory) error { return nil }

func (f *loyaltyCategoryRepoFake) Delete(id string) error { return nil }

type loyaltyBenefitRepoFake struct{}

func (f *loyaltyBenefitRepoFake) Create(benefit *entity.CRMBenefit) error { return nil }

func (f *loyaltyBenefitRepoFake) GetByID(id string) (*entity.CRMBenefit, error) { return nil, nil }

func (f *loyaltyBenefitRepoFake) ListByCategory(categoryID string, limit, offset int) ([]*entity.CRMBenefit, error) {
	return nil, nil
}

func (f *loyaltyBenefitRepoFake) Update(benefit *entity.CRMBenefit) error { return nil }

func (f *loyaltyBenefitRepoFake) Delete(id string) error { return nil }

type loyaltyInteractionRepoFake struct {
	events      []*entity.CRMInteraction
	createCalls int
}

func (f *loyaltyInteractionRepoFake) Create(interaction *entity.CRMInteraction) error {
	f.createCalls++
	return nil
}

func (f *loyaltyInteractionRepoFake) GetByID(id string) (*entity.CRMInteraction, error) {
	return nil, nil
}

func (f *loyaltyInteractionRepoFake) ListByCustomer(customerID string, limit, offset int) ([]*entity.CRMInteraction, error) {
	return nil, nil
}

func (f *loyaltyInteractionRepoFake) ListInteractions(customerID string, flt repository.InteractionFilters) ([]*entity.CRMInteraction, int64, error) {
	if flt.Offset > 0 {
		return []*entity.CRMInteraction{}, int64(len(f.events)), nil
	}
	return f.events, int64(len(f.events)), nil
}

func TestLoyaltyUseCase_RedeemPoints_InsufficientBalance(t *testing.T) {
	profileRepo := &loyaltyProfileRepoFake{profile: &entity.CRMCustomerProfile{
		ID:         "p1",
		CustomerID: "cust-1",
		CompanyID:  "comp-1",
		LTV:        decimal.NewFromInt(1000),
	}}
	customerRepo := &loyaltyCustomerRepoFake{customer: &entity.Customer{ID: "cust-1", CompanyID: "comp-1"}}
	interactionRepo := &loyaltyInteractionRepoFake{events: []*entity.CRMInteraction{
		{
			ID:         "e1",
			CompanyID:  "comp-1",
			CustomerID: "cust-1",
			Type:       entity.InteractionTypeOther,
			Subject:    "LOYALTY_POINTS_AWARD",
			Body:       `{"delta":30,"reason":"welcome"}`,
			CreatedAt:  time.Now(),
		},
	}}

	uc := NewLoyaltyUseCase(
		profileRepo,
		customerRepo,
		&loyaltyCategoryRepoFake{},
		&loyaltyBenefitRepoFake{},
		interactionRepo,
	)

	err := uc.RedeemPoints(context.Background(), "cust-1", 100, "canje premium")
	require.Error(t, err)
	assert.Equal(t, domain.ErrConflict, err)
	assert.Equal(t, 0, interactionRepo.createCalls)
}

func TestLoyaltyUseCase_GetBalance_NoProfile_ReturnsZero(t *testing.T) {
	profileRepo := &loyaltyProfileRepoFake{profile: nil}
	customerRepo := &loyaltyCustomerRepoFake{customer: &entity.Customer{ID: "cust-1", CompanyID: "comp-1"}}
	interactionRepo := &loyaltyInteractionRepoFake{events: []*entity.CRMInteraction{}}

	uc := NewLoyaltyUseCase(
		profileRepo,
		customerRepo,
		&loyaltyCategoryRepoFake{},
		&loyaltyBenefitRepoFake{},
		interactionRepo,
	)

	out, err := uc.GetBalance(context.Background(), "cust-1")
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, 0, out.Balance)
	assert.Equal(t, "", out.Tier)
	assert.Equal(t, 0, len(out.History))
}

func TestLoyaltyUseCase_GetBalanceByCompany_Forbidden(t *testing.T) {
	profileRepo := &loyaltyProfileRepoFake{profile: nil}
	customerRepo := &loyaltyCustomerRepoFake{customer: &entity.Customer{ID: "cust-1", CompanyID: "comp-1"}}
	interactionRepo := &loyaltyInteractionRepoFake{events: []*entity.CRMInteraction{}}

	uc := NewLoyaltyUseCase(
		profileRepo,
		customerRepo,
		&loyaltyCategoryRepoFake{},
		&loyaltyBenefitRepoFake{},
		interactionRepo,
	)

	out, err := uc.GetBalanceByCompany(context.Background(), "comp-2", "cust-1")
	require.Error(t, err)
	assert.Nil(t, out)
	assert.Equal(t, domain.ErrForbidden, err)
}
