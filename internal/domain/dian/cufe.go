// Package dian: cálculo del CUFE (Código Único de Factura Electrónica) según Anexo Técnico DIAN 1.9.
// Algoritmo: SHA-384. Fórmula de concatenación en el orden estricto definido por la DIAN.

package dian

import (
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"

	"github.com/shopspring/decimal"
)

// Códigos de impuesto DIAN para la cadena CUFE.
const (
	CodImpIVA         = "01" // IVA
	CodImpImpoconsumo = "04" // Impoconsumo (Impuesto Nacional al Consumo)
	CodImpICA         = "03" // ICA
)

// CufeParams contiene los datos para calcular el CUFE en el orden exigido por la DIAN.
type CufeParams struct {
	NumFac       string          // Número de factura (prefijo + número, sin espacios)
	FecFac       string          // Fecha de emisión YYYY-MM-DD
	ValFac       decimal.Decimal // Valor total sin impuestos (neto)
	ValImp_01    decimal.Decimal // Valor total IVA (código 01)
	ValImp_04    decimal.Decimal // Valor total Impoconsumo (código 04)
	ValImp_03    decimal.Decimal // Valor total ICA (código 03)
	ValPag       decimal.Decimal // Valor total a pagar (Grand Total)
	NitOfe       string          // NIT del facturador (solo dígitos)
	DocAdq       string          // Número de identificación del adquiriente (solo dígitos)
	ClTec        string          // Clave técnica de la resolución (DB)
	TipoAmb      string          // '1' = Producción, '2' = Pruebas
}

// CufeCalculatorService calcula el CUFE según el Anexo Técnico 1.9.
type CufeCalculatorService struct{}

// NewCufeCalculatorService crea el servicio.
func NewCufeCalculatorService() *CufeCalculatorService {
	return &CufeCalculatorService{}
}

// Calculate genera el CUFE (hash hexadecimal) a partir de los parámetros.
// Fórmula (sin separadores): NumFac + FecFac + ValFac + CodImp_01 + ValImp_01 + CodImp_04 + ValImp_04 + CodImp_03 + ValImp_03 + ValPag + NitOfe + DocAdq + ClTec + TipoAmb
// Algoritmo: SHA-384. Montos sin separador de miles, con punto decimal (ej: 1500.00).
func (s *CufeCalculatorService) Calculate(p *CufeParams) (string, error) {
	if p == nil {
		return "", fmt.Errorf("dian: CufeParams es obligatorio")
	}

	numFac := regexp.MustCompile(`\s+`).ReplaceAllString(strings.TrimSpace(p.NumFac), "")
	if numFac == "" {
		return "", fmt.Errorf("dian: NumFac es obligatorio")
	}
	if p.FecFac == "" {
		return "", fmt.Errorf("dian: FecFac es obligatorio (YYYY-MM-DD)")
	}

	nitOfe := onlyDigits(p.NitOfe)
	docAdq := onlyDigits(p.DocAdq)
	if nitOfe == "" {
		return "", fmt.Errorf("dian: NitOfe es obligatorio para el CUFE")
	}
	if docAdq == "" {
		return "", fmt.Errorf("dian: DocAdq es obligatorio para el CUFE")
	}
	if p.ClTec == "" {
		return "", fmt.Errorf("dian: ClTec es obligatoria para el CUFE")
	}
	tipoAmb := p.TipoAmb
	if tipoAmb == "" {
		tipoAmb = "1"
	}

	// Orden estricto DIAN (sin separadores)
	cadena := numFac +
		p.FecFac +
		formatAmount(p.ValFac) +
		CodImpIVA + formatAmount(p.ValImp_01) +
		CodImpImpoconsumo + formatAmount(p.ValImp_04) +
		CodImpICA + formatAmount(p.ValImp_03) +
		formatAmount(p.ValPag) +
		nitOfe +
		docAdq +
		p.ClTec +
		tipoAmb

	hash := sha512.Sum384([]byte(cadena))
	return hex.EncodeToString(hash[:]), nil
}

// formatAmount formatea montos para la cadena CUFE: sin separador de miles, punto decimal, 2 decimales (ej: 1500.00).
func formatAmount(d decimal.Decimal) string {
	return d.Round(2).StringFixed(2)
}

// onlyDigits deja solo dígitos 0-9 (para NIT y documento).
func onlyDigits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
