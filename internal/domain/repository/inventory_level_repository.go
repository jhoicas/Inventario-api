package repository

import (
	"context"

	"github.com/shopspring/decimal"
	"github.com/tu-usuario/inventory-pro/internal/domain/entity"
)

// ReplenishmentItem resultado crudo del repositorio para un producto bajo reorden.
type ReplenishmentItem struct {
	ProductID    string
	SKU          string
	ProductName  string
	CurrentStock decimal.Decimal
	ReorderPoint decimal.Decimal
	UnitCost     decimal.Decimal
	Price        decimal.Decimal
}

// InventoryLevelRepository define el puerto para consultar/actualizar stock por bodega+producto (DIP).
// Usado para lecturas rápidas y para mantener consistencia con movimientos.
type InventoryLevelRepository interface {
	Get(companyID, warehouseID, productID string) (*entity.InventoryLevel, error)
	Upsert(level *entity.InventoryLevel) error
	ListByWarehouse(warehouseID string, limit, offset int) ([]*entity.InventoryLevel, error)
	ListByProduct(productID string) ([]*entity.InventoryLevel, error)

	// GetProductsBelowReorderPoint devuelve los productos cuyo stock actual (en la bodega indicada)
	// es inferior a su punto de reorden, ordenados por mayor déficit primero.
	GetProductsBelowReorderPoint(ctx context.Context, companyID, warehouseID string) ([]ReplenishmentItem, error)
}
