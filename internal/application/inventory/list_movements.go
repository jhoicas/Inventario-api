package inventory

import (
	"context"
	"strings"

	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
)

// ListMovementsUseCase lista movimientos de inventario con filtros y paginación.
type ListMovementsUseCase struct {
	movementRepo repository.InventoryMovementRepository
}

// NewListMovementsUseCase construye el caso de uso.
func NewListMovementsUseCase(movementRepo repository.InventoryMovementRepository) *ListMovementsUseCase {
	return &ListMovementsUseCase{movementRepo: movementRepo}
}

// Execute lista movimientos de la empresa aplicando filtros opcionales.
func (uc *ListMovementsUseCase) Execute(ctx context.Context, companyID string, in dto.InventoryMovementFilter) (*dto.InventoryMovementListResponse, error) {
	if companyID == "" {
		return nil, domain.ErrInvalidInput
	}

	limit := in.Limit
	if limit <= 0 {
		limit = 20
	}
	offset := in.Offset
	if offset < 0 {
		offset = 0
	}

	items, total, err := uc.movementRepo.List(companyID, repository.MovementFilters{
		ProductID:   strings.TrimSpace(in.ProductID),
		WarehouseID: strings.TrimSpace(in.WarehouseID),
		Type:        strings.ToUpper(strings.TrimSpace(in.Type)),
		StartDate:   in.StartDate,
		EndDate:     in.EndDate,
		Limit:       limit,
		Offset:      offset,
	})
	if err != nil {
		return nil, err
	}

	out := &dto.InventoryMovementListResponse{
		Items: make([]dto.InventoryMovementDTO, 0, len(items)),
		Total: total,
	}
	for _, m := range items {
		out.Items = append(out.Items, dto.InventoryMovementDTO{
			ID:            m.ID,
			TransactionID: m.TransactionID,
			ProductID:     m.ProductID,
			WarehouseID:   m.WarehouseID,
			Type:          string(m.Type),
			Quantity:      m.Quantity,
			UnitCost:      m.UnitCost,
			TotalCost:     m.TotalCost,
			Date:          m.Date,
			CreatedAt:     m.CreatedAt,
			CreatedBy:     m.CreatedBy,
		})
	}

	return out, nil
}
