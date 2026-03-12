package dto

import "time"

// CreateCompanyRequest entrada para crear una empresa.
type CreateCompanyRequest struct {
	Name    string `json:"name" validate:"required,min=1,max=200"`
	NIT     string `json:"nit" validate:"required,min=1,max=20"`
	Address string `json:"address"`
	Phone   string `json:"phone"`
	Email   string `json:"email" validate:"omitempty,email"`
}

// UpdateCompanyRequest entrada para actualizar una empresa (campos opcionales).
type UpdateCompanyRequest struct {
	Name    *string `json:"name" validate:"omitempty,min=1,max=200"`
	Address *string `json:"address"`
	Phone   *string `json:"phone"`
	Email   *string `json:"email" validate:"omitempty,email"`
	Status  *string `json:"status" validate:"omitempty,oneof=active suspended inactive"`
}

// CompanyResponse salida de una empresa (sin datos sensibles).
type CompanyResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	NIT       string    `json:"nit"`
	Address   string    `json:"address"`
	Phone     string    `json:"phone"`
	Email     string    `json:"email"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CompanyListResponse lista paginada de empresas.
type CompanyListResponse struct {
	Items []CompanyResponse `json:"items"`
	Page  PageResponse      `json:"page"`
}

// CreateResolutionRequest entrada para crear resolución DIAN por empresa.
type CreateResolutionRequest struct {
	Prefix           string `json:"prefix"`
	ResolutionNumber string `json:"resolution_number"`
	FromNumber       int64  `json:"from_number"`
	ToNumber         int64  `json:"to_number"`
	ValidFrom        string `json:"valid_from"`  // formato YYYY-MM-DD
	ValidUntil       string `json:"valid_until"` // formato YYYY-MM-DD
	Environment      string `json:"environment"` // test|prod
}

// ResolutionResponse salida de resolución con alerta de umbral.
type ResolutionResponse struct {
	ID               string    `json:"id"`
	CompanyID        string    `json:"company_id"`
	Prefix           string    `json:"prefix"`
	ResolutionNumber string    `json:"resolution_number"`
	FromNumber       int64     `json:"from_number"`
	ToNumber         int64     `json:"to_number"`
	ValidFrom        time.Time `json:"valid_from"`
	ValidUntil       time.Time `json:"valid_until"`
	Environment      string    `json:"environment"`
	AlertThreshold   bool      `json:"alert_threshold"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}
