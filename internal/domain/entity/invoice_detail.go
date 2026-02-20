package entity

import "github.com/shopspring/decimal"

// InvoiceDetail representa una línea de detalle de una factura.
type InvoiceDetail struct {
	ID        string
	InvoiceID string
	ProductID string
	Quantity  decimal.Decimal
	UnitPrice decimal.Decimal
	TaxRate   decimal.Decimal
	Subtotal  decimal.Decimal
}
