package inventory

import (
	"context"

	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
	"github.com/shopspring/decimal"
)

// ReorderConfigRepository persiste configuración de reposición por producto y bodega.
type ReorderConfigRepository interface {
	UpsertProductReorderConfig(ctx context.Context, in dto.ReorderConfigRequest) error
}

// UpdateReorderConfigUseCase actualiza (upsert) configuración de reposición.
type UpdateReorderConfigUseCase struct {
	productRepo repository.ProductRepository
	reorderRepo ReorderConfigRepository
}

func NewUpdateReorderConfigUseCase(
	productRepo repository.ProductRepository,
	reorderRepo ReorderConfigRepository,
) *UpdateReorderConfigUseCase {
	return &UpdateReorderConfigUseCase{productRepo: productRepo, reorderRepo: reorderRepo}
}

// Execute valida y hace upsert en product_reorder_config.
func (uc *UpdateReorderConfigUseCase) Execute(ctx context.Context, companyID string, in dto.ReorderConfigRequest) error {
	if companyID == "" || in.ProductID == "" || in.WarehouseID == "" {
		return domain.ErrInvalidInput
	}
	if in.ReorderPoint.LessThan(decimal.Zero) || in.MinStock.LessThan(decimal.Zero) || in.MaxStock.LessThan(decimal.Zero) {
		return domain.ErrInvalidInput
	}
	if in.LeadTimeDays < 0 {
		return domain.ErrInvalidInput
	}
	if in.MaxStock.LessThan(in.MinStock) {
		return domain.ErrInvalidInput
	}

	product, err := uc.productRepo.GetByID(in.ProductID)
	if err != nil {
		return err
	}
	if product == nil {
		return domain.ErrNotFound
	}
	if product.CompanyID != companyID {
		return domain.ErrForbidden
	}

	if uc.reorderRepo == nil {
		return domain.ErrInvalidInput
	}
	return uc.reorderRepo.UpsertProductReorderConfig(ctx, in)
}
