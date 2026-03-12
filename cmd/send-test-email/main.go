package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"time"

	"gopkg.in/gomail.v2"

	dianws "github.com/jhoicas/Inventario-api/internal/billing"
	"github.com/jhoicas/Inventario-api/pkg/config"
)

// Stub repositories para pruebas sin DB real.
type stubInvoiceRepo struct{}

func main() {
	toEmail := flag.String("to", "jhoicas@gmail.com", "Email destino")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("cargar config: %v", err)
	}

	if cfg.SMTP.Host == "" {
		log.Fatal("ERROR: SMTP_HOST no configurado. Configura las variables de entorno:\n" +
			"  SMTP_HOST=smtp.gmail.com\n" +
			"  SMTP_PORT=587\n" +
			"  SMTP_USER=tu-email@gmail.com\n" +
			"  SMTP_PASSWORD=tu-aplicacion-password\n" +
			"  SMTP_FROM=tu-email@gmail.com")
	}

	// Configurar SMTP
	smtpCfg := dianws.SMTPConfig{
		Host:     cfg.SMTP.Host,
		Port:     cfg.SMTP.Port,
		User:     cfg.SMTP.User,
		Password: cfg.SMTP.Password,
		From:     cfg.SMTP.From,
	}

	fmt.Printf("📧 Enviando correo de prueba a: %s\n", *toEmail)
	fmt.Printf("   Host: %s:%d\n", smtpCfg.Host, smtpCfg.Port)
	fmt.Printf("   From: %s\n", smtpCfg.From)

	if err := sendTestEmail(*toEmail, smtpCfg); err != nil {
		log.Fatalf("❌ Error: %v", err)
	}

	fmt.Println("✅ Correo enviado correctamente")
}

func sendTestEmail(to string, smtpCfg dianws.SMTPConfig) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	msg := gomail.NewMessage()
	msg.SetHeader("From", smtpCfg.From)
	msg.SetHeader("To", to)
	msg.SetHeader("Subject", "🧪 Prueba de correo – Inventario API")
	msg.SetBody("text/html", fmt.Sprintf(`
		<h2>Prueba de envío exitosa</h2>
		<p>Este es un correo de prueba del sistema de facturación Inventario.</p>
		<p><strong>Hora:</strong> %s</p>
		<p>✅ Sistema de correo funcional.</p>
	`, time.Now().Format("2006-01-02 15:04:05")))

	_ = ctx // ctx no se usa directamente con gomail, pero lo mantenemos para future use
	dialer := gomail.NewDialer(smtpCfg.Host, smtpCfg.Port, smtpCfg.User, smtpCfg.Password)
	dialer.TLSConfig = &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         smtpCfg.Host,
	}
	return dialer.DialAndSend(msg)
}
