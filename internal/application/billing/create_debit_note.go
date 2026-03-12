package billing

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
	"github.com/shopspring/decimal"
)

// CreateDebitNoteUseCase crea una Nota Débito asociada a una factura existente.
// Replica el flujo de persistencia + DIAN de la Nota Crédito, usando el body
// explícito de items con precio unitario.
type CreateDebitNoteUseCase struct {
	txRunner         BillingTxRunner
	customerRepo     repository.CustomerRepository
	companyRepo      repository.CompanyRepository
	productRepo      repository.ProductRepository
	invoiceRepo      repository.InvoiceRepository
	dianOrchestrator *DIANOrchestrator
	dianConfig       DIANConfig
}

// NewCreateDebitNoteUseCase construye el caso de uso de nota débito.
func NewCreateDebitNoteUseCase(
	txRunner BillingTxRunner,
	customerRepo repository.CustomerRepository,
	companyRepo repository.CompanyRepository,
	productRepo repository.ProductRepository,
	invoiceRepo repository.InvoiceRepository,
	dianOrchestrator *DIANOrchestrator,
	dianConfig DIANConfig,
) *CreateDebitNoteUseCase {
	return &CreateDebitNoteUseCase{
		txRunner:         txRunner,
		customerRepo:     customerRepo,
		companyRepo:      companyRepo,
		productRepo:      productRepo,
		invoiceRepo:      invoiceRepo,
		dianOrchestrator: dianOrchestrator,
		dianConfig:       dianConfig,
	}
}

// CreateDebitNote registra una Nota Débito sobre una factura existente.
func (uc *CreateDebitNoteUseCase) CreateDebitNote(
	ctx context.Context,
	companyID, userID, invoiceID string,
	in dto.CreateDebitNoteRequest,
) (*dto.DebitNoteResponse, error) {
	if invoiceID == "" || companyID == "" || userID == "" {
		return nil, domain.ErrInvalidInput
	}
	if len(in.Items) == 0 {
		return nil, domain.ErrInvalidInput
	}

	origInv, err := uc.invoiceRepo.GetByID(invoiceID)
	if err != nil || origInv == nil {
		return nil, domain.ErrNotFound
	}
	if origInv.CompanyID != companyID {
		return nil, domain.ErrForbidden
	}

	if _, err := uc.customerRepo.GetByID(origInv.CustomerID); err != nil {
		return nil, domain.ErrNotFound
	}

	var netTotal, taxTotal decimal.Decimal
	for _, item := range in.Items {
		if item.ProductID == "" || !item.Quantity.GreaterThan(decimal.Zero) || !item.UnitPrice.GreaterThan(decimal.Zero) {
			return nil, domain.ErrInvalidInput
		}
		product, pErr := uc.productRepo.GetByID(item.ProductID)
		if pErr != nil || product == nil {
			return nil, domain.ErrNotFound
		}
		if product.CompanyID != companyID {
			return nil, domain.ErrForbidden
		}

		lineSubtotal := item.Quantity.Mul(item.UnitPrice)
		netTotal = netTotal.Add(lineSubtotal)
		taxTotal = taxTotal.Add(lineSubtotal.Mul(product.TaxRate))
	}
	if netTotal.IsZero() {
		return nil, domain.ErrInvalidInput
	}
	grandTotal := netTotal.Add(taxTotal)

	now := time.Now()
	debitNoteID := uuid.New().String()
	var debitInv *entity.Invoice
	var debitDetails []*entity.InvoiceDetail

	err = uc.txRunner.RunBilling(ctx, func(
		_ repository.InventoryMovementRepository,
		_ repository.StockRepository,
		_ repository.ProductRepository,
		_ repository.CustomerRepository,
		invoiceRepo repository.InvoiceRepository,
	) error {
		number := fmt.Sprintf("%s-ND-%d", origInv.Number, now.Unix())

		debitInv = &entity.Invoice{
			ID:                     debitNoteID,
			CompanyID:              companyID,
			CustomerID:             origInv.CustomerID,
			Prefix:                 origInv.Prefix,
			Number:                 number,
			Date:                   now,
			NetTotal:               netTotal,
			TaxTotal:               taxTotal,
			GrandTotal:             grandTotal,
			DIAN_Status:            entity.DIANStatusDraft,
			DocumentType:           "DEBIT_NOTE",
			OriginalInvoiceID:      origInv.ID,
			OriginalInvoiceNumber:  strings.TrimSpace(origInv.Prefix) + strings.TrimSpace(origInv.Number),
			OriginalInvoiceCUFE:    origInv.CUFE,
			OriginalInvoiceIssueOn: origInv.Date,
			DiscrepancyCode:        entity.CreditNoteConceptOtros,
			DiscrepancyReason:      in.Reason,
			CreatedAt:              now,
			UpdatedAt:              now,
		}

		for _, item := range in.Items {
			product, pErr := uc.productRepo.GetByID(item.ProductID)
			if pErr != nil || product == nil {
				return domain.ErrNotFound
			}
			subtotal := item.Quantity.Mul(item.UnitPrice)
			taxRate := product.TaxRate
			debitDetails = append(debitDetails, &entity.InvoiceDetail{
				ID:        uuid.New().String(),
				InvoiceID: debitInv.ID,
				ProductID: item.ProductID,
				Quantity:  item.Quantity,
				UnitPrice: item.UnitPrice,
				TaxRate:   taxRate,
				Subtotal:  subtotal,
			})
		}

		if err := invoiceRepo.Create(debitInv); err != nil {
			return err
		}
		for _, d := range debitDetails {
			if err := invoiceRepo.CreateDetail(d); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	if uc.dianConfig.TechnicalKey != "" {
		uc.dianOrchestrator.ProcessSync(debitNoteID)
		latest, getErr := uc.invoiceRepo.GetByID(debitNoteID)
		if getErr == nil && latest != nil {
			debitInv = latest
		}
	}

	return &dto.DebitNoteResponse{
		DebitNoteID: debitInv.ID,
		CUFE:        debitInv.CUFE,
		DIANStatus:  debitInv.DIAN_Status,
	}, nil
}
