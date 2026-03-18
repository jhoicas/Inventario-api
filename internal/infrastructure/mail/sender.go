package mail

import (
	"fmt"
	"net/smtp"
	"os"
	"strconv"
	"strings"
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

	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	auth := smtp.PlainAuth("", s.user, s.pass, s.host)

	headers := make(map[string]string)
	headers["From"] = s.from
	headers["To"] = to
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/plain; charset=\"UTF-8\""

	var msgBuilder strings.Builder
	for k, v := range headers {
		msgBuilder.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	msgBuilder.WriteString("\r\n")
	msgBuilder.WriteString(body)

	msg := []byte(msgBuilder.String())

	if err := smtp.SendMail(addr, auth, s.from, []string{to}, msg); err != nil {
		return fmt.Errorf("smtp: enviar correo: %w", err)
	}
	return nil
}

