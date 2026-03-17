//go:build ignore
// +build ignore

package main

import (
	"crypto/tls"
	"fmt"
	"log"

	"gopkg.in/gomail.v2"
)

func main() {
	// 1. Configuramos el correo
	m := gomail.NewMessage()
	m.SetHeader("From", "noresponder@ludoia.com")
	m.SetHeader("To", "jhoicas@gmail.com") // Tu correo destino
	m.SetHeader("Subject", "Prueba exitosa desde Resend y Go 🚀")
	m.SetBody("text/html", "<b>¡Hola!</b> Si estás leyendo esto, la configuración de SMTP funcionó perfectamente.")

	// 2. Credenciales exactas de Resend
	host := "smtp.resend.com"
	port := 587
	user := "resend"
	password := "re_X8Rpn4Jw_GTzeotxgr6rJHpra59pPoigq" // <-- Pon tu API Key aquí

	fmt.Printf("📧 Enviando correo de prueba a: jhoicas@gmail.com\n")
	fmt.Printf("   Host: %s:%d\n", host, port)
	fmt.Printf("   From: noresponder@ludoia.com\n")

	// 3. Preparamos el cliente SMTP
	d := gomail.NewDialer(host, port, user, password)

	// Resend requiere TLS, esto asegura que la conexión sea segura
	d.TLSConfig = &tls.Config{InsecureSkipVerify: false, ServerName: host}

	// 4. Enviamos
	if err := d.DialAndSend(m); err != nil {
		log.Fatalf("❌ Error: %v", err)
	}

	fmt.Println("✅ ¡Éxito! El correo ha sido enviado.")
}
