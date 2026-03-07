package crm

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
)

// LoyaltyUseCase gestiona categorías de fidelización y perfiles CRM.
type LoyaltyUseCase struct {
	profileRepo   repository.CRMProfileRepository
	customerRepo  repository.CustomerRepository
	categoryRepo  repository.CRMCategoryRepository
	benefitRepo   repository.CRMBenefitRepository
}

// NewLoyaltyUseCase construye el caso de uso.
func NewLoyaltyUseCase(
	profileRepo repository.CRMProfileRepository,
	customerRepo repository.CustomerRepository,
	categoryRepo repository.CRMCategoryRepository,
	benefitRepo repository.CRMBenefitRepository,
) *LoyaltyUseCase {
	return &LoyaltyUseCase{
		profileRepo:  profileRepo,
		customerRepo: customerRepo,
		categoryRepo: categoryRepo,
		benefitRepo:  benefitRepo,
	}
}

// GetProfile360 devuelve la vista 360 del cliente (datos base + perfil CRM + categoría y beneficios si aplica).
func (uc *LoyaltyUseCase) GetProfile360(ctx context.Context, companyID, customerID string) (*dto.Profile360Response, error) {
	p360, err := uc.profileRepo.GetProfile360(ctx, companyID, customerID)
	if err != nil {
		return nil, err
	}
	if p360 == nil {
		return nil, domain.ErrNotFound
	}
	if p360.Customer.CompanyID != companyID {
		return nil, domain.ErrForbidden
	}
	resp := &dto.Profile360Response{
		Customer: dto.CustomerResponse{
			ID:        p360.Customer.ID,
			CompanyID: p360.Customer.CompanyID,
			Name:      p360.Customer.Name,
			TaxID:     p360.Customer.TaxID,
			Email:     p360.Customer.Email,
			Phone:     p360.Customer.Phone,
		},
		ProfileID:   p360.ProfileID,
		CategoryID:  p360.CategoryID,
		LTV:         p360.LTV,
		Benefits:    []dto.BenefitResponse{},
	}
	if p360.CategoryID != "" {
		cat, _ := uc.categoryRepo.GetByID(p360.CategoryID)
		if cat != nil {
			resp.CategoryName = cat.Name
			benefits, _ := uc.benefitRepo.ListByCategory(p360.CategoryID, 50, 0)
			for _, b := range benefits {
				resp.Benefits = append(resp.Benefits, dto.BenefitResponse{
					ID:          b.ID,
					CompanyID:   b.CompanyID,
					CategoryID:  b.CategoryID,
					Name:        b.Name,
					Description: b.Description,
					CreatedAt:   b.CreatedAt,
					UpdatedAt:   b.UpdatedAt,
				})
			}
		}
	}
	return resp, nil
}

// AssignCategory asigna o actualiza la categoría y LTV del perfil CRM del cliente.
func (uc *LoyaltyUseCase) AssignCategory(ctx context.Context, companyID, customerID string, in dto.AssignCategoryRequest) error {
	customer, err := uc.customerRepo.GetByID(customerID)
	if err != nil || customer == nil {
		return domain.ErrNotFound
	}
	if customer.CompanyID != companyID {
		return domain.ErrForbidden
	}
	if in.CategoryID != "" {
		cat, _ := uc.categoryRepo.GetByID(in.CategoryID)
		if cat == nil || cat.CompanyID != companyID {
			return domain.ErrNotFound
		}
	}
	profile, _ := uc.profileRepo.GetByCustomerID(customerID)
	now := time.Now()
	if profile == nil {
		profile = &entity.CRMCustomerProfile{
			ID:         uuid.New().String(),
			CustomerID: customerID,
			CompanyID:  companyID,
			CategoryID: in.CategoryID,
			LTV:        in.LTV,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
	} else {
		profile.CategoryID = in.CategoryID
		profile.LTV = in.LTV
		profile.UpdatedAt = now
	}
	return uc.profileRepo.Upsert(profile)
}

// ListCategories lista categorías de fidelización de la empresa.
func (uc *LoyaltyUseCase) ListCategories(ctx context.Context, companyID string, limit, offset int) ([]dto.CategoryResponse, error) {
	list, err := uc.categoryRepo.ListByCompany(companyID, limit, offset)
	if err != nil {
		return nil, err
	}
	out := make([]dto.CategoryResponse, 0, len(list))
	for _, c := range list {
		out = append(out, dto.CategoryResponse{
			ID:        c.ID,
			CompanyID: c.CompanyID,
			Name:      c.Name,
			MinLTV:    c.MinLTV,
			CreatedAt: c.CreatedAt,
			UpdatedAt: c.UpdatedAt,
		})
	}
	return out, nil
}

// ListBenefitsByCategory lista beneficios de una categoría.
func (uc *LoyaltyUseCase) ListBenefitsByCategory(ctx context.Context, categoryID string, limit, offset int) ([]dto.BenefitResponse, error) {
	list, err := uc.benefitRepo.ListByCategory(categoryID, limit, offset)
	if err != nil {
		return nil, err
	}
	out := make([]dto.BenefitResponse, 0, len(list))
	for _, b := range list {
		out = append(out, dto.BenefitResponse{
			ID:          b.ID,
			CompanyID:   b.CompanyID,
			CategoryID:  b.CategoryID,
			Name:        b.Name,
			Description: b.Description,
			CreatedAt:   b.CreatedAt,
			UpdatedAt:   b.UpdatedAt,
		})
	}
	return out, nil
}
