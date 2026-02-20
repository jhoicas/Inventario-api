package billing

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/tu-usuario/inventory-pro/internal/application/dto"
	"github.com/tu-usuario/inventory-pro/internal/domain"
	"github.com/tu-usuario/inventory-pro/internal/domain/entity"
	"github.com/tu-usuario/inventory-pro/internal/domain/repository"
	infradian "github.com/tu-usuario/inventory-pro/internal/infrastructure/dian"
	"github.com/tu-usuario/inventory-pro/internal/infrastructure/dian/signer"
	"github.com/tu-usuario/inventory-pro/pkg/dian"
)

// DIANConfig para el caso de uso (clave técnica y rutas de certificado).
type DIANConfig struct {
	TechnicalKey  string
	Environment   string
	CertPath      string
	CertKeyPath   string
	CertPassword  string
}

// CreateInvoiceUseCase crea una factura y descuenta el inventario en una sola transacción.
type CreateInvoiceUseCase struct {
	txRunner      BillingTxRunner
	inventoryUC   InventoryUseCase
	customerRepo  repository.CustomerRepository
	companyRepo   repository.CompanyRepository
	productRepo   repository.ProductRepository
	warehouseRepo repository.WarehouseRepository
	invoiceRepo   repository.InvoiceRepository
	xmlBuilder    *infradian.XMLBuilderService
	signer        dian.Signer
	dianConfig    DIANConfig
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
	xmlBuilder *infradian.XMLBuilderService,
	signer dian.Signer,
	dianConfig DIANConfig,
) *CreateInvoiceUseCase {
	return &CreateInvoiceUseCase{
		txRunner:      txRunner,
		inventoryUC:   inventoryUC,
		customerRepo:  customerRepo,
		companyRepo:   companyRepo,
		productRepo:   productRepo,
		warehouseRepo: warehouseRepo,
		invoiceRepo:   invoiceRepo,
		xmlBuilder:    xmlBuilder,
		signer:        signer,
		dianConfig:    dianConfig,
	}
}

// CreateInvoice crea la factura, registra salidas de inventario por cada línea y guarda cabecera y detalles.
func (uc *CreateInvoiceUseCase) CreateInvoice(ctx context.Context, companyID, userID string, in dto.CreateInvoiceRequest) (*dto.InvoiceResponse, error) {
	if in.CustomerID == "" || in.WarehouseID == "" || len(in.Items) == 0 {
		return nil, domain.ErrInvalidInput
	}
	if in.Prefix == "" {
		return nil, domain.ErrInvalidInput
	}

	// Validar cliente y que sea de la empresa
	customer, err := uc.customerRepo.GetByID(in.CustomerID)
	if err != nil || customer == nil {
		return nil, domain.ErrNotFound
	}
	if customer.CompanyID != companyID {
		return nil, domain.ErrForbidden
	}

	// Empresa (para CUFE y XML DIAN)
	company, err := uc.companyRepo.GetByID(companyID)
	if err != nil || company == nil {
		return nil, domain.ErrNotFound
	}

	// Validar bodega
	wh, _ := uc.warehouseRepo.GetByID(in.WarehouseID)
	if wh == nil || wh.CompanyID != companyID {
		return nil, domain.ErrNotFound
	}

	// Validar productos y precios (fuera de la tx, solo lectura)
	productsByID := make(map[string]*entity.Product)
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

	now := time.Now()
	invoiceID := uuid.New().String() // ID de la factura; se usa como referencia en movimientos (TransactionID)
	var inv *entity.Invoice
	var details []*entity.InvoiceDetail

	err = uc.txRunner.RunBilling(ctx, func(
		movRepo repository.InventoryMovementRepository,
		stockRepo repository.StockRepository,
		productRepo repository.ProductRepository,
		_ repository.CustomerRepository,
		invoiceRepo repository.InvoiceRepository,
	) error {
		// 1) Por cada ítem, llamar InventoryUseCase.RegisterOUTInTx (tipo OUT, referencia a la factura).
		// Si inventario retorna error (ej: sin stock), se retorna y se hace rollback (atomicidad).
		for _, item := range in.Items {
			product := productsByID[item.ProductID]
			if err := uc.inventoryUC.RegisterOUTInTx(
				movRepo, stockRepo, productRepo,
				product,
				item.ProductID, in.WarehouseID, userID,
				item.Quantity,
				now,
				invoiceID, // referencia a la factura en inventory_movements.TransactionID
			); err != nil {
				return err
			}
		}

		// 2) Calcular impuestos (IVA 19% o 5% según el producto) y totales
		var netTotal, taxTotal decimal.Decimal
		taxRateDecimal := func(rate decimal.Decimal) decimal.Decimal {
			if rate.GreaterThan(decimal.NewFromInt(1)) {
				return rate.Div(decimal.NewFromInt(100))
			}
			return rate
		}
		for _, item := range in.Items {
			product := productsByID[item.ProductID]
			subtotal := item.Quantity.Mul(item.UnitPrice)
			rate := taxRateDecimal(product.TaxRate)
			taxAmount := subtotal.Mul(rate)
			netTotal = netTotal.Add(subtotal)
			taxTotal = taxTotal.Add(taxAmount)
		}
		grandTotal := netTotal.Add(taxTotal)

		// 3) Número de factura
		number := in.Number
		if number == "" {
			number = fmt.Sprintf("%s-%d", in.Prefix, now.Unix())
		}

		// 4) Entidad factura y detalles — estado DRAFT para reservar ID y consecutivo
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
			rate := taxRateDecimal(product.TaxRate)
			detail := &entity.InvoiceDetail{
				ID:        uuid.New().String(),
				InvoiceID: inv.ID,
				ProductID: item.ProductID,
				Quantity:  item.Quantity,
				UnitPrice: item.UnitPrice,
				TaxRate:   rate,
				Subtotal:  subtotal,
			}
			details = append(details, detail)
		}

		// 5) Persistencia inicial: guardar factura en DRAFT y detalles
		if err := invoiceRepo.Create(inv); err != nil {
			return err
		}
		for _, detail := range details {
			if err := invoiceRepo.CreateDetail(detail); err != nil {
				return err
			}
		}

		// 6) Si no hay clave técnica DIAN, la factura queda en DRAFT (sin CUFE/XML/firma)
		if uc.dianConfig.TechnicalKey == "" {
			return nil // factura ya guardada en DRAFT
		}
		envDIAN := uc.dianConfig.Environment
		if envDIAN == "" {
			envDIAN = "2"
		}
		// Cálculo CUFE (servicio CufeCalculator). TipoAmbiente "2" = Pruebas
		_, errCufe := infradian.CalculateCufeFromInvoice(&infradian.CufeContext{
			Invoice:       inv,
			Company:       company,
			Customer:      customer,
			ClaveTecnica:  uc.dianConfig.TechnicalKey,
			TipoAmbiente:  envDIAN,
		})
		if errCufe != nil {
			inv.DIAN_Status = entity.DIANStatusErrorGeneration
			inv.UpdatedAt = time.Now()
			_ = invoiceRepo.Update(inv)
			return fmt.Errorf("calcular CUFE: %w", errCufe)
		}

		// 7) Construcción XML UBL 2.1 (incluye CUFE en cbc:UUID)
		linesForXML := make([]infradian.InvoiceLineForXML, len(details))
		for i, d := range details {
			p := productsByID[d.ProductID]
			unitCode := p.UnitMeasure
			if unitCode == "" {
				unitCode = dian.UnitUnit
			}
			linesForXML[i] = infradian.InvoiceLineForXML{
				Detail:      d,
				ProductName: p.Name,
				ProductCode: p.SKU,
				UnitCode:    unitCode,
				Quantity:    d.Quantity,
				UnitPrice:   d.UnitPrice,
				TaxRate:     d.TaxRate,
				Subtotal:    d.Subtotal,
			}
		}
		buildCtx := &infradian.InvoiceBuildContext{
			Invoice:                          inv,
			Company:                          company,
			Customer:                         customer,
			Details:                          linesForXML,
			Resolution:                       nil,
			CustomerIdentificationTypeCode:  "31",
			CompanyIdentificationTypeCode:   "31",
		}
		xmlBytes, errXML := uc.xmlBuilder.Build(buildCtx)
		if errXML != nil {
			inv.DIAN_Status = entity.DIANStatusErrorGeneration
			inv.UpdatedAt = time.Now()
			_ = invoiceRepo.Update(inv)
			return fmt.Errorf("generar XML DIAN: %w", errXML)
		}

		// 8) Firma digital XAdES: cargar certificado (.p12 o PEM) y firmar
		var cert tls.Certificate
		if uc.dianConfig.CertPath != "" {
			if strings.HasSuffix(strings.ToLower(uc.dianConfig.CertPath), ".p12") || strings.HasSuffix(strings.ToLower(uc.dianConfig.CertPath), ".pfx") {
				cert, _ = signer.LoadFromP12(uc.dianConfig.CertPath, uc.dianConfig.CertPassword)
			} else {
				cert, _ = infradian.LoadCertFromPEM(uc.dianConfig.CertPath, uc.dianConfig.CertKeyPath)
			}
		}
		if len(cert.Certificate) == 0 || cert.PrivateKey == nil {
			inv.DIAN_Status = entity.DIANStatusErrorGeneration
			inv.UpdatedAt = time.Now()
			_ = invoiceRepo.Update(inv)
			return fmt.Errorf("no se pudo cargar el certificado DIAN (ruta o contraseña)")
		}
		signedXMLBytes, errSign := uc.signer.Sign(xmlBytes, cert)
		if errSign != nil {
			inv.DIAN_Status = entity.DIANStatusErrorGeneration
			inv.UpdatedAt = time.Now()
			_ = invoiceRepo.Update(inv)
			return fmt.Errorf("firmar XML: %w", errSign)
		}

		// 9) QR: NumFac|FecFac|ValFac|CodImp|...|Cufe|UrlValidacionDIAN
		inv.QRData = buildQRData(inv, envDIAN)
		inv.XMLSigned = string(signedXMLBytes)
		inv.DIAN_Status = entity.DIANStatusSigned
		inv.UpdatedAt = time.Now()
		if err := invoiceRepo.Update(inv); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return uc.toResponse(inv, customer.Name, details), nil
}

// URLs de validación DIAN (QR).
const (
	dianQRValidationURLPruebas = "https://catalogo-vpfe.dian.gov.co/document/searchqr?documentkey="
	dianQRValidationURLProd    = "https://catalogo-vpfe.dian.gov.co/document/searchqr?documentkey="
)

// buildQRData genera el string para el QR: NumFac|FecFac|ValFac|CodImp|ValImp|...|Cufe|UrlValidacionDIAN.
func buildQRData(inv *entity.Invoice, dianEnv string) string {
	numFac := strings.TrimSpace(inv.Prefix) + strings.TrimSpace(inv.Number)
	fecFac := inv.Date.Format("2006-01-02")
	valFac := inv.GrandTotal.Round(2).StringFixed(2)
	codImp := "01" // IVA
	valImp := inv.TaxTotal.Round(2).StringFixed(2)
	cufe := inv.CUFE
	base := dianQRValidationURLPruebas
	if dianEnv == "1" {
		base = dianQRValidationURLProd
	}
	urlValidacion := base + cufe
	return numFac + "|" + fecFac + "|" + valFac + "|" + codImp + "|" + valImp + "|" + cufe + "|" + urlValidacion
}

func (uc *CreateInvoiceUseCase) toResponse(inv *entity.Invoice, customerName string, details []*entity.InvoiceDetail) *dto.InvoiceResponse {
	resp := &dto.InvoiceResponse{
		ID:           inv.ID,
		CompanyID:   inv.CompanyID,
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
