package dto

import (
	"time"

	"github.com/shopspring/decimal"
)

// --- Categorías hub (crm_category_product_hub) ---

// CrmHubCategoryCreateRequest body POST /api/crm/categories-hub
type CrmHubCategoryCreateRequest struct {
	Name string `json:"name"`
	Code string `json:"code"` // opcional; si name vacío se usa code (compat)
}

// CrmHubCategoryUpdateRequest body PUT
type CrmHubCategoryUpdateRequest struct {
	Name     *string `json:"name"`
	IsActive *bool   `json:"is_active"`
}

// CrmHubCategoryResponse respuesta categoría hub.
type CrmHubCategoryResponse struct {
	ID        string    `json:"id"`
	CompanyID string    `json:"company_id"`
	Name      string    `json:"name"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CrmHubCategoryListResponse lista paginada.
type CrmHubCategoryListResponse struct {
	Items  []CrmHubCategoryResponse `json:"items"`
	Total  int                      `json:"total"`
	Limit  int                      `json:"limit"`
	Offset int                      `json:"offset"`
}

// --- Productos hub (crm_products_hub) ---

// CrmHubProductCreateRequest body POST
type CrmHubProductCreateRequest struct {
	CategoryID  string           `json:"category_id"`
	ProductCode string           `json:"product_code"`
	ProductName string           `json:"product_name"`
	UnitCost    *decimal.Decimal `json:"unit_cost"`
}

// CrmHubProductUpdateRequest body PUT
type CrmHubProductUpdateRequest struct {
	CategoryID  *string          `json:"category_id"`
	ProductCode *string          `json:"product_code"`
	ProductName *string          `json:"product_name"`
	UnitCost    *decimal.Decimal `json:"unit_cost"`
	IsActive    *bool            `json:"is_active"`
}

// CrmHubProductResponse respuesta producto hub.
type CrmHubProductResponse struct {
	ID          string           `json:"id"`
	CompanyID   string           `json:"company_id"`
	CategoryID  string           `json:"category_id"`
	ProductCode string           `json:"product_code"`
	ProductName string           `json:"product_name"`
	UnitCost    *decimal.Decimal `json:"unit_cost"`
	IsActive    bool             `json:"is_active"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

// CrmHubProductListResponse lista paginada.
type CrmHubProductListResponse struct {
	Items  []CrmHubProductResponse `json:"items"`
	Total  int                     `json:"total"`
	Limit  int                     `json:"limit"`
	Offset int                     `json:"offset"`
}
