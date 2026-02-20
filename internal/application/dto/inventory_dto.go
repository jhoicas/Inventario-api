package dto

import "github.com/shopspring/decimal"

// RegisterMovementRequest body para POST /api/inventory/movements.
type RegisterMovementRequest struct {
	ProductID       string           `json:"product_id"`
	WarehouseID     string           `json:"warehouse_id,omitempty"`
	FromWarehouseID string           `json:"from_warehouse_id,omitempty"`
	ToWarehouseID   string           `json:"to_warehouse_id,omitempty"`
	Type            string           `json:"type"`
	Quantity        decimal.Decimal  `json:"quantity"`
	UnitCost        *decimal.Decimal `json:"unit_cost,omitempty"`
}
