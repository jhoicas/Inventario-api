package entity

import "time"

// Tipos de movimiento de inventario (value object conceptual).
const (
	MovementTypeIn     = "in"     // entrada
	MovementTypeOut    = "out"    // salida
	MovementTypeAdjust = "adjust" // ajuste
	MovementTypeTransfer = "transfer" // entre bodegas (opcional)
)

// StockMovement representa un movimiento de inventario (entrada, salida o ajuste).
type StockMovement struct {
	ID          string
	CompanyID   string
	WarehouseID string
	ProductID   string
	Type        string    // in, out, adjust, transfer
	Quantity    float64   // positivo para in/adjust+, negativo para out
	Reference   string    // factura, orden, nota de ajuste, etc.
	Notes       string
	CreatedAt   time.Time
	CreatedBy   string    // UserID
}
