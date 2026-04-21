package usecase

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
)

// CrmHubProductUseCase catálogo CRM crm_products_hub (aislado por company_id).
type CrmHubProductUseCase struct {
	repo         repository.CrmHubProductRepository
	categoryRepo repository.CategoryRepository
}

// NewCrmHubProductUseCase constructor.
func NewCrmHubProductUseCase(
	repo repository.CrmHubProductRepository,
	categoryRepo repository.CategoryRepository,
) *CrmHubProductUseCase {
	return &CrmHubProductUseCase{repo: repo, categoryRepo: categoryRepo}
}

func (uc *CrmHubProductUseCase) assertCategoryForCompany(companyID, categoryID string) error {
	categoryID = strings.TrimSpace(categoryID)
	if categoryID == "" {
		return domain.ErrInvalidInput
	}
	if _, err := uuid.Parse(categoryID); err != nil {
		return domain.ErrInvalidInput
	}
	c, err := uc.categoryRepo.GetByID(categoryID)
	if err != nil {
		return err
	}
	if c == nil || c.CompanyID != companyID {
		return domain.ErrNotFound
	}
	return nil
}

// Create registra un producto hub.
func (uc *CrmHubProductUseCase) Create(companyID string, in dto.CrmHubProductCreateRequest) (*dto.CrmHubProductResponse, error) {
	code := strings.TrimSpace(in.ProductCode)
	name := strings.TrimSpace(in.ProductName)
	if code == "" || name == "" {
		return nil, domain.ErrInvalidInput
	}
	if err := uc.assertCategoryForCompany(companyID, in.CategoryID); err != nil {
		return nil, err
	}
	if existing, _ := uc.repo.GetByCompanyAndProductCode(companyID, code); existing != nil {
		return nil, domain.ErrDuplicate
	}
	now := time.Now()
	p := &entity.CrmProductHub{
		ID:          uuid.New().String(),
		CompanyID:   companyID,
		CategoryID:  strings.TrimSpace(in.CategoryID),
		ProductCode: code,
		ProductName: name,
		UnitCost:    in.UnitCost,
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := uc.repo.Create(p); err != nil {
		return nil, err
	}
	return toCrmHubProductResponse(p), nil
}

// GetByID obtiene un producto por id si pertenece a la empresa.
func (uc *CrmHubProductUseCase) GetByID(companyID, id string) (*dto.CrmHubProductResponse, error) {
	p, err := uc.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if p == nil || p.CompanyID != companyID {
		return nil, nil
	}
	return toCrmHubProductResponse(p), nil
}

// List lista con paginación y filtro opcional is_active (nil = todos).
func (uc *CrmHubProductUseCase) List(companyID string, limit, offset int, active *bool) (*dto.CrmHubProductListResponse, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	list, total, err := uc.repo.ListByCompany(companyID, limit, offset, active)
	if err != nil {
		return nil, err
	}
	items := make([]dto.CrmHubProductResponse, 0, len(list))
	for _, p := range list {
		items = append(items, *toCrmHubProductResponse(p))
	}
	return &dto.CrmHubProductListResponse{
		Items:  items,
		Total:  int(total),
		Limit:  limit,
		Offset: offset,
	}, nil
}

// Update actualiza un producto hub.
func (uc *CrmHubProductUseCase) Update(companyID, id string, in dto.CrmHubProductUpdateRequest) (*dto.CrmHubProductResponse, error) {
	p, err := uc.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if p == nil || p.CompanyID != companyID {
		return nil, nil
	}
	if in.CategoryID != nil {
		if err := uc.assertCategoryForCompany(companyID, *in.CategoryID); err != nil {
			return nil, err
		}
		p.CategoryID = strings.TrimSpace(*in.CategoryID)
	}
	if in.ProductCode != nil {
		c := strings.TrimSpace(*in.ProductCode)
		if c == "" {
			return nil, domain.ErrInvalidInput
		}
		if other, _ := uc.repo.GetByCompanyAndProductCode(companyID, c); other != nil && other.ID != id {
			return nil, domain.ErrDuplicate
		}
		p.ProductCode = c
	}
	if in.ProductName != nil {
		n := strings.TrimSpace(*in.ProductName)
		if n == "" {
			return nil, domain.ErrInvalidInput
		}
		p.ProductName = n
	}
	if in.UnitCost != nil {
		p.UnitCost = in.UnitCost
	}
	if in.IsActive != nil {
		p.IsActive = *in.IsActive
	}
	p.UpdatedAt = time.Now()
	if err := uc.repo.Update(p); err != nil {
		return nil, err
	}
	out, err := uc.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	return toCrmHubProductResponse(out), nil
}

// Deactivate marca is_active = false.
func (uc *CrmHubProductUseCase) Deactivate(companyID, id string) error {
	p, err := uc.repo.GetByID(id)
	if err != nil {
		return err
	}
	if p == nil || p.CompanyID != companyID {
		return domain.ErrNotFound
	}
	return uc.repo.Deactivate(companyID, id)
}

func toCrmHubProductResponse(p *entity.CrmProductHub) *dto.CrmHubProductResponse {
	if p == nil {
		return nil
	}
	return &dto.CrmHubProductResponse{
		ID:          p.ID,
		CompanyID:   p.CompanyID,
		CategoryID:  p.CategoryID,
		ProductCode: p.ProductCode,
		ProductName: p.ProductName,
		UnitCost:    p.UnitCost,
		IsActive:    p.IsActive,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}
