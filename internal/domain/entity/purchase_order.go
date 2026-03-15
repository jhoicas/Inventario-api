package entity

import (
	"time"

	"github.com/shopspring/decimal"
)

// Estados de una orden de compra.
const (
	PurchaseOrderStatusDraft          = "BORRADOR"
	PurchaseOrderStatusSent           = "ENVIADA"
	PurchaseOrderStatusConfirmed      = "CONFIRMADA"
	PurchaseOrderStatusPartialReceipt = "RECIBIDA_PARCIAL"
	PurchaseOrderStatusClosed         = "CERRADA"
)

// PurchaseOrder representa la cabecera de una orden de compra.
type PurchaseOrder struct {
	ID           string
	CompanyID    string
	SupplierID   string
	SupplierName string
	Number       string
	Date         time.Time
	Status       string
	Items        []PurchaseOrderItem
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// PurchaseOrderItem representa una línea de una orden de compra.
type PurchaseOrderItem struct {
	ProductID string
	Quantity  decimal.Decimal
	UnitCost  decimal.Decimal
}
