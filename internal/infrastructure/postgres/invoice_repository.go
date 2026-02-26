package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/tu-usuario/inventory-pro/internal/domain/entity"
	"github.com/tu-usuario/inventory-pro/internal/domain/repository"
)

var _ repository.InvoiceRepository = (*InvoiceRepo)(nil)

// InvoiceRepo implementación de InvoiceRepository (usable con pool o tx).
type InvoiceRepo struct {
	q Querier
}

// NewInvoiceRepository construye el adaptador. Pasar pool o tx (Querier).
func NewInvoiceRepository(q Querier) *InvoiceRepo {
	return &InvoiceRepo{q: q}
}

// Create persiste la cabecera de la factura.
func (r *InvoiceRepo) Create(invoice *entity.Invoice) error {
	if invoice.ID == "" {
		invoice.ID = uuid.New().String()
	}
	query := `
		INSERT INTO invoices (id, company_id, customer_id, prefix, number, date, net_total, tax_total, grand_total, dian_status, cufe, uuid, xml_signed, qr_data, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)`
	_, err := r.q.Exec(context.Background(), query,
		invoice.ID, invoice.CompanyID, invoice.CustomerID, invoice.Prefix, invoice.Number,
		invoice.Date, invoice.NetTotal, invoice.TaxTotal, invoice.GrandTotal,
		invoice.DIAN_Status, nullIfEmpty(invoice.CUFE), nullIfEmpty(invoice.UUID), nullIfEmpty(invoice.XMLSigned), nullIfEmpty(invoice.QRData),
		invoice.CreatedAt, invoice.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("invoice number already exists: %w", err)
		}
		return fmt.Errorf("insert invoice: %w", err)
	}
	return nil
}

// CreateDetail persiste una línea de detalle.
func (r *InvoiceRepo) CreateDetail(detail *entity.InvoiceDetail) error {
	if detail.ID == "" {
		detail.ID = uuid.New().String()
	}
	query := `
		INSERT INTO invoice_details (id, invoice_id, product_id, quantity, unit_price, tax_rate, subtotal)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.q.Exec(context.Background(), query,
		detail.ID, detail.InvoiceID, detail.ProductID, detail.Quantity, detail.UnitPrice,
		detail.TaxRate, detail.Subtotal,
	)
	if err != nil {
		return fmt.Errorf("insert invoice detail: %w", err)
	}
	return nil
}

// Update actualiza todos los campos DIAN de la factura.
func (r *InvoiceRepo) Update(invoice *entity.Invoice) error {
	query := `
		UPDATE invoices
		SET cufe          = COALESCE($2,  cufe),
		    uuid          = COALESCE($3,  uuid),
		    xml_signed    = $4,
		    dian_status   = $5,
		    qr_data       = COALESCE($6,  qr_data),
		    track_id_dian = COALESCE($7,  track_id_dian),
		    dian_errors   = COALESCE($8,  dian_errors),
		    updated_at    = $9
		WHERE id = $1`
	_, err := r.q.Exec(context.Background(), query,
		invoice.ID,
		nullIfEmpty(invoice.CUFE),
		nullIfEmpty(invoice.UUID),
		nullIfEmpty(invoice.XMLSigned),
		invoice.DIAN_Status,
		nullIfEmpty(invoice.QRData),
		nullIfEmpty(invoice.TrackID),
		nullIfEmpty(invoice.DIANErrors),
		invoice.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update invoice: %w", err)
	}
	return nil
}

// GetByID obtiene una factura completa por ID.
func (r *InvoiceRepo) GetByID(id string) (*entity.Invoice, error) {
	query := `
		SELECT id, company_id, customer_id, prefix, number, date,
		       net_total, tax_total, grand_total, dian_status,
		       cufe, uuid, xml_signed, qr_data, track_id_dian, dian_errors,
		       created_at, updated_at
		FROM invoices WHERE id = $1`
	var inv entity.Invoice
	var cufe, uuid, xmlSigned, qrData, trackID, dianErrors *string
	err := r.q.QueryRow(context.Background(), query, id).Scan(
		&inv.ID, &inv.CompanyID, &inv.CustomerID, &inv.Prefix, &inv.Number,
		&inv.Date, &inv.NetTotal, &inv.TaxTotal, &inv.GrandTotal,
		&inv.DIAN_Status, &cufe, &uuid, &xmlSigned, &qrData,
		&trackID, &dianErrors,
		&inv.CreatedAt, &inv.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get invoice: %w", err)
	}
	derefStr := func(p *string) string {
		if p != nil {
			return *p
		}
		return ""
	}
	inv.CUFE = derefStr(cufe)
	inv.UUID = derefStr(uuid)
	inv.XMLSigned = derefStr(xmlSigned)
	inv.QRData = derefStr(qrData)
	inv.TrackID = derefStr(trackID)
	inv.DIANErrors = derefStr(dianErrors)
	return &inv, nil
}

// GetDIANStatus devuelve solo los campos de estado DIAN (consulta ligera para polling).
func (r *InvoiceRepo) GetDIANStatus(id string) (*entity.Invoice, error) {
	const query = `
		SELECT id, company_id, dian_status,
		       COALESCE(cufe, ''), COALESCE(track_id_dian, ''), COALESCE(dian_errors, '')
		FROM invoices WHERE id = $1`
	var inv entity.Invoice
	err := r.q.QueryRow(context.Background(), query, id).Scan(
		&inv.ID, &inv.CompanyID, &inv.DIAN_Status,
		&inv.CUFE, &inv.TrackID, &inv.DIANErrors,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get invoice dian status: %w", err)
	}
	return &inv, nil
}

// GetDetailsByInvoiceID obtiene todas las líneas de una factura.
func (r *InvoiceRepo) GetDetailsByInvoiceID(invoiceID string) ([]*entity.InvoiceDetail, error) {
	query := `
		SELECT id, invoice_id, product_id, quantity, unit_price, tax_rate, subtotal
		FROM invoice_details WHERE invoice_id = $1 ORDER BY id`
	rows, err := r.q.Query(context.Background(), query, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("list invoice details: %w", err)
	}
	defer rows.Close()
	var list []*entity.InvoiceDetail
	for rows.Next() {
		var d entity.InvoiceDetail
		if err := rows.Scan(&d.ID, &d.InvoiceID, &d.ProductID, &d.Quantity, &d.UnitPrice, &d.TaxRate, &d.Subtotal); err != nil {
			return nil, fmt.Errorf("scan detail: %w", err)
		}
		list = append(list, &d)
	}
	return list, rows.Err()
}

func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
