package dto

import "github.com/shopspring/decimal"

// CreateCustomerRequest body para POST /api/customers.
type CreateCustomerRequest struct {
	Name  string `json:"name"`
	TaxID string `json:"tax_id"`
	Email string `json:"email,omitempty"`
	Phone string `json:"phone,omitempty"`
}

// CustomerResponse cliente en respuestas.
type CustomerResponse struct {
	ID        string `json:"id"`
	CompanyID string `json:"company_id"`
	Name      string `json:"name"`
	TaxID     string `json:"tax_id"`
	Email     string `json:"email,omitempty"`
	Phone     string `json:"phone,omitempty"`
}

// UpdateCustomerRequest body para actualizar un cliente.
type UpdateCustomerRequest struct {
	Name  string `json:"name"`
	TaxID string `json:"tax_id"`
	Email string `json:"email"`
	Phone string `json:"phone"`
}

// CreateInvoiceRequest body para POST /api/invoices.
// WarehouseID: bodega de la cual se descuenta el inventario.
type CreateInvoiceRequest struct {
	CustomerID  string               `json:"customer_id"`
	WarehouseID string               `json:"warehouse_id"`
	Prefix      string               `json:"prefix"`
	Number      string               `json:"number,omitempty"` // opcional; si va vacío se puede generar
	Items       []InvoiceItemRequest `json:"items"`
}

// InvoiceItemRequest línea de factura (producto, cantidad, precio unitario).
type InvoiceItemRequest struct {
	ProductID string          `json:"product_id"`
	Quantity  decimal.Decimal `json:"quantity"`
	UnitPrice decimal.Decimal `json:"unit_price"`
}

// ReturnItemRequest línea de devolución (producto y cantidad devuelta).
type ReturnItemRequest struct {
	ProductID string          `json:"product_id"`
	Quantity  decimal.Decimal `json:"quantity"`
}

// ReturnInvoiceRequest body para POST /api/invoices/{id}/return.
// WarehouseID: bodega a la que se reingresa el stock devuelto.
type ReturnInvoiceRequest struct {
	WarehouseID string              `json:"warehouse_id"`
	Items       []ReturnItemRequest `json:"items"`
	Reason      string              `json:"reason,omitempty"`
}

// DebitNoteItemRequest línea de nota débito (producto, cantidad y precio unitario).
type DebitNoteItemRequest struct {
	ProductID string          `json:"product_id"`
	Quantity  decimal.Decimal `json:"quantity"`
	UnitPrice decimal.Decimal `json:"unit_price"`
}

// CreateDebitNoteRequest body para POST /api/invoices/{id}/debit-note.
type CreateDebitNoteRequest struct {
	Reason string                 `json:"reason,omitempty"`
	Items  []DebitNoteItemRequest `json:"items"`
}

// DebitNoteResponse respuesta resumida de creación de nota débito.
type DebitNoteResponse struct {
	DebitNoteID string `json:"debit_note_id"`
	CUFE        string `json:"cufe,omitempty"`
	DIANStatus  string `json:"dian_status"`
}

// CreateVoidInvoiceRequest body para POST /api/invoices/{id}/void.
// concept_code: 1=Devolución, 2=Anulación, 3=Descuento, 4=Ajuste precio, 5=Otros.
type CreateVoidInvoiceRequest struct {
	ConceptCode int    `json:"concept_code"`
	Reason      string `json:"reason"`
}

// SendCustomEmailRequest body para POST /api/emails/send.
type SendCustomEmailRequest struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

// VoidInvoiceResponse respuesta resumida de anulación por nota crédito.
type VoidInvoiceResponse struct {
	CreditNoteID string `json:"credit_note_id"`
	CUFE         string `json:"cufe,omitempty"`
	DIANStatus   string `json:"dian_status"`
}

// InvoiceResponse factura con detalle para GET /api/invoices/:id.
type InvoiceResponse struct {
	ID           string                  `json:"id"`
	CompanyID    string                  `json:"company_id"`
	CustomerID   string                  `json:"customer_id"`
	CustomerName string                  `json:"customer_name,omitempty"`
	Prefix       string                  `json:"prefix"`
	Number       string                  `json:"number"`
	Date         string                  `json:"date"`
	NetTotal     decimal.Decimal         `json:"net_total"`
	TaxTotal     decimal.Decimal         `json:"tax_total"`
	GrandTotal   decimal.Decimal         `json:"grand_total"`
	DIAN_Status  string                  `json:"dian_status"`
	CUFE         string                  `json:"cufe,omitempty"`
	QRData       string                  `json:"qr_data,omitempty"` // String para generar QR (NumFac|FecFac|...|Cufe|UrlValidacionDIAN)
	Details      []InvoiceDetailResponse `json:"details"`
}

// InvoiceDetailResponse línea de detalle en la respuesta.
type InvoiceDetailResponse struct {
	ID        string          `json:"id"`
	ProductID string          `json:"product_id"`
	Quantity  decimal.Decimal `json:"quantity"`
	UnitPrice decimal.Decimal `json:"unit_price"`
	TaxRate   decimal.Decimal `json:"tax_rate"`
	Subtotal  decimal.Decimal `json:"subtotal"`
}

// InvoiceFilter parámetros de filtrado y paginación para GET /api/invoices.
type InvoiceFilter struct {
	StartDate  string `query:"start_date"` // YYYY-MM-DD
	EndDate    string `query:"end_date"`   // YYYY-MM-DD
	CustomerID string `query:"customer_id"`
	DIANStatus string `query:"dian_status"`
	Prefix     string `query:"prefix"`
	Limit      int    `query:"limit"`
	Offset     int    `query:"offset"`
}

// InvoiceListResponse respuesta paginada de facturas.
type InvoiceListResponse struct {
	Items  []InvoiceResponse `json:"items"`
	Total  int               `json:"total"`
	Limit  int               `json:"limit"`
	Offset int               `json:"offset"`
}

// InvoiceSummaryDTO resumen ligero de factura para uso en historial de compras por cliente.
type InvoiceSummaryDTO struct {
	ID           string          `json:"id"`
	Prefix       string          `json:"prefix"`
	Number       string          `json:"number"`
	Date         string          `json:"date"`
	GrandTotal   decimal.Decimal `json:"grand_total"`
	DocumentType string          `json:"document_type"`
	DIANStatus   string          `json:"dian_status"`
}

// CustomerPurchaseStatsDTO estadísticas de compra agregadas para un cliente.
type CustomerPurchaseStatsDTO struct {
	TotalPurchases   decimal.Decimal `json:"total_purchases"`
	AvgTicket        decimal.Decimal `json:"avg_ticket"`
	LastPurchaseDate string          `json:"last_purchase_date"` // RFC3339 o vacío
	InvoiceCount     int             `json:"invoice_count"`
}

// PurchaseHistoryResponse respuesta de GET /api/crm/customers/:id/purchase-history.
type PurchaseHistoryResponse struct {
	Stats    CustomerPurchaseStatsDTO `json:"stats"`
	Invoices []InvoiceSummaryDTO      `json:"invoices"`
	Total    int64                    `json:"total"`
}

// InvoiceDIANStatusDTO respuesta ligera para el endpoint de polling
// GET /api/invoices/:id/status.
// El frontend consulta este endpoint periódicamente hasta que dian_status sea
// "EXITOSO" o "RECHAZADO".
type InvoiceDIANStatusDTO struct {
	ID         string `json:"id"`
	DIANStatus string `json:"dian_status"` // DRAFT|SIGNED|EXITOSO|RECHAZADO|ERROR_GENERATION
	CUFE       string `json:"cufe"`        // Código único de factura (SHA-384)
	TrackID    string `json:"track_id"`    // ZipKey devuelto por el WS DIAN
	Errors     string `json:"errors"`      // Mensajes de rechazo de la DIAN (vacío si OK)
}

// DIANSummaryDTO resumen de estados DIAN para dashboard de facturación.
type DIANSummaryDTO struct {
	SentToday int `json:"sent_today"`
	Pending   int `json:"pending"`
	Rejected  int `json:"rejected"`
}
