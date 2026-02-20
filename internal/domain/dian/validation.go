// Package dian contiene validaciones de dominio para facturación electrónica DIAN (Colombia),
// según Anexo Técnico 1.9. Utiliza catálogos y reglas de pkg/dian.
package dian

import (
	"errors"
	"fmt"

	"github.com/tu-usuario/inventory-pro/internal/domain/entity"
	"github.com/tu-usuario/inventory-pro/pkg/dian"

	"github.com/shopspring/decimal"
)

// ErrInvalidInvoice agrupa errores de validación de factura.
var ErrInvalidInvoice = errors.New("factura inválida para DIAN")

// ValidateInvoice valida la factura y sus detalles según reglas del Anexo Técnico 1.9.
// Para clientes jurídicos (NIT, tipo 31) exige que customerTaxID tenga dígito de verificación válido.
// Comprueba que los totales de impuestos y netos coincidan con la suma de los ítems.
func ValidateInvoice(
	invoice *entity.Invoice,
	details []*entity.InvoiceDetail,
	customerIdentificationTypeCode string,
	customerTaxID string,
) error {
	if invoice == nil {
		return fmt.Errorf("%w: factura nula", ErrInvalidInvoice)
	}
	var errs []error

	// Cliente jurídico (NIT): debe tener dígito de verificación válido (Anexo 1.9).
	if customerIdentificationTypeCode == dian.IdentificationTypeNIT {
		if err := dian.ValidateNITVerificationDigit(customerTaxID); err != nil {
			errs = append(errs, fmt.Errorf("cliente NIT: %w", err))
		}
	}

	// Totales coherentes con los detalles.
	if len(details) == 0 {
		errs = append(errs, fmt.Errorf("%w: la factura debe tener al menos un detalle", ErrInvalidInvoice))
	} else {
		var sumSubtotal, sumTax decimal.Decimal
		for _, d := range details {
			sumSubtotal = sumSubtotal.Add(d.Subtotal)
			// Impuesto por línea = Subtotal * TaxRate (por ejemplo IVA 19% sobre base).
			lineTax := d.Subtotal.Mul(d.TaxRate).Round(2)
			sumTax = sumTax.Add(lineTax)
		}
		if !invoice.NetTotal.Equal(sumSubtotal.Round(2)) {
			errs = append(errs, fmt.Errorf("net total (%s) no coincide con la suma de subtotales de ítems (%s)", invoice.NetTotal.String(), sumSubtotal.Round(2).String()))
		}
		if !invoice.TaxTotal.Equal(sumTax) {
			errs = append(errs, fmt.Errorf("tax total (%s) no coincide con la suma de impuestos por ítems (%s)", invoice.TaxTotal.String(), sumTax.String()))
		}
		expectedGrand := sumSubtotal.Add(sumTax).Round(2)
		if !invoice.GrandTotal.Equal(expectedGrand) {
			errs = append(errs, fmt.Errorf("grand total (%s) no coincide con net + tax (%s)", invoice.GrandTotal.String(), expectedGrand.String()))
		}
	}

	if len(errs) > 0 {
		return errors.Join(append([]error{ErrInvalidInvoice}, errs...)...)
	}
	return nil
}
