package mail

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"crypto/tls"

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
		return fmt.Errorf("smtp: enviar correo: %w", err)
	}
	return nil
}

