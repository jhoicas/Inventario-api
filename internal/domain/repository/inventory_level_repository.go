package repository

import "github.com/tu-usuario/inventory-pro/internal/domain/entity"

// InventoryLevelRepository define el puerto para consultar/actualizar stock por bodega+producto (DIP).
// Usado para lecturas rápidas y para mantener consistencia con movimientos.
type InventoryLevelRepository interface {
	Get(companyID, warehouseID, productID string) (*entity.InventoryLevel, error)
	Upsert(level *entity.InventoryLevel) error
	ListByWarehouse(warehouseID string, limit, offset int) ([]*entity.InventoryLevel, error)
	ListByProduct(productID string) ([]*entity.InventoryLevel, error)
}
