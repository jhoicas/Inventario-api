package dian

import (
	"crypto/tls"
	"fmt"
)

// LoadCertFromPEM carga un certificado y llave privada desde archivos PEM.
// Si certPath está vacío retorna cert vacío y err nil (modo simulado: no firmar).
func LoadCertFromPEM(certPath, keyPath string) (tls.Certificate, error) {
	if certPath == "" {
		return tls.Certificate{}, nil
	}
	if keyPath == "" {
		// Un solo archivo puede contener cert+key en PEM
		return tls.LoadX509KeyPair(certPath, certPath)
	}
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("cargar certificado DIAN: %w", err)
	}
	return cert, nil
}
