package repository

import "github.com/jhoicas/Inventario-api/internal/domain/entity"

// InvoiceRepository define el puerto de persistencia para Invoice y detalles.
type InvoiceRepository interface {
	Create(invoice *entity.Invoice) error
	CreateDetail(detail *entity.InvoiceDetail) error
	// Update actualiza todos los campos DIAN de la factura:
	// cufe, uuid, xml_signed, dian_status, qr_data, track_id_dian, dian_errors.
	Update(invoice *entity.Invoice) error
	GetByID(id string) (*entity.Invoice, error)
	GetDetailsByInvoiceID(invoiceID string) ([]*entity.InvoiceDetail, error)
	// GetDIANStatus devuelve solo los campos de estado DIAN (ligero, para polling).
	GetDIANStatus(id string) (*entity.Invoice, error)

	// UpdateReturnStatus actualiza el estado de devolución lógico de la factura original
	// (por ejemplo, 'Returned' o 'Partially_Returned'). La implementación puede usar
	// una columna dedicada o el campo notes según el esquema de BD.
	UpdateReturnStatus(invoiceID string, status string) error

	// List devuelve facturas filtradas y paginadas para una empresa.
	// Los campos vacíos/cero de InvoiceListFilter se ignoran.
	List(filter InvoiceListFilter) ([]*entity.Invoice, int, error)
}

// InvoiceListFilter parámetros de consulta para el listado de facturas.
type InvoiceListFilter struct {
	CompanyID  string
	StartDate  string // YYYY-MM-DD; vacío = sin límite inferior
	EndDate    string // YYYY-MM-DD; vacío = sin límite superior
	CustomerID string
	DIANStatus string
	Prefix     string
	Limit      int
	Offset     int
}
