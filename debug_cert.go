//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"os"

	"golang.org/x/crypto/pkcs12"
)

func main() {
	// 1. Datos copiados EXACTAMENTE de tu .env
	// OJO: Asegúrate de que esta ruta sea 100% real. Copia y pega de tu .env
	certPath := "C:/Users/yoiner.castillo/source/repos/InvoryaBack/certificado_prueba.p12"
	certPass := "123456"

	fmt.Println("🔍 DIAGNÓSTICO DE CERTIFICADO DIAN")
	fmt.Println("----------------------------------")
	fmt.Printf("📂 Intentando leer: %s\n", certPath)

	// 2. Intentar leer el archivo (File System Check)
	p12Data, err := os.ReadFile(certPath)
	if err != nil {
		fmt.Println("\n❌ ERROR DE ARCHIVO:")
		fmt.Printf("   Go no puede encontrar o abrir el archivo.\n")
		fmt.Printf("   Detalle técnico: %v\n", err)
		return
	}
	fmt.Printf("✅ Archivo encontrado. Tamaño: %d bytes\n", len(p12Data))

	// 3. Intentar decodificar (Password Check)
	fmt.Println("\n🔐 Intentando decodificar PKCS#12 con la contraseña...")
	_, _, err = pkcs12.Decode(p12Data, certPass)
	if err != nil {
		fmt.Println("\n❌ ERROR DE CONTRASEÑA O FORMATO:")
		fmt.Printf("   El archivo existe, pero la contraseña '%s' falló o el archivo está corrupto.\n", certPass)
		fmt.Printf("   Detalle técnico: %v\n", err)
		return
	}

	fmt.Println("\n✨ ¡ÉXITO! El certificado y la contraseña son correctos.")
	fmt.Println("   El problema NO es el archivo, es cómo tu app carga el .env.")
}
