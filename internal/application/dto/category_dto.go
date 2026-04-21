package dto

import "time"

// CreateProductCategoryRequest crea una categoría en crm_category_product_hub.
// Si name viene vacío pero code no, se usa code como nombre (compatibilidad con clientes antiguos).
type CreateProductCategoryRequest struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

// UpdateProductCategoryRequest actualiza el nombre en el hub.
type UpdateProductCategoryRequest struct {
	Name     *string `json:"name"`
	IsActive *bool   `json:"is_active"`
}

// ProductCategoryResponse salida HTTP (crm_category_product_hub).
type ProductCategoryResponse struct {
	ID        string    `json:"id"`
	CompanyID string    `json:"company_id"`
	Name      string    `json:"name"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ProductCategoryListResponse lista paginada.
type ProductCategoryListResponse struct {
	Items  []ProductCategoryResponse `json:"items"`
	Total  int                       `json:"total"`
	Limit  int                       `json:"limit"`
	Offset int                       `json:"offset"`
}
