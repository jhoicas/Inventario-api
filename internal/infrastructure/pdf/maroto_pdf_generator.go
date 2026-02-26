// Package pdf implementa la generación de la Representación Gráfica de la
// Factura Electrónica DIAN (Resolución 000042/2020, Anexo Técnico 1.9).
//
// Layout de la página A4:
//
//	┌─────────────────────────────────────────────────────────────┐
//	│  HEADER: Razón Social + NIT  │  N° Factura + Fecha          │
//	│  ─────────────────────────────────────────────────────────  │
//	│  EMISOR: Dirección / Tel / Email                             │
//	│  RECEPTOR: Nombre + NIT/CC + contacto                       │
//	│  ─────────────────────────────────────────────────────────  │
//	│  TABLA: Cant | Descripción | P.Unit | IVA | Subtotal         │
//	│  ─────────────────────────────────────────────────────────  │
//	│  TOTALES: Subtotal neto / Impuestos / TOTAL A PAGAR          │
//	│  ─────────────────────────────────────────────────────────  │
//	│  FOOTER DIAN: CUFE + QR + Leyenda legal                      │
//	└─────────────────────────────────────────────────────────────┘
package pdf

import (
	"context"
	"fmt"

	maroto "github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/code"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/line"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/consts/pagesize"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/props"

	appbilling "github.com/jhoicas/Inventario-api/internal/application/billing"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
)

// ── Paleta de colores ─────────────────────────────────────────────────────────

var (
	colorPrimary = &props.Color{Red: 0, Green: 70, Blue: 127}
	colorGray    = &props.Color{Red: 100, Green: 100, Blue: 100}
	colorWhite   = &props.Color{Red: 255, Green: 255, Blue: 255}
)

// ── Generator ─────────────────────────────────────────────────────────────────

// MarotoPDFGenerator implementa billing.InvoicePDFGenerator usando Maroto v2.
type MarotoPDFGenerator struct{}

// NewMarotoPDFGenerator construye el generador.
func NewMarotoPDFGenerator() *MarotoPDFGenerator { return &MarotoPDFGenerator{} }

// GenerateInvoicePDF genera el PDF y devuelve sus bytes.
func (g *MarotoPDFGenerator) GenerateInvoicePDF(
	_ context.Context,
	invoice *entity.Invoice,
	company *entity.Company,
	customer *entity.Customer,
	details []appbilling.InvoiceDetailForPDF,
) ([]byte, error) {
	cfg := config.NewBuilder().
		WithPageSize(pagesize.A4).
		WithLeftMargin(10).WithRightMargin(10).
		WithTopMargin(10).WithBottomMargin(10).
		WithDefaultFont(&props.Font{Family: "helvetica", Size: 9}).
		WithTitle("Factura Electrónica DIAN", true).
		WithAuthor(company.Name, true).
		Build()

	m := maroto.New(cfg)

	// Header principal
	m.AddRows(headerRow(invoice, company))
	m.AddRows(line.NewRow(1, props.Line{Color: colorPrimary, Thickness: 0.5}))
	m.AddRows(emisorRow(company))
	m.AddRows(receptorRow(customer))
	m.AddRows(line.NewRow(1, props.Line{Color: colorPrimary, Thickness: 0.3}))

	// Tabla de detalles
	m.AddRows(tableHeaderRow())
	for _, r := range tableDetailRows(details) {
		m.AddRows(r)
	}

	// Totales
	m.AddRows(line.NewRow(1, props.Line{Color: colorPrimary, Thickness: 0.3}))
	m.AddRows(totalsRow(invoice))

	// Footer DIAN
	m.AddRows(line.NewRow(3))
	m.AddRows(line.NewRow(1, props.Line{Color: colorGray, Thickness: 0.3}))
	for _, r := range dianFooterRows(invoice) {
		m.AddRows(r)
	}

	doc, err := m.Generate()
	if err != nil {
		return nil, fmt.Errorf("pdf: generar documento: %w", err)
	}
	return doc.GetBytes(), nil
}

// ── Secciones ─────────────────────────────────────────────────────────────────

// headerRow: Razón social + NIT (izq) y N° Factura + Fecha (der).
func headerRow(invoice *entity.Invoice, company *entity.Company) core.Row {
	numFac := invoice.Prefix + invoice.Number
	fecha := invoice.Date.Format("02/01/2006")

	return row.New(18).Add(
		col.New(7).Add(
			text.New(company.Name, props.Text{
				Style: fontstyle.Bold, Size: 13, Color: colorPrimary, Top: 1,
			}),
			text.New("NIT: "+company.NIT, props.Text{
				Size: 9, Top: 9, Color: colorGray,
			}),
		),
		col.New(5).Add(
			text.New("FACTURA ELECTRÓNICA DE VENTA", props.Text{
				Style: fontstyle.Bold, Size: 8, Align: align.Right,
				Color: colorPrimary, Top: 1,
			}),
			text.New(numFac, props.Text{
				Style: fontstyle.Bold, Size: 12, Align: align.Right, Top: 7,
			}),
			text.New("Fecha: "+fecha, props.Text{
				Size: 8, Align: align.Right, Top: 14, Color: colorGray,
			}),
		),
	)
}

// emisorRow: datos del emisor (empresa).
func emisorRow(company *entity.Company) core.Row {
	return row.New(12).Add(
		col.New(12).Add(
			text.New("DATOS DEL EMISOR", props.Text{
				Style: fontstyle.Bold, Size: 8, Color: colorPrimary, Top: 1,
			}),
			text.New(fmt.Sprintf("Dirección: %s   |   Tel: %s   |   Email: %s",
				nonEmpty(company.Address, "—"),
				nonEmpty(company.Phone, "—"),
				nonEmpty(company.Email, "—"),
			), props.Text{Size: 8, Top: 7, Color: colorGray}),
		),
	)
}

// receptorRow: datos del comprador.
func receptorRow(customer *entity.Customer) core.Row {
	return row.New(14).Add(
		col.New(12).Add(
			text.New("RECEPTOR / ADQUIRIENTE", props.Text{
				Style: fontstyle.Bold, Size: 8, Color: colorPrimary, Top: 1,
			}),
			text.New(customer.Name, props.Text{
				Style: fontstyle.Bold, Size: 10, Top: 6,
			}),
			text.New(fmt.Sprintf("NIT/CC: %s   |   Email: %s   |   Tel: %s",
				customer.TaxID,
				nonEmpty(customer.Email, "—"),
				nonEmpty(customer.Phone, "—"),
			), props.Text{Size: 8, Top: 12, Color: colorGray}),
		),
	)
}

// tableHeaderRow: cabecera de la tabla de detalles con fondo azul simulado.
func tableHeaderRow() core.Row {
	h := func(label string, size int, a align.Type) core.Col {
		return col.New(size).Add(text.New(label, props.Text{
			Style: fontstyle.Bold, Size: 8, Align: a,
			Color: colorWhite, Top: 2, Left: 1, Right: 1,
		}))
	}
	return row.New(8).Add(
		h("Cant.", 1, align.Center),
		h("Descripción del producto/servicio", 5, align.Left),
		h("Precio Unit.", 2, align.Right),
		h("IVA%", 1, align.Center),
		h("Subtotal", 3, align.Right),
	)
}

// tableDetailRows: una fila por línea de detalle.
func tableDetailRows(details []appbilling.InvoiceDetailForPDF) []core.Row {
	result := make([]core.Row, 0, len(details))
	for _, d := range details {
		result = append(result, row.New(7).Add(
			col.New(1).Add(text.New(
				d.Quantity.StringFixed(0),
				props.Text{Size: 8, Align: align.Center, Top: 1},
			)),
			col.New(5).Add(text.New(
				d.ProductName,
				props.Text{Size: 8, Align: align.Left, Top: 1, Left: 1},
			)),
			col.New(2).Add(text.New(
				"$"+formatMoney(d.UnitPrice.StringFixed(0)),
				props.Text{Size: 8, Align: align.Right, Top: 1, Right: 1},
			)),
			col.New(1).Add(text.New(
				d.TaxRate.StringFixed(0)+"%",
				props.Text{Size: 8, Align: align.Center, Top: 1},
			)),
			col.New(3).Add(text.New(
				"$"+formatMoney(d.Subtotal.StringFixed(0)),
				props.Text{Size: 8, Align: align.Right, Top: 1, Right: 1},
			)),
		))
	}
	return result
}

// totalsRow: bloque de totales alineado a la derecha.
func totalsRow(invoice *entity.Invoice) core.Row {
	label := func(s string) core.Component {
		return text.New(s, props.Text{
			Style: fontstyle.Bold, Size: 9, Align: align.Right, Right: 2,
		})
	}
	value := func(s string) core.Component {
		return text.New(s, props.Text{Size: 9, Align: align.Right, Right: 1})
	}
	grandLabel := func(s string) core.Component {
		return text.New(s, props.Text{
			Style: fontstyle.Bold, Size: 10, Align: align.Right,
			Color: colorPrimary, Right: 2,
		})
	}
	grandValue := func(s string) core.Component {
		return text.New(s, props.Text{
			Style: fontstyle.Bold, Size: 10, Align: align.Right,
			Color: colorPrimary, Right: 1,
		})
	}

	return row.New(26).Add(
		col.New(3), // espacio izquierdo
		col.New(3).Add(
			label("Subtotal neto:"),
			label("Impuestos:"),
			grandLabel("TOTAL A PAGAR:"),
		),
		col.New(3).Add(
			value("$"+formatMoney(invoice.NetTotal.StringFixed(0))),
			value("$"+formatMoney(invoice.TaxTotal.StringFixed(0))),
			grandValue("$"+formatMoney(invoice.GrandTotal.StringFixed(0))),
		),
		col.New(3), // espacio derecho
	)
}

// dianFooterRows: CUFE partido + código QR + leyenda legal.
func dianFooterRows(invoice *entity.Invoice) []core.Row {
	rows := []core.Row{
		row.New(6).Add(col.New(12).Add(
			text.New("INFORMACIÓN ELECTRÓNICA DIAN", props.Text{
				Style: fontstyle.Bold, Size: 8, Color: colorPrimary, Top: 1,
			}),
		)),
	}

	// CUFE partido en fragmentos de 80 caracteres
	if invoice.CUFE != "" {
		rows = append(rows, row.New(5).Add(col.New(12).Add(
			text.New("CUFE (Código Único de Factura Electrónica):", props.Text{
				Style: fontstyle.Bold, Size: 7, Top: 1,
			}),
		)))
		for _, chunk := range splitEvery(invoice.CUFE, 80) {
			rows = append(rows, row.New(4).Add(col.New(12).Add(
				text.New(chunk, props.Text{Size: 6.5, Color: colorGray, Top: 0.5, Left: 2}),
			)))
		}
	}

	rows = append(rows, row.New(3))

	// QR + leyenda
	if invoice.QRData != "" {
		rows = append(rows, row.New(50).Add(
			col.New(4).Add(code.NewQr(invoice.QRData, props.Rect{
				Percent: 95,
				Center:  true,
			})),
			col.New(8).Add(
				text.New("Escanea el código QR para validar\nesta factura en el Portal DIAN.", props.Text{
					Size: 8, Top: 4, Left: 3, Color: colorGray,
				}),
				text.New("Documento equivalente a\nFACTURA ELECTRÓNICA DE VENTA", props.Text{
					Style: fontstyle.Bold, Size: 10, Top: 22,
					Left: 3, Color: colorPrimary,
				}),
			),
		))
	} else {
		rows = append(rows, row.New(10).Add(col.New(12).Add(
			text.New("Documento equivalente a FACTURA ELECTRÓNICA DE VENTA", props.Text{
				Style: fontstyle.Bold, Size: 9, Align: align.Center,
				Color: colorPrimary, Top: 2,
			}),
		)))
	}

	// Leyenda legal
	rows = append(rows, row.New(8).Add(col.New(12).Add(
		text.New(
			"Esta factura electrónica fue generada conforme a la normativa DIAN "+
				"(Decreto 2242/2015, Resolución 000042/2020). "+
				"Conserve este documento como soporte fiscal.",
			props.Text{Size: 6.5, Color: colorGray, Top: 2},
		),
	)))

	return rows
}

// ── helpers ───────────────────────────────────────────────────────────────────

func nonEmpty(s, fallback string) string {
	if s != "" {
		return s
	}
	return fallback
}

// formatMoney inserta puntos de miles en un string numérico sin decimales.
// Ej: "25000" → "25.000", "1000000" → "1.000.000"
func formatMoney(s string) string {
	n := len(s)
	if n <= 3 {
		return s
	}
	buf := make([]byte, 0, n+n/3)
	for i, c := range []byte(s) {
		if i > 0 && (n-i)%3 == 0 {
			buf = append(buf, '.')
		}
		buf = append(buf, c)
	}
	return string(buf)
}

// splitEvery divide s en trozos de max n caracteres.
func splitEvery(s string, n int) []string {
	var parts []string
	for len(s) > n {
		parts = append(parts, s[:n])
		s = s[n:]
	}
	if s != "" {
		parts = append(parts, s)
	}
	return parts
}
