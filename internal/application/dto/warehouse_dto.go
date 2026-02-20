package dto

import "time"

// CreateWarehouseRequest entrada para crear una bodega.
type CreateWarehouseRequest struct {
	Name    string `json:"name" validate:"required,min=1,max=200"`
	Address string `json:"address"`
}

// UpdateWarehouseRequest entrada para actualizar una bodega.
type UpdateWarehouseRequest struct {
	Name    *string `json:"name" validate:"omitempty,min=1,max=200"`
	Address *string `json:"address"`
}

// WarehouseResponse salida de una bodega.
type WarehouseResponse struct {
	ID        string    `json:"id"`
	CompanyID string    `json:"company_id"`
	Name      string    `json:"name"`
	Address   string    `json:"address"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// WarehouseListResponse lista paginada de bodegas.
type WarehouseListResponse struct {
	Items []WarehouseResponse `json:"items"`
	Page  PageResponse        `json:"page"`
}
