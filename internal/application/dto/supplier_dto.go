package dto

import "time"

// CreateSupplierRequest body para POST /api/suppliers.
type CreateSupplierRequest struct {
	Name            string `json:"name"`
	NIT             string `json:"nit"`
	Email           string `json:"email"`
	Phone           string `json:"phone"`
	PaymentTermDays int    `json:"payment_term_days"`
	LeadTimeDays    int    `json:"lead_time_days"`
}

// UpdateSupplierRequest body para PUT /api/suppliers/{id}.
type UpdateSupplierRequest struct {
	Name            *string `json:"name,omitempty"`
	NIT             *string `json:"nit,omitempty"`
	Email           *string `json:"email,omitempty"`
	Phone           *string `json:"phone,omitempty"`
	PaymentTermDays *int    `json:"payment_term_days,omitempty"`
	LeadTimeDays    *int    `json:"lead_time_days,omitempty"`
}

// SupplierFilters filtros para listado de proveedores.
type SupplierFilters struct {
	Search string `query:"search"`
	Limit  int    `query:"limit"`
	Offset int    `query:"offset"`
}

// SupplierResponse DTO de proveedor en respuestas HTTP.
type SupplierResponse struct {
	ID              string    `json:"id"`
	CompanyID       string    `json:"company_id"`
	Name            string    `json:"name"`
	NIT             string    `json:"nit"`
	Email           string    `json:"email"`
	Phone           string    `json:"phone"`
	PaymentTermDays int       `json:"payment_term_days"`
	LeadTimeDays    int       `json:"lead_time_days"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// SupplierListResponse respuesta paginada para listado de proveedores.
type SupplierListResponse struct {
	Items []SupplierResponse `json:"items"`
	Page  PageResponse       `json:"page"`
}
