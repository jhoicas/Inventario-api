package entity

import (
	"time"

	"github.com/shopspring/decimal"
)

// MovementType representa el tipo de movimiento de inventario.
type MovementType string

// Tipos de movimiento de inventario.
const (
	MovementTypeIN         MovementType = "IN"         // entrada
	MovementTypeOUT        MovementType = "OUT"        // salida
	MovementTypeADJUSTMENT MovementType = "ADJUSTMENT" // ajuste
	MovementTypeTRANSFER   MovementType = "TRANSFER"   // traslado entre bodegas
	MovementTypeReturn     MovementType = "RETURN"     // devolución de venta (entrada por devolución)
)

// InventoryMovement representa un movimiento de inventario (entrada, salida, ajuste o traslado).
type InventoryMovement struct {
	ID            string
	TransactionID string
	ProductID     string
	WarehouseID   string
	Type          MovementType
	Quantity      decimal.Decimal // positivo entrada/ajuste+, negativo salida
	UnitCost      decimal.Decimal
	TotalCost     decimal.Decimal
	Date          time.Time
	CreatedAt     time.Time
	CreatedBy     string
}
