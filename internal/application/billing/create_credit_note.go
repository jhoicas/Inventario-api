package billing

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
)

// CreateCreditNoteUseCase crea una Nota Crédito asociada a una factura existente.
//  1. Valida la factura original y las cantidades devueltas.
//  2. Si la empresa tiene módulo de inventario, registra movimientos RETURN dentro de la misma tx.
//  3. Persiste la Nota Crédito (cabecera + detalle) y marca la factura original como Returned/Partially_Returned.
//  4. Post-commit dispara el DIANOrchestrator para firmar y enviar la Nota Crédito.
type CreateCreditNoteUseCase struct {
	txRunner         BillingTxRunner
	inventoryUC      InventoryUseCase
	customerRepo     repository.CustomerRepository
	companyRepo      repository.CompanyRepository
	productRepo      repository.ProductRepository
	warehouseRepo    repository.WarehouseRepository
	invoiceRepo      repository.InvoiceRepository
	dianOrchestrator *DIANOrchestrator
	dianConfig       DIANConfig
}

// NewCreateCreditNoteUseCase construye el caso de uso para devoluciones.
func NewCreateCreditNoteUseCase(
	txRunner BillingTxRunner,
	inventoryUC InventoryUseCase,
	customerRepo repository.CustomerRepository,
	companyRepo repository.CompanyRepository,
	productRepo repository.ProductRepository,
	warehouseRepo repository.WarehouseRepository,
	invoiceRepo repository.InvoiceRepository,
	dianOrchestrator *DIANOrchestrator,
	dianConfig DIANConfig,
) *CreateCreditNoteUseCase {
	return &CreateCreditNoteUseCase{
		txRunner:         txRunner,
		inventoryUC:      inventoryUC,
		customerRepo:     customerRepo,
		companyRepo:      companyRepo,
		productRepo:      productRepo,
		warehouseRepo:    warehouseRepo,
		invoiceRepo:      invoiceRepo,
		dianOrchestrator: dianOrchestrator,
		dianConfig:       dianConfig,
	}
}

// CreateCreditNote registra una devolución parcial o total de una factura existente.
// companyID y userID provienen del JWT; invoiceID de la ruta; el body define ítems y bodega destino.
func (uc *CreateCreditNoteUseCase) CreateCreditNote(
	ctx context.Context,
	companyID, userID, invoiceID string,
	in dto.ReturnInvoiceRequest,
) (*dto.InvoiceResponse, error) {
	if invoiceID == "" || companyID == "" || userID == "" {
		return nil, domain.ErrInvalidInput
	}
	if len(in.Items) == 0 {
		return nil, domain.ErrInvalidInput
	}

	// ── Validaciones previas (fuera de tx) ───────────────────────────────────────
	origInv, err := uc.invoiceRepo.GetByID(invoiceID)
	if err != nil || origInv == nil {
		return nil, domain.ErrNotFound
	}
	if origInv.CompanyID != companyID {
		return nil, domain.ErrForbidden
	}

	customer, err := uc.customerRepo.GetByID(origInv.CustomerID)
	if err != nil || customer == nil {
		return nil, domain.ErrNotFound
	}

	// Verificar módulo de inventario y bodega a la que se reingresa stock.
	hasInventory, _ := uc.companyRepo.HasActiveModule(ctx, companyID, entity.ModuleInventory)
	if hasInventory {
		if in.WarehouseID == "" {
			return nil, domain.ErrInvalidInput
		}
		wh, _ := uc.warehouseRepo.GetByID(in.WarehouseID)
		if wh == nil || wh.CompanyID != companyID {
			return nil, domain.ErrNotFound
		}
	}

	// Cargar detalles originales para validar cantidades devueltas.
	origDetails, err := uc.invoiceRepo.GetDetailsByInvoiceID(invoiceID)
	if err != nil {
		return nil, err
	}
	if len(origDetails) == 0 {
		return nil, domain.ErrInvalidInput
	}
	soldByProduct := make(map[string]decimal.Decimal, len(origDetails))
	priceByProduct := make(map[string]decimal.Decimal, len(origDetails))
	taxRateByProduct := make(map[string]decimal.Decimal, len(origDetails))
	for _, d := range origDetails {
		soldByProduct[d.ProductID] = soldByProduct[d.ProductID].Add(d.Quantity)
		priceByProduct[d.ProductID] = d.UnitPrice
		taxRateByProduct[d.ProductID] = d.TaxRate
	}

	// Validar ítems devueltos y calcular totales esperados de la Nota Crédito.
	var netTotal, taxTotal decimal.Decimal
	returnQtyByProduct := make(map[string]decimal.Decimal, len(in.Items))
	for _, item := range in.Items {
		if item.ProductID == "" || !item.Quantity.GreaterThan(decimal.Zero) {
			return nil, domain.ErrInvalidInput
		}
		soldQty, ok := soldByProduct[item.ProductID]
		if !ok {
			return nil, domain.ErrInvalidInput
		}
		newReturned := returnQtyByProduct[item.ProductID].Add(item.Quantity)
		if newReturned.GreaterThan(soldQty) {
			return nil, domain.ErrInvalidInput
		}
		returnQtyByProduct[item.ProductID] = newReturned

		unitPrice := priceByProduct[item.ProductID]
		lineSubtotal := item.Quantity.Mul(unitPrice)
		netTotal = netTotal.Add(lineSubtotal)
		taxTotal = taxTotal.Add(lineSubtotal.Mul(taxRateByProduct[item.ProductID]))
	}
	if netTotal.IsZero() {
		return nil, domain.ErrInvalidInput
	}
	grandTotal := netTotal.Add(taxTotal)

	// Determinar si la devolución es total o parcial.
	fullReturn := true
	for pid, soldQty := range soldByProduct {
		retQty := returnQtyByProduct[pid]
		if !retQty.Equal(soldQty) {
			if retQty.IsZero() {
				continue
			}
			fullReturn = false
			break
		}
	}
	returnStatus := "Partially_Returned"
	if fullReturn {
		returnStatus = "Returned"
	}

	now := time.Now()
	creditNoteID := uuid.New().String()
	var creditInv *entity.Invoice
	var creditDetails []*entity.InvoiceDetail

	// ── Transacción atómica: inventario RETURN + Nota Crédito + marcar factura ──
	err = uc.txRunner.RunBilling(ctx, func(
		movRepo repository.InventoryMovementRepository,
		stockRepo repository.StockRepository,
		productRepo repository.ProductRepository,
		_ repository.CustomerRepository,
		invoiceRepo repository.InvoiceRepository,
	) error {
		// Reingreso de inventario (RETURN) si aplica.
		if hasInventory {
			for _, item := range in.Items {
				product, pErr := productRepo.GetByID(item.ProductID)
				if pErr != nil || product == nil {
					return domain.ErrNotFound
				}
				if product.CompanyID != companyID {
					return domain.ErrForbidden
				}
				if err := uc.inventoryUC.RegisterReturnInTx(
					ctx,
					movRepo, stockRepo, productRepo,
					product,
					item.ProductID, in.WarehouseID, userID,
					item.Quantity,
					now,
					creditNoteID,
				); err != nil {
					if errors.Is(err, domain.ErrInvalidInput) {
						return err
					}
					return err
				}
			}
		}

		// Construir cabecera de Nota Crédito.
		number := fmt.Sprintf("%s-NC-%d", origInv.Number, now.Unix())
		var concept entity.CreditNoteConcept
		if fullReturn {
			concept = entity.CreditNoteConceptAnulacion
		} else {
			concept = entity.CreditNoteConceptDevolucionParcial
		}

		creditInv = &entity.Invoice{
			ID:         creditNoteID,
			CompanyID:  companyID,
			CustomerID: origInv.CustomerID,
			Prefix:     origInv.Prefix,
			Number:     number,
			Date:       now,
			NetTotal:   netTotal,
			TaxTotal:   taxTotal,
			GrandTotal: grandTotal,
			// Se persiste inicialmente como DRAFT; el orquestador actualizará el estado DIAN.
			DIAN_Status:          entity.DIANStatusDraft,
			DocumentType:         "CREDIT_NOTE",
			OriginalInvoiceID:    origInv.ID,
			OriginalInvoiceNumber: strings.TrimSpace(origInv.Prefix) + strings.TrimSpace(origInv.Number),
			OriginalInvoiceCUFE:  origInv.CUFE,
			OriginalInvoiceIssueOn: origInv.Date,
			DiscrepancyCode:      concept,
			DiscrepancyReason:    in.Reason,
			CreatedAt:            now,
			UpdatedAt:            now,
		}

		for _, item := range in.Items {
			unitPrice := priceByProduct[item.ProductID]
			subtotal := item.Quantity.Mul(unitPrice)
			rate := taxRateByProduct[item.ProductID]
			creditDetails = append(creditDetails, &entity.InvoiceDetail{
				ID:        uuid.New().String(),
				InvoiceID: creditInv.ID,
				ProductID: item.ProductID,
				Quantity:  item.Quantity,
				UnitPrice: unitPrice,
				TaxRate:   rate,
				Subtotal:  subtotal,
			})
		}

		// Persistir Nota Crédito (cabecera + detalle).
		if err := invoiceRepo.Create(creditInv); err != nil {
			return err
		}
		for _, d := range creditDetails {
			if err := invoiceRepo.CreateDetail(d); err != nil {
				return err
			}
		}

		// Marcar la factura original con el estado de devolución.
		if err := invoiceRepo.UpdateReturnStatus(origInv.ID, returnStatus); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// ── Post-commit: disparar orquestador DIAN para la Nota Crédito ─────────────
	if uc.dianConfig.TechnicalKey != "" {
		uc.dianOrchestrator.ProcessAsync(creditNoteID)
	}

	// Reutilizamos el formato de respuesta de factura.
	resp := &dto.InvoiceResponse{
		ID:           creditInv.ID,
		CompanyID:    creditInv.CompanyID,
		CustomerID:   creditInv.CustomerID,
		CustomerName: customer.Name,
		Prefix:       creditInv.Prefix,
		Number:       creditInv.Number,
		Date:         creditInv.Date.Format("2006-01-02"),
		NetTotal:     creditInv.NetTotal,
		TaxTotal:     creditInv.TaxTotal,
		GrandTotal:   creditInv.GrandTotal,
		DIAN_Status:  creditInv.DIAN_Status,
		CUFE:         creditInv.CUFE,
		QRData:       creditInv.QRData,
		Details:      make([]dto.InvoiceDetailResponse, 0, len(creditDetails)),
	}
	for _, d := range creditDetails {
		resp.Details = append(resp.Details, dto.InvoiceDetailResponse{
			ID:        d.ID,
			ProductID: d.ProductID,
			Quantity:  d.Quantity,
			UnitPrice: d.UnitPrice,
			TaxRate:   d.TaxRate,
			Subtotal:  d.Subtotal,
		})
	}

	return resp, nil
}

