package repository

import "github.com/tu-usuario/inventory-pro/internal/domain/entity"

// InvoiceRepository define el puerto de persistencia para Invoice y detalles.
type InvoiceRepository interface {
	Create(invoice *entity.Invoice) error
	CreateDetail(detail *entity.InvoiceDetail) error
	Update(invoice *entity.Invoice) error // Actualiza cufe, uuid, xml_signed, dian_status, qr_data
	GetByID(id string) (*entity.Invoice, error)
	GetDetailsByInvoiceID(invoiceID string) ([]*entity.InvoiceDetail, error)
}
