package repository

import (
	"time"

	"github.com/jhoicas/Inventario-api/internal/domain/entity"
)

// MovementFilters filtros para consultar movimientos de inventario.
type MovementFilters struct {
	ProductID   string
	WarehouseID string
	Type        string
	StartDate   time.Time
	EndDate     time.Time
	Limit       int
	Offset      int
}

// InventoryMovementRepository define el puerto de persistencia para movimientos de inventario.
type InventoryMovementRepository interface {
	Create(movement *entity.InventoryMovement) error
	GetByID(id string) (*entity.InventoryMovement, error)
	List(companyID string, f MovementFilters) ([]*entity.InventoryMovement, int64, error)
	ListByWarehouse(warehouseID string, from, to *time.Time, limit, offset int) ([]*entity.InventoryMovement, error)
	ListByProduct(productID string, from, to *time.Time, limit, offset int) ([]*entity.InventoryMovement, error)
}
