package billing

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/tu-usuario/inventory-pro/internal/application/dto"
	"github.com/tu-usuario/inventory-pro/internal/domain"
	"github.com/tu-usuario/inventory-pro/internal/domain/entity"
	"github.com/tu-usuario/inventory-pro/internal/domain/repository"
)

// DIANConfig para el caso de uso (clave técnica, entorno de envío y rutas de certificado).
type DIANConfig struct {
	TechnicalKey string // Clave técnica DIAN (CUFE)
	Environment  string // "1" prod / "2" pruebas — TipoAmb en XML
	AppEnv       string // "dev" | "test" | "prod" — controla el envío al WS SOAP
	CertPath     string
	CertKeyPath  string
	CertPassword string
}

// CreateInvoiceUseCase crea una factura con lógica de inventario condicional según módulos activos.
// Si la empresa tiene el módulo "inventory" activo, valida stock y registra movimientos OUT.
// Si solo tiene "billing", la factura se genera sin tocar el inventario (ej: empresas de servicios).
// La firma electrónica DIAN se delega al DIANOrchestrator, que corre en goroutine post-commit.
type CreateInvoiceUseCase struct {
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

// NewCreateInvoiceUseCase construye el caso de uso.
func NewCreateInvoiceUseCase(
	txRunner BillingTxRunner,
	inventoryUC InventoryUseCase,
	customerRepo repository.CustomerRepository,
	companyRepo repository.CompanyRepository,
	productRepo repository.ProductRepository,
	warehouseRepo repository.WarehouseRepository,
	invoiceRepo repository.InvoiceRepository,
	dianOrchestrator *DIANOrchestrator,
	dianConfig DIANConfig,
) *CreateInvoiceUseCase {
	return &CreateInvoiceUseCase{
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

// CreateInvoice flujo principal:
//  1. Validaciones previas a la transacción (cliente, empresa, bodega si inventario, productos).
//  2. Verificar módulo "inventory" activo (lectura fuera de tx).
//  3. Transacción atómica:
//     a. Si hasInventory: validar stock y registrar salidas OUT por ítem.
//     b. Siempre: persistir cabecera DRAFT y detalles.
//  4. Post-commit: disparar DIANOrchestrator.ProcessAsync(invoiceID).
func (uc *CreateInvoiceUseCase) CreateInvoice(ctx context.Context, companyID, userID string, in dto.CreateInvoiceRequest) (*dto.InvoiceResponse, error) {
	if in.CustomerID == "" || len(in.Items) == 0 || in.Prefix == "" {
		return nil, domain.ErrInvalidInput
	}

	// ── Validaciones de solo lectura (fuera de tx) ────────────────────────────
	customer, err := uc.customerRepo.GetByID(in.CustomerID)
	if err != nil || customer == nil {
		return nil, domain.ErrNotFound
	}
	if customer.CompanyID != companyID {
		return nil, domain.ErrForbidden
	}

	_, err = uc.companyRepo.GetByID(companyID)
	if err != nil {
		return nil, domain.ErrNotFound
	}

	// ── Módulo de inventario (lectura fuera de tx) ────────────────────────────
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

	productsByID := make(map[string]*entity.Product, len(in.Items))
	for i := range in.Items {
		item := &in.Items[i]
		if item.ProductID == "" || !item.Quantity.GreaterThan(decimal.Zero) {
			return nil, domain.ErrInvalidInput
		}
		product, err := uc.productRepo.GetByID(item.ProductID)
		if err != nil || product == nil {
			return nil, domain.ErrNotFound
		}
		if product.CompanyID != companyID {
			return nil, domain.ErrForbidden
		}
		productsByID[item.ProductID] = product
		if item.UnitPrice.LessThan(decimal.Zero) {
			return nil, domain.ErrInvalidInput
		}
		if item.UnitPrice.IsZero() {
			in.Items[i].UnitPrice = product.Price
		}
	}

	// ── Transacción atómica ───────────────────────────────────────────────────
	now := time.Now()
	invoiceID := uuid.New().String()
	var inv *entity.Invoice
	var details []*entity.InvoiceDetail

	err = uc.txRunner.RunBilling(ctx, func(
		movRepo repository.InventoryMovementRepository,
		stockRepo repository.StockRepository,
		productRepo repository.ProductRepository,
		_ repository.CustomerRepository,
		invoiceRepo repository.InvoiceRepository,
	) error {

		// ── Bloque condicional: movimientos de inventario ─────────────────────
		if hasInventory {
			for _, item := range in.Items {
				product := productsByID[item.ProductID]
				if err := uc.inventoryUC.RegisterOUTInTx(
					ctx,
					movRepo, stockRepo, productRepo,
					product,
					item.ProductID, in.WarehouseID, userID,
					item.Quantity,
					now,
					invoiceID,
				); err != nil {
					if errors.Is(err, domain.ErrInsufficientStock) {
						return fmt.Errorf("stock insuficiente para SKU '%s': %w",
							product.SKU, domain.ErrInsufficientStock)
					}
					return err
				}
			}
		}

		// ── Calcular totales ──────────────────────────────────────────────────
		toRate := func(rate decimal.Decimal) decimal.Decimal {
			if rate.GreaterThan(decimal.NewFromInt(1)) {
				return rate.Div(decimal.NewFromInt(100))
			}
			return rate
		}
		var netTotal, taxTotal decimal.Decimal
		for _, item := range in.Items {
			product := productsByID[item.ProductID]
			subtotal := item.Quantity.Mul(item.UnitPrice)
			netTotal = netTotal.Add(subtotal)
			taxTotal = taxTotal.Add(subtotal.Mul(toRate(product.TaxRate)))
		}
		grandTotal := netTotal.Add(taxTotal)

		number := in.Number
		if number == "" {
			number = fmt.Sprintf("%s-%d", in.Prefix, now.Unix())
		}

		// ── Construir entidades ───────────────────────────────────────────────
		inv = &entity.Invoice{
			ID:          invoiceID,
			CompanyID:   companyID,
			CustomerID:  in.CustomerID,
			Prefix:      in.Prefix,
			Number:      number,
			Date:        now,
			NetTotal:    netTotal,
			TaxTotal:    taxTotal,
			GrandTotal:  grandTotal,
			DIAN_Status: entity.DIANStatusDraft,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		for _, item := range in.Items {
			product := productsByID[item.ProductID]
			subtotal := item.Quantity.Mul(item.UnitPrice)
			rate := toRate(product.TaxRate)
			details = append(details, &entity.InvoiceDetail{
				ID:        uuid.New().String(),
				InvoiceID: inv.ID,
				ProductID: item.ProductID,
				Quantity:  item.Quantity,
				UnitPrice: item.UnitPrice,
				TaxRate:   rate,
				Subtotal:  subtotal,
			})
		}

		// ── Persistencia inicial en DRAFT ─────────────────────────────────────
		if err := invoiceRepo.Create(inv); err != nil {
			return err
		}
		for _, d := range details {
			if err := invoiceRepo.CreateDetail(d); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// ── Post-commit: firma DIAN asíncrona ─────────────────────────────────────
	// La factura ya está committed en DRAFT. El orquestador re-fetcha todos los
	// datos frescos (empresa, cliente, resolución, productos) y ejecuta el ciclo
	// CUFE → XML → Firma → QR → Update con su propio context de 30 s.
	if uc.dianConfig.TechnicalKey != "" {
		uc.dianOrchestrator.ProcessAsync(invoiceID)
	}

	return uc.toResponse(inv, customer.Name, details), nil
}

func (uc *CreateInvoiceUseCase) toResponse(inv *entity.Invoice, customerName string, details []*entity.InvoiceDetail) *dto.InvoiceResponse {
	resp := &dto.InvoiceResponse{
		ID:           inv.ID,
		CompanyID:    inv.CompanyID,
		CustomerID:   inv.CustomerID,
		CustomerName: customerName,
		Prefix:       inv.Prefix,
		Number:       inv.Number,
		Date:         inv.Date.Format("2006-01-02"),
		NetTotal:     inv.NetTotal,
		TaxTotal:     inv.TaxTotal,
		GrandTotal:   inv.GrandTotal,
		DIAN_Status:  inv.DIAN_Status,
		CUFE:         inv.CUFE,
		QRData:       inv.QRData,
		Details:      make([]dto.InvoiceDetailResponse, 0, len(details)),
	}
	for _, d := range details {
		resp.Details = append(resp.Details, dto.InvoiceDetailResponse{
			ID:        d.ID,
			ProductID: d.ProductID,
			Quantity:  d.Quantity,
			UnitPrice: d.UnitPrice,
			TaxRate:   d.TaxRate,
			Subtotal:  d.Subtotal,
		})
	}
	return resp
}

// GetInvoiceDIANStatus devuelve solo los campos de estado DIAN de una factura.
// Es la llamada ligera usada por el frontend para hacer polling.
func (uc *CreateInvoiceUseCase) GetInvoiceDIANStatus(ctx context.Context, companyID, id string) (*dto.InvoiceDIANStatusDTO, error) {
	inv, err := uc.invoiceRepo.GetDIANStatus(id)
	if err != nil {
		return nil, err
	}
	if inv == nil {
		return nil, domain.ErrNotFound
	}
	if inv.CompanyID != companyID {
		return nil, domain.ErrForbidden
	}
	return &dto.InvoiceDIANStatusDTO{
		ID:         inv.ID,
		DIANStatus: inv.DIAN_Status,
		CUFE:       inv.CUFE,
		TrackID:    inv.TrackID,
		Errors:     inv.DIANErrors,
	}, nil
}

// GetInvoice obtiene una factura por ID con su detalle completo.
func (uc *CreateInvoiceUseCase) GetInvoice(ctx context.Context, companyID, id string) (*dto.InvoiceResponse, error) {
	inv, err := uc.invoiceRepo.GetByID(id)
	if err != nil || inv == nil {
		return nil, domain.ErrNotFound
	}
	if inv.CompanyID != companyID {
		return nil, domain.ErrForbidden
	}
	details, err := uc.invoiceRepo.GetDetailsByInvoiceID(id)
	if err != nil {
		return nil, err
	}
	customer, _ := uc.customerRepo.GetByID(inv.CustomerID)
	customerName := ""
	if customer != nil {
		customerName = customer.Name
	}
	return uc.toResponse(inv, customerName, details), nil
}
