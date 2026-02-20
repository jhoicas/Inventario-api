package inventory

import (
	"context"

	"github.com/tu-usuario/inventory-pro/internal/application/dto"
)

// RegisterMovementFromRequest adapta el request HTTP al caso de uso RegisterMovement(ctx, MovementInputDTO).
// Usar desde handlers HTTP o desde otros casos de uso que tengan companyID, userID y dto.RegisterMovementRequest.
func (uc *RegisterMovementUseCase) RegisterMovementFromRequest(ctx context.Context, companyID, userID string, in dto.RegisterMovementRequest) error {
	input := MovementInputDTO{
		CompanyID:       companyID,
		UserID:          userID,
		ProductID:       in.ProductID,
		WarehouseID:     in.WarehouseID,
		FromWarehouseID: in.FromWarehouseID,
		ToWarehouseID:   in.ToWarehouseID,
		Type:            in.Type,
		Quantity:        in.Quantity,
		UnitCost:        in.UnitCost,
	}
	return uc.RegisterMovement(ctx, input)
}
