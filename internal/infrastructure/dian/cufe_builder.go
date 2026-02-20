package dian

import (
	"errors"
	"regexp"
	"strings"

	"github.com/shopspring/decimal"
	domdian "github.com/tu-usuario/inventory-pro/internal/domain/dian"
	"github.com/tu-usuario/inventory-pro/internal/domain/entity"
)

// CufeContext agrupa factura, emisor, cliente y datos de resolución para calcular el CUFE.
type CufeContext struct {
	Invoice      *entity.Invoice
	Company      *entity.Company
	Customer     *entity.Customer
	ClaveTecnica string // Clave técnica de la resolución (DB)
	TipoAmbiente string // "1" = Producción, "2" = Pruebas
}

// CalculateCufeFromInvoice construye CufeParams desde el contexto y devuelve el CUFE (hex).
// Asigna el valor a inv.CUFE e inv.UUID. ValFac = NetTotal, ValImp_01 = IVA; Impoconsumo e ICA en 0 si no aplican.
func CalculateCufeFromInvoice(ctx *CufeContext) (string, error) {
	if ctx == nil || ctx.Invoice == nil || ctx.Company == nil || ctx.Customer == nil {
		return "", errors.New("dian: se requieren factura, empresa y cliente para calcular el CUFE")
	}
	inv := ctx.Invoice
	tipoAmb := ctx.TipoAmbiente
	if tipoAmb == "" {
		tipoAmb = "1"
	}
	params := &domdian.CufeParams{
		NumFac:    strings.TrimSpace(inv.Prefix) + strings.TrimSpace(inv.Number),
		FecFac:    inv.Date.Format("2006-01-02"), // YYYY-MM-DD
		ValFac:    inv.NetTotal,                   // Valor total sin impuestos
		ValImp_01: inv.TaxTotal,                   // IVA (código 01)
		ValImp_04: decimal.Zero,                   // Impoconsumo (04); poner valor si aplica
		ValImp_03: decimal.Zero,                   // ICA (03); poner valor si aplica
		ValPag:    inv.GrandTotal,
		NitOfe:    onlyDigitsNIT(ctx.Company.NIT),
		DocAdq:    onlyDigitsNIT(ctx.Customer.TaxID),
		ClTec:     ctx.ClaveTecnica,
		TipoAmb:   tipoAmb,
	}
	svc := domdian.NewCufeCalculatorService()
	cufe, err := svc.Calculate(params)
	if err != nil {
		return "", err
	}
	inv.CUFE = cufe
	inv.UUID = cufe
	return cufe, nil
}

func onlyDigitsNIT(s string) string {
	return regexp.MustCompile(`[^0-9]`).ReplaceAllString(s, "")
}
