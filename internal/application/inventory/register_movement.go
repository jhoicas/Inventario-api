package inventory

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
)

// RegisterMovementFromRequest adapta el request HTTP al caso de uso RegisterMovement(ctx, MovementInputDTO).
// Usar desde handlers HTTP o desde otros casos de uso que tengan companyID, userID y dto.RegisterMovementRequest.
func (uc *RegisterMovementUseCase) RegisterMovementFromRequest(ctx context.Context, companyID, userID string, in dto.RegisterMovementRequest) error {
	input := MovementInputDTO{
		CompanyID:        companyID,
		UserID:           userID,
		ProductID:        in.ProductID,
		WarehouseID:      in.WarehouseID,
		FromWarehouseID:  in.FromWarehouseID,
		ToWarehouseID:    in.ToWarehouseID,
		Type:             in.Type,
		Quantity:         in.Quantity,
		UnitCost:         in.UnitCost,
		AdjustmentReason: in.AdjustmentReason,
	}
	return uc.RegisterMovement(ctx, input)
}

// RegisterAdjustmentFromRequest valida la razón, pre-genera el movement_id, fuerza Type="ADJUSTMENT"
// y delega al caso de uso RegisterMovement. Devuelve el movement_id generado.
func (uc *RegisterMovementUseCase) RegisterAdjustmentFromRequest(ctx context.Context, companyID, userID string, in dto.RegisterMovementRequest) (string, error) {
	// Validar adjustment_reason
	if in.AdjustmentReason == "" {
		return "", fmt.Errorf("%w: adjustment_reason es obligatorio", domain.ErrInvalidInput)
	}
	validReason := false
	for _, r := range dto.AdjustmentReasons {
		if r == in.AdjustmentReason {
			validReason = true
			break
		}
	}
	if !validReason {
		return "", fmt.Errorf("%w: adjustment_reason inválida (MERMA|ROBO|VENCIMIENTO|CONTEO_FISICO|DETERIORO|OTRO)", domain.ErrInvalidInput)
	}

	movementID := uuid.New().String()
	input := MovementInputDTO{
		MovementID:       movementID,
		CompanyID:        companyID,
		UserID:           userID,
		ProductID:        in.ProductID,
		WarehouseID:      in.WarehouseID,
		Type:             string(entity.MovementTypeADJUSTMENT),
		Quantity:         in.Quantity,
		UnitCost:         in.UnitCost,
		AdjustmentReason: in.AdjustmentReason,
	}
	if err := uc.RegisterMovement(ctx, input); err != nil {
		return "", err
	}
	return movementID, nil
}
