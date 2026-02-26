package billing

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"strings"
	"time"

	infradian "github.com/tu-usuario/inventory-pro/internal/infrastructure/dian"
	"github.com/tu-usuario/inventory-pro/internal/infrastructure/dian/signer"

	"github.com/tu-usuario/inventory-pro/internal/domain/entity"
	"github.com/tu-usuario/inventory-pro/internal/domain/repository"
	pkgdian "github.com/tu-usuario/inventory-pro/pkg/dian"
)

// DIANOrchestrator orquesta el ciclo completo de firma y envío electrónico DIAN:
//
//	CUFE → XML UBL 2.1 → Firma XAdES-EPES → ZIP → Envío SOAP → Update DB
//
// Se ejecuta siempre en goroutine independiente (ProcessAsync) con su propio
// context.Background() + timeout 30 s, desacoplado del ciclo HTTP.
//
// Modos de operación (controlados por DIANConfig.AppEnv):
//   - "dev"  → Genera y firma el XML, NO envía al WS DIAN. Estado final: EXITOSO (mock).
//   - "test" → Envía al ambiente de habilitación vpfe-hab.dian.gov.co.
//   - "prod" → Envía al ambiente de producción vpfe.dian.gov.co.
type DIANOrchestrator struct {
	invoiceRepo    repository.InvoiceRepository
	companyRepo    repository.CompanyRepository
	customerRepo   repository.CustomerRepository
	productRepo    repository.ProductRepository
	resolutionRepo repository.BillingResolutionRepository
	xmlBuilder     *infradian.XMLBuilderService
	signer         pkgdian.Signer
	submitter      infradian.DIANSubmitter // cliente SOAP; nil en dev
	dianConfig     DIANConfig
}

// NewDIANOrchestrator construye el orquestador con todas sus dependencias.
// submitter puede ser nil: en ese caso el modo dev es el único que funciona.
func NewDIANOrchestrator(
	invoiceRepo repository.InvoiceRepository,
	companyRepo repository.CompanyRepository,
	customerRepo repository.CustomerRepository,
	productRepo repository.ProductRepository,
	resolutionRepo repository.BillingResolutionRepository,
	xmlBuilder *infradian.XMLBuilderService,
	signer pkgdian.Signer,
	submitter infradian.DIANSubmitter,
	dianConfig DIANConfig,
) *DIANOrchestrator {
	return &DIANOrchestrator{
		invoiceRepo:    invoiceRepo,
		companyRepo:    companyRepo,
		customerRepo:   customerRepo,
		productRepo:    productRepo,
		resolutionRepo: resolutionRepo,
		xmlBuilder:     xmlBuilder,
		signer:         signer,
		submitter:      submitter,
		dianConfig:     dianConfig,
	}
}

// ProcessAsync dispara el procesamiento DIAN en una goroutine independiente.
// invoiceID es el ID de la factura ya persistida en estado DRAFT.
func (o *DIANOrchestrator) ProcessAsync(invoiceID string) {
	go o.process(invoiceID)
}

// process es el núcleo síncrono del orquestador. Siempre termina actualizando
// dian_status en la DB (EXITOSO, RECHAZADO o ERROR_GENERATION).
func (o *DIANOrchestrator) process(invoiceID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// markError actualiza la factura a ERROR_GENERATION y hace log del problema.
	markError := func(inv *entity.Invoice, step, msg string) {
		inv.DIAN_Status = entity.DIANStatusErrorGeneration
		inv.UpdatedAt = time.Now()
		if err := o.invoiceRepo.Update(inv); err != nil {
			log.Printf("[DIAN][%s] no se pudo persistir ERROR_GENERATION: %v", invoiceID, err)
		}
		log.Printf("[DIAN][%s] ERROR en %s: %s", invoiceID, step, msg)
	}

	// ═══════════════════════════════════════════════════════════════════════════
	// 0. Re-fetch datos frescos (evita data races con el goroutine HTTP)
	// ═══════════════════════════════════════════════════════════════════════════
	inv, err := o.invoiceRepo.GetByID(invoiceID)
	if err != nil || inv == nil {
		log.Printf("[DIAN][%s] factura no encontrada: %v", invoiceID, err)
		return
	}
	if inv.DIAN_Status != entity.DIANStatusDraft {
		log.Printf("[DIAN][%s] estado %q inesperado (ya procesada?), saltando", invoiceID, inv.DIAN_Status)
		return
	}

	company, err := o.companyRepo.GetByID(inv.CompanyID)
	if err != nil || company == nil {
		markError(inv, "fetch-company", fmt.Sprintf("empresa %s no encontrada: %v", inv.CompanyID, err))
		return
	}

	customer, err := o.customerRepo.GetByID(inv.CustomerID)
	if err != nil || customer == nil {
		markError(inv, "fetch-customer", fmt.Sprintf("cliente %s no encontrado: %v", inv.CustomerID, err))
		return
	}

	resolution, err := o.resolutionRepo.GetActiveByCompanyAndPrefix(ctx, inv.CompanyID, inv.Prefix)
	if err != nil {
		markError(inv, "fetch-resolution", fmt.Sprintf("error consultando resolución: %v", err))
		return
	}

	details, err := o.invoiceRepo.GetDetailsByInvoiceID(invoiceID)
	if err != nil {
		markError(inv, "fetch-details", fmt.Sprintf("error obteniendo detalles: %v", err))
		return
	}

	// ═══════════════════════════════════════════════════════════════════════════
	// 1. Enriquecer líneas con datos de producto
	// ═══════════════════════════════════════════════════════════════════════════
	linesForXML := make([]infradian.InvoiceLineForXML, len(details))
	for i, d := range details {
		unitCode := pkgdian.UnitUnit
		product, pErr := o.productRepo.GetByID(d.ProductID)
		if pErr == nil && product != nil {
			if product.UnitMeasure != "" {
				unitCode = product.UnitMeasure
			}
			linesForXML[i] = infradian.InvoiceLineForXML{
				Detail: d, ProductName: product.Name, ProductCode: product.SKU,
				UnitCode: unitCode, Quantity: d.Quantity, UnitPrice: d.UnitPrice,
				TaxRate: d.TaxRate, Subtotal: d.Subtotal,
			}
		} else {
			linesForXML[i] = infradian.InvoiceLineForXML{
				Detail: d, ProductName: "Producto " + d.ProductID, ProductCode: d.ProductID,
				UnitCode: unitCode, Quantity: d.Quantity, UnitPrice: d.UnitPrice,
				TaxRate: d.TaxRate, Subtotal: d.Subtotal,
			}
		}
	}

	// ═══════════════════════════════════════════════════════════════════════════
	// 2. Calcular CUFE (SHA-384, Anexo Técnico 1.9)
	// ═══════════════════════════════════════════════════════════════════════════
	tipoAmb := o.dianConfig.Environment
	if tipoAmb == "" {
		tipoAmb = "2"
	}
	if _, err := infradian.CalculateCufeFromInvoice(&infradian.CufeContext{
		Invoice:      inv,
		Company:      company,
		Customer:     customer,
		ClaveTecnica: o.dianConfig.TechnicalKey,
		TipoAmbiente: tipoAmb,
	}); err != nil {
		markError(inv, "cufe", err.Error())
		return
	}

	// ═══════════════════════════════════════════════════════════════════════════
	// 3. Construir XML UBL 2.1 (incluye CUFE en cbc:UUID + DianExtensions)
	// ═══════════════════════════════════════════════════════════════════════════
	var resData *infradian.BillingResolutionData
	if resolution != nil {
		resData = &infradian.BillingResolutionData{
			Number: resolution.ResolutionNumber, Prefix: resolution.Prefix,
			From: resolution.RangeFrom, To: resolution.RangeTo,
			DateFrom: resolution.DateFrom, DateTo: resolution.DateTo,
		}
	}

	xmlBytes, errXML := o.xmlBuilder.Build(&infradian.InvoiceBuildContext{
		Invoice:                        inv,
		Company:                        company,
		Customer:                       customer,
		Details:                        linesForXML,
		Resolution:                     resData,
		CustomerIdentificationTypeCode: identTypeCode(customer.TaxID),
		CompanyIdentificationTypeCode:  "31",
	})
	if errXML != nil {
		markError(inv, "xml-build", errXML.Error())
		return
	}

	// ═══════════════════════════════════════════════════════════════════════════
	// 4. Firma digital XAdES-EPES
	// ═══════════════════════════════════════════════════════════════════════════
	cert, errCert := loadCertificate(o.dianConfig)
	if errCert != nil {
		markError(inv, "cert-load", errCert.Error())
		return
	}
	if len(cert.Certificate) == 0 || cert.PrivateKey == nil {
		markError(inv, "cert-load", "certificado vacío: verifica DIAN_CERT_PATH y DIAN_CERT_PASSWORD")
		return
	}

	signedXMLBytes, errSign := o.signer.Sign(xmlBytes, cert)
	if errSign != nil {
		markError(inv, "xml-sign", errSign.Error())
		return
	}

	// Actualizar en DB como SIGNED (XML firmado disponible para descarga)
	inv.QRData = buildDIANQR(inv, tipoAmb)
	inv.XMLSigned = string(signedXMLBytes)
	inv.DIAN_Status = entity.DIANStatusSigned
	inv.UpdatedAt = time.Now()
	if err := o.invoiceRepo.Update(inv); err != nil {
		log.Printf("[DIAN][%s] error persistiendo SIGNED: %v", invoiceID, err)
		return
	}

	// ═══════════════════════════════════════════════════════════════════════════
	// 5. Empaquetar en ZIP
	// ═══════════════════════════════════════════════════════════════════════════
	xmlName, zipName := infradian.DIANFilenames(company, inv)
	zipBytes, errZIP := infradian.CompressXMLToZip(signedXMLBytes, xmlName)
	if errZIP != nil {
		markError(inv, "zip", errZIP.Error())
		return
	}

	// ═══════════════════════════════════════════════════════════════════════════
	// 6. Envío condicional al WS DIAN
	// ═══════════════════════════════════════════════════════════════════════════
	appEnv := strings.ToLower(strings.TrimSpace(o.dianConfig.AppEnv))

	var finalStatus, trackID, dianErrors string

	switch appEnv {
	case infradian.AppEnvDev, "":
		// ── Modo desarrollo: simular respuesta, no enviar ──────────────────
		log.Printf("[DIAN][%s] [DEV] Simulando envío a DIAN — ZIP generado: %s (%d bytes)",
			invoiceID, zipName, len(zipBytes))
		trackID = "MOCK-TRACK-123"
		finalStatus = entity.DIANStatusExitoso

	case infradian.AppEnvTest, infradian.AppEnvProd:
		// ── Modo test/prod: llamada real al WS DIAN ────────────────────────
		if o.submitter == nil {
			markError(inv, "soap", "DIANSubmitter no inyectado para entorno "+appEnv)
			return
		}
		result, soapErr := o.submitter.SubmitZip(ctx, zipBytes, zipName, appEnv)
		if soapErr != nil {
			markError(inv, "soap", soapErr.Error())
			return
		}
		trackID = result.TrackID
		dianErrors = result.Errors
		if result.Accepted {
			finalStatus = entity.DIANStatusExitoso
			log.Printf("[DIAN][%s] Aceptada por la DIAN → TrackID: %s", invoiceID, trackID)
		} else {
			finalStatus = entity.DIANStatusRechazado
			log.Printf("[DIAN][%s] Rechazada por la DIAN — Errores: %s", invoiceID, dianErrors)
		}

	default:
		markError(inv, "config", fmt.Sprintf("DIAN_ENV desconocido: %q (usar dev|test|prod)", appEnv))
		return
	}

	// ═══════════════════════════════════════════════════════════════════════════
	// 7. Persistir resultado final en DB
	// ═══════════════════════════════════════════════════════════════════════════
	inv.DIAN_Status = finalStatus
	inv.TrackID = trackID
	inv.DIANErrors = dianErrors
	inv.UpdatedAt = time.Now()

	if err := o.invoiceRepo.Update(inv); err != nil {
		log.Printf("[DIAN][%s] error persistiendo estado final %s: %v", invoiceID, finalStatus, err)
		return
	}

	log.Printf("[DIAN][%s] procesada → %s (TrackID: %s)", invoiceID, finalStatus, trackID)
}

// ── helpers privados ──────────────────────────────────────────────────────────

func loadCertificate(cfg DIANConfig) (tls.Certificate, error) {
	if cfg.CertPath == "" {
		return tls.Certificate{}, fmt.Errorf("DIAN_CERT_PATH no configurado")
	}
	lower := strings.ToLower(cfg.CertPath)
	if strings.HasSuffix(lower, ".p12") || strings.HasSuffix(lower, ".pfx") {
		return signer.LoadFromP12(cfg.CertPath, cfg.CertPassword)
	}
	return infradian.LoadCertFromPEM(cfg.CertPath, cfg.CertKeyPath)
}

func identTypeCode(taxID string) string {
	digits := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, taxID)
	if len(digits) >= 9 {
		return "31"
	}
	return "13"
}

func buildDIANQR(inv *entity.Invoice, tipoAmb string) string {
	const (
		urlPruebas = "https://catalogo-vpfe.dian.gov.co/document/searchqr?documentkey="
		urlProd    = "https://catalogo-vpfe.dian.gov.co/document/searchqr?documentkey="
	)
	base := urlPruebas
	if tipoAmb == "1" {
		base = urlProd
	}
	return strings.Join([]string{
		strings.TrimSpace(inv.Prefix) + strings.TrimSpace(inv.Number),
		inv.Date.Format("2006-01-02"),
		inv.GrandTotal.Round(2).StringFixed(2),
		"01",
		inv.TaxTotal.Round(2).StringFixed(2),
		inv.CUFE,
		base + inv.CUFE,
	}, "|")
}
