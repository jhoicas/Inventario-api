package inventory

import (
	"context"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
)

// GetMovementsUseCase lista movimientos de inventario y calcula balance acumulado por período.
type GetMovementsUseCase struct {
	movementRepo repository.InventoryMovementRepository
}

// NewGetMovementsUseCase construye el caso de uso.
func NewGetMovementsUseCase(movementRepo repository.InventoryMovementRepository) *GetMovementsUseCase {
	return &GetMovementsUseCase{movementRepo: movementRepo}
}

// Execute devuelve movimientos paginados con balance acumulado en cada item.
func (uc *GetMovementsUseCase) Execute(ctx context.Context, companyID string, f dto.MovementFiltersDTO) (*dto.PaginatedMovementsDTO, error) {
	if companyID == "" {
		return nil, domain.ErrInvalidInput
	}

	limit := f.Limit
	if limit <= 0 {
		limit = 20
	}
	offset := f.Offset
	if offset < 0 {
		offset = 0
	}

	filters := repository.MovementFilters{
		ProductID:   strings.TrimSpace(f.ProductID),
		WarehouseID: strings.TrimSpace(f.WarehouseID),
		Type:        strings.ToUpper(strings.TrimSpace(f.Type)),
		StartDate:   f.StartDate,
		EndDate:     f.EndDate,
		Limit:       limit,
		Offset:      offset,
	}

	items, total, err := uc.movementRepo.List(companyID, filters)
	if err != nil {
		return nil, err
	}

	balance := decimal.Zero
	if offset > 0 {
		prevFilters := filters
		prevFilters.Limit = offset
		prevFilters.Offset = 0
		prevItems, _, err := uc.movementRepo.List(companyID, prevFilters)
		if err != nil {
			return nil, err
		}
		for _, m := range prevItems {
			balance = balance.Add(m.Quantity)
		}
	}

	out := &dto.PaginatedMovementsDTO{
		Items: make([]dto.MovementDTO, 0, len(items)),
		Total: total,
	}

	for _, m := range items {
		balance = balance.Add(m.Quantity)
		out.Items = append(out.Items, dto.MovementDTO{
			ID:            m.ID,
			TransactionID: m.TransactionID,
			ProductID:     m.ProductID,
			WarehouseID:   m.WarehouseID,
			Type:          string(m.Type),
			Quantity:      m.Quantity,
			Balance:       balance,
			UnitCost:      m.UnitCost,
			TotalCost:     m.TotalCost,
			Notes:         m.Notes,
			Date:          m.Date,
			CreatedAt:     m.CreatedAt,
			CreatedBy:     m.CreatedBy,
		})
	}

	return out, nil
}
