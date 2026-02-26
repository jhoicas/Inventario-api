package repository

import (
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"time"
)

// StockMovementRepository define el puerto de persistencia para movimientos de inventario (DIP).
type StockMovementRepository interface {
	Create(movement *entity.StockMovement) error
	GetByID(id string) (*entity.StockMovement, error)
	ListByWarehouse(warehouseID string, from, to *time.Time, limit, offset int) ([]*entity.StockMovement, error)
	ListByProduct(productID string, from, to *time.Time, limit, offset int) ([]*entity.StockMovement, error)
}
