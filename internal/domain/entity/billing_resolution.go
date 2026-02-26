package entity

import "time"

// BillingResolution representa la resolución de facturación autorizada por la DIAN.
// Es obligatoria en el nodo <sts:DianExtensions> del XML UBL 2.1.
// Cada empresa puede tener una o varias resoluciones; solo una activa por prefijo.
type BillingResolution struct {
	ID               string
	CompanyID        string
	ResolutionNumber string    // Número de resolución (ej: "18764000000001")
	Prefix           string    // Prefijo autorizado (ej: "SETP", "FE")
	RangeFrom        int64     // Número inicial del rango autorizado
	RangeTo          int64     // Número final del rango autorizado
	DateFrom         time.Time // Fecha de inicio de vigencia
	DateTo           time.Time // Fecha de vencimiento
	IsActive         bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
}
