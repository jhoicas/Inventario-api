package entity

import (
	"time"

	"github.com/shopspring/decimal"
)

// Estados de envío a la DIAN (Colombia).
const (
	DIANStatusDraft            = "DRAFT"             // Guardada inicialmente para reservar ID y consecutivo
	DIANStatusPending          = "Pending"
	DIANStatusSigned           = "SIGNED"            // XML firmado, listo para enviar
	DIANStatusSent             = "Sent"
	DIANStatusError            = "Error"
	DIANStatusErrorGeneration  = "ERROR_GENERATION" // Falló firma o generación XML
)

// Invoice representa la cabecera de una factura.
type Invoice struct {
	ID          string
	CompanyID   string
	CustomerID  string
	Prefix      string
	Number      string
	Date        time.Time
	NetTotal    decimal.Decimal
	TaxTotal    decimal.Decimal
	GrandTotal  decimal.Decimal
	DIAN_Status string
	CUFE        string // Código Único de Factura Electrónica (SHA-384)
	UUID        string // Mismo valor que CUFE; se envía en el nodo <cbc:UUID> del XML DIAN
	XMLSigned   string // XML firmado (contenido); alternativamente usar XMLURL si se guarda en S3/disco
	QRData      string // String para QR (NumFac|FecFac|...|Cufe|UrlValidacionDIAN)
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
