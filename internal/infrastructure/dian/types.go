// Package dian implementa la generación de XML UBL 2.1 para factura electrónica DIAN (Colombia).
package dian

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
)

// BillingResolutionData datos de la resolución de facturación DIAN (obligatorios en ExtensionContent).
type BillingResolutionData struct {
	Number      string    // Número de resolución (ej: 18764000000001)
	Prefix      string    // Prefijo autorizado (ej: SETP)
	From        int64     // Número desde
	To          int64     // Número hasta
	DateFrom    time.Time // Fecha desde
	DateTo      time.Time // Fecha hasta
}

// InvoiceLineForXML línea de factura con datos de producto para el XML (descripción, unidad, código).
type InvoiceLineForXML struct {
	Detail       *entity.InvoiceDetail
	ProductName  string
	ProductCode  string          // SKU o código
	UnitCode     string          // Código unidad medida DIAN (94, KGM, etc.)
	Quantity     decimal.Decimal
	UnitPrice    decimal.Decimal
	TaxRate      decimal.Decimal
	Subtotal     decimal.Decimal
}

// InvoiceBuildContext contexto con todos los datos necesarios para construir el XML de la factura.
type InvoiceBuildContext struct {
	Invoice   *entity.Invoice
	Company   *entity.Company   // Emisor (AccountingSupplierParty)
	Customer  *entity.Customer  // Cliente (AccountingCustomerParty)
	Details   []InvoiceLineForXML
	Resolution *BillingResolutionData

	// Opcionales (si la factura los tiene en BD)
	PaymentFormCode       string    // 1=Contado, 2=Crédito
	PaymentMethodCode     string    // 10=Efectivo, 47=Transferencia, etc.
	DueDate               *time.Time
	IssueDate             *time.Time // Si no se usa Invoice.Date
	CustomerIdentificationTypeCode string // 13=CC, 31=NIT
	CompanyIdentificationTypeCode  string
}
