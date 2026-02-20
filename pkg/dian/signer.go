// Package dian: interfaz para firma digital de documentos XML (XAdES-EPES, DIAN).

package dian

import "crypto/tls"

// Signer firma un XML de factura y devuelve el XML con la firma inyectada en el ExtensionContent.
type Signer interface {
	// Sign toma el XML de la factura (sin firma) y el certificado con llave privada,
	// y retorna el XML con el nodo ds:Signature dentro de ext:ExtensionContent.
	Sign(xmlBytes []byte, cert tls.Certificate) ([]byte, error)
}
