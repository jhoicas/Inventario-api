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

// ReplenishmentSuggestionDTO representa una sugerencia de reposición para un SKU
// que se encuentra por debajo de su punto de reorden.
type ReplenishmentSuggestionDTO struct {
	ProductID             string          `json:"product_id"`
	SKU                   string          `json:"sku"`
	ProductName           string          `json:"product_name"`
	CurrentStock          decimal.Decimal `json:"current_stock"`
	ReorderPoint          decimal.Decimal `json:"reorder_point"`
	IdealStock            decimal.Decimal `json:"ideal_stock"`           // ReorderPoint * 1.5
	SuggestedOrderQty     decimal.Decimal `json:"suggested_order_qty"`   // IdealStock - CurrentStock
	UnitCost              decimal.Decimal `json:"unit_cost"`             // costo promedio ponderado
	EstimatedOrderCost    decimal.Decimal `json:"estimated_order_cost"`  // SuggestedOrderQty * UnitCost
	GrossMarginPct        decimal.Decimal `json:"gross_margin_pct"`      // % margen histórico (puede ser 0 si sin ventas)
	UnitsSoldLast90Days   decimal.Decimal `json:"units_sold_last_90d"`   // volumen de ventas reciente
	Priority              int             `json:"priority"`              // 1 = más urgente
}
