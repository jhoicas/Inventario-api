package dto

import "github.com/shopspring/decimal"

// CreateCustomerRequest body para POST /api/customers.
type CreateCustomerRequest struct {
	Name   string `json:"name"`
	TaxID  string `json:"tax_id"`
	Email  string `json:"email,omitempty"`
	Phone  string `json:"phone,omitempty"`
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

// CreateInvoiceRequest body para POST /api/invoices.
// WarehouseID: bodega de la cual se descuenta el inventario.
type CreateInvoiceRequest struct {
	CustomerID   string               `json:"customer_id"`
	WarehouseID  string               `json:"warehouse_id"`
	Prefix       string               `json:"prefix"`
	Number       string               `json:"number,omitempty"` // opcional; si va vacío se puede generar
	Items        []InvoiceItemRequest `json:"items"`
}

// InvoiceItemRequest línea de factura (producto, cantidad, precio unitario).
type InvoiceItemRequest struct {
	ProductID  string          `json:"product_id"`
	Quantity   decimal.Decimal `json:"quantity"`
	UnitPrice  decimal.Decimal `json:"unit_price"`
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

// InvoiceDIANStatusDTO respuesta ligera para el endpoint de polling
// GET /api/invoices/:id/status.
// El frontend consulta este endpoint periódicamente hasta que dian_status sea
// "EXITOSO" o "RECHAZADO".
type InvoiceDIANStatusDTO struct {
	ID         string `json:"id"`
	DIANStatus string `json:"dian_status"` // DRAFT|SIGNED|EXITOSO|RECHAZADO|ERROR_GENERATION
	CUFE       string `json:"cufe"`         // Código único de factura (SHA-384)
	TrackID    string `json:"track_id"`     // ZipKey devuelto por el WS DIAN
	Errors     string `json:"errors"`       // Mensajes de rechazo de la DIAN (vacío si OK)
}
