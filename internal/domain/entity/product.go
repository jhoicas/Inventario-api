package entity

import (
	"encoding/json"
	"time"

	"github.com/shopspring/decimal"
)

// Product representa un producto o SKU del inventario (multi-bodega).
// Cost es promedio ponderado calculado desde movimientos; Stock se maneja por bodega en InventoryLevel.
type Product struct {
	ID           string
	CompanyID    string
	SKU          string    // código único por empresa
	Name         string
	Description  string
	Price        decimal.Decimal // precio de venta
	Cost         decimal.Decimal // costo promedio ponderado (inicia en 0)
	TaxRate      decimal.Decimal // IVA Colombia: 0, 0.05 (5%), 0.19 (19%)
	UNSPSC_Code  string
	UnitMeasure  string
	Attributes   json.RawMessage
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
