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
)

// CreateVoidInvoiceUseCase anula una factura emitida creando una Nota Crédito total.
type CreateVoidInvoiceUseCase struct {
	txRunner         BillingTxRunner
	customerRepo     repository.CustomerRepository
	invoiceRepo      repository.InvoiceRepository
	dianOrchestrator *DIANOrchestrator
	dianConfig       DIANConfig
}

// NewCreateVoidInvoiceUseCase construye el caso de uso de anulación.
func NewCreateVoidInvoiceUseCase(
	txRunner BillingTxRunner,
	customerRepo repository.CustomerRepository,
	invoiceRepo repository.InvoiceRepository,
	dianOrchestrator *DIANOrchestrator,
	dianConfig DIANConfig,
) *CreateVoidInvoiceUseCase {
	return &CreateVoidInvoiceUseCase{
		txRunner:         txRunner,
		customerRepo:     customerRepo,
		invoiceRepo:      invoiceRepo,
		dianOrchestrator: dianOrchestrator,
		dianConfig:       dianConfig,
	}
}

// VoidInvoice ejecuta la anulación de una factura en estado Sent.
func (uc *CreateVoidInvoiceUseCase) VoidInvoice(
	ctx context.Context,
	companyID, userID, invoiceID string,
	in dto.CreateVoidInvoiceRequest,
) (*dto.VoidInvoiceResponse, error) {
	if companyID == "" || userID == "" || invoiceID == "" {
		return nil, domain.ErrInvalidInput
	}
	if in.ConceptCode < 1 || in.ConceptCode > 5 {
		return nil, domain.ErrInvalidInput
	}

	origInv, err := uc.invoiceRepo.GetByID(invoiceID)
	if err != nil || origInv == nil {
		return nil, domain.ErrNotFound
	}
	if origInv.CompanyID != companyID {
		return nil, domain.ErrForbidden
	}
	if origInv.DIAN_Status != entity.DIANStatusSent {
		return nil, domain.ErrConflict
	}

	if _, err := uc.customerRepo.GetByID(origInv.CustomerID); err != nil {
		return nil, domain.ErrNotFound
	}

	origDetails, err := uc.invoiceRepo.GetDetailsByInvoiceID(invoiceID)
	if err != nil {
		return nil, err
	}
	if len(origDetails) == 0 {
		return nil, domain.ErrInvalidInput
	}

	now := time.Now()
	creditNoteID := uuid.New().String()

	creditInv := &entity.Invoice{
		ID:                     creditNoteID,
		CompanyID:              companyID,
		CustomerID:             origInv.CustomerID,
		Prefix:                 origInv.Prefix,
		Number:                 fmt.Sprintf("%s-NCV-%d", origInv.Number, now.Unix()),
		Date:                   now,
		NetTotal:               origInv.NetTotal,
		TaxTotal:               origInv.TaxTotal,
		GrandTotal:             origInv.GrandTotal,
		DIAN_Status:            entity.DIANStatusDraft,
		DocumentType:           "CREDIT_NOTE",
		OriginalInvoiceID:      origInv.ID,
		OriginalInvoiceNumber:  strings.TrimSpace(origInv.Prefix) + strings.TrimSpace(origInv.Number),
		OriginalInvoiceCUFE:    origInv.CUFE,
		OriginalInvoiceIssueOn: origInv.Date,
		DiscrepancyCode:        mapVoidConceptToCreditNote(in.ConceptCode),
		DiscrepancyReason:      strings.TrimSpace(in.Reason),
		CreatedAt:              now,
		UpdatedAt:              now,
	}

	creditDetails := make([]*entity.InvoiceDetail, 0, len(origDetails))
	for _, d := range origDetails {
		creditDetails = append(creditDetails, &entity.InvoiceDetail{
			ID:        uuid.New().String(),
			InvoiceID: creditInv.ID,
			ProductID: d.ProductID,
			Quantity:  d.Quantity,
			UnitPrice: d.UnitPrice,
			TaxRate:   d.TaxRate,
			Subtotal:  d.Subtotal,
		})
	}

	err = uc.txRunner.RunBilling(ctx, func(
		_ repository.InventoryMovementRepository,
		_ repository.StockRepository,
		_ repository.ProductRepository,
		_ repository.CustomerRepository,
		invoiceRepo repository.InvoiceRepository,
	) error {
		if err := invoiceRepo.Create(creditInv); err != nil {
			return err
		}
		for _, detail := range creditDetails {
			if err := invoiceRepo.CreateDetail(detail); err != nil {
				return err
			}
		}
		if err := invoiceRepo.UpdateReturnStatus(origInv.ID, "VOID"); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if uc.dianConfig.TechnicalKey != "" {
		uc.dianOrchestrator.ProcessSync(creditNoteID)
		latest, getErr := uc.invoiceRepo.GetByID(creditNoteID)
		if getErr == nil && latest != nil {
			creditInv = latest
		}
	}

	return &dto.VoidInvoiceResponse{
		CreditNoteID: creditInv.ID,
		CUFE:         creditInv.CUFE,
		DIANStatus:   creditInv.DIAN_Status,
	}, nil
}

func mapVoidConceptToCreditNote(conceptCode int) entity.CreditNoteConcept {
	switch conceptCode {
	case 1:
		return entity.CreditNoteConceptDevolucionParcial
	case 2:
		return entity.CreditNoteConceptAnulacion
	case 3:
		return entity.CreditNoteConceptRebaja
	case 4:
		return entity.CreditNoteConceptDescuento
	default:
		return entity.CreditNoteConceptOtros
	}
}
