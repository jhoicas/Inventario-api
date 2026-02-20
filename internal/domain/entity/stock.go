package entity

import (
	"time"

	"github.com/shopspring/decimal"
)

// Stock representa el stock actual de un producto en una bodega (tabla intermedia/materializada).
type Stock struct {
	ProductID   string
	WarehouseID string
	Quantity    decimal.Decimal
	UpdatedAt   time.Time
}
