package entity

import "time"

// CrmCategoryProductHub representa una fila en crm_category_product_hub.
type CrmCategoryProductHub struct {
	ID        string    `json:"id"`
	CompanyID string    `json:"company_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Category alias de compatibilidad para el catálogo inventario (/api/categories) que usa la misma tabla hub.
type Category = CrmCategoryProductHub
