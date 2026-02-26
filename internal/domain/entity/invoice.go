package entity

import (
	"time"

	"github.com/shopspring/decimal"
)

// Estados de envío a la DIAN (Colombia).
const (
	DIANStatusDraft           = "DRAFT"            // Guardada para reservar ID y consecutivo
	DIANStatusPending         = "Pending"           // En proceso
	DIANStatusSigned          = "SIGNED"            // XML firmado, pendiente de envío al WS
	DIANStatusSent            = "Sent"              // Enviada al WS DIAN, respuesta pendiente
	DIANStatusExitoso         = "EXITOSO"           // Aceptada por la DIAN (o simulada en dev)
	DIANStatusRechazado       = "RECHAZADO"         // Rechazada por la DIAN con errores
	DIANStatusError           = "Error"             // Error genérico
	DIANStatusErrorGeneration = "ERROR_GENERATION"  // Falló firma o generación XML
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
	UUID        string // Mismo valor que CUFE; en <cbc:UUID> del XML DIAN
	XMLSigned   string // XML firmado (contenido completo)
	QRData      string // String para QR (NumFac|FecFac|...|Cufe|UrlValidacionDIAN)
	TrackID     string // ZipKey / TrackID devuelto por el WS DIAN tras el envío
	DIANErrors  string // Mensajes de rechazo devueltos por la DIAN (JSON o texto plano)
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
