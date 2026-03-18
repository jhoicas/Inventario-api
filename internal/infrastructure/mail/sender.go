package mail

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"crypto/tls"
	"errors"

	"bytes"
	"encoding/json"
	"net"
	"net/http"
	"time"

	"gopkg.in/gomail.v2"
)

// Sender define el contrato mínimo para enviar correos electrónicos.
type Sender interface {
	Send(to string, subject string, body string) error
}

// SMTPSender implementación simple de Sender usando net/smtp.
type SMTPSender struct {
	host string
	port int
	user string
	pass string
	from string

	resendAPIKey string
	resendAPIURL string
}

// NewSMTPSenderFromEnv construye un SMTPSender leyendo variables de entorno:
// SMTP_HOST, SMTP_PORT, SMTP_USER, SMTP_PASS, SMTP_FROM.
func NewSMTPSenderFromEnv() (*SMTPSender, error) {
	host := strings.TrimSpace(os.Getenv("SMTP_HOST"))
	portStr := strings.TrimSpace(os.Getenv("SMTP_PORT"))
	user := strings.TrimSpace(os.Getenv("SMTP_USER"))
	pass := strings.TrimSpace(os.Getenv("SMTP_PASS"))
	// Compatibilidad: en algunos entornos se configura como SMTP_PASSWORD.
	if pass == "" {
		pass = strings.TrimSpace(os.Getenv("SMTP_PASSWORD"))
	}
	from := strings.TrimSpace(os.Getenv("SMTP_FROM"))

	resendAPIKey := strings.TrimSpace(os.Getenv("RESEND_API_KEY"))
	resendAPIURL := strings.TrimSpace(os.Getenv("RESEND_API_URL"))
	if resendAPIURL == "" {
		resendAPIURL = "https://api.resend.com/emails"
	}

	// Compatibilidad con el patrón que usa InvoiceMailer:
	// si RESEND_API_KEY no está, reutiliza SMTP_PASSWORD como token.
	if resendAPIKey == "" {
		resendAPIKey = pass
	}

	if host == "" || portStr == "" || user == "" || pass == "" || from == "" {
		return nil, fmt.Errorf("smtp: variables de entorno incompletas (requeridas: SMTP_HOST, SMTP_PORT, SMTP_USER, SMTP_PASS|SMTP_PASSWORD, SMTP_FROM)")
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("smtp: puerto inválido: %w", err)
	}

	return &SMTPSender{
		host: host,
		port: port,
		user: user,
		pass: pass,
		from: from,
		resendAPIKey: resendAPIKey,
		resendAPIURL: resendAPIURL,
	}, nil
}

// Send envía un email de texto plano (UTF-8) a un destinatario.
func (s *SMTPSender) Send(to string, subject string, body string) error {
	if strings.TrimSpace(to) == "" {
		return fmt.Errorf("smtp: destinatario vacío")
	}

	msg := gomail.NewMessage()
	msg.SetHeader("From", s.from)
	msg.SetHeader("To", to)
	msg.SetHeader("Subject", subject)
	msg.SetBody("text/plain", body)

	dialer := gomail.NewDialer(s.host, s.port, s.user, s.pass)
	dialer.TLSConfig = &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         s.host,
	}

	if err := dialer.DialAndSend(msg); err != nil {
		// Fallback a Resend si SMTP falla por conectividad/timeouts (consistente con InvoiceMailer).
		if s.hasResendAPIConfig() && isSMTPConnectivityError(err) {
			if resendErr := s.sendWithResendAPI(to, subject, body); resendErr == nil {
				return nil
			}
			// Si Resend falla también, se devuelve el error original de SMTP.
		}
		return fmt.Errorf("smtp: enviar correo: %w", err)
	}
	return nil
}

func (s *SMTPSender) hasResendAPIConfig() bool {
	return strings.TrimSpace(s.resendAPIKey) != ""
}

func (s *SMTPSender) sendWithResendAPI(to, subject, body string) error {
	reqBody := struct {
		From    string   `json:"from"`
		To      []string `json:"to"`
		Subject string   `json:"subject"`
		Text    string   `json:"text"`
	}{
		From:    s.from,
		To:      []string{to},
		Subject: subject,
		Text:    body,
	}

	b, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("resend: marshal: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, s.resendAPIURL, bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("resend: new request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.resendAPIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("resend: request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("resend: status %d", resp.StatusCode)
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

