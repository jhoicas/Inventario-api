package billing

import (
	"context"
	"fmt"

	"github.com/jhoicas/Inventario-api/internal/domain"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
)

// PDFUseCase genera la representación gráfica (PDF) de una factura electrónica.
// Solo se permite generar el PDF si la factura ya tiene CUFE (es decir, no está en DRAFT).
type PDFUseCase struct {
	invoiceRepo  repository.InvoiceRepository
	companyRepo  repository.CompanyRepository
	customerRepo repository.CustomerRepository
	productRepo  repository.ProductRepository
	generator    InvoicePDFGenerator
}

// NewPDFUseCase construye el caso de uso inyectando todas sus dependencias.
func NewPDFUseCase(
	invoiceRepo repository.InvoiceRepository,
	companyRepo repository.CompanyRepository,
	customerRepo repository.CustomerRepository,
	productRepo repository.ProductRepository,
	generator InvoicePDFGenerator,
) *PDFUseCase {
	return &PDFUseCase{
		invoiceRepo:  invoiceRepo,
		companyRepo:  companyRepo,
		customerRepo: customerRepo,
		productRepo:  productRepo,
		generator:    generator,
	}
}

// DownloadInvoicePDF recupera todos los datos de la factura, verifica que ya tiene
// CUFE/QR (no está en DRAFT) y genera el PDF con la representación gráfica DIAN.
//
// Retorna:
//   - (pdfBytes, filename, nil)  si todo sale bien.
//   - domain.ErrNotFound         si la factura no existe.
//   - domain.ErrForbidden        si la factura no pertenece a la empresa del token.
//   - domain.ErrInvalidInput     si la factura está en DRAFT (aún sin CUFE).
func (uc *PDFUseCase) DownloadInvoicePDF(
	ctx context.Context,
	companyID, invoiceID string,
) (pdfBytes []byte, filename string, err error) {
	// ── 1. Cargar factura ─────────────────────────────────────────────────────
	inv, err := uc.invoiceRepo.GetByID(invoiceID)
	if err != nil {
		return nil, "", fmt.Errorf("pdf: obtener factura: %w", err)
	}
	if inv == nil {
		return nil, "", domain.ErrNotFound
	}
	if inv.CompanyID != companyID {
		return nil, "", domain.ErrForbidden
	}

	// ── 2. Validar que ya fue procesada (tiene al menos CUFE) ─────────────────
	if inv.DIAN_Status == entity.DIANStatusDraft || inv.CUFE == "" {
		return nil, "", fmt.Errorf("%w: la factura está en estado %s, espere a que sea firmada antes de descargar el PDF",
			domain.ErrInvalidInput, inv.DIAN_Status)
	}

	// ── 3. Cargar empresa ─────────────────────────────────────────────────────
	company, err := uc.companyRepo.GetByID(companyID)
	if err != nil || company == nil {
		return nil, "", fmt.Errorf("pdf: obtener empresa: %w", err)
	}

	// ── 4. Cargar cliente ─────────────────────────────────────────────────────
	customer, err := uc.customerRepo.GetByID(inv.CustomerID)
	if err != nil || customer == nil {
		return nil, "", fmt.Errorf("pdf: obtener cliente: %w", err)
	}

	// ── 5. Cargar detalles + enriquecer con nombre de producto ────────────────
	rawDetails, err := uc.invoiceRepo.GetDetailsByInvoiceID(invoiceID)
	if err != nil {
		return nil, "", fmt.Errorf("pdf: obtener detalles: %w", err)
	}

	enriched := make([]InvoiceDetailForPDF, 0, len(rawDetails))
	for _, d := range rawDetails {
		name := "Producto " + d.ProductID // fallback
		if product, pErr := uc.productRepo.GetByID(d.ProductID); pErr == nil && product != nil {
			name = product.Name
		}
		enriched = append(enriched, InvoiceDetailForPDF{
			InvoiceDetail: *d,
			ProductName:   name,
		})
	}

	// ── 6. Generar PDF ────────────────────────────────────────────────────────
	pdfBytes, err = uc.generator.GenerateInvoicePDF(ctx, inv, company, customer, enriched)
	if err != nil {
		return nil, "", fmt.Errorf("pdf: generación fallida: %w", err)
	}

	filename = fmt.Sprintf("factura_%s%s.pdf", inv.Prefix, inv.Number)
	return pdfBytes, filename, nil
}
