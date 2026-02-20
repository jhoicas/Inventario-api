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
