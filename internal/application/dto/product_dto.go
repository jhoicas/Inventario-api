package dto

import (
	"encoding/json"
	"time"

	"github.com/shopspring/decimal"
)

// CreateProductRequest entrada para crear un producto.
type CreateProductRequest struct {
	SKU         string          `json:"sku" validate:"required,min=1,max=100"`
	Name        string          `json:"name" validate:"required,min=1,max=200"`
	Description string          `json:"description"`
	Price       decimal.Decimal `json:"price"`
	TaxRate     decimal.Decimal `json:"tax_rate"`
	UNSPSC_Code string          `json:"unspsc_code"`
	UnitMeasure string          `json:"unit_measure" validate:"required"`
	Attributes  json.RawMessage `json:"attributes"`
}

// UpdateProductRequest entrada para actualizar un producto (sin Cost ni Stock).
type UpdateProductRequest struct {
	Name        *string          `json:"name" validate:"omitempty,min=1,max=200"`
	Description *string          `json:"description"`
	Price       *decimal.Decimal `json:"price"`
	TaxRate     *decimal.Decimal `json:"tax_rate"`
	UNSPSC_Code *string         `json:"unspsc_code"`
	UnitMeasure *string         `json:"unit_measure"`
	Attributes  json.RawMessage `json:"attributes"`
}

// ProductResponse salida de un producto.
type ProductResponse struct {
	ID          string          `json:"id"`
	CompanyID   string          `json:"company_id"`
	SKU         string          `json:"sku"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Price       decimal.Decimal `json:"price"`
	Cost        decimal.Decimal `json:"cost"`
	TaxRate     decimal.Decimal `json:"tax_rate"`
	UNSPSC_Code string          `json:"unspsc_code"`
	UnitMeasure string          `json:"unit_measure"`
	Attributes  json.RawMessage `json:"attributes"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// ProductListResponse lista paginada de productos.
type ProductListResponse struct {
	Items []ProductResponse `json:"items"`
	Page  PageResponse      `json:"page"`
}

// ── Clasificación arancelaria por IA ─────────────────────────────────────────

// AIClassificationRequest cuerpo de POST /api/ai/suggest-classification.
type AIClassificationRequest struct {
	ProductName string `json:"product_name"`
	Description string `json:"description"`
}

// AIClassificationDTO respuesta del servicio de IA con la clasificación sugerida.
// SuggestedTaxRate puede ser 0, 5 o 19 (tarifas de IVA Colombia).
// ConfidenceScore va de 0.0 a 1.0; valores ≥ 0.8 se consideran alta confianza.
type AIClassificationDTO struct {
	SuggestedUNSPSC string          `json:"suggested_unspsc"` // código UNSPSC de 8 dígitos
	SuggestedTaxRate decimal.Decimal `json:"suggested_tax_rate"` // 0, 5 o 19
	ConfidenceScore  float64         `json:"confidence_score"`   // 0.0 – 1.0
	Reasoning        string          `json:"reasoning"`          // explicación del modelo
}
