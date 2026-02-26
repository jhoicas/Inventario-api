package repository

import (
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"time"
)

// InventoryMovementRepository define el puerto de persistencia para movimientos de inventario.
type InventoryMovementRepository interface {
	Create(movement *entity.InventoryMovement) error
	GetByID(id string) (*entity.InventoryMovement, error)
	ListByWarehouse(warehouseID string, from, to *time.Time, limit, offset int) ([]*entity.InventoryMovement, error)
	ListByProduct(productID string, from, to *time.Time, limit, offset int) ([]*entity.InventoryMovement, error)
}
