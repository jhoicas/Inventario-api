package dian

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strconv"

	"github.com/shopspring/decimal"
	"github.com/tu-usuario/inventory-pro/pkg/dian"
)

// Namespaces oficiales UBL 2.1 y DIAN (Anexo Técnico 1.9).
const (
	// Namespace por defecto (UBL Invoice)
	NsInvoice = "urn:oasis:names:specification:ubl:schema:xsd:Invoice-2"
	// Common Aggregate Components
	NsCac = "urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2"
	// Common Basic Components
	NsCbc = "urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2"
	// Extension Components
	NsExt = "urn:oasis:names:specification:ubl:schema:xsd:CommonExtensionComponents-2"
	// DIAN Extensions (valor oficial para Anexo 1.8/1.9)
	NsSts = "dian:gov:co:facturaelectronica:v1"
	// XML Digital Signature
	NsDs = "http://www.w3.org/2000/09/xmldsig#"
	// XAdES (para la firma)
	NsXades = "http://uri.etsi.org/01903/v1.3.2#"
	// XML Schema Instance (para schemaLocation)
	nsXsi = "http://www.w3.org/2001/XMLSchema-instance"
	// Schema location UBL Invoice 2.1
	schemaLocationInvoice = "urn:oasis:names:specification:ubl:schema:xsd:Invoice-2 http://docs.oasis-open.org/ubl/os-UBL-2.1/xsd/maindoc/UBL-Invoice-2.1.xsd"
)

// XMLBuilderService construye el XML UBL 2.1 de la factura (sin firma XAdES).
type XMLBuilderService struct{}

// NewXMLBuilderService crea el servicio.
func NewXMLBuilderService() *XMLBuilderService {
	return &XMLBuilderService{}
}

// Build genera el []byte del documento Invoice según UBL 2.1 y extensiones DIAN.
func (s *XMLBuilderService) Build(ctx *InvoiceBuildContext) ([]byte, error) {
	if ctx == nil || ctx.Invoice == nil || ctx.Company == nil || ctx.Customer == nil {
		return nil, fmt.Errorf("dian: faltan invoice, company o customer en el contexto")
	}
	var buf bytes.Buffer
	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")

	// Root <Invoice> con atributos obligatorios (Anexo 1.9). Id para Reference URI en firma XAdES.
	root := xml.StartElement{
		Name: xml.Name{Space: NsInvoice, Local: "Invoice"},
		Attr: []xml.Attr{
			{Name: xml.Name{Local: "Id"}, Value: "invoice-id"},
			{Name: xml.Name{Local: "xmlns"}, Value: NsInvoice},
			{Name: xml.Name{Local: "xmlns:cac"}, Value: NsCac},
			{Name: xml.Name{Local: "xmlns:cbc"}, Value: NsCbc},
			{Name: xml.Name{Local: "xmlns:ds"}, Value: NsDs},
			{Name: xml.Name{Local: "xmlns:ext"}, Value: NsExt},
			{Name: xml.Name{Local: "xmlns:sts"}, Value: NsSts},
			{Name: xml.Name{Local: "xmlns:xades"}, Value: NsXades},
			{Name: xml.Name{Local: "xmlns:xsi"}, Value: nsXsi},
			{Name: xml.Name{Space: nsXsi, Local: "schemaLocation"}, Value: schemaLocationInvoice},
		},
	}
	if err := enc.EncodeToken(root); err != nil {
		return nil, err
	}

	// ---- CRÍTICO: ext:UBLExtensions siempre como primer hijo de Invoice (requerido por el firmador)
	// Extensión 1: DIAN (resolución). Extensión 2: placeholder para ds:Signature (el signer inyecta aquí)
	if err := s.writeUBLExtensions(enc, ctx); err != nil {
		return nil, err
	}

	// ---- cbc: elementos obligatorios y opcionales del Invoice
	issueDate := ctx.Invoice.Date
	if ctx.IssueDate != nil {
		issueDate = *ctx.IssueDate
	}
	invoiceID := ctx.Invoice.Prefix + ctx.Invoice.Number
	if ctx.Invoice.Prefix == "" {
		invoiceID = ctx.Invoice.Number
	}

	writeCbc(enc, "UBLVersionID", "2.1")
	writeCbc(enc, "CustomizationID", "10")
	writeCbc(enc, "ProfileID", "DIAN 2.1: Factura Electrónica de Venta")
	writeCbc(enc, "ID", invoiceID)
	// cbc:UUID = CUFE (Código Único de Factura Electrónica)
	if u := ctx.Invoice.UUID; u != "" {
		writeCbc(enc, "UUID", u)
	} else if ctx.Invoice.CUFE != "" {
		writeCbc(enc, "UUID", ctx.Invoice.CUFE)
	}
	writeCbc(enc, "IssueDate", issueDate.Format("2006-01-02"))
	writeCbc(enc, "IssueTime", issueDate.Format("15:04:05-07:00"))
	writeCbc(enc, "DocumentCurrencyCode", "COP")
	writeCbc(enc, "LineCountNumeric", strconv.Itoa(len(ctx.Details)))

	if ctx.DueDate != nil {
		writeCbc(enc, "DueDate", ctx.DueDate.Format("2006-01-02"))
	}

	// ---- cac:AccountingSupplierParty
	if err := s.writeSupplierParty(enc, ctx); err != nil {
		return nil, err
	}
	// ---- cac:AccountingCustomerParty
	if err := s.writeCustomerParty(enc, ctx); err != nil {
		return nil, err
	}
	// ---- cac:PaymentMeans (forma y medio de pago)
	s.writePaymentMeans(enc, ctx)
	// ---- cac:TaxTotal
	if err := s.writeTaxTotal(enc, ctx); err != nil {
		return nil, err
	}
	// ---- cac:LegalMonetaryTotal
	if err := s.writeLegalMonetaryTotal(enc, ctx); err != nil {
		return nil, err
	}
	// ---- cac:InvoiceLine (cada detalle)
	for i, line := range ctx.Details {
		if err := s.writeInvoiceLine(enc, i+1, line); err != nil {
			return nil, err
		}
	}

	if err := enc.EncodeToken(root.End()); err != nil {
		return nil, err
	}
	if err := enc.Flush(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func writeCbc(enc *xml.Encoder, local, value string) {
	_ = enc.EncodeToken(xml.StartElement{Name: xml.Name{Space: NsCbc, Local: local}})
	_ = enc.EncodeToken(xml.CharData(value))
	_ = enc.EncodeToken(xml.EndElement{Name: xml.Name{Space: NsCbc, Local: local}})
}

func writeCbcAmount(enc *xml.Encoder, local, value string, currency string) {
	attr := []xml.Attr{}
	if currency != "" {
		attr = append(attr, xml.Attr{Name: xml.Name{Local: "currencyID"}, Value: currency})
	}
	_ = enc.EncodeToken(xml.StartElement{Name: xml.Name{Space: NsCbc, Local: local}, Attr: attr})
	_ = enc.EncodeToken(xml.CharData(value))
	_ = enc.EncodeToken(xml.EndElement{Name: xml.Name{Space: NsCbc, Local: local}})
}

// writeUBLExtensions escribe siempre ext:UBLExtensions como primer hijo de Invoice.
// Extensión 1: DIAN (DianExtensions si hay Resolution, si no ExtensionContent vacío).
// Extensión 2: ExtensionContent vacío; el firmador inyectará aquí <ds:Signature>.
func (s *XMLBuilderService) writeUBLExtensions(enc *xml.Encoder, ctx *InvoiceBuildContext) error {
	// Contenedor de extensiones (siempre presente)
	_ = enc.EncodeToken(xml.StartElement{Name: xml.Name{Space: NsExt, Local: "UBLExtensions"}})

	// 1. Extensión DIAN (datos de resolución o placeholder vacío)
	_ = enc.EncodeToken(xml.StartElement{Name: xml.Name{Space: NsExt, Local: "UBLExtension"}})
	_ = enc.EncodeToken(xml.StartElement{Name: xml.Name{Space: NsExt, Local: "ExtensionContent"}})
	if ctx.Resolution != nil {
		_ = enc.EncodeToken(xml.StartElement{Name: xml.Name{Space: NsSts, Local: "DianExtensions"}})
		_ = enc.EncodeToken(xml.StartElement{Name: xml.Name{Space: NsSts, Local: "InvoiceControl"}})
		writeSts(enc, "InvoiceAuthorization", ctx.Resolution.Number)
		_ = enc.EncodeToken(xml.StartElement{Name: xml.Name{Space: NsSts, Local: "AuthorizationPeriod"}})
		writeSts(enc, "StartDate", ctx.Resolution.DateFrom.Format("2006-01-02"))
		writeSts(enc, "EndDate", ctx.Resolution.DateTo.Format("2006-01-02"))
		_ = enc.EncodeToken(xml.EndElement{Name: xml.Name{Space: NsSts, Local: "AuthorizationPeriod"}})
		_ = enc.EncodeToken(xml.StartElement{Name: xml.Name{Space: NsSts, Local: "AuthorizedInvoices"}})
		writeSts(enc, "Prefix", ctx.Resolution.Prefix)
		writeSts(enc, "From", strconv.FormatInt(ctx.Resolution.From, 10))
		writeSts(enc, "To", strconv.FormatInt(ctx.Resolution.To, 10))
		_ = enc.EncodeToken(xml.EndElement{Name: xml.Name{Space: NsSts, Local: "AuthorizedInvoices"}})
		_ = enc.EncodeToken(xml.EndElement{Name: xml.Name{Space: NsSts, Local: "InvoiceControl"}})
		_ = enc.EncodeToken(xml.EndElement{Name: xml.Name{Space: NsSts, Local: "DianExtensions"}})
	}
	_ = enc.EncodeToken(xml.EndElement{Name: xml.Name{Space: NsExt, Local: "ExtensionContent"}})
	_ = enc.EncodeToken(xml.EndElement{Name: xml.Name{Space: NsExt, Local: "UBLExtension"}})

	// 2. Extensión para la firma (placeholder vacío; el signer inyectará <ds:Signature> aquí)
	_ = enc.EncodeToken(xml.StartElement{Name: xml.Name{Space: NsExt, Local: "UBLExtension"}})
	_ = enc.EncodeToken(xml.StartElement{Name: xml.Name{Space: NsExt, Local: "ExtensionContent"}})
	_ = enc.EncodeToken(xml.EndElement{Name: xml.Name{Space: NsExt, Local: "ExtensionContent"}})
	_ = enc.EncodeToken(xml.EndElement{Name: xml.Name{Space: NsExt, Local: "UBLExtension"}})

	_ = enc.EncodeToken(xml.EndElement{Name: xml.Name{Space: NsExt, Local: "UBLExtensions"}})
	return nil
}

func writeSts(enc *xml.Encoder, local, value string) {
	_ = enc.EncodeToken(xml.StartElement{Name: xml.Name{Space: NsSts, Local: local}})
	_ = enc.EncodeToken(xml.CharData(value))
	_ = enc.EncodeToken(xml.EndElement{Name: xml.Name{Space: NsSts, Local: local}})
}

func (s *XMLBuilderService) writeSupplierParty(enc *xml.Encoder, ctx *InvoiceBuildContext) error {
	_ = enc.EncodeToken(xml.StartElement{Name: xml.Name{Space: NsCac, Local: "AccountingSupplierParty"}})
	_ = enc.EncodeToken(xml.StartElement{Name: xml.Name{Space: NsCac, Local: "Party"}})

	// Identificación fiscal (NIT)
	_ = enc.EncodeToken(xml.StartElement{Name: xml.Name{Space: NsCac, Local: "PartyIdentification"}})
	_ = enc.EncodeToken(xml.StartElement{
		Name: xml.Name{Space: NsCbc, Local: "ID"},
		Attr: []xml.Attr{{Name: xml.Name{Local: "schemeID"}, Value: schemeIDFromCode(ctx.CompanyIdentificationTypeCode)}},
	})
	_ = enc.EncodeToken(xml.CharData(normalizeNIT(ctx.Company.NIT)))
	_ = enc.EncodeToken(xml.EndElement{Name: xml.Name{Space: NsCbc, Local: "ID"}})
	_ = enc.EncodeToken(xml.EndElement{Name: xml.Name{Space: NsCac, Local: "PartyIdentification"}})

	_ = enc.EncodeToken(xml.StartElement{Name: xml.Name{Space: NsCac, Local: "PartyName"}})
	writeCbc(enc, "Name", ctx.Company.Name)
	_ = enc.EncodeToken(xml.EndElement{Name: xml.Name{Space: NsCac, Local: "PartyName"}})

	if ctx.Company.Address != "" {
		_ = enc.EncodeToken(xml.StartElement{Name: xml.Name{Space: NsCac, Local: "PostalAddress"}})
		writeCbc(enc, "StreetName", ctx.Company.Address)
		_ = enc.EncodeToken(xml.EndElement{Name: xml.Name{Space: NsCac, Local: "PostalAddress"}})
	}

	_ = enc.EncodeToken(xml.EndElement{Name: xml.Name{Space: NsCac, Local: "Party"}})
	_ = enc.EncodeToken(xml.EndElement{Name: xml.Name{Space: NsCac, Local: "AccountingSupplierParty"}})
	return nil
}

func (s *XMLBuilderService) writeCustomerParty(enc *xml.Encoder, ctx *InvoiceBuildContext) error {
	_ = enc.EncodeToken(xml.StartElement{Name: xml.Name{Space: NsCac, Local: "AccountingCustomerParty"}})
	_ = enc.EncodeToken(xml.StartElement{Name: xml.Name{Space: NsCac, Local: "Party"}})

	_ = enc.EncodeToken(xml.StartElement{Name: xml.Name{Space: NsCac, Local: "PartyIdentification"}})
	_ = enc.EncodeToken(xml.StartElement{
		Name: xml.Name{Space: NsCbc, Local: "ID"},
		Attr: []xml.Attr{{Name: xml.Name{Local: "schemeID"}, Value: schemeIDFromCode(ctx.CustomerIdentificationTypeCode)}},
	})
	_ = enc.EncodeToken(xml.CharData(normalizeNIT(ctx.Customer.TaxID)))
	_ = enc.EncodeToken(xml.EndElement{Name: xml.Name{Space: NsCbc, Local: "ID"}})
	_ = enc.EncodeToken(xml.EndElement{Name: xml.Name{Space: NsCac, Local: "PartyIdentification"}})

	_ = enc.EncodeToken(xml.StartElement{Name: xml.Name{Space: NsCac, Local: "PartyName"}})
	writeCbc(enc, "Name", ctx.Customer.Name)
	_ = enc.EncodeToken(xml.EndElement{Name: xml.Name{Space: NsCac, Local: "PartyName"}})

	_ = enc.EncodeToken(xml.EndElement{Name: xml.Name{Space: NsCac, Local: "Party"}})
	_ = enc.EncodeToken(xml.EndElement{Name: xml.Name{Space: NsCac, Local: "AccountingCustomerParty"}})
	return nil
}

func (s *XMLBuilderService) writePaymentMeans(enc *xml.Encoder, ctx *InvoiceBuildContext) {
	form := ctx.PaymentFormCode
	if form == "" {
		form = dian.PaymentFormContado
	}
	method := ctx.PaymentMethodCode
	if method == "" {
		method = dian.PaymentMethodEfectivo
	}
	_ = enc.EncodeToken(xml.StartElement{Name: xml.Name{Space: NsCac, Local: "PaymentMeans"}})
	writeCbc(enc, "PaymentMeansCode", method)
	if ctx.DueDate != nil && form == dian.PaymentFormCredito {
		writeCbc(enc, "PaymentDueDate", ctx.DueDate.Format("2006-01-02"))
	}
	_ = enc.EncodeToken(xml.EndElement{Name: xml.Name{Space: NsCac, Local: "PaymentMeans"}})
}

func (s *XMLBuilderService) writeTaxTotal(enc *xml.Encoder, ctx *InvoiceBuildContext) error {
	percent := "19"
	if ctx.Invoice.NetTotal.IsPositive() {
		pct := ctx.Invoice.TaxTotal.Div(ctx.Invoice.NetTotal).Mul(decimal.NewFromInt(100)).Round(0)
		percent = pct.String()
	}
	_ = enc.EncodeToken(xml.StartElement{Name: xml.Name{Space: NsCac, Local: "TaxTotal"}})
	writeCbcAmount(enc, "TaxAmount", formatDecimal(ctx.Invoice.TaxTotal), "COP")
	_ = enc.EncodeToken(xml.StartElement{Name: xml.Name{Space: NsCac, Local: "TaxSubtotal"}})
	writeCbcAmount(enc, "TaxableAmount", formatDecimal(ctx.Invoice.NetTotal), "COP")
	writeCbcAmount(enc, "TaxAmount", formatDecimal(ctx.Invoice.TaxTotal), "COP")
	_ = enc.EncodeToken(xml.StartElement{Name: xml.Name{Space: NsCac, Local: "TaxCategory"}})
	writeCbc(enc, "ID", dian.TaxCodeIVA)
	writeCbc(enc, "Percent", percent)
	_ = enc.EncodeToken(xml.EndElement{Name: xml.Name{Space: NsCac, Local: "TaxCategory"}})
	_ = enc.EncodeToken(xml.EndElement{Name: xml.Name{Space: NsCac, Local: "TaxSubtotal"}})
	_ = enc.EncodeToken(xml.EndElement{Name: xml.Name{Space: NsCac, Local: "TaxTotal"}})
	return nil
}

func (s *XMLBuilderService) writeLegalMonetaryTotal(enc *xml.Encoder, ctx *InvoiceBuildContext) error {
	_ = enc.EncodeToken(xml.StartElement{Name: xml.Name{Space: NsCac, Local: "LegalMonetaryTotal"}})
	writeCbcAmount(enc, "LineExtensionAmount", formatDecimal(ctx.Invoice.NetTotal), "COP")
	writeCbcAmount(enc, "TaxExclusiveAmount", formatDecimal(ctx.Invoice.NetTotal), "COP")
	writeCbcAmount(enc, "TaxInclusiveAmount", formatDecimal(ctx.Invoice.GrandTotal), "COP")
	writeCbcAmount(enc, "PayableAmount", formatDecimal(ctx.Invoice.GrandTotal), "COP")
	_ = enc.EncodeToken(xml.EndElement{Name: xml.Name{Space: NsCac, Local: "LegalMonetaryTotal"}})
	return nil
}

func (s *XMLBuilderService) writeInvoiceLine(enc *xml.Encoder, lineNum int, line InvoiceLineForXML) error {
	unitCode := line.UnitCode
	if unitCode == "" {
		unitCode = dian.UnitUnit
	}
	_ = enc.EncodeToken(xml.StartElement{Name: xml.Name{Space: NsCac, Local: "InvoiceLine"}})
	writeCbc(enc, "ID", strconv.Itoa(lineNum))
	writeCbcWithAttr(enc, "InvoicedQuantity", formatDecimal(line.Quantity), "unitCode", unitCode)
	writeCbcAmount(enc, "LineExtensionAmount", formatDecimal(line.Subtotal), "COP")

	// cac:Item
	_ = enc.EncodeToken(xml.StartElement{Name: xml.Name{Space: NsCac, Local: "Item"}})
	desc := line.ProductName
	if desc == "" && line.Detail != nil {
		desc = "Item " + strconv.Itoa(lineNum)
	}
	writeCbc(enc, "Description", desc)
	if line.ProductCode != "" {
		_ = enc.EncodeToken(xml.StartElement{Name: xml.Name{Space: NsCac, Local: "SellersItemIdentification"}})
		writeCbc(enc, "ID", line.ProductCode)
		_ = enc.EncodeToken(xml.EndElement{Name: xml.Name{Space: NsCac, Local: "SellersItemIdentification"}})
	}
	_ = enc.EncodeToken(xml.EndElement{Name: xml.Name{Space: NsCac, Local: "Item"}})

	// cac:Price
	_ = enc.EncodeToken(xml.StartElement{Name: xml.Name{Space: NsCac, Local: "Price"}})
	writeCbcAmount(enc, "PriceAmount", formatDecimal(line.UnitPrice), "COP")
	writeCbcWithAttr(enc, "BaseQuantity", "1", "unitCode", unitCode)
	_ = enc.EncodeToken(xml.EndElement{Name: xml.Name{Space: NsCac, Local: "Price"}})

	_ = enc.EncodeToken(xml.EndElement{Name: xml.Name{Space: NsCac, Local: "InvoiceLine"}})
	return nil
}

func schemeIDFromCode(code string) string {
	if code == "31" {
		return "31"
	}
	if code == "13" {
		return "13"
	}
	return "31"
}

func normalizeNIT(nit string) string {
	var out []byte
	for _, b := range []byte(nit) {
		if b >= '0' && b <= '9' {
			out = append(out, b)
		}
	}
	return string(out)
}

func formatDecimal(d decimal.Decimal) string {
	return d.Round(2).StringFixed(2)
}

func writeCbcWithAttr(enc *xml.Encoder, local, value, attrLocal, attrValue string) {
	_ = enc.EncodeToken(xml.StartElement{
		Name: xml.Name{Space: NsCbc, Local: local},
		Attr: []xml.Attr{{Name: xml.Name{Local: attrLocal}, Value: attrValue}},
	})
	_ = enc.EncodeToken(xml.CharData(value))
	_ = enc.EncodeToken(xml.EndElement{Name: xml.Name{Space: NsCbc, Local: local}})
}
