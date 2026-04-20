package usecase

import (
	"time"

	"github.com/google/uuid"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
	"github.com/shopspring/decimal"
)

// ProductUseCase casos de uso CRUD para productos. Cost y Stock se manejan vía movimientos.
type ProductUseCase struct {
	repo repository.ProductRepository
}

// NewProductUseCase construye el caso de uso.
func NewProductUseCase(repo repository.ProductRepository) *ProductUseCase {
	return &ProductUseCase{repo: repo}
}

// Create crea un nuevo producto. Cost inicia en 0.
func (uc *ProductUseCase) Create(companyID string, in dto.CreateProductRequest) (*dto.ProductResponse, error) {
	existing, _ := uc.repo.GetByCompanyAndSKU(companyID, in.SKU)
	if existing != nil {
		return nil, domain.ErrDuplicate
	}
	// tax_rate se guarda como porcentaje (ej: 19, 5, 0, 7.5). En cálculos se normaliza.
	// Permitimos cualquier porcentaje entre 0 y 100 (inclusive).
	if in.TaxRate.LessThan(decimal.Zero) || in.TaxRate.GreaterThan(decimal.NewFromInt(100)) {
		return nil, domain.ErrInvalidInput
	}
	// UnitMeasure e información DIAN provienen exclusivamente del DTO (parametrización manual).
	now := time.Now()
	product := &entity.Product{
		ID:          uuid.New().String(),
		CompanyID:   companyID,
		SKU:         in.SKU,
		Name:        in.Name,
		Description: in.Description,
		Price:       in.Price,
		Cost:        decimal.Zero,
		TaxRate:     in.TaxRate,
		UNSPSC_Code: in.UNSPSC_Code,
		UnitMeasure: in.UnitMeasure,
		Attributes:  in.Attributes,
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := uc.repo.Create(product); err != nil {
		return nil, err
	}
	return toProductResponse(product), nil
}

// GetByID obtiene un producto por ID (solo si pertenece a la empresa).
func (uc *ProductUseCase) GetByID(companyID, id string) (*dto.ProductResponse, error) {
	product, err := uc.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if product == nil || product.CompanyID != companyID {
		return nil, nil
	}
	return toProductResponse(product), nil
}

// Update actualiza un producto. No permite modificar Cost ni Stock (se manejan vía movimientos).
func (uc *ProductUseCase) Update(companyID, id string, in dto.UpdateProductRequest) (*dto.ProductResponse, error) {
	product, err := uc.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if product == nil || product.CompanyID != companyID {
		return nil, nil
	}
	if in.Name != nil {
		product.Name = *in.Name
	}
	if in.Description != nil {
		product.Description = *in.Description
	}
	if in.Price != nil {
		product.Price = *in.Price
	}
	if in.TaxRate != nil {
		if in.TaxRate.LessThan(decimal.Zero) || in.TaxRate.GreaterThan(decimal.NewFromInt(100)) {
			return nil, domain.ErrInvalidInput
		}
		product.TaxRate = *in.TaxRate
	}
	if in.UNSPSC_Code != nil {
		product.UNSPSC_Code = *in.UNSPSC_Code
	}
	if in.UnitMeasure != nil {
		product.UnitMeasure = *in.UnitMeasure
	}
	if len(in.Attributes) > 0 {
		product.Attributes = in.Attributes
	}
	if in.IsActive != nil {
		product.IsActive = *in.IsActive
	}
	product.UpdatedAt = time.Now()
	if err := uc.repo.Update(product); err != nil {
		return nil, err
	}
	return toProductResponse(product), nil
}

// Deactivate marca el producto como inactivo (soft delete).
func (uc *ProductUseCase) Deactivate(companyID, id string) error {
	p, err := uc.repo.GetByID(id)
	if err != nil {
		return err
	}
	if p == nil || p.CompanyID != companyID {
		return domain.ErrNotFound
	}
	return uc.repo.Deactivate(companyID, id)
}

// List lista productos por empresa con paginación. Si activeOnly es true, excluye inactivos.
func (uc *ProductUseCase) List(companyID, search string, limit, offset int, activeOnly bool) (*dto.ProductListResponse, error) {
	list, total, err := uc.repo.ListByCompany(companyID, search, limit, offset, activeOnly)
	if err != nil {
		return nil, err
	}
	items := make([]dto.ProductResponse, 0, len(list))
	for _, p := range list {
		items = append(items, *toProductResponse(p))
	}
	return &dto.ProductListResponse{
		Items:  items,
		Total:  int(total),
		Limit:  limit,
		Offset: offset,
	}, nil
}

// Delete elimina un producto por ID.
func (uc *ProductUseCase) Delete(id string) error {
	return uc.repo.Delete(id)
}

func toProductResponse(p *entity.Product) *dto.ProductResponse {
	if p == nil {
		return nil
	}
	return &dto.ProductResponse{
		ID:          p.ID,
		CompanyID:   p.CompanyID,
		SKU:         p.SKU,
		Name:        p.Name,
		Description: p.Description,
		Price:       p.Price,
		Cost:        p.Cost,
		TaxRate:     p.TaxRate,
		UNSPSC_Code: p.UNSPSC_Code,
		UnitMeasure: p.UnitMeasure,
		Attributes:  p.Attributes,
		IsActive:    p.IsActive,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}
