package billing

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"gopkg.in/gomail.v2"

	appbilling "github.com/jhoicas/Inventario-api/internal/application/billing"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
)

// SMTPConfig parámetros del servidor SMTP saliente.
// Se leen desde variables de entorno: SMTP_HOST, SMTP_PORT, SMTP_USER, SMTP_PASSWORD, SMTP_FROM.
type SMTPConfig struct {
	Host         string
	Port         int
	User         string
	Password     string
	From         string
	ResendAPIKey string
	ResendAPIURL string
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

// SendCustomEmailSync envía un correo libre sin adjuntos.
func (m *InvoiceMailer) SendCustomEmailSync(ctx context.Context, companyID, to, subject, body string) error {
	if m.smtp.Host == "" && !m.hasResendAPIConfig() {
		return fmt.Errorf("no hay proveedor de correo configurado (SMTP_HOST o RESEND_API_KEY)")
	}

	from := m.smtp.From
	if from == "" {
		from = m.smtp.User
	}

	if m.smtp.Host == "" && m.hasResendAPIConfig() {
		return m.sendWithResendAPI(ctx, from, to, subject, body, "", nil, "", "")
	}

	err := m.sendWithSMTP(from, to, subject, body, "", nil, "", "")
	if err == nil {
		return nil
	}
	if m.hasResendAPIConfig() && isSMTPConnectivityError(err) {
		if apiErr := m.sendWithResendAPI(ctx, from, to, subject, body, "", nil, "", ""); apiErr == nil {
			return nil
		} else {
			return fmt.Errorf("error SMTP: %v; error Resend API: %w", err, apiErr)
		}
	}

	return fmt.Errorf("error al enviar correo SMTP: %w", err)
}

// send es la implementación interna del envío.
func (m *InvoiceMailer) send(ctx context.Context, invoiceID string) error {
	if m.smtp.Host == "" && !m.hasResendAPIConfig() {
		return fmt.Errorf("no hay proveedor de correo configurado (SMTP_HOST o RESEND_API_KEY)")
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

	// ── 5. Construir dirección "From" desde SMTP_FROM ────────────────────────
	from := m.smtp.From
	if from == "" {
		from = m.smtp.User
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

	pdfName := fmt.Sprintf("factura_%s.pdf", docNum)

	// ── 7. Adjuntar XML firmado (si existe) ───────────────────────────────────
	xmlName := ""
	if inv.XMLSigned != "" {
		xmlName = fmt.Sprintf("factura_%s.xml", docNum)
	}

	// ── 8. Enviar ─────────────────────────────────────────────────────────────
	if m.smtp.Host == "" && m.hasResendAPIConfig() {
		if err := m.sendWithResendAPI(ctx, from, customer.Email, subject, body, pdfName, pdfBytes, xmlName, inv.XMLSigned); err != nil {
			return fmt.Errorf("error al enviar correo Resend API: %w", err)
		}
		return nil
	}

	err = m.sendWithSMTP(from, customer.Email, subject, body, pdfName, pdfBytes, xmlName, inv.XMLSigned)
	if err == nil {
		return nil
	}

	if m.hasResendAPIConfig() && isSMTPConnectivityError(err) {
		if apiErr := m.sendWithResendAPI(ctx, from, customer.Email, subject, body, pdfName, pdfBytes, xmlName, inv.XMLSigned); apiErr == nil {
			log.Printf("[MAILER][%s] SMTP falló por conectividad, enviado vía Resend API", invoiceID)
			return nil
		} else {
			return fmt.Errorf("error SMTP: %v; error Resend API: %w", err, apiErr)
		}
	}

	return fmt.Errorf("error al enviar correo SMTP: %w", err)
}

func (m *InvoiceMailer) sendWithSMTP(from, to, subject, body, pdfName string, pdfBytes []byte, xmlName, xmlContent string) error {
	msg := gomail.NewMessage()
	msg.SetHeader("From", from)
	msg.SetHeader("To", to)
	msg.SetHeader("Subject", subject)
	msg.SetBody("text/plain", body)

	if pdfName != "" && len(pdfBytes) > 0 {
		pdfCopy := make([]byte, len(pdfBytes))
		copy(pdfCopy, pdfBytes)
		msg.Attach(pdfName, gomail.SetCopyFunc(func(w io.Writer) error {
			_, err := io.Copy(w, bytes.NewReader(pdfCopy))
			return err
		}))
	}

	if xmlName != "" && xmlContent != "" {
		xmlCopy := xmlContent
		msg.Attach(xmlName, gomail.SetCopyFunc(func(w io.Writer) error {
			_, err := io.Copy(w, strings.NewReader(xmlCopy))
			return err
		}))
	}

	dialer := gomail.NewDialer(m.smtp.Host, m.smtp.Port, m.smtp.User, m.smtp.Password)
	dialer.TLSConfig = &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         m.smtp.Host,
	}
	if err := dialer.DialAndSend(msg); err != nil {
		return err
	}
	return nil
}

type resendAttachment struct {
	Filename string `json:"filename"`
	Content  string `json:"content"`
}

type resendEmailRequest struct {
	From        string             `json:"from"`
	To          []string           `json:"to"`
	Subject     string             `json:"subject"`
	Text        string             `json:"text"`
	Attachments []resendAttachment `json:"attachments,omitempty"`
}

func (m *InvoiceMailer) sendWithResendAPI(ctx context.Context, from, to, subject, body, pdfName string, pdfBytes []byte, xmlName, xmlContent string) error {
	apiURL := strings.TrimSpace(m.smtp.ResendAPIURL)
	if apiURL == "" {
		apiURL = "https://api.resend.com/emails"
	}

	apiKey := strings.TrimSpace(m.smtp.ResendAPIKey)
	if apiKey == "" {
		apiKey = strings.TrimSpace(m.smtp.Password)
	}
	if apiKey == "" {
		return fmt.Errorf("RESEND_API_KEY no configurado")
	}

	reqBody := resendEmailRequest{
		From:    from,
		To:      []string{to},
		Subject: subject,
		Text:    body,
	}
	if pdfName != "" && len(pdfBytes) > 0 {
		reqBody.Attachments = append(reqBody.Attachments, resendAttachment{Filename: pdfName, Content: base64.StdEncoding.EncodeToString(pdfBytes)})
	}

	if xmlName != "" && xmlContent != "" {
		reqBody.Attachments = append(reqBody.Attachments, resendAttachment{
			Filename: xmlName,
			Content:  base64.StdEncoding.EncodeToString([]byte(xmlContent)),
		})
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("crear request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request Resend API: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if resp.StatusCode == http.StatusForbidden && strings.Contains(strings.ToLower(string(respBody)), "not authorized to send emails from") {
			fallbackFrom := strings.TrimSpace(m.smtp.From)
			if fallbackFrom != "" && !strings.EqualFold(fallbackFrom, from) {
				return m.sendWithResendAPI(ctx, fallbackFrom, to, subject, body, pdfName, pdfBytes, xmlName, xmlContent)
			}
		}
		return fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	return nil
}

func isSMTPConnectivityError(err error) bool {
	if err == nil {
		return false
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "i/o timeout") ||
		strings.Contains(msg, "connection timed out") ||
		strings.Contains(msg, "no route to host") ||
		strings.Contains(msg, "connection refused")
}

func (m *InvoiceMailer) hasResendAPIConfig() bool {
	return strings.TrimSpace(m.smtp.ResendAPIKey) != "" || strings.TrimSpace(m.smtp.Password) != ""
}
