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

// CategoryUseCase CRUD de categorías de producto (crm_category_product_hub).
type CategoryUseCase struct {
	repo repository.CategoryRepository
}

// NewCategoryUseCase constructor.
func NewCategoryUseCase(repo repository.CategoryRepository) *CategoryUseCase {
	return &CategoryUseCase{repo: repo}
}

func categoryNameFromCreate(in dto.CreateProductCategoryRequest) string {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		name = strings.TrimSpace(in.Code)
	}
	return name
}

func (uc *CategoryUseCase) Create(companyID string, in dto.CreateProductCategoryRequest) (*dto.ProductCategoryResponse, error) {
	name := categoryNameFromCreate(in)
	if name == "" {
		return nil, domain.ErrInvalidInput
	}
	if existing, _ := uc.repo.GetByCompanyAndName(companyID, name); existing != nil {
		return nil, domain.ErrDuplicate
	}
	now := time.Now()
	c := &entity.CrmCategoryProductHub{
		ID:        uuid.New().String(),
		CompanyID: companyID,
		Name:      name,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := uc.repo.Create(c); err != nil {
		return nil, err
	}
	return toCategoryResponse(c), nil
}

func (uc *CategoryUseCase) GetByID(companyID, id string) (*dto.ProductCategoryResponse, error) {
	c, err := uc.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if c == nil || c.CompanyID != companyID {
		return nil, nil
	}
	return toCategoryResponse(c), nil
}

func (uc *CategoryUseCase) List(companyID string, limit, offset int) (*dto.ProductCategoryListResponse, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	list, total, err := uc.repo.ListByCompany(companyID, limit, offset)
	if err != nil {
		return nil, err
	}
	items := make([]dto.ProductCategoryResponse, 0, len(list))
	for _, c := range list {
		items = append(items, *toCategoryResponse(c))
	}
	return &dto.ProductCategoryListResponse{Items: items, Total: int(total), Limit: limit, Offset: offset}, nil
}

func (uc *CategoryUseCase) Update(companyID, id string, in dto.UpdateProductCategoryRequest) (*dto.ProductCategoryResponse, error) {
	c, err := uc.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if c == nil || c.CompanyID != companyID {
		return nil, nil
	}
	if in.Name != nil {
		c.Name = strings.TrimSpace(*in.Name)
	}
	if in.IsActive != nil {
		c.IsActive = *in.IsActive
	}
	if c.Name == "" {
		return nil, domain.ErrInvalidInput
	}
	// Si el nombre cambia a uno ya usado por otra fila
	if other, _ := uc.repo.GetByCompanyAndName(companyID, c.Name); other != nil && other.ID != id {
		return nil, domain.ErrDuplicate
	}
	c.UpdatedAt = time.Now()
	if err := uc.repo.Update(c); err != nil {
		return nil, err
	}
	out, err := uc.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	return toCategoryResponse(out), nil
}

func (uc *CategoryUseCase) Deactivate(companyID, id string) error {
	c, err := uc.repo.GetByID(id)
	if err != nil {
		return err
	}
	if c == nil || c.CompanyID != companyID {
		return domain.ErrNotFound
	}
	if err := uc.repo.Deactivate(companyID, id); err != nil {
		return err
	}
	return nil
}

func toCategoryResponse(c *entity.CrmCategoryProductHub) *dto.ProductCategoryResponse {
	if c == nil {
		return nil
	}
	return &dto.ProductCategoryResponse{
		ID:        c.ID,
		CompanyID: c.CompanyID,
		Name:      c.Name,
		IsActive:  c.IsActive,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}
