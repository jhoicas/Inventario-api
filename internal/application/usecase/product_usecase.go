package usecase

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/tu-usuario/inventory-pro/internal/application/dto"
	"github.com/tu-usuario/inventory-pro/internal/domain"
	"github.com/tu-usuario/inventory-pro/internal/domain/entity"
	"github.com/tu-usuario/inventory-pro/internal/domain/repository"
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
	taxZero := decimal.Zero
	tax5 := decimal.NewFromInt(5)
	tax19 := decimal.NewFromInt(19)
	if !in.TaxRate.Equal(taxZero) && !in.TaxRate.Equal(tax5) && !in.TaxRate.Equal(tax19) {
		return nil, domain.ErrInvalidInput
	}
	if in.UnitMeasure == "" {
		in.UnitMeasure = "94"
	}
	now := time.Now()
	product := &entity.Product{
		ID:           uuid.New().String(),
		CompanyID:    companyID,
		SKU:          in.SKU,
		Name:         in.Name,
		Description:  in.Description,
		Price:        in.Price,
		Cost:         decimal.Zero,
		TaxRate:      in.TaxRate,
		UNSPSC_Code:  in.UNSPSC_Code,
		UnitMeasure:  in.UnitMeasure,
		Attributes:   in.Attributes,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := uc.repo.Create(product); err != nil {
		return nil, err
	}
	return toProductResponse(product), nil
}

// GetByID obtiene un producto por ID.
func (uc *ProductUseCase) GetByID(id string) (*dto.ProductResponse, error) {
	product, err := uc.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if product == nil {
		return nil, nil
	}
	return toProductResponse(product), nil
}

// Update actualiza un producto. No permite modificar Cost ni Stock (se manejan vía movimientos).
func (uc *ProductUseCase) Update(id string, in dto.UpdateProductRequest) (*dto.ProductResponse, error) {
	product, err := uc.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if product == nil {
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
		taxZero := decimal.Zero
		tax5 := decimal.NewFromInt(5)
		tax19 := decimal.NewFromInt(19)
		if !in.TaxRate.Equal(taxZero) && !in.TaxRate.Equal(tax5) && !in.TaxRate.Equal(tax19) {
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
	product.UpdatedAt = time.Now()
	if err := uc.repo.Update(product); err != nil {
		return nil, err
	}
	return toProductResponse(product), nil
}

// List lista productos por empresa con paginación.
func (uc *ProductUseCase) List(companyID string, limit, offset int) (*dto.ProductListResponse, error) {
	list, err := uc.repo.ListByCompany(companyID, limit, offset)
	if err != nil {
		return nil, err
	}
	items := make([]dto.ProductResponse, 0, len(list))
	for _, p := range list {
		items = append(items, *toProductResponse(p))
	}
	return &dto.ProductListResponse{
		Items: items,
		Page:  dto.PageResponse{Limit: limit, Offset: offset},
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
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}
