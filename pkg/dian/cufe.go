// Package dian: cálculo del CUFE (Código Único de Factura Electrónica) según Anexo Técnico DIAN 1.9.
// Algoritmo: SHA-384 sobre la cadena de concatenación en el orden estricto definido por la DIAN.

package dian

import (
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"

	"github.com/shopspring/decimal"
)

// CufeParams contiene los datos para calcular el CUFE (orden estricto DIAN).
// Se puede construir desde Invoice + Company + Customer + Resolution en la capa de aplicación.
type CufeParams struct {
	NumFac         string          // Número de factura (prefijo + número, sin espacios)
	FecFac         string          // Fecha emisión YYYY-MM-DD
	ValFac         decimal.Decimal // Valor total a pagar (GrandTotal)
	CodImp1        string          // Código impuesto 1 (01=IVA)
	ValImp1        decimal.Decimal
	CodImp2        string
	ValImp2        decimal.Decimal
	CodImp3        string
	ValImp3        decimal.Decimal
	ValPag         decimal.Decimal // Valor total a pagar (igual que ValFac)
	NitOferente    string          // NIT emisor, solo dígitos
	DocAdquiriente string          // Documento cliente, solo dígitos
	ClaveTecnica   string          // Clave técnica de la resolución
	TipoAmbiente   string          // "1" = producción, "2" = habilitación
}

// CufeCalculatorService calcula el CUFE según el Anexo Técnico DIAN.
type CufeCalculatorService struct{}

// NewCufeCalculatorService crea el servicio.
func NewCufeCalculatorService() *CufeCalculatorService {
	return &CufeCalculatorService{}
}

// Calculate genera el CUFE a partir de parámetros ya preparados.
// Orden estricto DIAN: NumFac + FecFac + ValFac + CodImp1 + ValImp1 + CodImp2 + ValImp2 + CodImp3 + ValImp3 + ValPag + NitOfe + DocAdq + ClTec + TipoAmb.
// Hash: SHA-384, salida en hexadecimal (minúsculas).
func (s *CufeCalculatorService) Calculate(p *CufeParams) (string, error) {
	if p == nil {
		return "", fmt.Errorf("dian: CufeParams es obligatorio")
	}

	numFac := regexp.MustCompile(`\s+`).ReplaceAllString(strings.TrimSpace(p.NumFac), "")
	if numFac == "" {
		return "", fmt.Errorf("dian: NumFac es obligatorio")
	}
	if p.FecFac == "" {
		return "", fmt.Errorf("dian: FecFac es obligatorio")
	}
	valFac := formatDecimalForCufe(p.ValFac)
	valPag := formatDecimalForCufe(p.ValPag)

	cod1, cod2, cod3 := p.CodImp1, p.CodImp2, p.CodImp3
	if cod1 == "" {
		cod1 = TaxCodeIVA
	}
	if cod2 == "" {
		cod2 = "00"
	}
	if cod3 == "" {
		cod3 = "00"
	}

	nitOfe := onlyDigits(p.NitOferente)
	docAdq := onlyDigits(p.DocAdquiriente)
	if nitOfe == "" {
		return "", fmt.Errorf("dian: NitOferente es obligatorio para el CUFE")
	}
	if docAdq == "" {
		return "", fmt.Errorf("dian: DocAdquiriente es obligatorio para el CUFE")
	}
	if p.ClaveTecnica == "" {
		return "", fmt.Errorf("dian: ClaveTecnica es obligatoria para el CUFE")
	}
	tipoAmb := p.TipoAmbiente
	if tipoAmb == "" {
		tipoAmb = "1"
	}

	cadena := numFac +
		p.FecFac +
		valFac +
		cod1 + formatDecimalForCufe(p.ValImp1) +
		cod2 + formatDecimalForCufe(p.ValImp2) +
		cod3 + formatDecimalForCufe(p.ValImp3) +
		valPag +
		nitOfe +
		docAdq +
		p.ClaveTecnica +
		tipoAmb

	hash := sha512.Sum384([]byte(cadena))
	return hex.EncodeToString(hash[:]), nil
}

// formatDecimalForCufe formatea el valor para la cadena CUFE: sin separador de miles, punto decimal, 2 decimales.
func formatDecimalForCufe(d decimal.Decimal) string {
	return d.Round(2).StringFixed(2)
}

// onlyDigits deja solo dígitos 0-9.
func onlyDigits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
