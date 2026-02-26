package dian

import (
	"archive/zip"
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/tu-usuario/inventory-pro/internal/domain/entity"
)

// CompressXMLToZip empaqueta el XML firmado en un archivo ZIP en memoria.
// La DIAN exige que el ZIP contenga un único archivo con el nombre:
//
//	{NIT_OFE}{PREFIX}{NUMBER}.xml  (sin guiones ni espacios)
//
// Devuelve los bytes del ZIP listo para enviar al WS DIAN.
func CompressXMLToZip(xmlBytes []byte, xmlFilename string) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	fw, err := zw.Create(xmlFilename)
	if err != nil {
		return nil, fmt.Errorf("zip: crear entrada %s: %w", xmlFilename, err)
	}
	if _, err := fw.Write(xmlBytes); err != nil {
		return nil, fmt.Errorf("zip: escribir XML: %w", err)
	}
	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("zip: cerrar archivo: %w", err)
	}
	return buf.Bytes(), nil
}

// onlyDigits elimina todo carácter que no sea dígito (útil para NIT).
var nonDigit = regexp.MustCompile(`[^0-9]`)

// DIANFilenames genera los nombres de archivo requeridos por la DIAN para el ZIP y el XML interno.
// Formato: {NIT_OFE}{PREFIX}{NUMBER}  (sin DV, solo dígitos del NIT)
// Ejemplo: 900123456SETP000001
func DIANFilenames(company *entity.Company, inv *entity.Invoice) (xmlName, zipName string) {
	nit := nonDigit.ReplaceAllString(company.NIT, "")
	// Quitar dígito de verificación si el NIT tiene más de 9 dígitos y termina en "-DV"
	if idx := strings.Index(nit, "-"); idx != -1 {
		nit = nit[:idx]
	}
	base := nit + strings.TrimSpace(inv.Prefix) + strings.TrimSpace(inv.Number)
	return base + ".xml", base + ".zip"
}
