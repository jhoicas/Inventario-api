package entity

import (
	"time"

	"github.com/shopspring/decimal"
)

// Tipos de movimiento de inventario.
const (
	MovementTypeIN         = "IN"         // entrada
	MovementTypeOUT        = "OUT"        // salida
	MovementTypeADJUSTMENT = "ADJUSTMENT" // ajuste
	MovementTypeTRANSFER   = "TRANSFER"   // traslado entre bodegas
)

// InventoryMovement representa un movimiento de inventario (entrada, salida, ajuste o traslado).
type InventoryMovement struct {
	ID            string
	TransactionID string
	ProductID     string
	WarehouseID   string
	Type          string
	Quantity      decimal.Decimal // positivo entrada/ajuste+, negativo salida
	UnitCost      decimal.Decimal
	TotalCost     decimal.Decimal
	Date          time.Time
	CreatedAt     time.Time
	CreatedBy     string
}
