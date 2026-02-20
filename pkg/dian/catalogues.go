// Package dian contiene catálogos y validaciones alineados al Anexo Técnico
// de Factura Electrónica de Venta DIAN (Colombia) v1.9.
package dian

// =============================================================================
// Tabla 17 - Tipos de Responsabilidad Fiscal (Anexo 1.9 - 13.2.7.1)
// Códigos que identifican las obligaciones tributarias del contribuyente en el RUT.
// En el anexo figuran como "0-XX"; en sistemas se usa también "O-XX" (letra O).
// =============================================================================

const (
	TaxLevelGranContribuyente     = "O-13"  // Gran contribuyente
	TaxLevelAutorretenedor        = "O-15"  // Autorretenedor
	TaxLevelAgenteRetencionIVA    = "O-23"  // Agente de retención en el impuesto sobre las ventas
	TaxLevelRégimenSimple         = "O-47"  // Régimen Simple de Tributación – SIMPLE
	TaxLevelResponsableIVA        = "O-48"  // Responsable de IVA (Impuesto sobre las Ventas)
	TaxLevelNoResponsableIVA      = "O-49"  // No responsable de IVA
	TaxLevelNoAplicaOtros         = "R-99-PN" // No Aplica - Otros
)

// ValidFiscalResponsibilityCodes contiene los códigos de responsabilidad fiscal válidos (DIAN).
var ValidFiscalResponsibilityCodes = map[string]bool{
	TaxLevelGranContribuyente:   true,
	TaxLevelAutorretenedor:      true,
	TaxLevelAgenteRetencionIVA:   true,
	TaxLevelRégimenSimple:       true,
	TaxLevelResponsableIVA:      true,
	TaxLevelNoResponsableIVA:    true,
	TaxLevelNoAplicaOtros:       true,
	"0-13": true, "0-15": true, "0-23": true, "0-47": true, // formato con cero
}

// =============================================================================
// Tabla 6 - Unidades de Medida (Anexo 1.9 - 13.3.6 Unidades de Cantidad @unitCode)
// Códigos ISO/UNECE usados en líneas de factura (cantidad, base unit measure).
// =============================================================================

const (
	UnitUnit       = "94"  // Unidad
	UnitKilogram   = "KGM"  // Kilogramo
	UnitGram       = "GRM"  // Gramo
	UnitLitre      = "LTR"  // Litro
	UnitMetre      = "MTR"  // Metro
	UnitSquareMetre = "MTK" // Metro cuadrado
	UnitCubicMetre = "MTQ" // Metro cúbico
	UnitDozen      = "DZN"  // Docena
	UnitHour       = "HUR"  // Hora
	UnitDay        = "DAY"  // Día
)

// ValidMeasurementUnitCodes códigos de unidad de medida válidos (uso común en facturación).
var ValidMeasurementUnitCodes = map[string]bool{
	UnitUnit: true, UnitKilogram: true, UnitGram: true, UnitLitre: true,
	UnitMetre: true, UnitSquareMetre: true, UnitCubicMetre: true,
	UnitDozen: true, UnitHour: true, UnitDay: true,
}

// =============================================================================
// Tabla 14 - Forma de Pago (Anexo 1.9 - 13.3.4.1)
// =============================================================================

const (
	PaymentFormContado  = "1" // Contado
	PaymentFormCredito  = "2" // Crédito
)

// =============================================================================
// Tabla 13 - Medios de Pago (Anexo 1.9 - 13.3.4.2) - códigos de uso frecuente
// =============================================================================

const (
	PaymentMethodEfectivo           = "10" // Efectivo
	PaymentMethodTransferencia     = "47" // Transferencia Débito Bancaria
	PaymentMethodTarjetaCredito    = "48" // Tarjeta Crédito
	PaymentMethodTarjetaDebito     = "49" // Tarjeta Débito
)

// =============================================================================
// Tabla 11 - Tipos de Impuesto (Anexo 1.9 - 13.2.2)
// =============================================================================

const (
	TaxCodeIVA     = "01" // IVA
	TaxCodeINC     = "04" // Impuesto Nacional al Consumo
	TaxCodeReteIVA = "05" // Retención sobre el IVA
)

// =============================================================================
// Tabla 3 - Tipos de identificación (Anexo 1.9 - 13.2.1)
// =============================================================================

const (
	IdentificationTypeNIT = "31" // NIT - requiere dígito de verificación
	IdentificationTypeCC = "13" // Cédula de ciudadanía
)
