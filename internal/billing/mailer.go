package billing

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"strings"

	"gopkg.in/gomail.v2"

	appbilling "github.com/jhoicas/Inventario-api/internal/application/billing"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
)

// SMTPConfig parámetros del servidor SMTP saliente.
// Se leen desde variables de entorno: SMTP_HOST, SMTP_PORT, SMTP_USER, SMTP_PASSWORD, SMTP_FROM.
type SMTPConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
}

// InvoiceMailer envía la factura electrónica (PDF + XML) al correo del cliente
// tras ser validada por la DIAN.
type InvoiceMailer struct {
	invoiceRepo  repository.InvoiceRepository
	companyRepo  repository.CompanyRepository
	customerRepo repository.CustomerRepository
	productRepo  repository.ProductRepository
	pdfGen       appbilling.InvoicePDFGenerator
	smtp         SMTPConfig
}

// NewInvoiceMailer construye el mailer inyectando todas sus dependencias.
func NewInvoiceMailer(
	invoiceRepo repository.InvoiceRepository,
	companyRepo repository.CompanyRepository,
	customerRepo repository.CustomerRepository,
	productRepo repository.ProductRepository,
	pdfGen appbilling.InvoicePDFGenerator,
	smtpCfg SMTPConfig,
) *InvoiceMailer {
	return &InvoiceMailer{
		invoiceRepo:  invoiceRepo,
		companyRepo:  companyRepo,
		customerRepo: customerRepo,
		productRepo:  productRepo,
		pdfGen:       pdfGen,
		smtp:         smtpCfg,
	}
}

// SendInvoiceEmail dispara el envío del correo en una goroutine independiente.
// Los errores se registran en el log; nunca bloquea el flujo principal.
func (m *InvoiceMailer) SendInvoiceEmail(invoiceID string) {
	go func() {
		if err := m.send(context.Background(), invoiceID); err != nil {
			log.Printf("[MAILER][%s] error enviando correo: %v", invoiceID, err)
		} else {
			log.Printf("[MAILER][%s] correo enviado correctamente", invoiceID)
		}
	}()
}

// SendInvoiceEmailSync envía el correo de forma síncrona y devuelve el error si lo hay.
// Usado por el endpoint manual POST /api/invoices/{id}/send-email.
func (m *InvoiceMailer) SendInvoiceEmailSync(ctx context.Context, companyID, invoiceID string) error {
	return m.send(ctx, invoiceID)
}

// send es la implementación interna del envío.
func (m *InvoiceMailer) send(ctx context.Context, invoiceID string) error {
	if m.smtp.Host == "" {
		return fmt.Errorf("SMTP_HOST no configurado; se omite el envío de correo")
	}

	// ── 1. Cargar factura ─────────────────────────────────────────────────────
	inv, err := m.invoiceRepo.GetByID(invoiceID)
	if err != nil || inv == nil {
		return fmt.Errorf("factura %s no encontrada: %w", invoiceID, err)
	}

	// ── 2. Cargar cliente (necesitamos el email) ──────────────────────────────
	customer, err := m.customerRepo.GetByID(inv.CustomerID)
	if err != nil || customer == nil {
		return fmt.Errorf("cliente %s no encontrado: %w", inv.CustomerID, err)
	}
	if customer.Email == "" {
		return fmt.Errorf("cliente %s sin email configurado; se omite el correo", customer.ID)
	}

	// ── 3. Cargar empresa ─────────────────────────────────────────────────────
	company, err := m.companyRepo.GetByID(inv.CompanyID)
	if err != nil || company == nil {
		return fmt.Errorf("empresa %s no encontrada: %w", inv.CompanyID, err)
	}

	// ── 4. Generar PDF ────────────────────────────────────────────────────────
	rawDetails, err := m.invoiceRepo.GetDetailsByInvoiceID(invoiceID)
	if err != nil {
		return fmt.Errorf("detalles de factura no encontrados: %w", err)
	}
	enriched := make([]appbilling.InvoiceDetailForPDF, 0, len(rawDetails))
	for _, d := range rawDetails {
		name := "Producto " + d.ProductID
		if product, pErr := m.productRepo.GetByID(d.ProductID); pErr == nil && product != nil {
			name = product.Name
		}
		enriched = append(enriched, appbilling.InvoiceDetailForPDF{InvoiceDetail: *d, ProductName: name})
	}
	pdfBytes, err := m.pdfGen.GenerateInvoicePDF(ctx, inv, company, customer, enriched)
	if err != nil {
		return fmt.Errorf("error generando PDF para correo: %w", err)
	}

	// ── 5. Construir dirección "From" usando el dominio de la empresa ─────────
	from := buildSenderEmail(company.Email)
	if from == "" {
		from = m.smtp.From
		if from == "" {
			from = m.smtp.User
		}
	}

	// ── 6. Construir mensaje ──────────────────────────────────────────────────
	docNum := strings.TrimSpace(inv.Prefix) + strings.TrimSpace(inv.Number)
	subject := fmt.Sprintf("Factura electrónica %s – %s", docNum, company.Name)
	body := fmt.Sprintf(
		"Estimado(a) %s,\n\n"+
			"Le informamos que la factura electrónica %s emitida por %s "+
			"ha sido validada correctamente por la DIAN.\n\n"+
			"CUFE: %s\n\n"+
			"Adjunto encontrará el PDF y el archivo XML de la factura.\n\n"+
			"Gracias por su preferencia.\n\n"+
			"— %s",
		customer.Name, docNum, company.Name, inv.CUFE, company.Name,
	)

	msg := gomail.NewMessage()
	msg.SetHeader("From", from)
	msg.SetHeader("To", customer.Email)
	msg.SetHeader("Subject", subject)
	msg.SetBody("text/plain", body)

	// ── 6. Adjuntar PDF ───────────────────────────────────────────────────────
	pdfName := fmt.Sprintf("factura_%s.pdf", docNum)
	pdfCopy := make([]byte, len(pdfBytes))
	copy(pdfCopy, pdfBytes)
	msg.Attach(pdfName, gomail.SetCopyFunc(func(w io.Writer) error {
		_, err := io.Copy(w, bytes.NewReader(pdfCopy))
		return err
	}))

	// ── 7. Adjuntar XML firmado (si existe) ───────────────────────────────────
	if inv.XMLSigned != "" {
		xmlName := fmt.Sprintf("factura_%s.xml", docNum)
		xmlContent := inv.XMLSigned
		msg.Attach(xmlName, gomail.SetCopyFunc(func(w io.Writer) error {
			_, err := io.Copy(w, strings.NewReader(xmlContent))
			return err
		}))
	}

	// ── 8. Enviar ─────────────────────────────────────────────────────────────
	dialer := gomail.NewDialer(m.smtp.Host, m.smtp.Port, m.smtp.User, m.smtp.Password)
	if err := dialer.DialAndSend(msg); err != nil {
		return fmt.Errorf("error al enviar correo SMTP: %w", err)
	}
	return nil
}

// buildSenderEmail extrae el dominio del email de la empresa y construye noresponde@dominio.
// Ejemplo: contacto@artemisa.co → noresponder@artemisa.co
func buildSenderEmail(companyEmail string) string {
	if companyEmail == "" {
		return ""
	}
	// Buscar el último @ para extraer la parte del dominio
	atIndex := strings.LastIndex(companyEmail, "@")
	if atIndex == -1 || atIndex == len(companyEmail)-1 {
		// No hay @ o está al final (dominio vacío)
		return ""
	}
	domain := companyEmail[atIndex+1:]
	if domain == "" {
		return ""
	}
	return "noresponder@" + domain
}
