package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
	"github.com/shopspring/decimal"
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
		INSERT INTO invoices (
			id, company_id, customer_id, prefix, number, date,
			net_total, tax_total, grand_total, dian_status,
			cufe, uuid, xml_signed, qr_data, track_id_dian, dian_errors,
			document_type,
			original_invoice_id,
			original_invoice_number,
			original_invoice_cufe,
			original_invoice_issue_on,
			discrepancy_code,
			discrepancy_reason,
			created_at, updated_at
		)
		VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16,
			$17,
			$18,
			$19,
			$20,
			$21,
			$22,
			$23,
			$24, $25
		)`
	_, err := r.q.Exec(context.Background(), query,
		invoice.ID, invoice.CompanyID, invoice.CustomerID, invoice.Prefix, invoice.Number,
		invoice.Date, invoice.NetTotal, invoice.TaxTotal, invoice.GrandTotal,
		invoice.DIAN_Status,
		nullIfEmpty(invoice.CUFE),
		nullIfEmpty(invoice.UUID),
		nullIfEmpty(invoice.XMLSigned),
		nullIfEmpty(invoice.QRData),
		nullIfEmpty(invoice.TrackID),
		nullIfEmpty(invoice.DIANErrors),
		nullIfEmpty(invoice.DocumentType),
		nullIfEmpty(invoice.OriginalInvoiceID),
		nullIfEmpty(invoice.OriginalInvoiceNumber),
		nullIfEmpty(invoice.OriginalInvoiceCUFE),
		invoice.OriginalInvoiceIssueOn,
		func() *string {
			if invoice.DiscrepancyCode == "" {
				return nil
			}
			s := string(invoice.DiscrepancyCode)
			return &s
		}(),
		nullIfEmpty(invoice.DiscrepancyReason),
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
		       document_type,
		       original_invoice_id,
		       original_invoice_number,
		       original_invoice_cufe,
		       original_invoice_issue_on,
		       discrepancy_code,
		       discrepancy_reason,
		       created_at, updated_at
		FROM invoices WHERE id = $1`
	var inv entity.Invoice
	var cufe, uuid, xmlSigned, qrData, trackID, dianErrors *string
	var docType, origInvID, origInvNumber, origInvCUFE, discCode, discReason *string
	var origIssueOn *time.Time
	err := r.q.QueryRow(context.Background(), query, id).Scan(
		&inv.ID, &inv.CompanyID, &inv.CustomerID, &inv.Prefix, &inv.Number,
		&inv.Date, &inv.NetTotal, &inv.TaxTotal, &inv.GrandTotal,
		&inv.DIAN_Status, &cufe, &uuid, &xmlSigned, &qrData,
		&trackID, &dianErrors,
		&docType,
		&origInvID,
		&origInvNumber,
		&origInvCUFE,
		&origIssueOn,
		&discCode,
		&discReason,
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
	inv.DocumentType = derefStr(docType)
	inv.OriginalInvoiceID = derefStr(origInvID)
	inv.OriginalInvoiceNumber = derefStr(origInvNumber)
	inv.OriginalInvoiceCUFE = derefStr(origInvCUFE)
	if origIssueOn != nil {
		inv.OriginalInvoiceIssueOn = *origIssueOn
	}
	if discCode != nil {
		inv.DiscrepancyCode = entity.CreditNoteConcept(*discCode)
	}
	inv.DiscrepancyReason = derefStr(discReason)
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

// GetDIANSummary devuelve los contadores DIAN para tablero de facturación.
func (r *InvoiceRepo) GetDIANSummary(companyID string) (*repository.DIANSummary, error) {
	const query = `
		SELECT
			COUNT(*) FILTER (WHERE date = CURRENT_DATE AND dian_status = 'Sent') AS sent_today,
			COUNT(*) FILTER (WHERE dian_status IN ('Pending', 'DRAFT'))         AS pending,
			COUNT(*) FILTER (WHERE dian_status = 'Error')                        AS rejected
		FROM invoices
		WHERE company_id = $1`

	var out repository.DIANSummary
	err := r.q.QueryRow(context.Background(), query, companyID).Scan(
		&out.SentToday,
		&out.Pending,
		&out.Rejected,
	)
	if err != nil {
		return nil, fmt.Errorf("get dian summary: %w", err)
	}
	return &out, nil
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

// UpdateReturnStatus marca una factura como devuelta total o parcialmente.
// Esta implementación almacena el estado en la columna notes, preservando cualquier contenido previo.
func (r *InvoiceRepo) UpdateReturnStatus(invoiceID string, status string) error {
	query := `
		UPDATE invoices
		SET notes = CASE
		                WHEN notes IS NULL OR notes = '' THEN $2
		                ELSE notes || E'\n' || $2
		            END,
		    updated_at = now()
		WHERE id = $1`
	_, err := r.q.Exec(context.Background(), query, invoiceID, status)
	if err != nil {
		return fmt.Errorf("update invoice return status: %w", err)
	}
	return nil
}

func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// List devuelve facturas paginadas y filtradas para una empresa.
func (r *InvoiceRepo) List(filter repository.InvoiceListFilter) ([]*entity.Invoice, int, error) {
	args := []any{filter.CompanyID}
	conds := []string{"company_id = $1"}
	idx := 2

	if filter.StartDate != "" {
		args = append(args, filter.StartDate)
		conds = append(conds, fmt.Sprintf("date >= $%d::date", idx))
		idx++
	}
	if filter.EndDate != "" {
		args = append(args, filter.EndDate)
		conds = append(conds, fmt.Sprintf("date <= $%d::date", idx))
		idx++
	}
	if filter.CustomerID != "" {
		args = append(args, filter.CustomerID)
		conds = append(conds, fmt.Sprintf("customer_id = $%d", idx))
		idx++
	}
	if filter.DIANStatus != "" {
		args = append(args, filter.DIANStatus)
		conds = append(conds, fmt.Sprintf("dian_status = $%d", idx))
		idx++
	}
	if filter.DocumentType != "" {
		args = append(args, filter.DocumentType)
		conds = append(conds, fmt.Sprintf("COALESCE(document_type, 'INVOICE') = $%d", idx))
		idx++
	}
	if filter.Prefix != "" {
		args = append(args, filter.Prefix)
		conds = append(conds, fmt.Sprintf("prefix = $%d", idx))
		idx++
	}

	where := strings.Join(conds, " AND ")

	// total count
	var total int
	countQ := fmt.Sprintf("SELECT COUNT(1) FROM invoices WHERE %s", where)
	if err := r.q.QueryRow(context.Background(), countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count invoices: %w", err)
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	args = append(args, limit, offset)
	dataQ := fmt.Sprintf(`
		SELECT id, company_id, customer_id, prefix, number, date,
		       net_total, tax_total, grand_total, dian_status,
		       COALESCE(cufe, ''), COALESCE(uuid, ''), COALESCE(xml_signed, ''),
		       COALESCE(qr_data, ''), COALESCE(track_id_dian, ''), COALESCE(dian_errors, ''),
		       COALESCE(document_type, ''),
		       COALESCE(original_invoice_id, ''),
		       COALESCE(original_invoice_number, ''),
		       COALESCE(original_invoice_cufe, ''),
		       original_invoice_issue_on,
		       COALESCE(discrepancy_code, ''),
		       COALESCE(discrepancy_reason, ''),
		       created_at, updated_at
		FROM invoices
		WHERE %s
		ORDER BY date DESC, created_at DESC
		LIMIT $%d OFFSET $%d`, where, idx, idx+1)

	rows, err := r.q.Query(context.Background(), dataQ, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list invoices: %w", err)
	}
	defer rows.Close()

	var list []*entity.Invoice
	for rows.Next() {
		var inv entity.Invoice
		var origIssueOn *time.Time
		var discCode string
		err := rows.Scan(
			&inv.ID, &inv.CompanyID, &inv.CustomerID, &inv.Prefix, &inv.Number,
			&inv.Date, &inv.NetTotal, &inv.TaxTotal, &inv.GrandTotal,
			&inv.DIAN_Status, &inv.CUFE, &inv.UUID, &inv.XMLSigned,
			&inv.QRData, &inv.TrackID, &inv.DIANErrors,
			&inv.DocumentType,
			&inv.OriginalInvoiceID,
			&inv.OriginalInvoiceNumber,
			&inv.OriginalInvoiceCUFE,
			&origIssueOn,
			&discCode,
			&inv.DiscrepancyReason,
			&inv.CreatedAt, &inv.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scan invoice row: %w", err)
		}
		if origIssueOn != nil {
			inv.OriginalInvoiceIssueOn = *origIssueOn
		}
		if discCode != "" {
			inv.DiscrepancyCode = entity.CreditNoteConcept(discCode)
		}
		list = append(list, &inv)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate invoices: %w", err)
	}

	return list, total, nil
}

// ListByCustomer devuelve las facturas de un cliente con paginación.
func (r *InvoiceRepo) ListByCustomer(customerID string, limit, offset int) ([]*entity.Invoice, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	var total int64
	const countQ = `SELECT COUNT(1) FROM invoices WHERE customer_id = $1`
	if err := r.q.QueryRow(context.Background(), countQ, customerID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count invoices by customer: %w", err)
	}

	const dataQ = `
		SELECT id, company_id, customer_id, prefix, number, date,
		       net_total, tax_total, grand_total, dian_status,
		       COALESCE(cufe, ''), COALESCE(uuid, ''), COALESCE(xml_signed, ''),
		       COALESCE(qr_data, ''), COALESCE(track_id_dian, ''), COALESCE(dian_errors, ''),
		       COALESCE(document_type, ''),
		       COALESCE(original_invoice_id, ''),
		       COALESCE(original_invoice_number, ''),
		       COALESCE(original_invoice_cufe, ''),
		       original_invoice_issue_on,
		       COALESCE(discrepancy_code, ''),
		       COALESCE(discrepancy_reason, ''),
		       created_at, updated_at
		FROM invoices
		WHERE customer_id = $1
		ORDER BY date DESC, created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.q.Query(context.Background(), dataQ, customerID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list invoices by customer: %w", err)
	}
	defer rows.Close()

	var list []*entity.Invoice
	for rows.Next() {
		var inv entity.Invoice
		var origIssueOn *time.Time
		var discCode string
		if err := rows.Scan(
			&inv.ID, &inv.CompanyID, &inv.CustomerID, &inv.Prefix, &inv.Number,
			&inv.Date, &inv.NetTotal, &inv.TaxTotal, &inv.GrandTotal,
			&inv.DIAN_Status, &inv.CUFE, &inv.UUID, &inv.XMLSigned,
			&inv.QRData, &inv.TrackID, &inv.DIANErrors,
			&inv.DocumentType,
			&inv.OriginalInvoiceID,
			&inv.OriginalInvoiceNumber,
			&inv.OriginalInvoiceCUFE,
			&origIssueOn,
			&discCode,
			&inv.DiscrepancyReason,
			&inv.CreatedAt, &inv.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan invoice by customer: %w", err)
		}
		if origIssueOn != nil {
			inv.OriginalInvoiceIssueOn = *origIssueOn
		}
		if discCode != "" {
			inv.DiscrepancyCode = entity.CreditNoteConcept(discCode)
		}
		list = append(list, &inv)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate invoices by customer: %w", err)
	}
	return list, total, nil
}

// GetCustomerStats retorna estadísticas de compra agregadas para un cliente.
func (r *InvoiceRepo) GetCustomerStats(customerID string) (*repository.CustomerPurchaseStats, error) {
	const q = `
		SELECT
			COALESCE(SUM(grand_total), 0)                    AS total_purchases,
			COALESCE(AVG(grand_total), 0)                    AS avg_ticket,
			COALESCE(MAX(date), '0001-01-01'::date)          AS last_purchase_date,
			COALESCE(COUNT(1), 0)                            AS invoice_count
		FROM invoices
		WHERE customer_id = $1`

	var stats repository.CustomerPurchaseStats
	var lastDate *time.Time
	var totalPurchases, avgTicket decimal.Decimal
	var count int64

	if err := r.q.QueryRow(context.Background(), q, customerID).Scan(
		&totalPurchases,
		&avgTicket,
		&lastDate,
		&count,
	); err != nil {
		return nil, fmt.Errorf("get customer stats: %w", err)
	}

	stats.TotalPurchases = totalPurchases
	stats.AvgTicket = avgTicket
	stats.InvoiceCount = int(count)
	if lastDate != nil {
		stats.LastPurchaseDate = *lastDate
	}
	return &stats, nil
}
