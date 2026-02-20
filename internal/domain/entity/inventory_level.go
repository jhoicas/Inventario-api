package entity

import "time"

// InventoryLevel representa el stock actual de un producto en una bodega.
// Derivado de los movimientos; puede materializarse para consultas rápidas.
type InventoryLevel struct {
	CompanyID   string
	WarehouseID string
	ProductID   string
	Quantity    float64
	UpdatedAt   time.Time
}
