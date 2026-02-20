// Carga de certificado desde .p12 (PKCS#12) o par PEM.

package signer

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"os"

	"golang.org/x/crypto/pkcs12"
)

// LoadFromP12 carga certificado y llave privada desde un archivo .p12/.pfx.
// El password puede ser vacío si el archivo no está protegido.
func LoadFromP12(path, password string) (tls.Certificate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("leer p12: %w", err)
	}
	priv, cert, err := pkcs12.Decode(data, password)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("decodificar p12: %w", err)
	}
	// pkcs12.Decode devuelve un solo certificado; tls.Certificate espera una cadena.
	// Para DIAN suele bastar el certificado hoja.
	return tls.Certificate{
		Certificate: [][]byte{cert.Raw},
		PrivateKey:  priv,
		Leaf:        cert,
	}, nil
}

// LoadFromPEM carga certificado y llave desde archivos PEM (certificado y llave por separado, o combinados).
func LoadFromPEM(certPath, keyPath string) (tls.Certificate, error) {
	if certPath == "" {
		return tls.Certificate{}, nil
	}
	if keyPath == "" {
		return tls.LoadX509KeyPair(certPath, certPath)
	}
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("cargar PEM: %w", err)
	}
	return cert, nil
}

// CertDigestAndIssuerSerial devuelve el digest SHA-256 del certificado (Base64) y el serial en hex para XAdES.
func CertDigestAndIssuerSerial(cert *x509.Certificate) (digestB64 string, issuerName string, serialHex string) {
	h := sha256.Sum256(cert.Raw)
	digestB64 = base64.StdEncoding.EncodeToString(h[:])
	issuerName = cert.Issuer.String()
	serialHex = cert.SerialNumber.Text(16)
	return digestB64, issuerName, serialHex
}
