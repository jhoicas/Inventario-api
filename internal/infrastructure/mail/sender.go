package mail

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"time"
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

	// net/smtp no expone context; aquí ponemos timeouts a nivel de conexión
	// para evitar que el endpoint se quede "colgado" en redes lentas o caídas.
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 10 * time.Second}
	conn, err := dialer.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("smtp: dial %s: %w", addr, err)
	}
	defer func() { _ = conn.Close() }()

	_ = conn.SetDeadline(time.Now().Add(20 * time.Second))

	// 465 suele ser SMTP sobre TLS implícito.
	if s.port == 465 {
		tlsConn := tls.Client(conn, &tls.Config{ServerName: s.host})
		if err := tlsConn.Handshake(); err != nil {
			return fmt.Errorf("smtp: TLS handshake: %w", err)
		}
		conn = tlsConn
		_ = conn.SetDeadline(time.Now().Add(20 * time.Second))
	}

	client, err := smtp.NewClient(conn, s.host)
	if err != nil {
		return fmt.Errorf("smtp: new client: %w", err)
	}
	defer func() { _ = client.Close() }()

	// 587 suele usar STARTTLS.
	if s.port == 587 {
		if ok, _ := client.Extension("STARTTLS"); ok {
			// Reintento de deadline posterior.
			_ = conn.SetDeadline(time.Now().Add(20 * time.Second))
			if err := client.StartTLS(&tls.Config{ServerName: s.host}); err != nil {
				return fmt.Errorf("smtp: starttls: %w", err)
			}
		}
	}

	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("smtp: auth: %w", err)
	}
	if err := client.Mail(s.from); err != nil {
		return fmt.Errorf("smtp: mail from: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp: rcpt: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp: data: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		_ = w.Close()
		return fmt.Errorf("smtp: write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp: close data: %w", err)
	}
	_ = client.Quit()
	return nil
}

