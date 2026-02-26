package repository

import "github.com/tu-usuario/inventory-pro/internal/domain/entity"

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
}
